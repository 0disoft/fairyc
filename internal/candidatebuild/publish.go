package candidatebuild

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func result(packet []byte, units, size int, profile string) Result {
	hash := sha256.Sum256(packet)
	return Result{hex.EncodeToString(hash[:]), units, size, profile}
}

type publishHooks struct {
	write       func(*os.File, []byte) error
	sync, close func(*os.File) error
	rename      func(*os.Root, string, string) error
}

func Publish(path string, image []byte) error { return publish(path, image, publishHooks{}) }

// The Fairy caller owns the root and relative destination selection. All
// directory creation and staged publication stay under the opened root.
func PublishWithin(directory, path string, image []byte) error {
	if !filepath.IsLocal(path) || filepath.Base(path) == "." || len(image) == 0 || len(image) > 64*1024*1024 {
		return fmt.Errorf("invalid rooted output image or path")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return publishOpened(root, path, image, publishHooks{})
}

// Stage, sync and close inside the destination's opened parent before rename.
// A failed rename never removes the existing image to make replacement work.
func publish(path string, image []byte, hooks publishHooks) error {
	if path == "" || len(image) == 0 || len(image) > 64*1024*1024 {
		return fmt.Errorf("invalid output image or path")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	directory, leaf := filepath.Dir(absolute), filepath.Base(absolute)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	return publishOpened(root, leaf, image, hooks)
}

func publishOpened(root *os.Root, leaf string, image []byte, hooks publishHooks) error {
	if info, err := root.Lstat(leaf); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("output is not a regular file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	temporary := filepath.Join(filepath.Dir(leaf), ".fairyc-candidate-"+hex.EncodeToString(nonce[:])+".tmp")
	file, err := root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	defer file.Close()
	if hooks.write == nil {
		hooks.write = func(f *os.File, data []byte) error {
			n, err := f.Write(data)
			if err == nil && n != len(data) {
				return io.ErrShortWrite
			}
			return err
		}
	}
	if hooks.sync == nil {
		hooks.sync = (*os.File).Sync
	}
	if hooks.close == nil {
		hooks.close = (*os.File).Close
	}
	if hooks.rename == nil {
		hooks.rename = (*os.Root).Rename
	}
	if err := hooks.write(file, image); err != nil {
		return fmt.Errorf("stage candidate image: %w", err)
	}
	if err := hooks.sync(file); err != nil {
		return fmt.Errorf("sync candidate image: %w", err)
	}
	if err := hooks.close(file); err != nil {
		return fmt.Errorf("close candidate image: %w", err)
	}
	if err := hooks.rename(root, temporary, leaf); err != nil {
		return fmt.Errorf("replace candidate image: %w", err)
	}
	return nil
}
