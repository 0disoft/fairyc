package packagecache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/0disoft/fairyc/internal/durablefile"
	"github.com/0disoft/fairyc/internal/moduleinterface"
)

const (
	DiskRecordSchema  = "fairyc.package-cache-record/v1"
	DiskRecordVersion = "fairyc-package-cache-record/1"
)

const diskLayoutDirectory = "v1"

// DiskLimits are mandatory caller policy. MaxEntries counts immutable generations across all
// supported namespaces, MaxBytes counts their combined encoded bytes, MaxRecordBytes bounds each
// store's reads and writes, and TempMaxAge controls
// recovery of abandoned unpublished files without guessing a universal cleanup interval.
type DiskLimits struct {
	MaxEntries     int
	MaxBytes       uint64
	MaxRecordBytes uint64
	TempMaxAge     time.Duration
}

// DiskStats combines a current retained-state scan with process-local operation counters.
type DiskStats struct {
	Entries   int
	Bytes     uint64
	Hits      uint64
	Misses    uint64
	Rejected  uint64
	Evictions uint64
}

// DiskStore persists immutable content-addressed record generations. It uses no lock file and no
// mutable index; concurrent processes can only publish identical names or independent generations.
type DiskStore struct {
	root          *os.Root
	retentionRoot *os.Root
	rootName      string
	limits        DiskLimits
	mu            sync.Mutex
	hits          uint64
	misses        uint64
	rejected      uint64
	evicted       uint64
}

type diskArtifact struct {
	Kind string `json:"kind"`
	Data []byte `json:"data"`
}

type diskRecord struct {
	Key             Key                  `json:"key"`
	PublicInterface moduleinterface.Hash `json:"publicInterface"`
	Artifacts       []diskArtifact       `json:"artifacts"`
}

type diskEnvelope struct {
	Schema  string     `json:"schema"`
	Version string     `json:"version"`
	Record  diskRecord `json:"record"`
}

type diskGeneration struct {
	path    string
	name    string
	size    uint64
	modTime time.Time
}

type diskPutHooks struct {
	afterTemporaryClosed func(string)
	afterPublished       func(string)
}

// NewDiskStore creates the versioned root and rejects every missing bound.
func NewDiskStore(root string, limits DiskLimits) (*DiskStore, error) {
	return NewNamespacedDiskStore(root, ".", limits)
}

// NewNamespacedDiskStore isolates records while sharing retention bounds with
// the ordinary, generic-v0 and storage-transfer-v0 stores beneath root.
func NewNamespacedDiskStore(root, namespace string, limits DiskLimits) (*DiskStore, error) {
	if err := validateDiskStoreInputs(root, limits); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve package cache disk root: %w", err)
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("create package cache disk root: %w", err)
	}
	boundary, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, fmt.Errorf("open package cache disk root: %w", err)
	}
	defer boundary.Close()
	return NewNamespacedDiskStoreInRoot(boundary, ".", namespace, limits)
}

// NewDiskStoreInRoot creates the versioned cache beneath an already-open filesystem boundary.
// Every later operation stays relative to the returned opened root, so a symlink, junction, or
// rename race cannot redirect cache I/O outside the caller's boundary.
func NewDiskStoreInRoot(boundary *os.Root, root string, limits DiskLimits) (*DiskStore, error) {
	return NewNamespacedDiskStoreInRoot(boundary, root, ".", limits)
}

// NewNamespacedDiskStoreInRoot retains the caller's filesystem boundary for
// both namespace-local access and collection across the supported namespaces.
func NewNamespacedDiskStoreInRoot(boundary *os.Root, root, namespace string, limits DiskLimits) (*DiskStore, error) {
	if boundary == nil {
		return nil, fmt.Errorf("package cache filesystem boundary must be initialized")
	}
	if err := validateDiskStoreInputs(root, limits); err != nil {
		return nil, err
	}
	if namespace != "." && namespace != "generic-v0" && namespace != "storage-transfer-v0" {
		return nil, fmt.Errorf("unsupported package cache namespace %q", namespace)
	}
	for _, directory := range []string{filepath.Join(root, namespace), filepath.Join(root, namespace, diskLayoutDirectory)} {
		info, err := boundary.Lstat(directory)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return nil, fmt.Errorf("package cache namespace must be a real directory")
		}
	}
	versionRoot := filepath.Join(root, namespace, diskLayoutDirectory)
	if err := boundary.MkdirAll(versionRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create package cache disk root in boundary: %w", err)
	}
	opened, err := boundary.OpenRoot(versionRoot)
	if err != nil {
		return nil, fmt.Errorf("open package cache disk root in boundary: %w", err)
	}
	retentionRoot, err := boundary.OpenRoot(root)
	if err != nil {
		opened.Close()
		return nil, fmt.Errorf("open shared cache retention root: %w", err)
	}
	store := &DiskStore{root: opened, retentionRoot: retentionRoot, rootName: opened.Name(), limits: limits}
	if err := store.Sweep(); err != nil {
		store.Close()
		return nil, fmt.Errorf("collect retained cache on open: %w", err)
	}
	return store, nil
}

