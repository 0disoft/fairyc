package atlasmap

import (
	"encoding/hex"
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
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const mapStoreLayoutDirectory = "fymap-v1"
const mapStoreLayoutDirectoryV2 = "fymap-v2"
const mapStoreLayoutDirectoryV3 = "fymap-v3"
const maxReadableMapBytes = uint64(1<<63 - 2)

// MapStoreLimits are mandatory caller policy for retained immutable map generations.
type MapStoreLimits struct {
	MaxGenerations int
	MaxBytes       uint64
	MaxMapBytes    uint64
	TempMaxAge     time.Duration
}

type MapStoreStats struct {
	Generations int
	Bytes       uint64
	Hits        uint64
	Misses      uint64
	Rejected    uint64
	Evictions   uint64
}

// MapStore publishes content-addressed immutable maps under an opened filesystem boundary.
type MapStore struct {
	root     *os.Root
	rootName string
	limits   MapStoreLimits
	version  mapStoreVersion
	mu       sync.Mutex
	hits     uint64
	misses   uint64
	rejected uint64
	evicted  uint64
}

type mapGeneration struct {
	path    string
	size    uint64
	modTime time.Time
}

type mapPublishHooks struct {
	afterTemporaryClosed func(string)
	afterPublished       func(string)
}

func NewMapStore(root string, limits MapStoreLimits) (*MapStore, error) {
	return newMapStore(root, limits, mapStoreV1)
}

// NewMapStoreV2 opens an opt-in generic-aware store in a separate version directory.
func NewMapStoreV2(root string, limits MapStoreLimits) (*MapStore, error) {
	return newMapStore(root, limits, mapStoreV2)
}

func newMapStore(root string, limits MapStoreLimits, version mapStoreVersion) (*MapStore, error) {
	if _, err := version.layoutDirectory(); err != nil {
		return nil, err
	}
	if root == "" {
		return nil, fmt.Errorf(".fymap store root must be nonempty")
	}
	if err := validateMapStoreLimits(limits); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve .fymap store root: %w", err)
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("create .fymap store root: %w", err)
	}
	boundary, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, fmt.Errorf("open .fymap store root: %w", err)
	}
	defer boundary.Close()
	return newMapStoreInRoot(boundary, ".", limits, version)
}

// NewMapStoreInRoot binds all later map I/O beneath boundary and a canonical relative root.
func NewMapStoreInRoot(boundary *os.Root, root string, limits MapStoreLimits) (*MapStore, error) {
	return newMapStoreInRoot(boundary, root, limits, mapStoreV1)
}

// NewMapStoreV2InRoot keeps v2 publication and cleanup below the opened boundary.
func NewMapStoreV2InRoot(boundary *os.Root, root string, limits MapStoreLimits) (*MapStore, error) {
	return newMapStoreInRoot(boundary, root, limits, mapStoreV2)
}

func newMapStoreInRoot(boundary *os.Root, root string, limits MapStoreLimits, version mapStoreVersion) (*MapStore, error) {
	if boundary == nil {
		return nil, fmt.Errorf(".fymap filesystem boundary must be initialized")
	}
	if !validMapStoreRelativePath(root) {
		return nil, fmt.Errorf(".fymap store root must be a canonical relative path")
	}
	if err := validateMapStoreLimits(limits); err != nil {
		return nil, err
	}
	layout, err := version.layoutDirectory()
	if err != nil {
		return nil, err
	}
	versionRoot := filepath.Join(root, layout)
	if err := boundary.MkdirAll(versionRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create .fymap version root: %w", err)
	}
	before, err := boundary.Lstat(versionRoot)
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf(".fymap version root must be a real directory")
	}
	opened, err := boundary.OpenRoot(versionRoot)
	if err != nil {
		return nil, fmt.Errorf("open .fymap version root: %w", err)
	}
	after, err := opened.Stat(".")
	if err != nil || !os.SameFile(before, after) {
		_ = opened.Close()
		return nil, fmt.Errorf(".fymap version root changed while opening")
	}
	return &MapStore{root: opened, rootName: opened.Name(), limits: limits, version: version}, nil
}

