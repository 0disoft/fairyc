package packagesource

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

type countedGitReader struct {
	input io.Reader
	read  int
}

func (r *countedGitReader) Read(p []byte) (int, error) {
	n, err := r.input.Read(p)
	r.read += n
	return n, err
}

func TestGitSnapshotStreamRejectsBeforeBlobAllocation(t *testing.T) {
	object := strings.Repeat("a", 40)
	for _, tc := range []struct {
		name string
		size int
	}{{"src/large.fy", MaxFileBytes + 1}, {"fairy.toml", projectmanifest.MaxBytes + 1}} {
		input := &countedGitReader{input: io.MultiReader(strings.NewReader(fmt.Sprintf("%s blob %d\n", object, tc.size)), io.LimitReader(gitSnapshotZeros{}, 64<<20))}
		writer := zip.NewWriter(io.Discard)
		if _, err := writeGitSnapshotBlobs(t.Context(), input, []gitTreeBlob{{name: tc.name, object: object}}, writer); err == nil {
			t.Fatal("oversized blob accepted")
		}
		if input.read > 128 {
			t.Fatalf("read oversized body before admission: %d bytes", input.read)
		}
	}
}

func TestGitSnapshotStreamFraming(t *testing.T) {
	object := strings.Repeat("a", 40)
	entry := []gitTreeBlob{{name: "src/main.fy", object: object}}
	for _, raw := range []string{"", strings.Repeat("x", 129), object + " blob 03\nabc\n", object + " blob -1\n", object + " commit 0\n\n", strings.Repeat("b", 40) + " blob 0\n\n", object + " blob 3\nab", object + " blob 3\nabc!", object + " blob 0\n\ntrailing"} {
		if _, err := writeGitSnapshotBlobs(t.Context(), strings.NewReader(raw), entry, zip.NewWriter(io.Discard)); err == nil {
			t.Fatalf("bad framing accepted: %q", raw)
		}
	}
}

func TestGitSnapshotStreamAggregateBudget(t *testing.T) {
	entries, source := snapshotBlobWorkload(9, MaxFileBytes)
	reader := &countedGitReader{input: source}
	if _, err := writeGitSnapshotBlobs(t.Context(), reader, entries, zip.NewWriter(io.Discard)); err == nil {
		t.Fatal("aggregate overflow accepted")
	}
	if reader.read > MaxTotalBytes+9*128 {
		t.Fatalf("ninth payload consumed: %d", reader.read)
	}
}

func TestGitSnapshotStreamReapsRejectedProducer(t *testing.T) {
	if os.Getenv("FAIRYC_TEST_REJECTED_BLOB_PRODUCER") == "exit-after-eof" {
		fmt.Fprintf(os.Stdout, "%s blob 0\n\n", "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391")
		os.Exit(7)
	}
	if os.Getenv("FAIRYC_TEST_REJECTED_BLOB_PRODUCER") == "1" {
		fmt.Fprintf(os.Stdout, "%s blob %d\n", strings.Repeat("a", 40), MaxFileBytes+1)
		io.Copy(os.Stdout, gitSnapshotZeros{})
		os.Exit(0)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGitSnapshotStreamReapsRejectedProducer$")
	command.Env = append(os.Environ(), "FAIRYC_TEST_REJECTED_BLOB_PRODUCER=1")
	command.WaitDelay = time.Second
	err := consumeGitProcess(command, cancel, func(reader io.Reader) error {
		_, err := writeGitSnapshotBlobs(ctx, reader, []gitTreeBlob{{name: "src/main.fy", object: strings.Repeat("a", 40)}}, zip.NewWriter(io.Discard))
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "byte bound") {
		t.Fatalf("expected admission refusal: %v", err)
	}
	if command.ProcessState == nil || command.ProcessState.Success() {
		t.Fatal("rejected producer not killed and waited")
	}
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatal("producer cleanup waited for operation timeout")
	}
}

func TestGitSnapshotStreamPropagatesProducerFailureAfterEOF(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGitSnapshotStreamReapsRejectedProducer$")
	command.Env = append(os.Environ(), "FAIRYC_TEST_REJECTED_BLOB_PRODUCER=exit-after-eof")
	command.WaitDelay = time.Second
	err := consumeGitProcess(command, cancel, func(reader io.Reader) error {
		_, err := writeGitSnapshotBlobs(ctx, reader, []gitTreeBlob{{name: "src/main.fy", object: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"}}, zip.NewWriter(io.Discard))
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "local Git object command failed") {
		t.Fatalf("producer failure accepted after valid EOF: %v", err)
	}
	if command.ProcessState == nil || command.ProcessState.ExitCode() != 7 {
		t.Fatal("producer was not waited")
	}
}
