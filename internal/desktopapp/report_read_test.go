package desktopapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/durablefile"
)

func reportReceipt(t testing.TB, root string, body []byte) []byte {
	t.Helper()
	sum := sha256.Sum256(body)
	data, err := json.Marshal(map[string]any{
		"ok": true, "archive": filepath.Join(root, "dist", "app.zip"),
		"zipSha256": hex.EncodeToString(sum[:]), "exeBytes": 100, "zipBytes": len(body),
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func linkReportDirectory(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err == nil {
		t.Cleanup(func() { _ = os.Remove(link) })
		return
	} else if runtime.GOOS != "windows" {
		t.Skipf("directory symlinks unavailable: %v", err)
	}
	// Junctions work on Windows without the symbolic-link privilege. Only
	// test-owned temporary paths are passed to the native directory-link tool.
	if out, err := exec.Command("cmd.exe", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Skipf("directory junctions unavailable: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = os.Remove(link) })
}

func TestParseReportRejectsLinkedDist(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	body := []byte("outside archive bytes")
	archive := filepath.Join(outside, "app.zip")
	if err := os.WriteFile(archive, body, 0600); err != nil {
		t.Fatal(err)
	}
	linkReportDirectory(t, outside, filepath.Join(root, "dist"))
	report, err := ParseReport(reportReceipt(t, root, body), root)
	if err == nil || report != (Report{}) {
		t.Fatalf("accepted archive outside project through linked dist: report=%+v error=%v", report, err)
	}
	got, readErr := os.ReadFile(archive)
	if readErr != nil || !bytes.Equal(got, body) {
		t.Fatalf("outside archive changed: %q %v", got, readErr)
	}
}

func reportFixture(t *testing.T) (root string, body, receipt []byte) {
	t.Helper()
	root = t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "dist"), 0700); err != nil {
		t.Fatal(err)
	}
	body = []byte("packaged archive bytes")
	if err := os.WriteFile(filepath.Join(root, "dist", "app.zip"), body, 0600); err != nil {
		t.Fatal(err)
	}
	return root, body, reportReceipt(t, root, body)
}

func TestParseReportRejectsDirectoryReplacement(t *testing.T) {
	root, body, receipt := reportFixture(t)
	read := false
	report, err := ParseReportWithOptions(receipt, root, ReportReadOptions{
		OpenDirectory: func(project *os.Root, name string) (*os.Root, error) {
			if err := project.Rename(name, "original-dist"); err != nil {
				return nil, err
			}
			if err := project.Mkdir(name, 0700); err != nil {
				return nil, err
			}
			if err := project.WriteFile(filepath.Join(name, "app.zip"), body, 0600); err != nil {
				return nil, err
			}
			return project.OpenRoot(name)
		},
		Read: func(reader io.Reader, buffer []byte) (int, error) { read = true; return reader.Read(buffer) },
	})
	if !errors.Is(err, durablefile.ErrIdentityChanged) || report != (Report{}) || read {
		t.Fatalf("directory replacement accepted/read: report=%+v read=%v error=%v", report, read, err)
	}
	for _, directory := range []string{"dist", "original-dist"} {
		got, err := os.ReadFile(filepath.Join(root, directory, "app.zip"))
		if err != nil || !bytes.Equal(got, body) {
			t.Fatalf("verification changed %s: %q %v", directory, got, err)
		}
	}
}

func TestParseReportRejectsArchiveLink(t *testing.T) {
	root, body, receipt := reportFixture(t)
	archive := filepath.Join(root, "dist", "app.zip")
	target := filepath.Join(root, "dist", "original.zip")
	if err := os.Rename(archive, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("original.zip", archive); err != nil {
		t.Skipf("file symlinks unavailable: %v", err)
	}
	report, err := ParseReport(receipt, root)
	if !errors.Is(err, durablefile.ErrNotRegularFile) || report != (Report{}) {
		t.Fatalf("linked archive certified: %+v %v", report, err)
	}
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("archive link target changed: %q %v", got, err)
	}
}

func TestParseReportArchiveByteBoundaries(t *testing.T) {
	root, body, receipt := reportFixture(t)
	reachedRead := errors.New("admitted file reached bounded reader")
	// Metadata fixtures test the actual 33 MiB +/- 1 admission boundary
	// without allocating or hashing three archive-sized byte slices.
	for _, size := range []int64{maxReportArchiveBytes - 1, maxReportArchiveBytes, maxReportArchiveBytes + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			if err := os.Truncate(filepath.Join(root, "dist", "app.zip"), size); err != nil {
				t.Fatal(err)
			}
			reads := 0
			report, err := ParseReportWithOptions(receipt, root, ReportReadOptions{Read: func(io.Reader, []byte) (int, error) {
				reads++
				return 0, reachedRead
			}})
			if report != (Report{}) || err == nil || (size <= maxReportArchiveBytes) != errors.Is(err, reachedRead) || reads != intBool(size <= maxReportArchiveBytes) {
				t.Fatalf("size=%d reads=%d report=%+v error=%v", size, reads, report, err)
			}
		})
	}
	// Declared size has the same bound and must reject invalid values before I/O.
	if err := os.WriteFile(filepath.Join(root, "dist", "app.zip"), body, 0600); err != nil {
		t.Fatal(err)
	}
	for _, size := range []int64{-1, 0, 1, maxReportArchiveBytes - 1, maxReportArchiveBytes, maxReportArchiveBytes + 1} {
		var fields map[string]any
		if err := json.Unmarshal(receipt, &fields); err != nil {
			t.Fatal(err)
		}
		fields["zipBytes"] = size
		data, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ParseReportWithOptions(data, root, ReportReadOptions{Read: func(io.Reader, []byte) (int, error) { return 0, reachedRead }})
		if (size > 0 && size <= maxReportArchiveBytes) != errors.Is(err, reachedRead) {
			t.Fatalf("declared size=%d error=%v", size, err)
		}
	}
}

func intBool(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestParseReportReadFailuresAndCancellation(t *testing.T) {
	root, body, receipt := reportFixture(t)
	before := bytes.Clone(receipt)
	readFailure := errors.New("archive I/O failure")
	cause := errors.New("caller stopped verification")
	for _, tc := range []struct {
		name   string
		data   []byte
		err    error
		cancel bool
	}{
		{"partial I/O failure", body[:len(body)-1], readFailure, false},
		{"full I/O failure", body, readFailure, false},
		{"short read", body[:len(body)-1], nil, false},
		{"long read", append(bytes.Clone(body), '!'), nil, false},
		{"wrong hash", bytes.Repeat([]byte{'x'}, len(body)), nil, false},
		{"cancel at final read", body, nil, true},
		{"cancel with I/O failure", body, readFailure, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			report, err := ParseReportWithOptions(receipt, root, ReportReadOptions{Context: ctx, Read: func(_ io.Reader, buffer []byte) (int, error) {
				if tc.cancel {
					cancel(cause)
				}
				if tc.err != nil {
					return copy(buffer, tc.data), tc.err
				}
				return copy(buffer, tc.data), io.EOF
			}})
			if err == nil || report != (Report{}) || tc.err != nil && !errors.Is(err, tc.err) || tc.cancel && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatalf("failure certified archive: report=%+v error=%v", report, err)
			}
		})
	}
	if !bytes.Equal(receipt, before) {
		t.Fatal("verification mutated receipt input")
	}
	got, err := os.ReadFile(filepath.Join(root, "dist", "app.zip"))
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("failed verification changed archive: %q %v", got, err)
	}
}

