package packagesource

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/durablefile"
)

func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// directory rejects links and case aliases for every existing path component.
func directory(root *os.Root, name string, create bool) error {
	if name == "." {
		return nil
	}
	if path.Clean(name) != name || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || name == ".." || strings.HasPrefix(name, "../") {
		return fmt.Errorf("invalid contained directory")
	}
	current := ""
	for _, part := range strings.Split(name, "/") {
		parent := current
		if parent == "" {
			parent = "."
		}
		current = path.Join(current, part)
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) && create {
			if err := root.Mkdir(current, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			info, err = root.Lstat(current)
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("directory must not be a link or special file")
		}
		if err := exactName(root, parent, part); err != nil {
			return err
		}
	}
	return nil
}

func exactName(root *os.Root, parent, name string) error {
	f, err := root.Open(parent)
	if err != nil {
		return err
	}
	defer f.Close()
	for {
		entries, err := f.ReadDir(128)
		for _, entry := range entries {
			if entry.Name() == name {
				return nil
			}
		}
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("noncanonical path case")
			}
			return err
		}
	}
}

func readRegular(root *os.Root, name string, max int64) ([]byte, error) {
	if err := directory(root, path.Dir(name), false); err != nil {
		return nil, err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > max {
		return nil, fmt.Errorf("expected bounded regular source file")
	}
	if err := exactName(root, path.Dir(name), path.Base(name)); err != nil {
		return nil, err
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("source file changed during open")
	}
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("source file byte limit exceeded")
	}
	return data, nil
}

func storeArchive(root *os.Root, pin Pin, data []byte) error {
	if err := directory(root, CachePath, true); err != nil {
		return err
	}
	f, tmp, err := durablefile.CreateTemporary(root, filepath.FromSlash(CachePath))
	if err != nil {
		return err
	}
	defer root.Remove(tmp)
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	// A hard-link publication never replaces a concurrent existing cache object.
	name := path.Join(CachePath, pin.SHA256+".zip")
	if err := root.Link(tmp, name); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		existing, err := readRegular(root, name, MaxArchiveBytes)
		if err != nil || hash(existing) != pin.SHA256 {
			return fmt.Errorf("source cache publication conflict")
		}
	}
	return nil
}