func validateDiskStoreInputs(root string, limits DiskLimits) error {
	if root == "" {
		return fmt.Errorf("package cache disk root must be nonempty")
	}
	if limits.MaxEntries <= 0 || limits.MaxBytes == 0 || limits.MaxRecordBytes == 0 || limits.TempMaxAge <= 0 {
		return fmt.Errorf("package cache disk limits must all be positive")
	}
	if limits.MaxRecordBytes > limits.MaxBytes {
		return fmt.Errorf("package cache MaxRecordBytes cannot exceed MaxBytes")
	}
	return nil
}

// Close releases the opened cache-directory handle. Close must not race another store operation.
func (store *DiskStore) Close() error {
	if store == nil || store.root == nil {
		return nil
	}
	return errors.Join(store.root.Close(), store.retentionRoot.Close())
}

// Put validates and canonically encodes one record, durably publishes one immutable digest-named
// file through the host contract, and runs bounded collection. An oversized record cannot disturb
// an existing generation. Durability failure is a cache-store failure even if a verifiable file
// became visible; callers must never treat cache retention as source truth.
func (store *DiskStore) Put(record Record) error {
	return store.put(record, diskPutHooks{})
}

func (store *DiskStore) put(record Record, hooks diskPutHooks) error {
	if store == nil {
		return fmt.Errorf("package cache disk store is not initialized")
	}
	encoded, err := encodeDiskRecord(record, store.limits.MaxRecordBytes)
	if err != nil {
		return err
	}
	if uint64(len(encoded)) > store.limits.MaxBytes {
		return ErrRecordTooLarge
	}
	keyDirectory := store.keyDirectoryName(record.Key)
	createdKeyDirectory := false
	if err := store.root.Mkdir(keyDirectory, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create package cache key directory: %w", err)
		}
		info, statErr := store.root.Lstat(keyDirectory)
		if statErr != nil {
			return fmt.Errorf("inspect package cache key directory: %w", statErr)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("package cache key path is not a real directory")
		}
	} else {
		createdKeyDirectory = true
	}
	if createdKeyDirectory {
		if err := durablefile.SyncDirectory(store.root, "."); err != nil {
			return fmt.Errorf("sync package cache key-directory parent: %w", err)
		}
	}
	digest := sha256.Sum256(encoded)
	finalPath := filepath.Join(keyDirectory, hex.EncodeToString(digest[:])+".json")
	temporary, temporaryPath, err := durablefile.CreateTemporary(store.root, keyDirectory)
	if err != nil {
		return fmt.Errorf("create package cache temporary file: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		_ = store.root.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("protect package cache temporary file: %w", err)
	}
	if _, err := temporary.Write(encoded); err != nil {
		return fmt.Errorf("write package cache temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync package cache temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close package cache temporary file: %w", err)
	}
	closed = true
	if hooks.afterTemporaryClosed != nil {
		hooks.afterTemporaryClosed(store.absolutePath(temporaryPath))
	}
	if err := durablefile.PublishImmutable(store.root, temporaryPath, finalPath, encoded); err != nil {
		return err
	}
	if hooks.afterPublished != nil {
		hooks.afterPublished(store.absolutePath(finalPath))
	}
	if err := store.Sweep(); err != nil {
		return fmt.Errorf("collect package cache after put: %w", err)
	}
	return nil
}