func TestParseReportRejectsCorruptedArchive(t *testing.T) {
	root, body, receipt := reportFixture(t)
	body[0] ^= 1
	if err := os.WriteFile(filepath.Join(root, "dist", "app.zip"), body, 0600); err != nil {
		t.Fatal(err)
	}
	report, err := ParseReport(receipt, root)
	if report != (Report{}) || err == nil || err.Error() != "packaged archive checksum mismatch" {
		t.Fatalf("corrupted archive certified: %+v %v", report, err)
	}
}

func TestParseReportOptionsAndCloseFailures(t *testing.T) {
	root, body, receipt := reportFixture(t)
	report, err := ParseReportWithOptions(receipt, root, ReportReadOptions{})
	if err != nil || report.Archive != "dist/app.zip" || report.ZipBytes != int64(len(body)) {
		t.Fatalf("default options failed: %+v %v", report, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if report, err := ParseReportWithOptions(receipt, filepath.Join(root, "missing"), ReportReadOptions{Context: ctx}); !errors.Is(err, context.Canceled) || report != (Report{}) {
		t.Fatalf("pre-cancelled verification performed I/O: %+v %v", report, err)
	}
	if _, err := ParseReportWithOptions(receipt, root, ReportReadOptions{}, ReportReadOptions{}); err == nil {
		t.Fatal("multiple options values accepted")
	}
	want := errors.New("directory open failed")
	if report, err := ParseReportWithOptions(receipt, root, ReportReadOptions{OpenDirectory: func(*os.Root, string) (*os.Root, error) { return nil, want }}); !errors.Is(err, want) || report != (Report{}) {
		t.Fatalf("lost directory I/O failure: %+v %v", report, err)
	}
	if _, err := ParseReportWithOptions(receipt, root, ReportReadOptions{OpenDirectory: func(*os.Root, string) (*os.Root, error) { return nil, nil }}); err == nil {
		t.Fatal("missing directory handle accepted")
	}
	closeFailure := errors.New("resource close failed")
	closed := 0
	report, err = ParseReportWithOptions(receipt, root, ReportReadOptions{
		Close: func(resource io.Closer) error {
			closed++
			return errors.Join(resource.Close(), closeFailure)
		},
	})
	if !errors.Is(err, closeFailure) || report != (Report{}) || closed != 3 {
		t.Fatalf("close failure certified archive or stopped cleanup: report=%+v closed=%d error=%v", report, closed, err)
	}
}

func reportArchiveFixture(t testing.TB, size int) (root string, receipt []byte) {
	t.Helper()
	root = t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "dist"), 0700); err != nil {
		t.Fatal(err)
	}
	body := bytes.Repeat([]byte{0xA5}, size)
	if err := os.WriteFile(filepath.Join(root, "dist", "app.zip"), body, 0600); err != nil {
		t.Fatal(err)
	}
	return root, reportReceipt(t, root, body)
}

