package prevrevision

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func revisionFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "--quiet")
	manifest := []byte(`{"cases":[{"id":"fixture","sourceRoot":".","span":{"source":"main.fy","startByte":0,"endByte":6}}]}`)
	for name, data := range map[string][]byte{"one": []byte("one\x00\n"), "two space": []byte("two"), "empty": {}, "main.fy": []byte("module main;\nfn i32 main() { return 0; }\n"), "manifest.json": manifest, "other.json": manifest} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	run("add", ".")
	run("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--quiet", "-m", "fixture")
	return root
}

func TestRefreshFixtureSpansBatchesReadsAndPlansBeforeWriting(t *testing.T) {
	root := revisionFixture(t)
	name := filepath.Join(root, "manifest.json")
	baseline, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.ReplaceAll(baseline, []byte(`"startByte":0`), []byte(`"startByte":12`))
	if err := os.WriteFile(name, changed, 0600); err != nil {
		t.Fatal(err)
	}
	report, err := RefreshFixtureSpans(root, "HEAD", []string{"manifest.json", "other.json"}, true)
	if err != nil || report.Spans != 2 || report.Written != 1 {
		t.Fatalf("refresh = %+v, %v", report, err)
	}
	got, _ := os.ReadFile(name)
	if !bytes.Equal(got, baseline) {
		t.Fatal("refresh did not restore mapped spans")
	}
	if err := os.WriteFile(name, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "other.json"), []byte(`{"cases":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	report, err = RefreshFixtureSpans(root, "HEAD", []string{"manifest.json", "other.json"}, true)
	got, _ = os.ReadFile(name)
	if err == nil || report.Written != 0 || !bytes.Equal(got, changed) {
		t.Fatalf("failed refresh published files: %+v, %v", report, err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, err := RefreshFixtureSpansContext(ctx, root, "HEAD", []string{"manifest.json"}, true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline cause = %v", err)
	}
}

func TestRevisionReaderReusesOneProcessAndFramesBinaryBlobs(t *testing.T) {
	root := revisionFixture(t)
	r := newRevisionReader(context.Background(), root, "HEAD")
	defer r.Close()
	var process *os.Process
	for _, name := range []string{"one", "two space", "empty", "one"} {
		got, err := r.Read(name)
		want, readErr := os.ReadFile(filepath.Join(root, name))
		if err != nil || readErr != nil || !bytes.Equal(got, want) {
			t.Fatalf("blob %s = %q, %v", name, got, err)
		}
		if process == nil {
			process = r.command.Process
		} else if process != r.command.Process {
			t.Fatal("started another Git process")
		}
	}
	if _, err := r.Read("missing"); err == nil {
		t.Fatal("accepted missing blob")
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if !r.command.ProcessState.Exited() {
		t.Fatal("Git process not reaped")
	}
	if _, err := r.Read("one"); err == nil {
		t.Fatal("read after close succeeded")
	}
}

func TestRevisionReaderCancellationReleasesBlockedRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := newRevisionReader(ctx, revisionFixture(t), "HEAD")
	defer r.Close()
	if err := r.start(); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { _, err := r.output.ReadByte(); finished <- err }()
	cancel()
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("blocked read returned no error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled read remained blocked")
	}
	if _, err := r.Read("one"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel cause = %v", err)
	}
	if err := r.Close(); !errors.Is(err, context.Canceled) {
		t.Fatalf("close cause = %v", err)
	}
}

func TestRevisionBlobBoundsAndProtocolFailures(t *testing.T) {
	for _, test := range []struct {
		wire  string
		limit int64
		ok    bool
	}{
		{"oid blob 0\x00\x00", 0, true},
		{"oid blob 3\x00abc\x00", 3, true},
		{"oid blob 3\x00abc\x00", 4, true},
		{"oid blob 3\x00abc\x00", 2, false},
		{"oid blob -1\x00", 8, false},
		{"oid blob 9999999999999999999999999\x00", 8, false},
		{"oid tree 0\x00\x00", 8, false},
		{"name missing\x00", 8, false},
		{"oid blob 3\x00ab", 8, false},
		{"oid blob 3\x00abc!", 8, false},
		{strings.Repeat("x", 5000), 8, false},
	} {
		_, err := readRevisionBlob(bufio.NewReaderSize(strings.NewReader(test.wire), 4096), test.limit)
		if (err == nil) != test.ok {
			t.Fatalf("protocol %q limit %d: %v", test.wire[:min(len(test.wire), 40)], test.limit, err)
		}
	}
	var diagnostics revisionDiagnostics
	for range 3 {
		fmt.Fprint(&diagnostics, strings.Repeat("x", maxRevisionDiagnostics))
	}
	if len(diagnostics) != maxRevisionDiagnostics {
		t.Fatal("unbounded Git diagnostics")
	}
}