// Get returns the sole valid immutable generation for key. Corrupt files are removed best-effort;
// multiple distinct valid generations are a conflict and therefore a safe miss.
func (store *DiskStore) Get(key Key) (Record, bool) {
	if store == nil || isZero(key[:]) {
		return Record{}, false
	}
	keyDirectory := store.keyDirectoryName(key)
	entries, err := readCacheDirectory(store.root, keyDirectory)
	if err != nil {
		store.recordMiss()
		return Record{}, false
	}
	slices.SortFunc(entries, func(left, right os.DirEntry) int { return strings.Compare(left.Name(), right.Name()) })
	valid := make([]Record, 0, 1)
	for _, entry := range entries {
		if entry.IsDir() || !validGenerationName(entry.Name()) {
			continue
		}
		path := filepath.Join(keyDirectory, entry.Name())
		record, readErr := store.readGeneration(path, entry.Name(), key)
		if readErr != nil {
			store.recordRejected()
			_ = store.root.Remove(path)
			continue
		}
		valid = append(valid, record)
	}
	if len(valid) != 1 {
		if len(valid) > 1 {
			store.recordRejectedBy(uint64(len(valid)))
			for _, entry := range entries {
				if validGenerationName(entry.Name()) {
					_ = store.root.Remove(filepath.Join(keyDirectory, entry.Name()))
				}
			}
		}
		store.recordMiss()
		return Record{}, false
	}
	store.recordHit()
	return cloneRecord(valid[0]), true
}

