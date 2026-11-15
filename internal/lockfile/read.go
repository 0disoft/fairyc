package lockfile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/durablefile"
)

// ReadOptions replaces dependencies for one read only. Nil fields retain the
// host filesystem and Background context. Open transfers a detached file to
// the reader, which closes it on every path after a successful open.
// Lstat must snapshot file identity at inspection time, as os.Root.Lstat does.
type ReadOptions struct {
	Context context.Context
	Lstat   func(string) (os.FileInfo, error)
	Open    func(string) (fs.File, error)
}

func lockReadPolicy(options []ReadOptions) (ReadOptions, error) {
	if len(options) > 1 {
		return ReadOptions{}, fmt.Errorf("lockfile read accepts at most one options value")
	}
	var policy ReadOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	if policy.Lstat == nil {
		policy.Lstat = inspectLockReadFile
	}
	if policy.Open == nil {
		policy.Open = openLockReadFile
	}
	return policy, nil
}

func inspectLockReadFile(path string) (os.FileInfo, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	// Root.Lstat captures handle identity on Windows. Plain os.Lstat may load
	// its file ID later, after a path replacement, when SameFile is called.
	return root.Lstat(filepath.Base(path))
}

func openLockReadFile(path string) (fs.File, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	// The returned handle is detached from root. The shared regular-file opener
	// also prevents a checked regular file's replacement with a FIFO blocking on Linux.
	return durablefile.OpenRegular(root, filepath.Base(path))
}

func lockReadFailure(ctx context.Context, err error) error {
	if stop := ctx.Err(); stop != nil {
		return errors.Join(err, stop, context.Cause(ctx))
	}
	return err
}

// ReadWithOptions retains Read's required inputs and result shape.
// Cancellation is cooperative between I/O operations, not syscall preemption.
func ReadWithOptions(path string, options ...ReadOptions) (document Document, contents []byte, err error) {
	policy, err := lockReadPolicy(options)
	if err != nil {
		return Document{}, nil, err
	}
	ctx := policy.Context
	if err := lockReadFailure(ctx, nil); err != nil {
		return Document{}, nil, err
	}
	info, err := policy.Lstat(path)
	if err = lockReadFailure(ctx, err); err != nil {
		return Document{}, nil, err
	}
	if info == nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > MaxBytes {
		return Document{}, nil, fmt.Errorf("lockfile is not a bounded regular file")
	}
	file, err := policy.Open(path)
	if file != nil {
		defer func() {
			closeErr := file.Close()
			if closeErr != nil || ctx.Err() != nil {
				err = fmt.Errorf("finish bounded lockfile read: %w", lockReadFailure(ctx, errors.Join(err, closeErr)))
				document, contents = Document{}, nil
			}
		}()
	}
	if err = lockReadFailure(ctx, err); err != nil {
		return Document{}, nil, err
	}
	if file == nil {
		return Document{}, nil, fmt.Errorf("lockfile opener returned no file")
	}
	opened, err := file.Stat()
	if err = lockReadFailure(ctx, err); err != nil {
		return Document{}, nil, err
	}
	if opened == nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return Document{}, nil, fmt.Errorf("lockfile changed during open: %w", durablefile.ErrIdentityChanged)
	}
	if opened.Size() < 0 || opened.Size() > MaxBytes {
		return Document{}, nil, fmt.Errorf("lockfile is not a bounded regular file")
	}
	contents, err = io.ReadAll(io.LimitReader(&lockFileReader{ctx: ctx, reader: file}, MaxBytes+1))
	if err = lockReadFailure(ctx, err); err != nil {
		return Document{}, nil, fmt.Errorf("read bounded lockfile: %w", err)
	}
	if len(contents) > MaxBytes {
		return Document{}, nil, fmt.Errorf("lockfile exceeds %d bytes", MaxBytes)
	}
	document, err = Parse(contents)
	return document, contents, err
}

const lockReadChunkBytes = 32 << 10
const maxLockEmptyReads = 100

type lockFileReader struct {
	ctx        context.Context
	reader     io.Reader
	emptyReads int
}

func (reader *lockFileReader) Read(buffer []byte) (int, error) {
	if err := lockReadFailure(reader.ctx, nil); err != nil {
		return 0, err
	}
	n, err := reader.reader.Read(buffer[:min(len(buffer), lockReadChunkBytes)])
	if stop := reader.ctx.Err(); stop != nil {
		return n, errors.Join(err, stop, context.Cause(reader.ctx))
	}
	if n == 0 && err == nil {
		reader.emptyReads++
		if reader.emptyReads >= maxLockEmptyReads {
			return 0, io.ErrNoProgress
		}
	} else {
		reader.emptyReads = 0
	}
	return n, err
}
