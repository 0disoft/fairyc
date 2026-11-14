package packagesource

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	stdhash "hash"
	"io"
)

// I/O defaults to ReadFull and the supplied ZIP writer. A nil parent context
// means Background. Cancellation is cooperative; blocked host I/O cannot be
// interrupted here. The ZIP timestamp remains fixed, with no ambient clock.
type gitBlobOptions struct {
	ReadFull    func(context.Context, io.Reader, []byte) (int, error)
	CreateEntry func(context.Context, *zip.FileHeader) (io.Writer, error)
}

func resolveGitBlobOptions(writer *zip.Writer, options []gitBlobOptions) (gitBlobOptions, error) {
	if len(options) > 1 {
		return gitBlobOptions{}, fmt.Errorf("at most one Git blob options value is supported")
	}
	var selected gitBlobOptions
	if len(options) == 1 {
		selected = options[0]
	}
	if selected.ReadFull == nil {
		selected.ReadFull = func(_ context.Context, reader io.Reader, data []byte) (int, error) { return io.ReadFull(reader, data) }
	}
	if selected.CreateEntry == nil {
		if writer == nil {
			return selected, fmt.Errorf("Git ZIP writer is missing")
		}
		selected.CreateEntry = func(ctx context.Context, header *zip.FileHeader) (io.Writer, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return writer.CreateHeader(header)
		}
	}
	return selected, nil
}

func verifyGitBlob(ctx context.Context, object string, data []byte) error {
	var digest stdhash.Hash
	switch len(object) {
	case sha1.Size * 2:
		digest = sha1.New()
	case sha256.Size * 2:
		digest = sha256.New()
	default:
		return fmt.Errorf("invalid Git blob object identifier")
	}
	expected, err := canonicalHex(object, digest.Size())
	if err != nil {
		return err
	}
	// Git hashes the object kind and decimal byte count, NUL, then payload.
	// SHA-1 is required for legacy Git identifiers, not publisher authentication.
	fmt.Fprintf(digest, "blob %d%c", len(data), 0)
	for len(data) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		count := min(len(data), 32<<10)
		digest.Write(data[:count])
		data = data[count:]
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !bytes.Equal(digest.Sum(nil), expected) {
		return fmt.Errorf("Git blob object digest mismatch")
	}
	return nil
}

type gitBlobReader struct {
	ctx        context.Context
	input      io.Reader
	emptyReads int
}

func (reader *gitBlobReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	count, err := reader.input.Read(buffer[:min(len(buffer), 32<<10)])
	if canceled := reader.ctx.Err(); canceled != nil {
		return count, canceled
	}
	if count == 0 && err == nil {
		reader.emptyReads++
		if reader.emptyReads >= 100 {
			return 0, io.ErrNoProgress
		}
	} else {
		reader.emptyReads = 0
	}
	return count, err
}
