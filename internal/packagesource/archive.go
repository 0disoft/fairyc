package packagesource

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

const (
	MaxArchiveBytes       = 16 << 20
	MaxTotalBytes         = 64 << 20
	MaxFileBytes          = 8 << 20
	MaxFiles              = 4096
	maxArchivePathBytes   = 1024
	maxArchivePathDepth   = 32
	maxArchivePathEntries = MaxFiles * 2
)

type bundle struct {
	files map[string][]byte
	bytes int
}

func portableEntry(name string) bool {
	if len(name) == 0 || len(name) > maxArchivePathBytes || path.Clean(name) != name || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
		return false
	}
	parts := strings.Split(name, "/")
	if len(parts) > maxArchivePathDepth {
		return false
	}
	for _, part := range parts {
		if part == "." || part == ".." {
			return false
		}
		switch strings.ToLower(part) {
		case ".git", ".hg", ".svn":
			return false // Source archives cannot inject nested VCS configuration/hooks.
		}
		// Dotfiles are data too; never execute hooks. Retain the portable reserved-name checks.
		if strings.HasPrefix(part, ".") {
			part = "x" + part
		}
		if !projectmanifest.ValidPortablePathSegment(part) {
			return false
		}
	}
	return true
}

func unpack(ctx context.Context, data []byte, pin Pin, language string) (bundle, error) {
	return unpackWithin(ctx, data, pin, language, MaxTotalBytes, MaxFiles)
}

func unpackWithin(ctx context.Context, data []byte, pin Pin, language string, expandedLimit, fileLimit int) (bundle, error) {
	if err := admitArchiveDirectory(data); err != nil {
		if err == errArchiveEntryLimit {
			return bundle{}, err
		}
		return bundle{}, fmt.Errorf("invalid source ZIP: %w", err)
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return bundle{}, fmt.Errorf("invalid source ZIP: %w", err)
	}
	if len(r.File) == 0 || len(r.File) > maxArchivePathEntries {
		return bundle{}, fmt.Errorf("ZIP entry limit exceeded")
	}
	b, err := unpackArchiveFiles(ctx, r.File, expandedLimit, fileLimit)
	if err != nil {
		return bundle{}, err
	}
	return validateArchiveBundle(b, pin, language)
}

func unpackArchiveFiles(ctx context.Context, files []*zip.File, expandedLimit, fileLimit int) (bundle, error) {
	b := bundle{files: map[string][]byte{}}
	index := archivePathIndex{seen: map[string]archivePathEntry{}, explicit: map[string]bool{}}
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return bundle{}, err
		}
		name, dir, err := archiveEntryIdentity(f)
		if err != nil {
			return bundle{}, err
		}
		if err := index.add(name, dir); err != nil {
			return bundle{}, err
		}
		if dir {
			continue
		}
		remaining, err := archiveRemainingBudget(f.UncompressedSize64, len(b.files), b.bytes, expandedLimit, fileLimit)
		if err != nil {
			return bundle{}, err
		}
		contents, err := readArchiveEntry(f, remaining)
		if err != nil {
			return bundle{}, err
		}
		b.bytes += len(contents)
		if b.bytes > expandedLimit {
			return bundle{}, fmt.Errorf("ZIP expanded byte limit exceeded")
		}
		b.files[name] = contents
	}
	return b, nil
}

// Header checks and budget arithmetic do not mutate zip.File or caller data.
func archiveEntryIdentity(f *zip.File) (string, bool, error) {
	name := strings.TrimSuffix(f.Name, "/")
	if !portableEntry(name) {
		return "", false, fmt.Errorf("nonportable ZIP path")
	}
	dir := f.FileInfo().IsDir()
	if dir != strings.HasSuffix(f.Name, "/") || (f.Mode().Type() != 0 && f.Mode().Type() != fs.ModeDir) || (!dir && !f.Mode().IsRegular()) {
		return "", false, fmt.Errorf("ZIP special entry rejected")
	}
	return name, dir, nil
}

type archivePathEntry struct {
	name string
	dir  bool
}

// This index belongs to one unpack operation; in-place updates avoid copying a
// growing path map for every entry. It never aliases a caller-owned map.
type archivePathIndex struct {
	seen     map[string]archivePathEntry
	explicit map[string]bool
}

func (index *archivePathIndex) add(name string, dir bool) error {
	if index.explicit[strings.ToLower(name)] {
		return fmt.Errorf("duplicate ZIP entry")
	}
	index.explicit[strings.ToLower(name)] = true
	parts := strings.Split(name, "/")
	for i := range parts {
		part := strings.Join(parts[:i+1], "/")
		isDir := i < len(parts)-1 || dir
		fold := strings.ToLower(part)
		if prior, ok := index.seen[fold]; ok && (prior.name != part || prior.dir != isDir) {
			return fmt.Errorf("ZIP case or file/directory conflict")
		}
		index.seen[fold] = archivePathEntry{part, isDir}
		if len(index.seen) > maxArchivePathEntries {
			return fmt.Errorf("ZIP path entry limit exceeded")
		}
	}
	return nil
}

func archiveRemainingBudget(size uint64, fileCount, expanded, expandedLimit, fileLimit int) (int, error) {
	remaining := min(MaxFileBytes, expandedLimit-expanded)
	if fileCount >= fileLimit || remaining < 0 || size > uint64(remaining) {
		return 0, fmt.Errorf("ZIP file limit exceeded")
	}
	return remaining, nil
}

func readArchiveEntry(f *zip.File, remaining int) ([]byte, error) {
	file, err := f.Open()
	if err != nil {
		return nil, err
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, int64(remaining)+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(contents) > remaining || uint64(len(contents)) != f.UncompressedSize64 {
		return nil, fmt.Errorf("ZIP entry integrity or byte limit failed")
	}
	return contents, nil
}

func validateArchiveBundle(b bundle, pin Pin, language string) (bundle, error) {
	manifest, err := projectmanifest.Parse(b.files[projectmanifest.FileName])
	if err != nil {
		return bundle{}, fmt.Errorf("archive manifest: %w", err)
	}
	if manifest.Workspace.Present || manifest.Package.UID != pin.PackageUID || manifest.Package.Version != pin.Version || manifest.Package.Language != language {
		return bundle{}, fmt.Errorf("archive package identity, language or workspace mismatch")
	}
	for name := range b.files {
		if strings.HasPrefix(name, manifest.Package.Source+"/") && strings.HasSuffix(name, ".fy") {
			return b, nil
		}
	}
	return bundle{}, fmt.Errorf("archive source directory has no .fy files")
}
