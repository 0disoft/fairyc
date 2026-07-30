package architecturecontract

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRepositoryKeepsAtlasImportsOneWay(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	violations, err := CheckAtlasOneWayImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("Atlas import boundary violations = %+v", violations)
	}
}

func TestAtlasImportBoundaryDetectsForbiddenImportsAndKeepsAllowedAdapters(t *testing.T) {
	root := t.TempDir()
	writeArchitectureFile(t, root, "internal/compiler/compiler.go", `package compiler
// github.com/0disoft/fairyc/internal/atlasmap is not an import here.
import "github.com/0disoft/fairyc/internal/contextpack"
`)
	writeArchitectureFile(t, root, "internal/mir/mir_test.go", `package mir
import "github.com/0disoft/fairyc/internal/attributedtext"
`)
	writeArchitectureFile(t, root, "internal/cli/atlas.go", `package cli
import "github.com/0disoft/fairyc/internal/atlasmap"
`)
	writeArchitectureFile(t, root, "internal/contextpack/pack.go", `package contextpack
import "github.com/0disoft/fairyc/internal/semanticsnapshot"
`)
	writeArchitectureFile(t, root, "internal/workspacecallcatalog/catalog.go", `package workspacecallcatalog
import "github.com/0disoft/fairyc/internal/atlasmap"
`)
	writeArchitectureFile(t, root, "internal/runtime/atlas.go", `package runtime
import "github.com/0disoft/fairyc/internal/workspacecallcatalog"
`)
	writeArchitectureFile(t, root, "cmd/fairyc-windows-host-smoke/main.go", `package main
import "github.com/0disoft/fairyc/internal/atlasmap"
`)
	writeArchitectureFile(t, root, "cmd/fairyc/main.go", `package main
import "example.com/github.com/0disoft/fairyc/internal/atlasmap"
`)

	violations, err := CheckAtlasOneWayImports(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []ImportViolation{
		{
			Importer: modulePath + "/internal/compiler",
			Imported: modulePath + "/internal/contextpack",
			File:     "internal/compiler/compiler.go",
		},
		{
			Importer: modulePath + "/internal/mir",
			Imported: modulePath + "/internal/attributedtext",
			File:     "internal/mir/mir_test.go",
		},
		{
			Importer: modulePath + "/internal/runtime",
			Imported: modulePath + "/internal/workspacecallcatalog",
			File:     "internal/runtime/atlas.go",
		},
	}
	if !reflect.DeepEqual(violations, want) {
		t.Fatalf("violations = %+v, want %+v", violations, want)
	}
}

func TestAtlasImportBoundaryRejectsMissingRootsAndMalformedGo(t *testing.T) {
	if _, err := CheckAtlasOneWayImports(""); err == nil || !strings.Contains(err.Error(), "repository root") {
		t.Fatalf("empty-root error = %v", err)
	}
	missingSubtree := t.TempDir()
	if err := os.Mkdir(filepath.Join(missingSubtree, "internal"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckAtlasOneWayImports(missingSubtree); err == nil ||
		!strings.Contains(err.Error(), `repository subtree "cmd"`) || strings.Contains(err.Error(), missingSubtree) {
		t.Fatalf("missing-subtree error = %v", err)
	}
	root := t.TempDir()
	writeArchitectureFile(t, root, "cmd/fairyc/main.go", "package main\n")
	writeArchitectureFile(t, root, "internal/compiler/broken.go", "package compiler\nimport (\n")
	if _, err := CheckAtlasOneWayImports(root); err == nil ||
		!strings.Contains(err.Error(), "internal/compiler/broken.go") || strings.Contains(err.Error(), root) {
		t.Fatalf("malformed-Go error = %v", err)
	}
}

func writeArchitectureFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
