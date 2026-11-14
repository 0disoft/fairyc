package packagesource

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testGitBlobObject(data []byte, digits int) string {
	header := []byte(fmt.Sprintf("blob %d%c", len(data), 0))
	object := append(header, data...)
	if digits == 64 {
		sum := sha256.Sum256(object)
		return hex.EncodeToString(sum[:])
	}
	sum := sha1.Sum(object)
	return hex.EncodeToString(sum[:])
}

func TestGitBlobHashAndIdentifierBoundaries(t *testing.T) {
	if err := verifyGitBlob(t.Context(), "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", nil); err != nil {
		t.Fatal(err)
	}
	for _, digits := range []int{40, 64} {
		for _, size := range []int{0, (32 << 10) - 1, 32 << 10, (32 << 10) + 1} {
			data := bytes.Repeat([]byte("x"), size)
			object := testGitBlobObject(data, digits)
			if err := verifyGitBlob(t.Context(), object, data); err != nil {
				t.Fatalf("%d/%d: %v", digits, size, err)
			}
			if size > 0 {
				changed := bytes.Clone(data)
				changed[size-1] = 'y'
				if err := verifyGitBlob(t.Context(), object, changed); err == nil {
					t.Fatal("same-size substitution accepted")
				}
			}
			for _, delta := range []int{-1, 1} {
				invalid := object[:digits-1]
				if delta > 0 {
					invalid = object + "0"
				}
				if err := verifyGitBlob(t.Context(), invalid, data); err == nil {
					t.Fatalf("identifier length %d accepted", len(invalid))
				}
			}
		}
	}
}

func TestGitBlobRejectsMismatchBeforeZIPEntry(t *testing.T) {
	for _, digits := range []int{40, 64} {
		t.Run(fmt.Sprint(digits), func(t *testing.T) {
			object := testGitBlobObject([]byte("original"), digits)
			input := fmt.Sprintf("%s blob 8\nmodified\n", object)
			entries := []gitTreeBlob{{name: "src/main.fy", object: object}}
			created := 0
			_, err := writeGitSnapshotBlobsWithOptions(t.Context(), strings.NewReader(input), entries, nil, gitBlobOptions{CreateEntry: func(context.Context, *zip.FileHeader) (io.Writer, error) { created++; return io.Discard, nil }})
			if err == nil || !strings.Contains(err.Error(), "digest mismatch") || created != 0 {
				t.Fatalf("unverified ZIP entry: created=%d err=%v", created, err)
			}
		})
	}
}

func TestGitBlobInjectedIOAndCancellation(t *testing.T) {
	data := []byte("payload")
	object := testGitBlobObject(data, 40)
	raw := fmt.Sprintf("%s blob %d\n%s\n", object, len(data), data)
	entries := []gitTreeBlob{{name: "src/main.fy", object: object}}
	failure := errors.New("injected blob I/O failure")
	for _, scenario := range []string{"read", "short-read", "create", "write", "short-write", "pre-cancel", "cancel-read", "cancel-create", "cancel-write"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			reads, creates, writes := 0, 0, 0
			if scenario == "pre-cancel" {
				cancel()
			}
			_, err := writeGitSnapshotBlobsWithOptions(ctx, strings.NewReader(raw), entries, nil, gitBlobOptions{
				ReadFull: func(ctx context.Context, reader io.Reader, out []byte) (int, error) {
					reads++
					if scenario == "read" {
						return 0, failure
					}
					if scenario == "short-read" {
						return 0, nil
					}
					count, err := io.ReadFull(reader, out)
					if scenario == "cancel-read" {
						cancel()
					}
					return count, err
				},
				CreateEntry: func(ctx context.Context, header *zip.FileHeader) (io.Writer, error) {
					creates++
					if scenario == "create" {
						return nil, failure
					}
					if scenario == "cancel-create" {
						cancel()
					}
					return gitBlobTestWriter(func(input []byte) (int, error) {
						writes++
						if scenario == "write" {
							return 0, failure
						}
						if scenario == "short-write" {
							return 0, nil
						}
						if scenario == "cancel-write" {
							cancel()
						}
						return len(input), nil
					}), nil
				},
			})
			want := failure
			switch scenario {
			case "short-read":
				want = io.ErrUnexpectedEOF
			case "short-write":
				want = io.ErrShortWrite
			case "pre-cancel", "cancel-read", "cancel-create", "cancel-write":
				want = context.Canceled
			}
			if !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			if scenario == "pre-cancel" && (reads != 0 || creates != 0 || writes != 0) {
				t.Fatal("pre-canceled operation touched I/O")
			}
			if (scenario == "read" || scenario == "short-read" || scenario == "cancel-read") && creates != 0 {
				t.Fatal("rejected blob reached ZIP writer")
			}
			if scenario == "cancel-create" && writes != 0 {
				t.Fatal("canceled entry wrote payload")
			}
		})
	}
}

type gitBlobTestWriter func([]byte) (int, error)

func (write gitBlobTestWriter) Write(data []byte) (int, error) { return write(data) }

func TestGitSnapshotRejectsCorruptedLooseBlob(t *testing.T) {
	directory, commit := snapshotFixture(t)
	object := snapshotGit(t, directory, "rev-parse", commit+":src/main.fy")
	// Valid zlib framing and the original object pathname, but different blob
	// bytes of the same length. Git cat-file does not itself establish identity.
	payload := []byte("fn main() -> i32 { return 2; }\n")
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := fmt.Fprintf(writer, "blob %d%c", len(payload), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, ".git", "objects", object[:2], object[2:])
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, compressed.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := PrepareGitSnapshot(t.Context(), directory, commit)
	if err == nil {
		t.Fatalf("corrupted blob accepted under original commit: archive=%d bytes", len(result.Archive))
	}
	if len(result.Archive) != 0 {
		t.Fatal("rejected object exposed an archive")
	}
}
