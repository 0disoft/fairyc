package candidatebuild

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/durablefile"
)

// Zero fields retain Background, the host regular-file opener and the 96 MiB
// ceiling. MaxBytes can only lower that ceiling. Open transfers a detached file
// to this reader, including when it returns both a file and an error. Cancellation
// is cooperative between I/O operations; it cannot preempt a blocked syscall.
type cachedPacketReadOptions struct {
	Context  context.Context
	Open     func(string) (fs.File, error)
	MaxBytes int
}

func cachedPacket(path, expected string) ([]byte, error) {
	return cachedPacketWithOptions(path, expected)
}

func openCachedPacket(path string) (fs.File, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := durablefile.OpenRegular(root, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func cachedPacketContextError(ctx context.Context, err error) error {
	if stop := ctx.Err(); stop != nil {
		return errors.Join(err, stop, context.Cause(ctx))
	}
	return err
}

func cachedPacketWithOptions(path, expected string, options ...cachedPacketReadOptions) (data []byte, err error) {
	if len(options) > 1 {
		return nil, fmt.Errorf("at most one cached packet read options value is supported")
	}
	var policy cachedPacketReadOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	if policy.Open == nil {
		policy.Open = openCachedPacket
	}
	if policy.MaxBytes == 0 {
		policy.MaxBytes = maxPacket
	}
	if policy.MaxBytes < 1 || policy.MaxBytes > maxPacket {
		return nil, fmt.Errorf("cached packet byte limit must be in [1,%d]", maxPacket)
	}
	want, err := hex.DecodeString(expected)
	if err != nil || len(want) != sha256.Size || path == "" {
		return nil, fmt.Errorf("packet path and SHA256 digest are required")
	}
	ctx := policy.Context
	if err := cachedPacketContextError(ctx, nil); err != nil {
		return nil, err
	}
	file, err := policy.Open(path)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil || ctx.Err() != nil {
				err = cachedPacketContextError(ctx, errors.Join(err, closeErr))
				data = nil
			}
		}()
	}
	if err = cachedPacketContextError(ctx, err); err != nil {
		return nil, err
	}
	if file == nil {
		return nil, fmt.Errorf("cached packet opener returned no file")
	}
	info, err := file.Stat()
	if err = cachedPacketContextError(ctx, err); err != nil {
		return nil, err
	}
	if info == nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > int64(policy.MaxBytes) {
		return nil, fmt.Errorf("cached packet must be a regular file within %d bytes", policy.MaxBytes)
	}
	reader := &cachedPacketReader{ctx: ctx, reader: file}
	data, err = io.ReadAll(io.LimitReader(reader, int64(policy.MaxBytes)+1))
	if err = cachedPacketContextError(ctx, err); err != nil {
		return nil, err
	}
	if len(data) > policy.MaxBytes {
		return nil, fmt.Errorf("cached packet exceeds %d bytes", policy.MaxBytes)
	}
	got := sha256.Sum256(data)
	if !bytes.Equal(got[:], want) {
		return nil, fmt.Errorf("cached packet SHA256 mismatch")
	}
	return data, nil
}

type cachedPacketReader struct {
	ctx        context.Context
	reader     io.Reader
	emptyReads int
}

func (reader *cachedPacketReader) Read(buffer []byte) (int, error) {
	if err := cachedPacketContextError(reader.ctx, nil); err != nil {
		return 0, err
	}
	n, err := reader.reader.Read(buffer[:min(len(buffer), 32<<10)])
	if stop := reader.ctx.Err(); stop != nil {
		return n, errors.Join(err, stop, context.Cause(reader.ctx))
	}
	if n == 0 && err == nil {
		reader.emptyReads++
		if reader.emptyReads >= 100 {
			return 0, io.ErrNoProgress
		}
	} else {
		reader.emptyReads = 0
	}
	return n, err
}
