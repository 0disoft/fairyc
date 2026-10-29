package cli

import (
	"bytes"
	"context"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func checkLinuxNew(t *testing.T, artifact cdriver.NativeArtifact, backend []byte) {
	t.Helper()
	install, parent, tmp := t.TempDir(), filepath.Join(t.TempDir(), "parent 한글 space"), t.TempDir()
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	image := filepath.Join(install, "fairyc")
	if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
		t.Fatal(err)
	}
	run := func(cwd string, args []string, want int) (string, string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		command := exec.CommandContext(ctx, image, args...)
		command.Dir = cwd
		command.Env = append(os.Environ(), "TMPDIR="+tmp)
		command.WaitDelay = 2 * time.Second
		var stdout, stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		err := command.Run()
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != want {
			t.Fatalf("Linux new pipeline %v code=%d want=%d stdout=%q stderr=%q", args, code, want, stdout.String(), stderr.String())
		}
		entries, err := os.ReadDir(tmp)
		if err != nil || len(entries) != 0 {
			t.Fatalf("new pipeline scratch leak: %v %v", entries, err)
		}
		return stdout.String(), stderr.String()
	}
	args := []string{"new", "hello-fairy", "--uid", "example.hello-fairy"}
	out, err := run(parent, args, 0)
	reference := t.TempDir()
	code, refOut, refErr := runForTest(args, reference)
	if code != 0 || out != refOut || err != refErr {
		t.Fatalf("new reference: (%q,%q) vs (%d,%q,%q)", out, err, code, refOut, refErr)
	}
	root := filepath.Join(parent, "hello-fairy")
	paths := []string{".gitignore", "README.md", "fairy.toml", "src/main.fy", "tests/main.fy"}
	for _, path := range paths {
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join(reference, "hello-fairy", filepath.FromSlash(path)))
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatal("starter bytes differ", path, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, "fairy.lock")); !os.IsNotExist(err) {
		t.Fatal("new created a lock", err)
	}
	run(parent, args, 7)
	for _, args := range [][]string{
		{"new"}, {"new", "name"}, {"new", "../escape", "--uid", "example.good"},
		{"new", "Upper", "--uid", "example.good"}, {"new", "nul", "--uid", "example.good"},
		{"new", "valid", "--uid", "bad"}, {"new", "valid", "--uid", ""},
		{"new", "valid", "--uid", "example.good", "--force"},
	} {
		before, err := os.ReadDir(parent)
		if err != nil {
			t.Fatal(err)
		}
		run(parent, args, 2)
		after, err := os.ReadDir(parent)
		if err != nil || len(before) != len(after) {
			t.Fatal("invalid new mutated parent", err)
		}
	}
	writeTestSource(t, parent, "present", "sentinel")
	writeTestSource(t, parent, "directory/marker", "sentinel")
	outside := t.TempDir()
	writeTestSource(t, outside, "marker", "outside")
	if err := os.Symlink(outside, filepath.Join(parent, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("absent", filepath.Join(parent, "dangling")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"present", "directory", "link", "dangling"} {
		before, err := os.Lstat(filepath.Join(parent, name))
		if err != nil {
			t.Fatal(err)
		}
		run(parent, []string{"new", name, "--uid", "example.good"}, 7)
		after, err := os.Lstat(filepath.Join(parent, name))
		if err != nil || !os.SameFile(before, after) {
			t.Fatal("existing target changed", name, err)
		}
	}
	marker, readErr := os.ReadFile(filepath.Join(outside, "marker"))
	if readErr != nil || string(marker) != "outside" {
		t.Fatal("outside target changed", readErr)
	}
	run(root, []string{"check"}, 0)
	run(root, []string{"check", "--locked"}, 0)
	out, err = run(root, []string{"test", "--locked"}, 0)
	if !strings.Contains(out, "test tests::main::answer_is_42 ... ok") || err != "" {
		t.Fatalf("new semantic starter: %q %q", out, err)
	}
	if err := os.WriteFile(filepath.Join(install, "fairyc-candidate-linux-build"), backend, 0700); err != nil {
		t.Fatal(err)
	}
	run(root, []string{"test", "--native", "--locked"}, 0)
	run(root, []string{"run", "--locked"}, 0)
	run(root, []string{"build", "--release", "--locked"}, 0)
	if _, err := os.Stat(filepath.Join(root, "build", "hello-fairy")); err != nil {
		t.Fatal(err)
	}
}
