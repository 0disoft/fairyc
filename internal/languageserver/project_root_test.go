package languageserver

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProjectRootFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func projectRootManifest(uid, name, source, extra string) string {
	return "schema = \"fairyc.manifest/v1\"\n\n[package]\nuid = \"" + uid +
		"\"\nname = \"" + name + "\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"" + source + "\"\n\n" + extra
}

func siblingWorkspaceFixture(t *testing.T) (root, app, library string) {
	t.Helper()
	root = t.TempDir()
	app = filepath.Join(root, "a")
	library = filepath.Join(root, "b")
	writeProjectRootFile(t, filepath.Join(root, "fairy.toml"), projectRootManifest(
		"test.lsp-root.root", "root", "src",
		"[workspace]\nmembers = [\"a\", \"b\"]\n\n[dependencies.a]\npath = \"a\"\nversion = \"1.0.0\"\nfeatures = []\n",
	))
	writeProjectRootFile(t, filepath.Join(root, "src", "main.fy"),
		"module main; import a as a; fn i32 main() { return a::value(); }")
	writeProjectRootFile(t, filepath.Join(app, "fairy.toml"), projectRootManifest(
		"test.lsp-root.app", "app", "src",
		"[dependencies.b]\npath = \"b\"\nversion = \"1.0.0\"\nfeatures = []\n",
	))
	writeProjectRootFile(t, filepath.Join(app, "src", "a.fy"),
		"module a; import b as b; pub fn i32 value() { return b::value(); }")
	writeProjectRootFile(t, filepath.Join(library, "fairy.toml"), projectRootManifest(
		"test.lsp-root.library", "library", "src", "",
	))
	writeProjectRootFile(t, filepath.Join(library, "src", "b.fy"),
		"module b; pub fn i32 value() { return 42; }")
	return root, app, library
}

func TestAncestorCatalogServesSiblingMembersInLSP(t *testing.T) {
	root, appDirectory, libraryDirectory := siblingWorkspaceFixture(t)
	appPath := filepath.Join(appDirectory, "src", "a.fy")
	libraryPath := filepath.Join(libraryDirectory, "src", "b.fy")
	appURI, libraryURI := fileURI(appPath), fileURI(libraryPath)
	appText := "module a; import b as b; pub fn i32 value() { return b::value(); }"
	libraryText := "module b; pub fn i32 value() { return 42; }"
	var output bytes.Buffer
	s := server{out: &output, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	if got := s.projectRoot(appPath); got != root {
		t.Fatalf("app project root = %q, want catalog %q", got, root)
	}
	if got := s.projectRoot(libraryPath); got != root {
		t.Fatalf("library project root = %q, want catalog %q", got, root)
	}
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": libraryURI, "languageId": "fairyc", "version": 1, "text": libraryText,
	}})
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": appURI, "languageId": "fairyc", "version": 1, "text": appText,
	}})
	appDocument := s.docs[appURI]
	if appDocument.View == nil || !appDocument.View.clean || appDocument.View.root != root {
		t.Fatalf("catalog analysis = %+v", appDocument.View)
	}
	if len(appDocument.View.files) != 3 {
		t.Fatalf("catalog source views = %d, want root, app and library", len(appDocument.View.files))
	}
	definition := s.definition(appDocument, strings.Index(appText, "::value")+3)
	if len(definition) != 1 || definition[0].URI != libraryURI {
		t.Fatalf("sibling definition = %+v", definition)
	}
	completed := false
	for _, item := range s.completion(appDocument, strings.Index(appText, "::value")+5) {
		if item.Label == "value" {
			completed = true
		}
	}
	if !completed {
		t.Fatal("sibling completion missing")
	}
	unsavedLibrary := "module b; pub fn bool value() { return false; }"
	send(t, &s, 0, "textDocument/didChange", map[string]any{
		"textDocument":   map[string]any{"uri": libraryURI, "version": 2},
		"contentChanges": []any{map[string]any{"text": unsavedLibrary}},
	})
	if len(appDocument.View.diagnostics[appPath]) == 0 {
		t.Fatal("sibling overlay signature change did not update app diagnostics")
	}
	foundOverlay := false
	for _, source := range appDocument.View.files {
		if source.Path == libraryPath && string(source.Text) == unsavedLibrary {
			foundOverlay = true
		}
	}
	if !foundOverlay {
		t.Fatal("catalog analysis did not consume the current sibling overlay")
	}
	disk, err := os.ReadFile(libraryPath)
	if err != nil || string(disk) != libraryText {
		t.Fatalf("LSP changed library source on disk: %q, %v", disk, err)
	}
}

func TestStandaloneNestedProjectKeepsItsOwnScope(t *testing.T) {
	root, _, _ := siblingWorkspaceFixture(t)
	standalone := filepath.Join(root, "standalone")
	writeProjectRootFile(t, filepath.Join(standalone, "fairy.toml"), projectRootManifest(
		"test.lsp-root.standalone", "standalone", "src", "",
	))
	filename := filepath.Join(standalone, "src", "main.fy")
	text := "module main; fn i32 main() { return 7; }"
	writeProjectRootFile(t, filename, text)
	var output bytes.Buffer
	s := server{out: &output, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	if got := s.projectRoot(filename); got != standalone {
		t.Fatalf("standalone project root = %q, want %q", got, standalone)
	}
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": fileURI(filename), "languageId": "fairyc", "version": 1, "text": text,
	}})
	document := s.docs[fileURI(filename)]
	if document.View == nil || !document.View.clean || document.View.root != standalone || len(document.View.files) != 1 {
		t.Fatalf("standalone scope = %+v", document.View)
	}
}

func TestProjectRootDoesNotWidenAboveInitializedFolder(t *testing.T) {
	_, appDirectory, _ := siblingWorkspaceFixture(t)
	filename := filepath.Join(appDirectory, "src", "a.fy")
	var output bytes.Buffer
	s := server{workspaceRoots: []string{appDirectory}, out: &output, docs: map[string]*document{}}
	if got := s.projectRoot(filename); got != appDirectory {
		t.Fatalf("initialized member project root = %q, want %q", got, appDirectory)
	}
}

func TestProjectRootRejectsSymlinkedPackageDirectory(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	writeProjectRootFile(t, filepath.Join(external, "fairy.toml"), projectRootManifest(
		"test.lsp-root.external", "external", "src", "",
	))
	writeProjectRootFile(t, filepath.Join(external, "src", "main.fy"),
		"module main; fn i32 main() { return 0; }")
	linked := filepath.Join(root, "linked")
	if err := os.Symlink(external, linked); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	var output bytes.Buffer
	s := server{workspaceRoots: []string{root}, out: &output, docs: map[string]*document{}}
	if got := s.projectRoot(filepath.Join(linked, "src", "main.fy")); got != "" {
		t.Fatalf("symlinked package selected project root %q", got)
	}
}
