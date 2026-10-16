package projectscaffold

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

func TestCreateWritesExactGuideManifestSourceTestAndIgnoreFiles(t *testing.T) {
	parent := t.TempDir()
	spec := Spec{Name: "hello-fairy", UID: "example.hello-fairy"}
	if err := Create(parent, spec); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, spec.Name)
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{".", ".gitignore", "README.md", "fairy.toml", "src", "src/main.fy", "tests", "tests/main.fy"}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("scaffold paths = %q, want %q", paths, wantPaths)
	}

	manifest := readScaffoldFile(t, root, projectmanifest.FileName)
	wantManifest := `schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
`
	if manifest != wantManifest {
		t.Fatalf("manifest = %q, want %q", manifest, wantManifest)
	}
	parsed, err := projectmanifest.Parse([]byte(manifest))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Package.Name != spec.Name || parsed.Package.UID != spec.UID ||
		parsed.Package.Version != DefaultVersion || parsed.Package.Language != DefaultLanguage {
		t.Fatalf("parsed manifest = %+v", parsed.Package)
	}
	if got := readScaffoldFile(t, root, "src/main.fy"); got != mainSource {
		t.Fatalf("main source = %q, want %q", got, mainSource)
	}
	if got := readScaffoldFile(t, root, "tests/main.fy"); got != testSource {
		t.Fatalf("test source = %q, want %q", got, testSource)
	}
	if got := readScaffoldFile(t, root, ".gitignore"); got != "/build/\n" {
		t.Fatalf(".gitignore = %q", got)
	}
	wantReadme := `# hello-fairy

A Fairy C starter package.

## Check and test

Run these commands manually from this directory. The first check validates the workspace and
creates ` + "`fairy.lock`" + `.

` + "```text" + `
fairyc check
fairyc test --locked
` + "```" + `

The default test command uses the fast semantic interpreter.

## Run on Windows

With x64 MSVC and a Windows SDK installed:

` + "```text" + `
fairyc test --native --locked
fairyc run --locked
fairyc build --release --locked
.\build\hello-fairy.exe
` + "```" + `

` + "`fairyc test --native`" + ` compiles every test into a separate temporary Windows image.
` + "`fairyc run`" + ` forwards program output and returns ` + "`main`" + `'s ` + "`i32`" + ` as the process status.
The generated starter returns zero after checking that ` + "`answer()`" + ` is 42.

## Run on Linux

With Linux GNU amd64, glibc and GCC or Clang on PATH:

` + "```sh" + `
fairyc test --native --locked
fairyc run --locked
fairyc build --release --locked
` + "```" + `

Run the executable under ` + "`build/`" + ` with this package's name and no extension.
Linux process status preserves the low eight bits of ` + "`main`" + `'s return.
Use default semantic tests for fast checks or ` + "`test --native`" + ` for isolated ELF execution.

## Write to standard output

Replace ` + "`src/main.fy`" + ` with:

` + "```fairyc" + `
module main;

import std::io as io;

@effects(console)
fn i32 main() {
    let ref str message = "Fairy C says hi\n";
    return switch (io::write(message)) {
        case .ok(written) : {
            if (written == message.len) {
                0
            } else {
                1
            }
        }
        case .err(error) : {
            if (error.status != 0 && error.written <= message.len) {
                2
            } else {
                3
            }
        }
    };
}
` + "```" + `

Then refresh the lock and run the program directly:

` + "```text" + `
fairyc check
fairyc run --release --locked
` + "```" + `
`
	if got := readScaffoldFile(t, root, "README.md"); got != wantReadme {
		t.Fatalf("README = %q, want %q", got, wantReadme)
	}
}

func TestCreateRejectsInvalidIdentityWithoutWriting(t *testing.T) {
	for _, test := range []struct {
		name   string
		spec   Spec
		reason FailureReason
	}{
		{name: "name", spec: Spec{Name: "Bad_Name", UID: "example.valid"}, reason: ReasonInvalidName},
		{name: "reserved device name", spec: Spec{Name: "con", UID: "example.valid"}, reason: ReasonInvalidName},
		{name: "uid", spec: Spec{Name: "valid", UID: "single"}, reason: ReasonInvalidUID},
	} {
		t.Run(test.name, func(t *testing.T) {
			parent := t.TempDir()
			err := Create(parent, test.spec)
			var failure *Error
			if !errors.As(err, &failure) || failure.Reason != test.reason {
				t.Fatalf("Create() error = %v, want reason %q", err, test.reason)
			}
			entries, readErr := os.ReadDir(parent)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("invalid scaffold wrote entries: %v", entries)
			}
		})
	}
}

