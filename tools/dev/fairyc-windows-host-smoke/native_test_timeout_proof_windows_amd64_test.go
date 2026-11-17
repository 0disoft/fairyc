package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStableNativeTestTimeoutProof(t *testing.T) {
	root := t.TempDir()
	relative := filepath.Join("examples", "native_test_timeout_quickstart")
	source := filepath.Join("..", "..", "..", relative)
	if err := os.CopyFS(filepath.Join(root, relative), os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
	executions, err := runStableNativeTestTimeoutProof(root)
	if err != nil {
		t.Fatal(err)
	}
	if executions != 2 {
		t.Fatalf("timeout executions = %d; want development and release", executions)
	}
}
