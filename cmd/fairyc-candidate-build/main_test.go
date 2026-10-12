package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceSelectionRejectsBeforeOutput(t *testing.T) {
	output := filepath.Join(t.TempDir(), "previous.exe")
	old := []byte("previous image")
	if err := os.WriteFile(output, old, 0600); err != nil {
		t.Fatal(err)
	}
	for _, arguments := range [][]string{
		{"--candidate", "must-not-run"},
		{"--candidate", "must-not-run", "--manifest", "m", "--workspace", "w"},
		{"--candidate", "must-not-run", "--manifest", "m", "--locked"},
		{"--packet", "must-not-read", "--packet-sha256", "bad", "--workspace", "w"},
		{"--packet", "must-not-read", "--packet-sha256", "bad", "--locked"},
		{"--workspace", "w"},
	} {
		if code := run(append(arguments, "--output", output)); code != 2 {
			t.Fatalf("invalid selection: %v code=%d", arguments, code)
		}
		got, err := os.ReadFile(output)
		if err != nil || string(got) != string(old) {
			t.Fatalf("invalid selection changed output: %v", err)
		}
	}
}
