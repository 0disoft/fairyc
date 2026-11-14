package workspaceloader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/source"
)

var errInputsChanged = errors.New("workspace inputs changed")

// FileSnapshot compares current inputs with a cache-owned, immutable loaded
// workspace. It neither reparses source nor certifies caller-mutated ASTs.
// Callers must keep workspace and its retained source slices immutable.
type FileSnapshot struct {
	workspace   Workspace
	lock        [sha256.Size]byte
	lockPresent bool
	overlays    [sha256.Size]byte
}

func CaptureFileSnapshot(ctx context.Context, workspace Workspace, overlays map[string][]byte) (*FileSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := openWorkspaceRoot(workspace.Root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	lock, present, err := snapshotLock(root)
	if err != nil {
		return nil, err
	}
	return &FileSnapshot{workspace: workspace, lock: lock, lockPresent: present, overlays: overlayFingerprint(overlays)}, nil
}

func snapshotLock(root *os.Root) ([sha256.Size]byte, bool, error) {
	data, err := readBoundedRootFile(root, "fairy.lock", 8<<20)
	if errors.Is(err, os.ErrNotExist) {
		return [sha256.Size]byte{}, false, nil
	}
	if err != nil {
		return [sha256.Size]byte{}, false, err
	}
	return sha256.Sum256(data), true, nil
}

// Matches checks every catalog manifest and source, including closed documents
// and newly added/deleted files. A metadata timestamp is never a freshness proof.
func (snapshot *FileSnapshot) Matches(ctx context.Context, overlays map[string][]byte) (bool, error) {
	if snapshot == nil {
		return false, nil
	}
	if overlayFingerprint(overlays) != snapshot.overlays {
		return false, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	workspace := snapshot.workspace
	root, err := openWorkspaceRoot(workspace.Root)
	if err != nil {
		return false, err
	}
	defer root.Close()
	budget := &limits{ctx: ctx, root: root}
	directories := newDirectoryResolver()
	directories.root, directories.budget = root, budget
	var comparisonBuffer []byte
	for _, pkg := range workspace.Packages {
		if pkg.SourceKind != SourceKindLocal {
			continue
		}
		directory := workspace.Root
		if pkg.Path != "." {
			directory, err = directories.resolve(workspace.Root, pkg.Path)
			if err != nil {
				return false, err
			}
		}
		manifest, err := readManifest(root, directory, pkg.Path)
		if err != nil {
			return false, err
		}
		if !reflect.DeepEqual(manifest, pkg.Manifest) {
			return false, nil
		}
		sourceRoot, err := directories.resolve(directory, manifest.Package.Source)
		if err != nil {
			return false, err
		}
		if err := matchPackageFiles(root, sourceRoot, pkg.Sources, overlays, budget, &comparisonBuffer); err != nil {
			if errors.Is(err, errInputsChanged) {
				return false, nil
			}
			return false, err
		}
	}
	lock, present, err := snapshotLock(root)
	if err != nil {
		return false, err
	}
	return lock == snapshot.lock && present == snapshot.lockPresent, ctx.Err()
}

func overlayFingerprint(overlays map[string][]byte) [sha256.Size]byte {
	keys := make([]string, 0, len(overlays))
	for key := range overlays {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	digest := sha256.New()
	var size [8]byte
	for _, key := range keys {
		binary.LittleEndian.PutUint64(size[:], uint64(len(key)))
		digest.Write(size[:])
		digest.Write([]byte(key))
		binary.LittleEndian.PutUint64(size[:], uint64(len(overlays[key])))
		digest.Write(size[:])
		digest.Write(overlays[key])
	}
	var result [sha256.Size]byte
	copy(result[:], digest.Sum(nil))
	return result
}

func matchPackageFiles(root *os.Root, sourceRoot string, sources []SourceFile, overlays map[string][]byte, budget *limits, comparisonBuffer *[]byte) error {
	boundary, err := openContainedDirectory(root, sourceRoot)
	if err != nil {
		return err
	}
	defer boundary.Close()
	expected := make(map[string][]byte, len(sources))
	for _, file := range sources {
		if file.Test {
			return errInputsChanged
		}
		expected[file.Path] = file.Bytes
	}
	seen := make(map[string]bool, len(sources))
	check := func(name string, data []byte) error {
		want, found := expected[name]
		if !found || !bytes.Equal(data, want) {
			return errInputsChanged
		}
		seen[name] = true
		return nil
	}
	err = walkSourceTree(boundary.FS(), budget, MaxSourceEntries, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errInputsChanged
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".fy" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errInputsChanged
		}
		absolute := filepath.Join(sourceRoot, filepath.FromSlash(name))
		data, overlay := overlays[absolute]
		if overlay {
			return check(name, data)
		}
		if *comparisonBuffer == nil {
			*comparisonBuffer = make([]byte, snapshotCompareBufferBytes)
		}
		want, found := expected[name]
		same, err := matchBoundedRootFile(budget.ctx, boundary, filepath.FromSlash(name), source.DefaultLimits().MaxBytes, want, *comparisonBuffer)
		if err != nil {
			return err
		}
		if !found || !same {
			return errInputsChanged
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		return err
	}
	for absolute, data := range overlays {
		name, err := filepath.Rel(sourceRoot, absolute)
		if err != nil || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			continue
		}
		name = filepath.ToSlash(name)
		if !seen[name] {
			if err := check(name, data); err != nil {
				return err
			}
		}
	}
	if len(seen) != len(expected) {
		return errInputsChanged
	}
	return nil
}
