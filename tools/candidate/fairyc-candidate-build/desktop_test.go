package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopUtilityRejectsBeforeFilesystem(t *testing.T) {
	root := filepath.Join(t.TempDir(), "must-not-be-created")
	for _, args := range [][]string{nil, {"unknown"}, {"new"}, {"new", "../escape", "test.app"}, {"new", "app", "invalid"}, {"prepare", "extra"}, {"package", "--locked"}, {"package", "--json", "--json"}} {
		var out, diagnostic bytes.Buffer
		if code := runDesktop(args, root, &out, &diagnostic); code != 2 || out.Len() != 0 || !strings.HasPrefix(diagnostic.String(), "fairyc: ") {
			t.Fatalf("%v: code=%d stdout=%q stderr=%q", args, code, out.String(), diagnostic.String())
		}
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Fatal("invalid utility invocation touched the filesystem", err)
		}
	}
}

func TestDesktopUtilityCreatesEmbeddedWin32WithoutOverwrite(t *testing.T) {
	root := t.TempDir()
	var out, diagnostic bytes.Buffer
	args := []string{"new", "desktop-probe", "test.desktop-probe"}
	if code := runDesktop(args, root, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 || out.String() != "created Fairy C package \"desktop-probe\" at desktop-probe\n" {
		t.Fatalf("create: code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
	project := filepath.Join(root, "desktop-probe")
	for _, name := range []string{"fairy.toml", "app.toml", "fairy.windows.json", "prepare_resources.py", "package_release.py"} {
		if info, err := os.Stat(filepath.Join(project, name)); err != nil || !info.Mode().IsRegular() {
			t.Fatal("missing embedded Win32 file", name, err)
		}
	}
	manifest := filepath.Join(project, "fairy.toml")
	before, err := os.ReadFile(manifest)
	if err != nil || !bytes.Contains(before, []byte(`uid = "test.desktop-probe"`)) {
		t.Fatal("wrong project identity", err)
	}
	out.Reset()
	diagnostic.Reset()
	if code := runDesktop(args, root, &out, &diagnostic); code != 7 || out.Len() != 0 {
		t.Fatalf("existing project: code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
	after, err := os.ReadFile(manifest)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("existing project changed", err)
	}
}

func TestDesktopPrepareLeavesOrdinaryProjectsIndependentOfPython(t *testing.T) {
	t.Setenv("PATH", "")
	var out, diagnostic bytes.Buffer
	if code := runDesktop([]string{"prepare"}, t.TempDir(), &out, &diagnostic); code != 0 || out.Len() != 0 || diagnostic.Len() != 0 {
		t.Fatalf("ordinary project preparation: %d %q %q", code, out.String(), diagnostic.String())
	}
}
