package candidatebuild

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublishWithinReplacesOnlyContainedOutput(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join("build", "program.exe")
	for _, image := range [][]byte{[]byte("first"), []byte("replacement")} {
		if err := PublishWithin(root, output, image); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(root, output))
		if err != nil || string(got) != string(image) {
			t.Fatalf("publication: %q %v", got, err)
		}
	}
	for _, invalid := range []string{"", ".", "../outside.exe", filepath.Join(root, "absolute.exe")} {
		if err := PublishWithin(root, invalid, []byte("bad")); err == nil {
			t.Fatalf("admitted %q", invalid)
		}
	}
	if err := PublishWithin(root, output, nil); err == nil {
		t.Fatal("empty image admitted")
	}
	got, err := os.ReadFile(filepath.Join(root, output))
	if err != nil || string(got) != "replacement" {
		t.Fatalf("prior image changed: %q %v", got, err)
	}
}

func TestPublishWithinRejectsDirectoryEscape(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	target := filepath.Join(outside, "program.exe")
	if err := os.WriteFile(target, []byte("prior"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "build")); err != nil {
		t.Skipf("host cannot create directory symlink: %v", err)
	}
	if err := PublishWithin(root, filepath.Join("build", "program.exe"), []byte("replacement")); err == nil {
		t.Fatal("followed directory outside publication root")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "prior" {
		t.Fatalf("outside image changed: %q %v", got, err)
	}
}