func TestParseReportStreamsArchiveAtLimit(t *testing.T) {
	root, receipt := reportArchiveFixture(t, maxReportArchiveBytes)
	var expected struct {
		SHA256 string `json:"zipSha256"`
	}
	if err := json.Unmarshal(receipt, &expected); err != nil {
		t.Fatal(err)
	}

	for _, grow := range []bool{false, true} {
		t.Run(fmt.Sprintf("grow=%t", grow), func(t *testing.T) {
			var readBytes int
			report, err := ParseReportWithOptions(receipt, root, ReportReadOptions{
				Read: func(reader io.Reader, buffer []byte) (int, error) {
					if len(buffer) > 32<<10 {
						t.Fatalf("read asked for %d bytes", len(buffer))
					}
					n, err := reader.Read(buffer)
					if grow && err == io.EOF {
						if readBytes != maxReportArchiveBytes || n != 0 || len(buffer) != 1 {
							t.Fatalf("expected a one-byte probe at the limit, got offset=%d read=%d buffer=%d", readBytes, n, len(buffer))
						}
						buffer[0] = 1 // Simulate growth after the file size check.
						n, err = 1, nil
					}
					readBytes += n
					return n, err
				},
			})
			if grow {
				if readBytes != maxReportArchiveBytes+1 || report != (Report{}) || err == nil || err.Error() != "packaged archive checksum mismatch" {
					t.Fatalf("excess stream accepted: bytes=%d report=%+v error=%v", readBytes, report, err)
				}
			} else if readBytes != maxReportArchiveBytes || err != nil || report.ZipBytes != maxReportArchiveBytes || report.SHA256 != expected.SHA256 {
				t.Fatalf("stream differs from whole-file SHA-256: bytes=%d report=%+v error=%v", readBytes, report, err)
			}
		})
	}
}

func BenchmarkParseReportArchive(b *testing.B) {
	for _, mib := range []int{1, maxReportArchiveBytes >> 20} {
		b.Run(fmt.Sprintf("%dMiB", mib), func(b *testing.B) {
			root, receipt := reportArchiveFixture(b, mib<<20)
			b.SetBytes(int64(mib) << 20)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := ParseReport(receipt, root); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
