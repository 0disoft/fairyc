package candidatebuild

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/durablefile"
)

func result(packet []byte, units, size int, profile string) Result {
	hash := sha256.Sum256(packet)
	return Result{hex.EncodeToString(hash[:]), units, size, profile}
}

type publishHooks struct {
	context     context.Context
	random      func([]byte) (int, error)
	write       func(*os.File, []byte) error
	sync, close func(*os.File) error
	rename      func(*os.Root, string, string) error
}

func Publish(path string, image []byte) error { return PublishWithOptions(path, image) }

// The Fairy caller owns the root and relative destination selection. All
// directory creation and staged publication stay under the opened root.
func PublishWithin(directory, path string, image []byte) error {
	return PublishWithinWithOptions(directory, path, image)
}

// PublishWithinWithOptions retains PublishWithin's rooted destination boundary
// and uses the same publication options and rename commit point as Publish.
func PublishWithinWithOptions(directory, path string, image []byte, options ...PublicationOptions) error {
	hooks, err := publicationPolicy(options)
	if err != nil {
		return err
	}
	if !filepath.IsLocal(path) || filepath.Base(path) == "." || len(image) == 0 || len(image) > maxPublishedImageBytes {
		return fmt.Errorf("invalid rooted output image or path")
	}
	if err := hooks.check(nil); err != nil {
		return err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return hooks.check(err)
	}
	defer root.Close()
	if err := hooks.check(nil); err != nil {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return hooks.check(err)
	}
	return publishOpened(root, path, image, hooks)
}

// Stage, sync and close inside the destination's opened parent before rename.
// A failed rename never removes the existing image to make replacement work.
func publish(path string, image []byte, hooks publishHooks) error {
	if path == "" || len(image) == 0 || len(image) > maxPublishedImageBytes {
		return fmt.Errorf("invalid output image or path")
	}
	if err := hooks.check(nil); err != nil {
		return err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	directory, leaf := filepath.Dir(absolute), filepath.Base(absolute)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return hooks.check(err)
	}
	if err := hooks.check(nil); err != nil {
		return err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return hooks.check(err)
	}
	defer root.Close()
	return publishOpened(root, leaf, image, hooks)
}

func publishOpened(root *os.Root, leaf string, image []byte, hooks publishHooks) (err error) {
	if err := hooks.check(nil); err != nil {
		return err
	}
	if info, err := root.Lstat(leaf); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("output is not a regular file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return hooks.check(err)
	}
	if err := hooks.check(nil); err != nil {
		return err
	}
	var nonce [16]byte
	if hooks.random == nil {
		hooks.random = rand.Read
	}
	n, err := hooks.random(nonce[:])
	if err = hooks.check(err); err != nil {
		return err
	}
	if n != len(nonce) {
		return fmt.Errorf("candidate staging nonce: %w", io.ErrUnexpectedEOF)
	}
	temporary := filepath.Join(filepath.Dir(leaf), ".fairyc-candidate-"+hex.EncodeToString(nonce[:])+".tmp")
	file, err := root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return hooks.check(err)
	}
	identity, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	staged := true
	defer func() {
		if staged {
			err = errors.Join(err, removeCandidateStage(root, temporary, identity))
		}
	}()
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
	if err := hooks.check(nil); err != nil {
		return err
	}
	if err := hooks.check(hooks.write(file, image)); err != nil {
		return fmt.Errorf("stage candidate image: %w", err)
	}
	if err := hooks.check(hooks.sync(file)); err != nil {
		return fmt.Errorf("sync candidate image: %w", err)
	}
	if err := hooks.check(hooks.close(file)); err != nil {
		return fmt.Errorf("close candidate image: %w", err)
	}
	current, err := root.Lstat(temporary)
	if err = hooks.check(err); err != nil {
		return fmt.Errorf("inspect staged candidate image: %w", err)
	}
	if !current.Mode().IsRegular() || !os.SameFile(identity, current) || current.Size() != int64(len(image)) {
		return fmt.Errorf("candidate staging image changed: %w", durablefile.ErrIdentityChanged)
	}
	if err := hooks.rename(root, temporary, leaf); err != nil {
		return fmt.Errorf("replace candidate image: %w", hooks.check(err))
	}
	// A successful rename is the commit point. The old stage name no longer
	// belongs to this operation, even if another file appears there immediately.
	staged = false
	return hooks.check(nil)
}

func removeCandidateStage(root *os.Root, name string, identity os.FileInfo) error {
	current, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !current.Mode().IsRegular() || !os.SameFile(identity, current) {
		return fmt.Errorf("candidate staging path replaced; preserved: %w", durablefile.ErrIdentityChanged)
	}
	return root.Remove(name)
}
