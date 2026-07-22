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

	"github.com/0disoft/fairyc/internal/moduleinterface"
)

const (
	DiskRecordSchema  = "fairyc.package-cache-record/v1"
	DiskRecordVersion = "fairyc-package-cache-record/1"
)

const diskLayoutDirectory = "v1"

// DiskLimits are mandatory caller policy. MaxEntries counts immutable generation files, MaxBytes
// counts their encoded bytes, MaxRecordBytes bounds every read and write, and TempMaxAge controls
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
	root     string
	limits   DiskLimits
	mu       sync.Mutex
	hits     uint64
	misses   uint64
	rejected uint64
	evicted  uint64
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

// NewDiskStore creates the versioned root and rejects every missing bound.
func NewDiskStore(root string, limits DiskLimits) (*DiskStore, error) {
	if root == "" {
		return nil, fmt.Errorf("package cache disk root must be nonempty")
	}
	if limits.MaxEntries <= 0 || limits.MaxBytes == 0 || limits.MaxRecordBytes == 0 || limits.TempMaxAge <= 0 {
		return nil, fmt.Errorf("package cache disk limits must all be positive")
	}
	if limits.MaxRecordBytes > limits.MaxBytes {
		return nil, fmt.Errorf("package cache MaxRecordBytes cannot exceed MaxBytes")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve package cache disk root: %w", err)
	}
	versionRoot := filepath.Join(absolute, diskLayoutDirectory)
	if err := os.MkdirAll(versionRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create package cache disk root: %w", err)
	}
	return &DiskStore{root: versionRoot, limits: limits}, nil
}

// Put validates and canonically encodes one record, publishes one immutable digest-named file, and
// runs bounded collection. An oversized record cannot disturb an existing generation.
func (store *DiskStore) Put(record Record) error {
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
	keyDirectory := store.keyDirectory(record.Key)
	if err := os.MkdirAll(keyDirectory, 0o700); err != nil {
		return fmt.Errorf("create package cache key directory: %w", err)
	}
	digest := sha256.Sum256(encoded)
	finalPath := filepath.Join(keyDirectory, hex.EncodeToString(digest[:])+".json")
	temporary, err := os.CreateTemp(keyDirectory, ".tmp-")
	if err != nil {
		return fmt.Errorf("create package cache temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryPath)
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
	if err := publishImmutableGeneration(temporaryPath, finalPath, encoded); err != nil {
		return err
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
	entries, err := os.ReadDir(store.keyDirectory(key))
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
		path := filepath.Join(store.keyDirectory(key), entry.Name())
		record, readErr := store.readGeneration(path, entry.Name(), key)
		if readErr != nil {
			store.recordRejected()
			_ = os.Remove(path)
			continue
		}
		valid = append(valid, record)
	}
	if len(valid) != 1 {
		if len(valid) > 1 {
			store.recordRejectedBy(uint64(len(valid)))
			for _, entry := range entries {
				if validGenerationName(entry.Name()) {
					_ = os.Remove(filepath.Join(store.keyDirectory(key), entry.Name()))
				}
			}
		}
		store.recordMiss()
		return Record{}, false
	}
	store.recordHit()
	return cloneRecord(valid[0]), true
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
		if err := os.Remove(oldest.path); err != nil && !errors.Is(err, os.ErrNotExist) {
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

func (store *DiskStore) scan(clean bool) ([]diskGeneration, error) {
	keyDirectories, err := os.ReadDir(store.root)
	if err != nil {
		return nil, fmt.Errorf("scan package cache root: %w", err)
	}
	now := time.Now()
	result := make([]diskGeneration, 0)
	for _, keyDirectory := range keyDirectories {
		if !keyDirectory.IsDir() || !validDigestDirectory(keyDirectory.Name()) {
			continue
		}
		directoryPath := filepath.Join(store.root, keyDirectory.Name())
		entries, readErr := os.ReadDir(directoryPath)
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(directoryPath, entry.Name())
			info, infoErr := entry.Info()
			if infoErr != nil {
				continue
			}
			if strings.HasPrefix(entry.Name(), ".tmp-") {
				if clean && now.Sub(info.ModTime()) >= store.limits.TempMaxAge {
					_ = os.Remove(path)
				}
				continue
			}
			if !validGenerationName(entry.Name()) {
				continue
			}
			if info.Size() <= 0 || uint64(info.Size()) > store.limits.MaxRecordBytes {
				if clean {
					_ = os.Remove(path)
					store.rejected++
				}
				continue
			}
			result = append(result, diskGeneration{path: path, name: entry.Name(), size: uint64(info.Size()), modTime: info.ModTime()})
		}
	}
	return result, nil
}

func (store *DiskStore) readGeneration(path, name string, expected Key) (Record, error) {
	file, err := os.Open(path)
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
	return filepath.Join(store.root, hex.EncodeToString(key[:]))
}

func publishImmutableGeneration(temporaryPath, finalPath string, expected []byte) error {
	if err := os.Rename(temporaryPath, finalPath); err == nil {
		return nil
	}
	existing, readErr := os.ReadFile(finalPath)
	if readErr == nil && bytes.Equal(existing, expected) {
		return nil
	}
	// A digest-named valid destination can contain only expected. Replacing any other bytes is safe:
	// a racing valid writer has the same payload, and losing cache state is never a build failure.
	if removeErr := os.Remove(finalPath); removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
		if renameErr := os.Rename(temporaryPath, finalPath); renameErr == nil {
			return nil
		}
		existing, readErr = os.ReadFile(finalPath)
		if readErr == nil && bytes.Equal(existing, expected) {
			return nil
		}
	}
	return fmt.Errorf("publish immutable package cache generation: destination conflict")
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