func TestCreateNeverOverwritesExistingTarget(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "existing")
	if err := os.WriteFile(target, []byte("keep-me"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Create(parent, Spec{Name: "existing", UID: "example.existing"})
	var failure *Error
	if !errors.As(err, &failure) || failure.Reason != ReasonTargetExists || failure.Path != "existing" {
		t.Fatalf("Create() error = %v, want target-exists", err)
	}
	if got := readScaffoldFile(t, parent, "existing"); got != "keep-me" {
		t.Fatalf("existing target changed to %q", got)
	}
}

func TestCreatePreservesExistingDirectoriesAndSymlinks(t *testing.T) {
	for _, test := range []struct {
		name      string
		directory bool
		symlink   bool
		dangling  bool
	}{
		{name: "directory", directory: true},
		{name: "live-directory-link", directory: true, symlink: true},
		{name: "dangling-directory-link", directory: true, symlink: true, dangling: true},
		{name: "live-file-link", symlink: true},
		{name: "dangling-file-link", symlink: true, dangling: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			parent := t.TempDir()
			target := filepath.Join(parent, "existing")
			referent := target
			if test.symlink {
				referent = filepath.Join(t.TempDir(), "referent")
			}
			if test.directory {
				if err := os.Mkdir(referent, 0o755); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(referent, []byte("keep-me"), 0o600); err != nil {
				t.Fatal(err)
			}
			if test.symlink {
				// Create the referent first so Windows creates a directory symlink,
				// then remove it to reproduce the dangling-directory-link failure.
				if err := os.Symlink(referent, target); err != nil {
					t.Fatal(err)
				}
				if test.dangling {
					if err := os.Remove(referent); err != nil {
						t.Fatal(err)
					}
					if _, err := os.Stat(target); !errors.Is(err, fs.ErrNotExist) {
						t.Fatalf("link is not dangling: %v", err)
					}
				}
			}
			if test.directory && !test.dangling {
				if err := os.WriteFile(filepath.Join(referent, "keep"), []byte("keep-me"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.Lstat(target)
			if err != nil {
				t.Fatal(err)
			}
			err = Create(parent, Spec{Name: "existing", UID: "example.existing"})
			var failure *Error
			if !errors.As(err, &failure) || failure.Reason != ReasonTargetExists || failure.Path != "existing" {
				t.Errorf("Create() error = %v, want target-exists", err)
			}
			if !errors.Is(err, fs.ErrExist) || err.Error() != `project path "existing" already exists` {
				t.Errorf("Create() error = %v, want canonical existing-target error", err)
			}
			after, err := os.Lstat(target)
			if err != nil {
				t.Fatalf("existing target was removed: %v", err)
			}
			if !os.SameFile(before, after) || before.Mode() != after.Mode() {
				t.Fatal("existing target was replaced or its mode changed")
			}
			if test.symlink {
				if got, err := os.Readlink(target); err != nil || got != referent {
					t.Fatalf("symlink = %q, %v; want %q", got, err, referent)
				}
			}
			if !test.dangling {
				if test.directory {
					entries, err := os.ReadDir(referent)
					if err != nil || len(entries) != 1 || entries[0].Name() != "keep" {
						t.Fatalf("existing directory contents changed: %v, %v", entries, err)
					}
					referent = filepath.Join(referent, "keep")
				}
				if content, err := os.ReadFile(referent); err != nil || string(content) != "keep-me" {
					t.Fatalf("existing contents changed: %q, %v", content, err)
				}
			}
		})
	}
}

func TestCreateErrorsDoNotExposeParentPath(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "missing-parent")
	err := Create(parent, Spec{Name: "hello", UID: "example.hello"})
	var failure *Error
	if !errors.As(err, &failure) || failure.Reason != ReasonParentUnavailable {
		t.Fatalf("Create() error = %v, want parent-unavailable", err)
	}
	if strings.Contains(err.Error(), filepath.ToSlash(parent)) || strings.Contains(err.Error(), parent) {
		t.Fatalf("error leaked parent path: %q", err)
	}
}

func readScaffoldFile(t *testing.T, root, relative string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}
