package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStableNewWin32CreatesChecksAndTestsWithoutGUI(t *testing.T) {
	for _, options := range [][]string{
		{"--uid", "example.desktop", "--template", "win32"},
		{"--template", "win32", "--uid", "example.desktop"},
	} {
		parent := t.TempDir()
		args := append([]string{"new", "desktop"}, options...)
		code, out, diag := runForTest(args, parent)
		if code != ExitSuccess || out != "created Fairy C package \"desktop\" at desktop\n" || diag != "" {
			t.Fatalf("new: %d %s %s", code, out, diag)
		}
		root := filepath.Join(parent, "desktop")
		for _, command := range [][]string{{"check"}, {"check", "--locked"}, {"test", "--locked"}} {
			code, out, diag = runForTest(command, root)
			if code != ExitSuccess || diag != "" {
				t.Fatalf("%v: %d %s %s", command, code, out, diag)
			}
		}
		for _, path := range []string{"win32_app/portable.py", "native/app.ico", "native/app.rc", "fairy.windows.json"} {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestStableNewRejectsInvalidTemplateOptionsBeforeFilesystem(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	for _, args := range [][]string{
		{"new", "app", "--template", "win32"},
		{"new", "app", "--uid", "example.app", "--template"},
		{"new", "app", "--uid", "example.app", "--template", "unknown"},
		{"new", "app", "--uid", "example.app", "--template", ""},
		{"new", "app", "--uid", "example.app", "--uid", "example.other"},
		{"new", "app", "--uid", "example.app", "--force", "win32"},
		{"new", "../escape", "--uid", "example.app", "--template", "win32"},
		{"new", "con", "--uid", "example.app", "--template", "win32"},
		{"new", strings.Repeat("a", 128), "--uid", "example.app", "--template", "win32"},
	} {
		code, out, diag := runForTest(args, missing)
		if code != ExitInvalidInvocation || out != "" || strings.Contains(diag, missing) || strings.Contains(diag, "parent directory") {
			t.Fatalf("%v: %d %s %s", args, code, out, diag)
		}
	}
}

func TestStableNewExplicitConsoleRetainsDefaultBytes(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	for _, entry := range []struct {
		root    string
		options []string
	}{{first, nil}, {second, []string{"--template", "console"}}} {
		args := append([]string{"new", "basic", "--uid", "example.basic"}, entry.options...)
		if code, out, diag := runForTest(args, entry.root); code != 0 || diag != "" {
			t.Fatalf("new: %d %s %s", code, out, diag)
		}
	}
	for _, name := range []string{".gitignore", "README.md", "fairy.toml", "src/main.fy", "tests/main.fy"} {
		a, err := os.ReadFile(filepath.Join(first, "basic", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(second, "basic", filepath.FromSlash(name)))
		if err != nil || string(a) != string(b) {
			t.Fatalf("default starter changed: %s %v", name, err)
		}
	}
}
