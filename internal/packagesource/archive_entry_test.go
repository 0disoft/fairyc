package packagesource

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"testing/iotest"
)

func TestArchiveEntryByteAndDeclaredSizeBoundaries(t *testing.T) {
	for _, size := range []int{0, archiveReadChunkBytes - 1, archiveReadChunkBytes, archiveReadChunkBytes + 1} {
		for _, kind := range []string{"budget", "declared"} {
			for _, delta := range []int{-1, 0, 1} {
				if kind == "declared" && size+delta < 0 {
					continue
				}
				t.Run(fmt.Sprintf("%d/%s/%+d", size, kind, delta), func(t *testing.T) {
					data := bytes.Repeat([]byte("x"), size)
					remaining := size
					declared := size
					if kind == "budget" {
						remaining += delta
					} else {
						declared += delta
					}
					file := &zip.File{FileHeader: zip.FileHeader{UncompressedSize64: uint64(declared)}}
					reader := &archiveTestCloser{reader: bytes.NewReader(data)}
					opens := 0
					got, err := readArchiveEntry(file, remaining, archiveReadOptions{Open: func(*zip.File) (io.ReadCloser, error) { opens++; return reader, nil }})
					wantSuccess := kind == "budget" && delta >= 0 || kind == "declared" && delta == 0
					if (err == nil) != wantSuccess {
						t.Fatalf("success=%v error=%v", wantSuccess, err)
					}
					if err != nil && got != nil {
						t.Fatal("failed read returned partial bytes")
					}
					if wantSuccess && !bytes.Equal(got, data) {
						t.Fatal("payload changed")
					}
					if reader.closes != opens || reader.largest > archiveReadChunkBytes || reader.total > max(remaining+1, 0) {
						t.Fatalf("unbounded read or leaked handle: %+v opens=%d", reader, opens)
					}
				})
			}
		}
	}
	for _, remaining := range []int{-1, 0, MaxFileBytes - 1, MaxFileBytes, MaxFileBytes + 1} {
		t.Run(fmt.Sprintf("max-budget/%d", remaining), func(t *testing.T) {
			opens := 0
			file := &zip.File{}
			_, err := readArchiveEntry(file, remaining, archiveReadOptions{Open: func(*zip.File) (io.ReadCloser, error) { opens++; return io.NopCloser(bytes.NewReader(nil)), nil }})
			valid := remaining >= 0 && remaining <= MaxFileBytes
			wantOpens := 0
			if valid {
				wantOpens = 1
			}
			if (err == nil) != valid || opens != wantOpens {
				t.Fatalf("invalid budget admitted: opens=%d err=%v", opens, err)
			}
		})
	}
}

func TestArchiveEntryCancellationAndIOFailures(t *testing.T) {
	failure := errors.New("injected ZIP I/O failure")
	for _, scenario := range []string{"open", "nil-reader", "read", "checksum", "close", "pre-cancel", "cancel-open", "cancel-read", "cancel-last-read", "cancel-close"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			data := bytes.Repeat([]byte("x"), archiveReadChunkBytes+1)
			file := &zip.File{FileHeader: zip.FileHeader{UncompressedSize64: uint64(len(data))}}
			reader := &archiveTestCloser{reader: bytes.NewReader(data)}
			want := failure
			switch scenario {
			case "read":
				reader.reader = iotest.ErrReader(failure)
			case "checksum":
				reader.reader = io.MultiReader(reader.reader, iotest.ErrReader(zip.ErrChecksum))
				want = zip.ErrChecksum
			case "close":
				reader.closeErr = failure
			case "pre-cancel":
				cancel()
				want = context.Canceled
			case "cancel-open":
				want = context.Canceled
			case "cancel-read", "cancel-last-read":
				want = context.Canceled
				source := bytes.NewReader(data)
				reader.reader = archiveReadFunc(func(buffer []byte) (int, error) {
					count, err := source.Read(buffer)
					if scenario == "cancel-read" || source.Len() == 0 {
						cancel()
					}
					return count, err
				})
			case "cancel-close":
				want = context.Canceled
				reader.onClose = cancel
			}
			opens := 0
			got, err := readArchiveEntry(file, len(data), archiveReadOptions{Context: ctx, Open: func(*zip.File) (io.ReadCloser, error) {
				opens++
				if scenario == "open" {
					return nil, failure
				}
				if scenario == "nil-reader" {
					return nil, nil
				}
				if scenario == "cancel-open" {
					cancel()
				}
				return reader, nil
			}})
			if got != nil || err == nil {
				t.Fatalf("failed/canceled read accepted: bytes=%d err=%v", len(got), err)
			}
			if scenario != "nil-reader" && !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			wantCloses := 1
			if scenario == "pre-cancel" || scenario == "open" || scenario == "nil-reader" {
				wantCloses = 0
			}
			if reader.closes != wantCloses {
				t.Fatalf("close calls=%d want=%d", reader.closes, wantCloses)
			}
			if scenario == "pre-cancel" && opens != 0 {
				t.Fatal("canceled request opened ZIP entry")
			}
			if scenario == "cancel-read" && reader.total > archiveReadChunkBytes {
				t.Fatal("cancellation did not stop at read boundary")
			}
		})
	}
}

