package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInventoryDisjointBuckets(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	state := filepath.Join(root, "state")
	binary := filepath.Join(project, "build", "bench.exe")
	files := map[string]string{"build/bench.exe": "123", "build/bench.pdb": "12", "build/main.obj": "1234", "target/debug/incremental/a/query.bin": "12345", ".zig-cache/o/a": "123456"}
	if err := writeSources(project, files); err != nil {
		t.Fatal(err)
	}
	if err := writeSources(state, map[string]string{"shared/entry": "1234567", "tmp/retained": "12345678"}); err != nil {
		t.Fatal(err)
	}
	f, err := inventory(project, state, binary)
	if err != nil {
		t.Fatal(err)
	}
	if f.Binary != 3 || f.Debug != 2 || f.Intermediate != 4 || f.Incremental != 5 || f.LocalCache != 6 || f.SharedCache != 7 || f.Temporary != 8 || f.Total != 35 || f.Files != 7 {
		t.Fatalf("incorrect inventory: %+v", f)
	}
	if f.Total != f.Binary+f.Debug+f.Intermediate+f.Incremental+f.LocalCache+f.SharedCache+f.Temporary {
		t.Fatal("double counted files")
	}
}

func TestSourceIdentityAndEdits(t *testing.T) {
	for _, language := range []string{"fairy", "zig", "rust", "c"} {
		a, _, _ := sourceIdentity(sources(language, 1, 0))
		b, _, _ := sourceIdentity(sources(language, 1, 1))
		c, _, _ := sourceIdentity(sources(language, 1, 5))
		if a == b || a == c || b == c {
			t.Fatalf("source mutation not material for %s", language)
		}
		root := t.TempDir()
		s := sources(language, 1, 0)
		if err := writeSources(root, s); err != nil {
			t.Fatal(err)
		}
		var path string
		for name := range s {
			path = filepath.Join(root, filepath.FromSlash(name))
			break
		}
		before, _ := os.Stat(path)
		if err := writeSources(root, s); err != nil {
			t.Fatal(err)
		}
		after, _ := os.Stat(path)
		if !before.ModTime().Equal(after.ModTime()) {
			t.Fatal("no-op rewrote source files")
		}
	}
}

func TestReference(t *testing.T) {
	// 'a' and '12' are two tokens; '+' is a third. Whitespace is ignored.
	if got := reference([]byte("a 12+"), 0, 1); got != "tokens=3 checksum=583262\n" {
		t.Fatalf("reference changed: %s", got)
	}
	if reference(workloadInput(), 0, 1) == reference(workloadInput(), 1, 1) {
		t.Fatal("edit leaves output unchanged")
	}
}

func TestSelectionAndOutputBoundary(t *testing.T) {
	for _, bad := range []string{"", "rust,rust", "rust,unknown"} {
		if _, err := selection(bad, "rust"); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	root := t.TempDir()
	sentinel := filepath.Join(root, "keep")
	os.WriteFile(sentinel, []byte("keep"), 0o644)
	if err := run(context.Background(), Options{Output: root}); err == nil {
		t.Fatal("reused existing output directory")
	}
	if b, err := os.ReadFile(sentinel); err != nil || string(b) != "keep" {
		t.Fatal("existing output changed")
	}
}

func TestProcessHelper(t *testing.T) {
	if os.Getenv("FAIRYC_BENCH_TEST_HELPER") != "1" {
		return
	}
	if os.Getenv("FAIRYC_BENCH_TEST_SLEEP") == "1" {
		time.Sleep(10 * time.Second)
	}
	os.Stdout.WriteString("ok\n")
	os.Exit(0)
}

func TestCommandTimeoutAndPortableReceipt(t *testing.T) {
	cmd, out, err := execute(context.Background(), 50*time.Millisecond, t.TempDir(), []string{"FAIRYC_BENCH_TEST_HELPER=1", "FAIRYC_BENCH_TEST_SLEEP=1"}, nil, os.Args[0], "-test.run=^TestProcessHelper$")
	if err == nil || !cmd.TimedOut || out != "" {
		t.Fatalf("timeout not detected: %+v %q %v", cmd, out, err)
	}
	root := t.TempDir()
	cmd = portable(Command{Argv: []string{filepath.Join(root, "compiler")}, Env: []string{"TMP=" + root}, Log: root}, root)
	if !strings.Contains(cmd.Argv[0], "$RUN") || cmd.Log != "$RUN" || cmd.Env[0] != "TMP=$RUN" {
		t.Fatal("raw run path leaked")
	}
}
