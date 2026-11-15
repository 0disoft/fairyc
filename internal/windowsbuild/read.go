package windowsbuild

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/durablefile"
)

// LoadOptions replaces dependencies for one load. Nil fields use Background
// and the host filesystem. Open transfers a file that the loader must close.
// Lstat must capture stable file identity, as os.Root.Lstat does on Windows.
// Cancellation is checked between I/O calls; blocked syscalls are not preempted.
type LoadOptions struct {
	Context  context.Context
	OpenRoot func(string) (*os.Root, error)
	Lstat    func(*os.Root, string) (os.FileInfo, error)
	Open     func(*os.Root, string) (fs.File, error)
}

func loadPolicy(options []LoadOptions) (LoadOptions, error) {
	if len(options) > 1 {
		return LoadOptions{}, fmt.Errorf("Windows build load accepts at most one options value")
	}
	var policy LoadOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	if policy.OpenRoot == nil {
		policy.OpenRoot = os.OpenRoot
	}
	if policy.Lstat == nil {
		policy.Lstat = (*os.Root).Lstat
	}
	if policy.Open == nil {
		policy.Open = func(root *os.Root, name string) (fs.File, error) {
			file, err := durablefile.OpenRegular(root, filepath.FromSlash(name))
			if err != nil {
				return nil, err
			}
			return file, nil
		}
	}
	return policy, nil
}

func inputReadError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	return err
}

func read(root *os.Root, name string, limit int, policy LoadOptions) (data []byte, err error) {
	ctx := policy.Context
	if err := inputReadError(ctx, nil); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > MaxInputBytes {
		return nil, fmt.Errorf("invalid Windows build input limit")
	}
	var before os.FileInfo
	parts := strings.Split(name, "/")
	for i := range parts {
		info, err := policy.Lstat(root, strings.Join(parts[:i+1], "/"))
		if err = inputReadError(ctx, err); err != nil {
			return nil, err
		}
		if info == nil {
			return nil, fmt.Errorf("missing metadata for %s", name)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink build input %s", name)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, fmt.Errorf("non-directory build input component %s", name)
		}
		before = info
	}
	if !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > int64(limit) {
		return nil, fmt.Errorf("invalid size/type for %s", name)
	}
	f, err := policy.Open(root, name)
	if f != nil {
		defer func() {
			err = inputReadError(ctx, errors.Join(err, f.Close()))
			if err != nil {
				data = nil
			}
		}()
	}
	if err = inputReadError(ctx, err); err != nil {
		// Only absence at initial inspection means an optional config is absent.
		return nil, fmt.Errorf("open build input %s: %w", name, err)
	}
	if f == nil {
		return nil, fmt.Errorf("missing build input handle for %s", name)
	}
	s, err := f.Stat()
	if err = inputReadError(ctx, err); err != nil {
		return nil, fmt.Errorf("stat build input %s: %w", name, err)
	}
	if !sameInput(before, s) {
		return nil, fmt.Errorf("build input %s changed during open: %w", name, durablefile.ErrIdentityChanged)
	}
	data, err = io.ReadAll(io.LimitReader(&inputReader{ctx: ctx, reader: f}, int64(limit)+1))
	if err = inputReadError(ctx, err); err != nil {
		return nil, fmt.Errorf("read build input %s: %w", name, err)
	}
	if len(data) != int(s.Size()) || len(data) > limit || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return nil, fmt.Errorf("invalid or changed UTF-8 build input %s", name)
	}
	after, err := f.Stat()
	if err = inputReadError(ctx, err); err != nil {
		return nil, fmt.Errorf("restat build input %s: %w", name, err)
	}
	if !sameInput(s, after) {
		return nil, fmt.Errorf("build input %s changed while reading: %w", name, durablefile.ErrIdentityChanged)
	}
	return data, nil
}

func sameInput(before, after os.FileInfo) bool {
	return after != nil && after.Mode().IsRegular() && os.SameFile(before, after) &&
		before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

const inputReadChunkBytes = 32 << 10
const maxInputEmptyReads = 100

type inputReader struct {
	ctx        context.Context
	reader     io.Reader
	emptyReads int
}

func (reader *inputReader) Read(buffer []byte) (int, error) {
	if err := inputReadError(reader.ctx, nil); err != nil {
		return 0, err
	}
	n, err := reader.reader.Read(buffer[:min(len(buffer), inputReadChunkBytes)])
	if err = inputReadError(reader.ctx, err); err != nil {
		return n, err
	}
	if n == 0 {
		reader.emptyReads++
		if reader.emptyReads >= maxInputEmptyReads {
			return 0, io.ErrNoProgress
		}
	} else {
		reader.emptyReads = 0
	}
	return n, nil
}
