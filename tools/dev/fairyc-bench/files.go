package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func writeSources(root string, sources map[string]string) error {
	for name, source := range sources {
		path := filepath.Join(root, filepath.FromSlash(name))
		if old, err := os.ReadFile(path); err == nil && string(old) == source {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func sourceIdentity(sources map[string]string) (string, int64, int) {
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	var data strings.Builder
	var bytes int64
	for _, name := range names {
		fmt.Fprintf(&data, "%s\x00%d\x00%s\x00", name, len(sources[name]), sources[name])
		bytes += int64(len(sources[name]))
	}
	return digest([]byte(data.String())), bytes, len(names)
}

// Walk only known output/cache roots. Refuse symlinks rather than following
// paths outside the experiment or silently claiming an incomplete inventory.
func inventory(project, state, binary string) (Footprint, error) {
	var result Footprint
	for _, root := range []string{filepath.Join(project, "build"), filepath.Join(project, "target"), filepath.Join(project, "zig-out"), filepath.Join(project, ".zig-cache"), state} {
		if _, err := os.Lstat(root); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return result, err
		}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing symlink in measured root: %s", path)
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("non-regular measured file: %s", path)
			}
			size := info.Size()
			lower := strings.ToLower(filepath.ToSlash(path))
			switch {
			case path == binary:
				result.Binary += size
			case strings.HasPrefix(path, filepath.Join(state, "tmp")+string(filepath.Separator)):
				result.Temporary += size
			case strings.HasPrefix(path, state+string(filepath.Separator)):
				result.SharedCache += size
			case strings.Contains(lower, "/incremental/"):
				result.Incremental += size
			case strings.HasSuffix(lower, ".pdb") || strings.HasSuffix(lower, ".dwo") || strings.HasSuffix(lower, ".debug") || strings.Contains(lower, ".dsym/"):
				result.Debug += size
			case strings.Contains(lower, "/.zig-cache/"):
				result.LocalCache += size
			default:
				result.Intermediate += size
			}
			result.Total += size
			result.Files++
			return nil
		})
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func executableSuffix() string {
	if os.PathSeparator == '\\' {
		return ".exe"
	}
	return ""
}
