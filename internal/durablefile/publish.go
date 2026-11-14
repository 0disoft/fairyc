// Package durablefile owns opened-root-contained publication of immutable derived files.
// It provides crash-safe observation boundaries, not universal physical-media retention.
package durablefile

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateTemporary creates one permission-restricted exclusive temporary inside directory.
// The caller owns file closure and best-effort path removal on every outcome.
func CreateTemporary(root *os.Root, directory string) (*os.File, string, error) {
	if root == nil {
		return nil, "", fmt.Errorf("durable file root must be initialized")
	}
	if !validRelativePath(directory) {
		return nil, "", fmt.Errorf("durable file directory must be canonical and nonempty")
	}
	for range 100 {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return nil, "", err
		}
		name := filepath.Join(directory, ".tmp-"+hex.EncodeToString(nonce[:]))
		file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return file, name, err
	}
	return nil, "", fmt.Errorf("exhausted durable file temporary names")
}

// PublishImmutable publishes fully written, synced, and closed expected bytes through a
// same-directory opened-root rename. An existing identical destination converges. Any different
// bytes under the same content-derived name are replaced and independently reverified by readers.
func PublishImmutable(root *os.Root, temporaryPath, finalPath string, expected []byte) error {
	return PublishImmutableWithOptions(root, temporaryPath, finalPath, expected)
}

// Retain the private action adapter used by existing concurrent-publication
// tests. Default publication uses host actions; options may override them.
type publicationAction func(*os.Root, string, string) error

func publishImmutable(root *os.Root, temporaryPath, finalPath string, expected []byte, move, flush publicationAction) error {
	return PublishImmutableWithOptions(root, temporaryPath, finalPath, expected, PublicationOptions{Move: publicationActionWithContext(move), Flush: publicationActionWithContext(flush)})
}

// PublishImmutableWithOptions preserves the default publication contract while
// allowing deterministic I/O failures and cooperative cancellation.
func PublishImmutableWithOptions(root *os.Root, temporaryPath, finalPath string, expected []byte, options ...PublicationOptions) error {
	if root == nil {
		return fmt.Errorf("durable file root must be initialized")
	}
	if temporaryPath == "" || finalPath == "" || len(expected) == 0 {
		return fmt.Errorf("durable file publication requires paths and expected bytes")
	}
	if !validRelativePath(temporaryPath) || !validRelativePath(finalPath) ||
		filepath.Dir(temporaryPath) != filepath.Dir(finalPath) {
		return fmt.Errorf("durable file publication paths must be canonical and share one directory")
	}
	policy, err := publicationOptions(options)
	if err != nil {
		return err
	}
	ctx := policy.Context
	directory := filepath.Dir(finalPath)
	for attempt := 0; attempt < 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		moveErr := policy.Move(ctx, root, temporaryPath, finalPath)
		if err := ctx.Err(); err != nil {
			return errors.Join(moveErr, err)
		}
		if moveErr == nil {
			break
		}
		matched, compareErr := policy.Compare(ctx, root, finalPath, expected)
		if err := errors.Join(compareErr, ctx.Err()); err != nil {
			return fmt.Errorf("inspect durable file destination: %w", errors.Join(moveErr, err))
		}
		if matched {
			break
		}
		if attempt == 1 {
			return fmt.Errorf("publish immutable durable file: destination conflict: %w", moveErr)
		}
		if removeErr := policy.Remove(ctx, root, finalPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("remove conflicting durable file: %w", errors.Join(moveErr, removeErr))
		}
	}
	// Every successful publication, including a concurrent publisher's win,
	// must cross the same host sync boundary before success is reported.
	if err := ctx.Err(); err != nil {
		return err
	}
	if syncErr := policy.Flush(ctx, root, finalPath, directory); syncErr != nil {
		return fmt.Errorf("sync published durable file: %w", syncErr)
	}
	return ctx.Err()
}

func matchesFile(root *os.Root, path string, expected []byte) bool {
	same, err := compareFile(context.Background(), root, path, expected)
	return same && err == nil
}

func compareFile(ctx context.Context, root *os.Root, path string, expected []byte) (same bool, err error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	file, err := OpenRegular(root, path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer func() {
		err = errors.Join(err, file.Close(), ctx.Err())
		if err != nil {
			same = false
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return false, err
	}
	if info.Size() != int64(len(expected)) {
		return false, nil
	}
	return compareReader(ctx, file, expected)
}

const compareChunkBytes = 32 << 10

// Never trust a destination's length to size an allocation. The final byte
// check also rejects a file that grows after matchesFile's size inspection.
func matchesReader(reader io.Reader, expected []byte) bool {
	same, err := compareReader(context.Background(), reader, expected)
	return same && err == nil
}

func compareReader(ctx context.Context, reader io.Reader, expected []byte) (bool, error) {
	checked := &contextComparisonReader{ctx: ctx, reader: reader}
	reader = checked
	chunk := make([]byte, min(len(expected), compareChunkBytes))
	for len(expected) > 0 {
		size := min(len(expected), len(chunk))
		_, err := io.ReadFull(reader, chunk[:size])
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if checked.failure != nil {
			return false, checked.failure
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !bytes.Equal(chunk[:size], expected[:size]) {
			return false, nil
		}
		expected = expected[size:]
	}
	var extra [1]byte
	count, err := io.ReadFull(reader, extra[:])
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if checked.failure != nil {
		return false, checked.failure
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	return count == 0 && errors.Is(err, io.EOF), nil
}

type contextComparisonReader struct {
	ctx        context.Context
	reader     io.Reader
	failure    error
	emptyReads int
}

const maxComparisonEmptyReads = 100

func (reader *contextComparisonReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	count, err := reader.reader.Read(buffer)
	if count == 0 && err == nil {
		reader.emptyReads++
		if reader.emptyReads >= maxComparisonEmptyReads {
			err = io.ErrNoProgress
		}
	} else {
		reader.emptyReads = 0
	}
	// ReadFull discards errors accompanying a full buffer. Retain non-EOF
	// failures so an exact chunk boundary cannot authorize convergence/removal.
	if err != nil && !errors.Is(err, io.EOF) {
		reader.failure = err
	}
	return count, err
}

// SyncDirectory persists a newly created directory entry where the host exposes such a contract.
func SyncDirectory(root *os.Root, path string) error {
	if root == nil || !validRelativePath(path) {
		return fmt.Errorf("durable directory sync requires a root and canonical path")
	}
	return syncDirectory(root, path)
}

func validRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) || filepath.VolumeName(value) != "" || filepath.Clean(value) != value {
		return false
	}
	return value != ".." && !strings.HasPrefix(value, ".."+string(filepath.Separator))
}
