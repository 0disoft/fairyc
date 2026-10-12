package candidatebuild

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFreshPacketInputRejectsBeforePublication(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "previous.exe")
	previous := []byte("previous image")
	if err := os.WriteFile(output, previous, 0600); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"empty": nil, "invalid": []byte("not a packet"), "truncated": packetFixture("valid")[:15]} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := BuildInput(Options{Output: output}, path); err == nil {
			t.Fatalf("invalid input admitted: %s", name)
		}
		got, err := os.ReadFile(output)
		if err != nil || !bytes.Equal(got, previous) {
			t.Fatalf("input failure changed prior artifact: %v", err)
		}
	}
	for _, path := range []string{"", root, filepath.Join(root, "missing")} {
		if _, err := readPacketInput(path); err == nil {
			t.Fatalf("nonfile input admitted: %q", path)
		}
	}
	for _, options := range []Options{{Candidate: "must-not-run"}, {Manifest: "m"}, {Workspace: "w"}, {Locked: true}} {
		if _, err := BuildInput(options, "must-not-read"); err == nil {
			t.Fatalf("mixed packet input admitted: %+v", options)
		}
	}
}

func TestFreshPacketInputBuildsWithoutFrontend(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires MSVC")
	}
	root := t.TempDir()
	path := filepath.Join(root, "fresh.bin")
	if err := os.WriteFile(path, packetFixture("valid"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		output := filepath.Join(root, "program.exe")
		result, err := BuildInput(Options{Output: output, Release: release}, path)
		if err != nil {
			t.Fatal(err)
		}
		image, err := os.ReadFile(output)
		if err != nil || len(image) != result.ImageBytes || !bytes.HasPrefix(image, []byte("MZ")) {
			t.Fatalf("invalid native image: %v", err)
		}
		streams, err := exec.CommandContext(t.Context(), output).CombinedOutput()
		if err != nil || len(streams) != 0 {
			t.Fatalf("fresh packet image failed: %v %s", err, streams)
		}
	}
}
