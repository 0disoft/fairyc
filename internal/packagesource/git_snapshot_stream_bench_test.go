package packagesource

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/0disoft/fairyc/internal/projectmanifest"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Frozen pre-refactor consumer for byte-equivalence and allocation comparison.
// This is a test baseline, never selected by production.
func legacyGitSnapshotBlobs(ctx context.Context, input io.Reader, entries []gitTreeBlob, writer *zip.Writer) ([]byte, error) {
	buffer := &boundedGitBuffer{limit: MaxTotalBytes + MaxFiles*128}
	if _, err := io.Copy(buffer, input); err != nil {
		return nil, err
	}
	blobs := buffer.Bytes()
	var manifestBytes []byte
	expanded := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := bytes.IndexByte(blobs, '\n')
		if end < 0 || end > 128 {
			return nil, fmt.Errorf("invalid Git blob framing")
		}
		header := strings.Split(string(blobs[:end]), " ")
		blobs = blobs[end+1:]
		if len(header) != 3 || header[0] != entry.object || header[1] != "blob" {
			return nil, fmt.Errorf("missing or mismatched Git blob")
		}
		size, err := strconv.ParseUint(header[2], 10, 32)
		if err != nil || strconv.FormatUint(size, 10) != header[2] || size > MaxFileBytes || int(size) > MaxTotalBytes-expanded || uint64(len(blobs)) <= size || blobs[size] != '\n' {
			return nil, fmt.Errorf("Git blob byte bound or framing")
		}
		data := blobs[:size]
		blobs = blobs[size+1:]
		expanded += int(size)
		if entry.name == projectmanifest.FileName {
			manifestBytes = data
		}
		fileHeader := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		fileHeader.SetMode(0644)
		fileHeader.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		file, err := writer.CreateHeader(fileHeader)
		if err != nil {
			return nil, err
		}
		if _, err = file.Write(data); err != nil {
			return nil, err
		}
	}
	if len(blobs) != 0 {
		return nil, fmt.Errorf("trailing Git blob data")
	}
	return manifestBytes, nil
}

type gitSnapshotZeros struct{}

func (gitSnapshotZeros) Read(p []byte) (int, error) { clear(p); return len(p), nil }
func snapshotBlobWorkload(count, size int) ([]gitTreeBlob, io.Reader) {
	var entries []gitTreeBlob
	var readers []io.Reader
	for i := 0; i < count; i++ {
		object := fmt.Sprintf("%040x", i+1)
		entries = append(entries, gitTreeBlob{name: fmt.Sprintf("chunk%d.fy", i), object: object})
		readers = append(readers, strings.NewReader(fmt.Sprintf("%s blob %d\n", object, size)), io.LimitReader(gitSnapshotZeros{}, int64(size)), strings.NewReader("\n"))
	}
	return entries, io.MultiReader(readers...)
}
func BenchmarkGitSnapshotBlobStaging(b *testing.B) {
	for _, tc := range []struct {
		name    string
		consume func(context.Context, io.Reader, []gitTreeBlob, *zip.Writer) ([]byte, error)
	}{{"before", legacyGitSnapshotBlobs}, {"after", writeGitSnapshotBlobs}} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(MaxTotalBytes)
			for i := 0; i < b.N; i++ {
				entries, input := snapshotBlobWorkload(8, MaxFileBytes)
				writer := zip.NewWriter(io.Discard)
				if _, err := tc.consume(b.Context(), input, entries, writer); err != nil {
					b.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
func TestGitSnapshotStreamMatchesPreviousZIP(t *testing.T) {
	object := strings.Repeat("a", 40)
	entries := []gitTreeBlob{{name: "fairy.toml", object: object}, {name: "src/main.fy", object: object}, {name: "empty", object: object}}
	raw := ""
	for _, body := range []string{gitSnapshotFixtureManifest, "fn main() -> i32 { return 1; }\n", ""} {
		raw += fmt.Sprintf("%s blob %d\n%s\n", object, len(body), body)
	}
	var previous []byte
	for i, consume := range []func(context.Context, io.Reader, []gitTreeBlob, *zip.Writer) ([]byte, error){legacyGitSnapshotBlobs, writeGitSnapshotBlobs} {
		var output bytes.Buffer
		writer := zip.NewWriter(&output)
		manifest, err := consume(t.Context(), strings.NewReader(raw), entries, writer)
		if err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if string(manifest) != gitSnapshotFixtureManifest {
			t.Fatal("manifest changed after scratch reuse")
		}
		if i == 0 {
			previous = bytes.Clone(output.Bytes())
		} else if !bytes.Equal(previous, output.Bytes()) {
			t.Fatal("ZIP bytes changed")
		}
	}
}
