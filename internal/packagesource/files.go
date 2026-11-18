package packagesource

import (
	"bytes"
	"context"
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

const sourceCompareBufferBytes = 32 << 10

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
		if errors.Is(err, os.ErrNotExist) {
			if caseErr := rejectCaseAlias(root, parent, part); caseErr != nil {
				return caseErr
			}
			if create {
				if err := root.Mkdir(current, 0700); err != nil && !errors.Is(err, os.ErrExist) {
					return err
				}
				info, err = root.Lstat(current)
			}
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

// A case-sensitive host can report an alias as missing. Check the directory
// before treating the path as available for a download or publication.
func rejectCaseAlias(root *os.Root, parent, name string) error {
	f, err := root.Open(parent)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	for {
		entries, err := f.ReadDir(128)
		for _, entry := range entries {
			if actual := entry.Name(); actual != name && strings.EqualFold(actual, name) {
				return fmt.Errorf("noncanonical path case: %s", path.Join(parent, name))
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
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
	f, err := openRegular(root, name, max)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("source file byte limit exceeded")
	}
	return data, nil
}

// openRegular preserves the same path, file kind, size and identity checks for
// readers and comparisons. The caller owns the returned file.
func openRegular(root *os.Root, name string, max int64) (*os.File, error) {
	return openRegularWithNames(root, name, max, nil)
}

func openRegularWithNames(root *os.Root, name string, max int64, names *treeNameIndex) (*os.File, error) {
	checkDirectory := func(name string) error { return directory(root, name, false) }
	checkName := func(parent, name string) error { return exactName(root, parent, name) }
	if names != nil {
		checkDirectory, checkName = names.directory, names.exactName
	}
	if err := checkDirectory(path.Dir(name)); err != nil {
		return nil, err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > max {
		return nil, fmt.Errorf("expected bounded regular source file")
	}
	if err := checkName(path.Dir(name), path.Base(name)); err != nil {
		return nil, err
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		f.Close()
		return nil, fmt.Errorf("source file changed during open")
	}
	return f, nil
}

func readRegularMatches(ctx context.Context, root *os.Root, name string, max int64, expected []byte) (bool, error) {
	return readRegularMatchesWithNames(ctx, root, name, max, expected, nil)
}

func readRegularMatchesWithNames(ctx context.Context, root *os.Root, name string, max int64, expected []byte, names *treeNameIndex) (bool, error) {
	f, err := openRegularWithNames(root, name, max, names)
	if err != nil {
		return false, err
	}
	defer f.Close()
	// Small pinned files need no full-sized buffer. If the installed file has
	// grown, expand only up to the fixed cap while preserving bounded reads.
	buffer := make([]byte, min(sourceCompareBufferBytes-1, len(expected))+1)
	reader := io.LimitReader(f, max+1)
	offset, matches := 0, true
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		n, err := reader.Read(buffer[:])
		if offset > len(expected) || n > len(expected)-offset || !bytes.Equal(buffer[:n], expected[offset:offset+n]) {
			matches = false
		}
		offset += n
		if err != nil {
			if err != io.EOF {
				return false, err
			}
			break
		}
		if n == len(buffer) && len(buffer) < sourceCompareBufferBytes {
			buffer = make([]byte, min(len(buffer)*2, sourceCompareBufferBytes))
		}
	}
	// Keep reading after a mismatch so later I/O/size errors retain precedence.
	if int64(offset) > max {
		return false, fmt.Errorf("source file byte limit exceeded")
	}
	return matches && offset == len(expected), nil
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
		existing, err := readRegularHash(root, name, MaxArchiveBytes)
		if err != nil || existing != pin.SHA256 {
			return fmt.Errorf("source cache publication conflict")
		}
	}
	return nil
}

// A concurrent publication only needs the existing archive's digest, not a
// second archive-sized byte slice. Keep the same regular-file and size checks.
func readRegularHash(root *os.Root, name string, max int64) (string, error) {
	f, err := openRegular(root, name, max)
	if err != nil {
		return "", err
	}
	defer f.Close()
	digest := sha256.New()
	n, err := io.CopyBuffer(digest, io.LimitReader(f, max+1), make([]byte, sourceCompareBufferBytes))
	if err != nil {
		return "", err
	}
	if n > max {
		return "", fmt.Errorf("source file byte limit exceeded")
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
