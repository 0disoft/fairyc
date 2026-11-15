package sourceedit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/0disoft/fairyc/internal/durablefile"
)

func writeEditStage(file *os.File, data []byte, mode os.FileMode, policy ApplyOptions) (err error) {
	ctx := policy.Context
	defer func() { err = errors.Join(err, policy.Close(file), ctx.Err(), context.Cause(ctx)) }()
	if err := ctx.Err(); err != nil {
		return err
	}
	n, err := policy.Write(file, data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err := errors.Join(err, ctx.Err()); err != nil {
		return err
	}
	if err := errors.Join(policy.Chmod(file, mode), ctx.Err()); err != nil {
		return err
	}
	return policy.Sync(file)
}

// One extra bounded streaming pass verifies the bytes actually on disk, not
// only the immutable overlay that was checked before staging.
func verifyEditStage(ctx context.Context, root *os.Root, name string, identity os.FileInfo, expected []byte) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := durablefile.OpenRegular(root, name)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close(), ctx.Err(), context.Cause(ctx)) }()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if identity == nil || !os.SameFile(identity, info) || info.Size() != int64(len(expected)) {
		return fmt.Errorf("edit staging file identity or size changed: %w", durablefile.ErrIdentityChanged)
	}
	digest := sha256.New()
	reader := &editStageReader{ctx: ctx, input: file}
	n, err := io.CopyBuffer(digest, io.LimitReader(reader, int64(len(expected))+1), make([]byte, 32<<10))
	if err := errors.Join(err, ctx.Err()); err != nil {
		return err
	}
	want := sha256.Sum256(expected)
	if n != int64(len(expected)) || !bytes.Equal(digest.Sum(nil), want[:]) {
		return fmt.Errorf("edit staging bytes changed: %w", durablefile.ErrIdentityChanged)
	}
	return nil
}

type editStageReader struct {
	ctx   context.Context
	input io.Reader
}

func (reader *editStageReader) Read(data []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := reader.input.Read(data)
	if reader.ctx.Err() != nil {
		return n, errors.Join(err, reader.ctx.Err(), context.Cause(reader.ctx))
	}
	return n, err
}

func removeEditStage(root *os.Root, name string, identity os.FileInfo) error {
	current, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if identity == nil || !current.Mode().IsRegular() || !os.SameFile(identity, current) {
		return fmt.Errorf("edit staging path replaced; preserved: %w", durablefile.ErrIdentityChanged)
	}
	return root.Remove(name)
}