func TestArchiveEntryNoProgressBoundaries(t *testing.T) {
	for _, empty := range []int{archiveMaxEmptyReads - 1, archiveMaxEmptyReads, archiveMaxEmptyReads + 1} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			calls := 0
			source := bytes.NewReader([]byte("x"))
			reader := &archiveTestCloser{reader: archiveReadFunc(func(buffer []byte) (int, error) {
				calls++
				if calls <= empty {
					return 0, nil
				}
				return source.Read(buffer)
			})}
			file := &zip.File{FileHeader: zip.FileHeader{UncompressedSize64: 1}}
			got, err := readArchiveEntry(file, 1, archiveReadOptions{Open: func(*zip.File) (io.ReadCloser, error) { return reader, nil }})
			if empty < archiveMaxEmptyReads {
				if string(got) != "x" || err != nil {
					t.Fatalf("progress rejected: %q %v", got, err)
				}
			} else if got != nil || !errors.Is(err, io.ErrNoProgress) || calls != archiveMaxEmptyReads {
				t.Fatalf("empty read loop: calls=%d err=%v", calls, err)
			}
			if reader.closes != 1 {
				t.Fatal("reader leaked")
			}
		})
	}
}

// Exercise the real unpack caller: cancel from the last entry's decompressor
// without replacing global ZIP hooks, sleeping, or touching the desktop.
func TestUnpackLastEntryCancellation(t *testing.T) {
	data := archiveBytes(t)
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	last := archive.File[len(archive.File)-1]
	var original []byte
	opened, err := last.Open()
	if err != nil {
		t.Fatal(err)
	}
	original, err = io.ReadAll(opened)
	closeErr := opened.Close()
	if err != nil || closeErr != nil {
		t.Fatal(errors.Join(err, closeErr))
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	archive.RegisterDecompressor(last.Method, func(io.Reader) io.ReadCloser {
		return &archiveTestCloser{reader: bytes.NewReader(original), onClose: cancel}
	})
	// A one-entry slice makes the cancellation boundary unambiguous.
	result, err := unpackArchiveFiles(ctx, []*zip.File{last}, MaxTotalBytes, MaxFiles)
	if !errors.Is(err, context.Canceled) || len(result.files) != 0 {
		t.Fatalf("last-entry cancellation accepted: files=%d err=%v", len(result.files), err)
	}
}

type archiveReadFunc func([]byte) (int, error)

func (read archiveReadFunc) Read(buffer []byte) (int, error) { return read(buffer) }

type archiveTestCloser struct {
	reader                 io.Reader
	closeErr               error
	onClose                func()
	closes, total, largest int
}

func (reader *archiveTestCloser) Read(buffer []byte) (int, error) {
	reader.largest = max(reader.largest, len(buffer))
	count, err := reader.reader.Read(buffer)
	reader.total += count
	return count, err
}
func (reader *archiveTestCloser) Close() error {
	reader.closes++
	if reader.onClose != nil {
		reader.onClose()
	}
	return reader.closeErr
}
