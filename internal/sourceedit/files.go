package sourceedit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/durablefile"
	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// ReadFile admits one bounded, portable, root-contained regular input without link traversal.
func ReadFile(root *os.Root, name string, max int64) ([]byte, error) {
	return readFile(root, name, max, nil)
}

func readFile(root *os.Root, name string, max int64, cases *pathCaseIndex) ([]byte, error) {
	if filepath.ToSlash(filepath.Clean(filepath.FromSlash(name))) != name || name == "." || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, "\\") {
		return nil, fmt.Errorf("noncanonical edit path")
	}
	current := ""
	for _, part := range strings.Split(name, "/") {
		if !projectmanifest.ValidPortablePathSegment(part) {
			return nil, fmt.Errorf("nonportable edit path")
		}
		parent := current
		if parent == "" {
			parent = "."
		}
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("edit path contains a symlink")
		}
		if err := checkPathCase(root, cases, parent, part); err != nil {
			return nil, err
		}
	}
	info, err := root.Lstat(current)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > max {
		return nil, fmt.Errorf("edit input is not a bounded regular file")
	}
	f, err := root.Open(current)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("edit input changed during open")
	}
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(data)) > max {
		return nil, fmt.Errorf("edit input read/size failed")
	}
	return data, nil
}

func snapshot(w workspaceloader.Workspace) (string, error) {
	root, err := os.OpenRoot(w.Root)
	if err != nil {
		return "", err
	}
	defer root.Close()
	cases := newPathCaseIndex(root)
	records := map[string]string{}
	for _, p := range w.Packages {
		if p.SourceKind != workspaceloader.SourceKindLocal {
			continue
		}
		manifestPath := filepath.ToSlash(filepath.Join(p.Path, projectmanifest.FileName))
		data, err := readFile(root, manifestPath, projectmanifest.MaxBytes, cases)
		if err != nil {
			return "", err
		}
		records[manifestPath] = Digest(data)
		for _, s := range p.Sources {
			absolute := filepath.Join(p.SourceRoot, filepath.FromSlash(s.Path))
			relative, err := filepath.Rel(w.Root, absolute)
			if err != nil {
				return "", err
			}
			records[filepath.ToSlash(relative)] = Digest(s.Bytes)
		}
	}
	lock, err := readFile(root, "fairy.lock", 8<<20, cases)
	if err != nil {
		return "", err
	}
	records["fairy.lock"] = Digest(lock)
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteByte(0)
		b.WriteString(records[key])
		b.WriteByte(0)
	}
	return Digest([]byte(b.String())), nil
}

// Apply commits exactly one already-verified authored file. It never removes an old destination
// to recover from rename failure and never refreshes the production graph lock.
func (p *Plan) Apply() (err error) {
	if p == nil || len(p.before) == 0 || len(p.after) == 0 || p.Report.Applied {
		return fmt.Errorf("invalid or already applied edit plan")
	}
	root, err := os.OpenRoot(p.root)
	if err != nil {
		return err
	}
	defer root.Close()
	const lock = ".fairyc-edit.lock"
	if err := root.Mkdir(lock, 0700); err != nil {
		return fmt.Errorf("exclusive edit lock unavailable: %w", err)
	}
	defer func() { err = errors.Join(err, root.Remove(lock)) }()
	w, err := workspaceloader.Load(p.root)
	if err != nil {
		return err
	}
	current, err := snapshot(w)
	if err != nil {
		return err
	}
	if current != p.fingerprint {
		return fmt.Errorf("workspace changed since edit preview")
	}
	if err := validateOverlay(p.root, filepath.Join(p.root, filepath.FromSlash(p.path)), p.after); err != nil {
		return err
	}
	info, err := root.Lstat(filepath.FromSlash(p.path))
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("source is read-only")
	}
	f, tmp, err := durablefile.CreateTemporary(root, filepath.Dir(filepath.FromSlash(p.path)))
	if err != nil {
		return err
	}
	defer root.Remove(tmp)
	_, writeErr := f.Write(p.after)
	chmodErr := f.Chmod(info.Mode().Perm())
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, chmodErr, syncErr, closeErr); err != nil {
		return err
	}
	// Re-read the complete closure after staging, immediately before the atomic replacement.
	w, err = workspaceloader.Load(p.root)
	if err != nil {
		return err
	}
	current, err = snapshot(w)
	if err != nil || current != p.fingerprint {
		return fmt.Errorf("workspace changed while staging edit")
	}
	before, err := ReadFile(root, p.path, int64(source.DefaultLimits().MaxBytes))
	if err != nil || !bytes.Equal(before, p.before) {
		return fmt.Errorf("source changed while staging edit")
	}
	if err := root.Rename(tmp, filepath.FromSlash(p.path)); err != nil {
		return err
	}
	p.Report.Applied = true
	return nil
}
