package packagesource

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
)

const archiveReadChunkBytes = 32 << 10
const archiveMaxEmptyReads = 100

// Nil dependencies preserve the existing call shape and host ZIP reader. The
// context is cooperative: a blocked Open/Read/Close cannot itself be interrupted.
type archiveReadOptions struct {
	Context context.Context
	Open    func(*zip.File) (io.ReadCloser, error)
}

func readArchiveEntry(f *zip.File, remaining int, options ...archiveReadOptions) ([]byte, error) {
	if len(options) > 1 {
		return nil, fmt.Errorf("at most one ZIP entry read options value is supported")
	}
	var policy archiveReadOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	ctx := policy.Context
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f == nil || remaining < 0 || remaining > MaxFileBytes || f.UncompressedSize64 > uint64(remaining) {
		return nil, fmt.Errorf("ZIP entry byte budget invalid or exhausted")
	}
	if policy.Open == nil {
		policy.Open = func(entry *zip.File) (io.ReadCloser, error) { return entry.Open() }
	}
	file, err := policy.Open(f)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, fmt.Errorf("ZIP entry opener returned no reader")
	}
	reader := &archiveEntryReader{ctx: ctx, reader: file}
	contents, readErr := io.ReadAll(io.LimitReader(reader, int64(remaining)+1))
	closeErr := file.Close()
	if err := errors.Join(readErr, closeErr, ctx.Err()); err != nil {
		return nil, fmt.Errorf("read ZIP entry: %w", err)
	}
	if len(contents) > remaining || uint64(len(contents)) != f.UncompressedSize64 {
		return nil, fmt.Errorf("ZIP entry integrity or byte limit failed")
	}
	return contents, nil
}

type archiveEntryReader struct {
	ctx        context.Context
	reader     io.Reader
	emptyReads int
}

func (reader *archiveEntryReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	count, err := reader.reader.Read(buffer[:min(len(buffer), archiveReadChunkBytes)])
	if canceled := reader.ctx.Err(); canceled != nil {
		return count, errors.Join(err, canceled)
	}
	if count == 0 && err == nil {
		reader.emptyReads++
		if reader.emptyReads >= archiveMaxEmptyReads {
			return 0, io.ErrNoProgress
		}
	} else {
		reader.emptyReads = 0
	}
	return count, err
}
