package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeInvocationPrecedesIO(t *testing.T) {
	directory := t.TempDir()
	valid := []string{"execute", directory, filepath.Join(directory, "missing.fy"), "--engine", "native-graph", "--profile", "low"}
	for _, change := range []struct {
		index int
		text  string
	}{
		{0, "reset"}, {1, ""}, {2, "missing.txt"}, {3, "--backend"}, {4, "go-mir"}, {5, "--release"}, {6, "fast"},
	} {
		args := append([]string(nil), valid...)
		args[change.index] = change.text
		var out, diagnostics bytes.Buffer
		if code := run(args, &out, &diagnostics); code != 2 || out.Len() != 0 {
			t.Fatalf("invocation %v: %d %s", args, code, &diagnostics)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, ctx := range []context.Context{nil, ctx} {
		if code := runContext(ctx, valid, io.Discard, io.Discard); code != 4 {
			t.Fatal("cancelled/nil native context", code)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal("invalid invocation accessed/created store", err, entries)
	}
}
