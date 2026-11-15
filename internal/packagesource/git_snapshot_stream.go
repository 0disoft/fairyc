package packagesource

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

func streamGitObjectCommand(ctx context.Context, repository string, input []byte, consume func(io.Reader) error, args ...string) error {
	processContext, cancel := context.WithCancel(ctx)
	defer cancel()
	return consumeGitProcess(configuredGitObjectCommand(processContext, repository, input, args...), cancel, consume, gitProcessOptions{Context: ctx})
}

func writeGitSnapshotBlobs(ctx context.Context, input io.Reader, entries []gitTreeBlob, writer *zip.Writer) ([]byte, error) {
	return writeGitSnapshotBlobsWithOptions(ctx, input, entries, writer)
}

func writeGitSnapshotBlobsWithOptions(ctx context.Context, input io.Reader, entries []gitTreeBlob, writer *zip.Writer, options ...gitBlobOptions) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	policy, err := resolveGitBlobOptions(writer, options)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReaderSize(&gitBlobReader{ctx: ctx, input: input}, 128)
	var scratch, manifest []byte
	expanded := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, err := reader.ReadSlice('\n')
		if err != nil {
			return nil, fmt.Errorf("invalid Git blob framing: %w", err)
		}
		header := strings.Split(string(line[:len(line)-1]), " ")
		if len(header) != 3 || header[0] != entry.object || header[1] != "blob" {
			return nil, fmt.Errorf("missing or mismatched Git blob")
		}
		size, err := strconv.ParseUint(header[2], 10, 32)
		if err != nil || strconv.FormatUint(size, 10) != header[2] || size > MaxFileBytes || int(size) > MaxTotalBytes-expanded {
			return nil, fmt.Errorf("Git blob byte bound or framing")
		}
		if entry.name == projectmanifest.FileName && size > projectmanifest.MaxBytes {
			return nil, fmt.Errorf("Git package manifest byte bound")
		}
		if cap(scratch) < int(size) {
			scratch = make([]byte, max(int(size), min(MaxFileBytes, 2*cap(scratch))))
		}
		data := scratch[:int(size)]
		count, err := policy.ReadFull(ctx, reader, data)
		if canceled := ctx.Err(); canceled != nil {
			return nil, canceled
		}
		if err != nil {
			return nil, fmt.Errorf("truncated Git blob: %w", err)
		}
		if count != len(data) {
			return nil, fmt.Errorf("truncated Git blob: %w", io.ErrUnexpectedEOF)
		}
		terminator, err := reader.ReadByte()
		if canceled := ctx.Err(); canceled != nil {
			return nil, canceled
		}
		if err != nil {
			return nil, fmt.Errorf("read Git blob terminator: %w", err)
		}
		if terminator != '\n' {
			return nil, fmt.Errorf("invalid Git blob terminator")
		}
		if err := verifyGitBlob(ctx, entry.object, data); err != nil {
			return nil, err
		}
		expanded += int(size)
		if entry.name == projectmanifest.FileName {
			manifest = append([]byte(nil), data...)
		}
		fileHeader := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		fileHeader.SetMode(0644)
		fileHeader.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		file, err := policy.CreateEntry(ctx, fileHeader)
		if canceled := ctx.Err(); canceled != nil {
			return nil, canceled
		}
		if err != nil {
			return nil, err
		}
		if file == nil {
			return nil, fmt.Errorf("Git ZIP entry writer is missing")
		}
		count, err = file.Write(data)
		if canceled := ctx.Err(); canceled != nil {
			return nil, canceled
		}
		if err != nil {
			return nil, err
		}
		if count != len(data) {
			return nil, io.ErrShortWrite
		}
	}
	_, err = reader.ReadByte()
	if canceled := ctx.Err(); canceled != nil {
		return nil, canceled
	}
	if err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("read Git blob tail: %w", err)
		}
		return nil, fmt.Errorf("trailing Git blob data or read failure")
	}
	return manifest, nil
}
