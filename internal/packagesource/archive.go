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
	MaxArchiveBytes = 16 << 20
	MaxTotalBytes   = 64 << 20
	MaxFileBytes    = 8 << 20
	MaxFiles        = 4096
)

type bundle struct {
	files map[string][]byte
	bytes int
}

func portableEntry(name string) bool {
	if len(name) == 0 || len(name) > 1024 || path.Clean(name) != name || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
		return false
	}
	parts := strings.Split(name, "/")
	if len(parts) > 32 {
		return false
	}
	for _, part := range parts {
		if part == "." || part == ".." {
			return false
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
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return bundle{}, fmt.Errorf("invalid source ZIP: %w", err)
	}
	if len(r.File) == 0 || len(r.File) > MaxFiles*2 {
		return bundle{}, fmt.Errorf("ZIP entry limit exceeded")
	}
	b := bundle{files: map[string][]byte{}}
	type entry struct {
		name string
		dir  bool
	}
	seen := map[string]entry{}
	explicit := map[string]bool{}
	for _, f := range r.File {
		if err := ctx.Err(); err != nil {
			return bundle{}, err
		}
		name := strings.TrimSuffix(f.Name, "/")
		if !portableEntry(name) {
			return bundle{}, fmt.Errorf("nonportable ZIP path")
		}
		dir := f.FileInfo().IsDir()
		if dir != strings.HasSuffix(f.Name, "/") || (f.Mode().Type() != 0 && f.Mode().Type() != fs.ModeDir) || (!dir && !f.Mode().IsRegular()) {
			return bundle{}, fmt.Errorf("ZIP special entry rejected")
		}
		if explicit[strings.ToLower(name)] {
			return bundle{}, fmt.Errorf("duplicate ZIP entry")
		}
		explicit[strings.ToLower(name)] = true
		parts := strings.Split(name, "/")
		for i := range parts {
			part := strings.Join(parts[:i+1], "/")
			isDir := i < len(parts)-1 || dir
			fold := strings.ToLower(part)
			if prior, ok := seen[fold]; ok && (prior.name != part || prior.dir != isDir) {
				return bundle{}, fmt.Errorf("ZIP case or file/directory conflict")
			}
			seen[fold] = entry{part, isDir}
		}
		if dir {
			continue
		}
		if len(b.files) >= MaxFiles || f.UncompressedSize64 > MaxFileBytes {
			return bundle{}, fmt.Errorf("ZIP file limit exceeded")
		}
		file, err := f.Open()
		if err != nil {
			return bundle{}, err
		}
		contents, readErr := io.ReadAll(io.LimitReader(file, MaxFileBytes+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil || len(contents) > MaxFileBytes || uint64(len(contents)) != f.UncompressedSize64 {
			return bundle{}, fmt.Errorf("ZIP entry integrity or byte limit failed")
		}
		b.bytes += len(contents)
		if b.bytes > MaxTotalBytes {
			return bundle{}, fmt.Errorf("ZIP expanded byte limit exceeded")
		}
		b.files[name] = contents
	}
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
