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

func TestCreateWritesExactManifestSourceTestAndIgnoreFiles(t *testing.T) {
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
	wantPaths := []string{".", ".gitignore", "fairy.toml", "src", "src/main.fy", "tests", "tests/main.fy"}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("scaffold paths = %q, want %q", paths, wantPaths)
	}

	manifest := readScaffoldFile(t, root, projectmanifest.FileName)
	wantManifest := `schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/0"
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