// DeleteIfMatch removes only the immutable generation whose canonical bytes exactly match
// expected. A cache consumer can therefore quarantine a semantically rejected payload without
// deleting another generation that a concurrent producer published for the same key.
func (store *DiskStore) DeleteIfMatch(expected Record) bool {
	if store == nil {
		return false
	}
	encoded, err := encodeDiskRecord(expected, store.limits.MaxRecordBytes)
	if err != nil {
		return false
	}
	digest := sha256.Sum256(encoded)
	path := filepath.Join(store.keyDirectoryName(expected.Key), hex.EncodeToString(digest[:])+".json")
	file, err := store.root.Open(path)
	if err != nil {
		return false
	}
	actual, readErr := io.ReadAll(io.LimitReader(file, int64(store.limits.MaxRecordBytes)+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(actual, encoded) {
		return false
	}
	return store.root.Remove(path) == nil
}

// Sweep removes expired temporary files and oldest immutable generations until both caller bounds
// hold. Creation/modification time is only an eviction hint; cache correctness never depends on it.
func (store *DiskStore) Sweep() error {
	if store == nil {
		return fmt.Errorf("package cache disk store is not initialized")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	generations, err := store.scan(true)
	if err != nil {
		return err
	}
	slices.SortFunc(generations, func(left, right diskGeneration) int {
		if byTime := left.modTime.Compare(right.modTime); byTime != 0 {
			return byTime
		}
		return strings.Compare(left.path, right.path)
	})
	var total uint64
	for _, generation := range generations {
		if ^uint64(0)-total < generation.size {
			total = ^uint64(0)
		} else {
			total += generation.size
		}
	}
	for len(generations) > store.limits.MaxEntries || total > store.limits.MaxBytes {
		oldest := generations[0]
		generations = generations[1:]
		if err := store.retentionRoot.Remove(oldest.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("evict package cache generation: %w", err)
		}
		if total >= oldest.size {
			total -= oldest.size
		} else {
			total = 0
		}
		store.evicted++
	}
	return nil
}

// Stats scans retained state so other-process writes are visible, then adds local counters.
func (store *DiskStore) Stats() (DiskStats, error) {
	if store == nil {
		return DiskStats{}, fmt.Errorf("package cache disk store is not initialized")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	generations, err := store.scan(false)
	if err != nil {
		return DiskStats{}, err
	}
	stats := DiskStats{Entries: len(generations), Hits: store.hits, Misses: store.misses, Rejected: store.rejected, Evictions: store.evicted}
	for _, generation := range generations {
		if ^uint64(0)-stats.Bytes < generation.size {
			stats.Bytes = ^uint64(0)
		} else {
			stats.Bytes += generation.size
		}
	}
	return stats, nil
}

func (store *DiskStore) readGeneration(path, name string, expected Key) (Record, error) {
	file, err := store.root.Open(path)
	if err != nil {
		return Record{}, err
	}
	defer file.Close()
	encoded, err := io.ReadAll(io.LimitReader(file, int64(store.limits.MaxRecordBytes)+1))
	if err != nil {
		return Record{}, err
	}
	if uint64(len(encoded)) > store.limits.MaxRecordBytes {
		return Record{}, ErrRecordTooLarge
	}
	digest := sha256.Sum256(encoded)
	if hex.EncodeToString(digest[:])+".json" != name {
		return Record{}, fmt.Errorf("package cache generation digest does not match its name")
	}
	record, err := decodeDiskRecord(encoded, store.limits.MaxRecordBytes)
	if err != nil {
		return Record{}, err
	}
	if record.Key != expected {
		return Record{}, fmt.Errorf("package cache generation key does not match its directory")
	}
	return record, nil
}

func (store *DiskStore) keyDirectory(key Key) string {
	return store.absolutePath(store.keyDirectoryName(key))
}

func (store *DiskStore) keyDirectoryName(key Key) string {
	return hex.EncodeToString(key[:])
}

func (store *DiskStore) absolutePath(relative string) string {
	return filepath.Join(store.rootName, relative)
}

func readCacheDirectory(root *os.Root, name string) ([]os.DirEntry, error) {
	directory, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	entries, readErr := directory.ReadDir(-1)
	closeErr := directory.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return entries, nil
}

func encodeDiskRecord(record Record, maxBytes uint64) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("package cache disk record byte limit must be positive")
	}
	if _, err := validateRecord(record); err != nil {
		return nil, err
	}
	wire := diskRecord{Key: record.Key, PublicInterface: record.PublicInterface}
	for _, artifact := range record.Artifacts {
		wire.Artifacts = append(wire.Artifacts, diskArtifact{Kind: artifact.Kind, Data: slices.Clone(artifact.Data)})
	}
	encoded, err := json.Marshal(diskEnvelope{Schema: DiskRecordSchema, Version: DiskRecordVersion, Record: wire})
	if err != nil {
		return nil, fmt.Errorf("encode package cache disk record: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrRecordTooLarge
	}
	return encoded, nil
}

func decodeDiskRecord(encoded []byte, maxBytes uint64) (Record, error) {
	if maxBytes == 0 || len(encoded) == 0 || uint64(len(encoded)) > maxBytes {
		return Record{}, ErrRecordTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope diskEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return Record{}, fmt.Errorf("decode package cache disk record: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Record{}, fmt.Errorf("decode package cache disk record trailing data")
	}
	if envelope.Schema != DiskRecordSchema || envelope.Version != DiskRecordVersion {
		return Record{}, fmt.Errorf("package cache disk record schema or version is incompatible")
	}
	record := Record{Key: envelope.Record.Key, PublicInterface: envelope.Record.PublicInterface}
	for _, artifact := range envelope.Record.Artifacts {
		record.Artifacts = append(record.Artifacts, Artifact{Kind: artifact.Kind, Data: slices.Clone(artifact.Data)})
	}
	if _, err := validateRecord(record); err != nil {
		return Record{}, err
	}
	canonical, err := encodeDiskRecord(record, maxBytes)
	if err != nil {
		return Record{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return Record{}, fmt.Errorf("package cache disk record is not canonically encoded")
	}
	return record, nil
}

func validDigestDirectory(name string) bool {
	return len(name) == sha256.Size*2 && strings.ToLower(name) == name && validHex(name)
}

func validGenerationName(name string) bool {
	return len(name) == sha256.Size*2+len(".json") && strings.HasSuffix(name, ".json") && validDigestDirectory(strings.TrimSuffix(name, ".json"))
}

func validHex(value string) bool {
	_, err := hex.DecodeString(value)
	return err == nil
}

func (store *DiskStore) recordHit() {
	store.mu.Lock()
	store.hits++
	store.mu.Unlock()
}

func (store *DiskStore) recordMiss() {
	store.mu.Lock()
	store.misses++
	store.mu.Unlock()
}

func (store *DiskStore) recordRejected() {
	store.recordRejectedBy(1)
}

func (store *DiskStore) recordRejectedBy(count uint64) {
	store.mu.Lock()
	store.rejected += count
	store.mu.Unlock()
}