func validateMapStoreLimits(limits MapStoreLimits) error {
	if limits.MaxGenerations <= 0 || limits.MaxBytes == 0 || limits.MaxMapBytes == 0 || limits.TempMaxAge <= 0 {
		return fmt.Errorf(".fymap store limits must all be positive")
	}
	if limits.MaxMapBytes > limits.MaxBytes {
		return fmt.Errorf(".fymap MaxMapBytes cannot exceed MaxBytes")
	}
	if limits.MaxMapBytes > maxReadableMapBytes {
		return fmt.Errorf(".fymap MaxMapBytes exceeds the bounded reader range")
	}
	return nil
}

func validMapStoreRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) || filepath.VolumeName(value) != "" || filepath.Clean(value) != value {
		return false
	}
	return value != ".." && !strings.HasPrefix(value, ".."+string(filepath.Separator))
}

func (store *MapStore) Close() error {
	if store == nil || store.root == nil {
		return nil
	}
	return store.root.Close()
}

// Publish encodes, syncs, and immutably publishes one exact package map by its logical checksum.
func (store *MapStore) Publish(partition Partition) (Checksum, error) {
	return store.publish(partition, mapPublishHooks{})
}

func (store *MapStore) publish(partition Partition, hooks mapPublishHooks) (Checksum, error) {
	if store == nil || store.root == nil {
		return Checksum{}, fmt.Errorf(".fymap store is not initialized")
	}
	schema, encoding, logical := store.transportVersion()
	encoded, err := encodeVersion(partition, store.limits.MaxMapBytes, schema, encoding, logical)
	if err != nil {
		return Checksum{}, err
	}
	checksum, err := computeVersionChecksum(documentFromPartition(partition), schema, encoding)
	if err != nil {
		return Checksum{}, err
	}
	header := partition.Header()
	instance, err := packagegraph.ParseInstanceID(header.PackageInstance)
	if err != nil || instance == (packagegraph.InstanceID{}) {
		return Checksum{}, fmt.Errorf("publish .fymap package instance is invalid")
	}
	directory := header.PackageInstance
	createdDirectory := false
	if err := store.root.Mkdir(directory, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return Checksum{}, fmt.Errorf("create .fymap package directory: %w", err)
		}
		info, statErr := store.root.Lstat(directory)
		if statErr != nil {
			return Checksum{}, fmt.Errorf("inspect .fymap package directory: %w", statErr)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return Checksum{}, fmt.Errorf(".fymap package path is not a real directory")
		}
	} else {
		createdDirectory = true
	}
	if createdDirectory {
		if err := durablefile.SyncDirectory(store.root, "."); err != nil {
			return Checksum{}, fmt.Errorf("sync .fymap package-directory parent: %w", err)
		}
	}
	temporary, temporaryPath, err := durablefile.CreateTemporary(store.root, directory)
	if err != nil {
		return Checksum{}, fmt.Errorf("create .fymap temporary: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		_ = store.root.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return Checksum{}, fmt.Errorf("protect .fymap temporary: %w", err)
	}
	if _, err := temporary.Write(encoded); err != nil {
		return Checksum{}, fmt.Errorf("write .fymap temporary: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return Checksum{}, fmt.Errorf("sync .fymap temporary: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return Checksum{}, fmt.Errorf("close .fymap temporary: %w", err)
	}
	closed = true
	if hooks.afterTemporaryClosed != nil {
		hooks.afterTemporaryClosed(filepath.Join(store.rootName, temporaryPath))
	}
	finalPath := filepath.Join(directory, mapGenerationName(checksum))
	if err := durablefile.PublishImmutable(store.root, temporaryPath, finalPath, encoded); err != nil {
		return Checksum{}, err
	}
	if hooks.afterPublished != nil {
		hooks.afterPublished(filepath.Join(store.rootName, finalPath))
	}
	if err := store.Sweep(); err != nil {
		return Checksum{}, fmt.Errorf("collect .fymap store after publication: %w", err)
	}
	return checksum, nil
}

// LoadCurrent returns exactly one compatible, current-source-validated generation.
func (store *MapStore) LoadCurrent(expectation LoadExpectation) (Partition, bool, error) {
	if store == nil || store.root == nil {
		return Partition{}, false, fmt.Errorf(".fymap store is not initialized")
	}
	if err := validateLoadExpectation(expectation); err != nil {
		return Partition{}, false, err
	}
	directory := expectation.Current.Instance.String()
	entries, err := readMapDirectory(store.root, directory)
	if errors.Is(err, os.ErrNotExist) {
		store.recordMiss()
		return Partition{}, false, nil
	}
	if err != nil {
		return Partition{}, false, fmt.Errorf("read .fymap package directory: %w", err)
	}
	slices.SortFunc(entries, func(left, right os.DirEntry) int { return strings.Compare(left.Name(), right.Name()) })
	compatible := make([]Partition, 0, 1)
	sawStale := false
	sawIncompatible := false
	sawCorrupt := false
	for _, entry := range entries {
		if entry.IsDir() || !validMapGenerationName(entry.Name()) {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		partition, readErr := store.readGeneration(path, entry.Name(), directory)
		if readErr != nil {
			sawCorrupt = true
			store.recordRejected()
			_ = store.root.Remove(path)
			continue
		}
		if compatibilityErr := compatibilityError(partition, expectation); compatibilityErr != nil {
			switch {
			case errors.Is(compatibilityErr, ErrStaleMap):
				sawStale = true
			case errors.Is(compatibilityErr, ErrIncompatibleMap):
				sawIncompatible = true
			default:
				sawCorrupt = true
				store.recordRejected()
			}
			continue
		}
		compatible = append(compatible, partition)
	}
	if len(compatible) == 1 {
		store.recordHit()
		return compatible[0], true, nil
	}
	store.recordMiss()
	if len(compatible) > 1 {
		store.recordRejectedBy(uint64(len(compatible)))
		return Partition{}, false, ErrMapConflict
	}
	if sawStale {
		return Partition{}, false, ErrStaleMap
	}
	if sawIncompatible {
		return Partition{}, false, ErrIncompatibleMap
	}
	if sawCorrupt {
		return Partition{}, false, ErrStoredMapCorrupt
	}
	return Partition{}, false, nil
}

func (store *MapStore) readGeneration(path, name, expectedInstance string) (Partition, error) {
	file, err := store.root.Open(path)
	if err != nil {
		return Partition{}, err
	}
	encoded, readErr := io.ReadAll(io.LimitReader(file, int64(store.limits.MaxMapBytes)+1))
	closeErr := file.Close()
	if readErr != nil {
		return Partition{}, readErr
	}
	if closeErr != nil {
		return Partition{}, closeErr
	}
	if uint64(len(encoded)) > store.limits.MaxMapBytes {
		return Partition{}, ErrMapTooLarge
	}
	schema, encoding, logical := store.transportVersion()
	partition, err := decodeVersion(encoded, store.limits.MaxMapBytes, schema, encoding, logical)
	if err != nil {
		return Partition{}, err
	}
	checksum, err := computeVersionChecksum(documentFromPartition(partition), schema, encoding)
	if err != nil {
		return Partition{}, err
	}
	if mapGenerationName(checksum) != name {
		return Partition{}, fmt.Errorf(".fymap checksum does not match generation name")
	}
	if partition.Header().PackageInstance != expectedInstance {
		return Partition{}, fmt.Errorf(".fymap package does not match its directory")
	}
	return partition, nil
}

// Sweep removes expired temporaries and oldest generations until both caller bounds hold.
func (store *MapStore) Sweep() error {
	if store == nil || store.root == nil {
		return fmt.Errorf(".fymap store is not initialized")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	generations, err := store.scan(true)
	if err != nil {
		return err
	}
	slices.SortFunc(generations, func(left, right mapGeneration) int {
		if byTime := left.modTime.Compare(right.modTime); byTime != 0 {
			return byTime
		}
		return strings.Compare(left.path, right.path)
	})
	total := generationBytes(generations)
	for len(generations) > store.limits.MaxGenerations || total > store.limits.MaxBytes {
		oldest := generations[0]
		generations = generations[1:]
		if err := store.root.Remove(oldest.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("evict .fymap generation: %w", err)
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

func (store *MapStore) Stats() (MapStoreStats, error) {
	if store == nil || store.root == nil {
		return MapStoreStats{}, fmt.Errorf(".fymap store is not initialized")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	generations, err := store.scan(false)
	if err != nil {
		return MapStoreStats{}, err
	}
	return MapStoreStats{
		Generations: len(generations), Bytes: generationBytes(generations), Hits: store.hits,
		Misses: store.misses, Rejected: store.rejected, Evictions: store.evicted,
	}, nil
}

func (store *MapStore) scan(clean bool) ([]mapGeneration, error) {
	directories, err := readMapDirectory(store.root, ".")
	if err != nil {
		return nil, fmt.Errorf("scan .fymap store: %w", err)
	}
	now := time.Now()
	result := make([]mapGeneration, 0)
	for _, directory := range directories {
		if !directory.IsDir() || !validMapPackageDirectory(directory.Name()) {
			continue
		}
		entries, readErr := readMapDirectory(store.root, directory.Name())
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(directory.Name(), entry.Name())
			info, infoErr := entry.Info()
			if infoErr != nil {
				continue
			}
			if strings.HasPrefix(entry.Name(), ".tmp-") {
				if clean && now.Sub(info.ModTime()) >= store.limits.TempMaxAge {
					_ = store.root.Remove(path)
				}
				continue
			}
			if !validMapGenerationName(entry.Name()) {
				continue
			}
			if info.Size() <= 0 || uint64(info.Size()) > store.limits.MaxMapBytes {
				if clean {
					_ = store.root.Remove(path)
					store.rejected++
				}
				continue
			}
			result = append(result, mapGeneration{path: path, size: uint64(info.Size()), modTime: info.ModTime()})
		}
	}
	return result, nil
}

func readMapDirectory(root *os.Root, name string) ([]os.DirEntry, error) {
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

func mapGenerationName(checksum Checksum) string {
	return hex.EncodeToString(checksum[:]) + ".fymap"
}

func validMapPackageDirectory(name string) bool {
	identity, err := packagegraph.ParseInstanceID(name)
	return err == nil && identity != (packagegraph.InstanceID{})
}

func validMapGenerationName(name string) bool {
	if !strings.HasSuffix(name, ".fymap") {
		return false
	}
	digest := strings.TrimSuffix(name, ".fymap")
	if len(digest) != 64 || digest != strings.ToLower(digest) {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}

func generationBytes(values []mapGeneration) uint64 {
	var total uint64
	for _, value := range values {
		if ^uint64(0)-total < value.size {
			return ^uint64(0)
		}
		total += value.size
	}
	return total
}

func (store *MapStore) recordHit() {
	store.mu.Lock()
	store.hits++
	store.mu.Unlock()
}

func (store *MapStore) recordMiss() {
	store.mu.Lock()
	store.misses++
	store.mu.Unlock()
}

func (store *MapStore) recordRejected() {
	store.recordRejectedBy(1)
}

func (store *MapStore) recordRejectedBy(count uint64) {
	store.mu.Lock()
	store.rejected += count
	store.mu.Unlock()
}
