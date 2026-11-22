package languageserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestReferencesResolveCurrentLocalAndImportedNames(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	providerPath := filepath.Join(root, "src", "answer.fy")
	providerURI := fileURI(providerPath)
	provider := "module answer; pub fn i32 value() { return 42; }"
	if err := os.WriteFile(providerPath, []byte(provider), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	var init struct {
		Capabilities struct {
			ReferencesProvider bool `json:"referencesProvider"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(editorResult(t, &out), &init); err != nil || !init.Capabilities.ReferencesProvider {
		t.Fatal("references capability missing", err)
	}
	text := "module main; import answer as answer; fn i32 unused() { return 0; } fn i32 main() { let i32 value = answer::value(); /* 😀 */ return value + answer::value(); }"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": text,
	}})
	query := func(input string, offset int, include bool) []Location {
		t.Helper()
		out.Reset()
		send(t, &s, 2, "textDocument/references", map[string]any{
			"textDocument": documentID{URI: uri},
			"position":     bytePosition([]byte(input), offset),
			"context":      map[string]bool{"includeDeclaration": include},
		})
		var got []Location
		if err := json.Unmarshal(editorResult(t, &out), &got); err != nil || got == nil {
			t.Fatalf("references result must be an array: %s (%v)", out.Bytes(), err)
		}
		return got
	}
	location := func(uri, input string, start, length int) Location {
		return Location{uri, textRange([]byte(input), start, start+length)}
	}
	first := strings.Index(text, "answer::value")
	second := strings.LastIndex(text, "answer::value")
	calls := []Location{
		location(uri, text, first, len("answer::value")),
		location(uri, text, second, len("answer::value")),
	}
	if got := query(text, second+len("answer::"), false); !slices.Equal(got, calls) {
		t.Fatalf("imported references = %+v, want %+v", got, calls)
	}
	withDecl := append([]Location{location(providerURI, provider, strings.Index(provider, "value"), len("value"))}, calls...)
	if got := query(text, first, true); !slices.Equal(got, withDecl) {
		t.Fatalf("imported references with declaration = %+v, want %+v", got, withDecl)
	}
	local := strings.Index(text, "value =")
	localUse := strings.Index(text, "value +")
	locals := []Location{location(uri, text, local, len("value")), location(uri, text, localUse, len("value"))}
	if got := query(text, local, true); !slices.Equal(got, locals) {
		t.Fatalf("local references = %+v, want %+v", got, locals)
	}
	unused := strings.Index(text, "unused")
	if got := query(text, unused, true); !slices.Equal(got, []Location{location(uri, text, unused, len("unused"))}) {
		t.Fatalf("unused declaration result = %+v", got)
	}
	broken := text[:first+len("answer::")]
	send(t, &s, 0, "textDocument/didChange", map[string]any{
		"textDocument":   documentID{URI: uri, Version: 2},
		"contentChanges": []map[string]string{{"text": broken}},
	})
	if got := query(broken, first, true); len(got) != 0 {
		t.Fatalf("stale references = %+v", got)
	}
}

func TestScratchOverlayPreservesProjectHoverAndReferences(t *testing.T) {
	root, mainPath := editorFixture(t)
	mainURI := fileURI(mainPath)
	providerPath := filepath.Join(root, "src", "answer.fy")
	providerURI := fileURI(providerPath)
	scratchPath := filepath.Join(root, "scratch.fy")
	scratchURI := fileURI(scratchPath)
	const provider = "module answer; pub fn i32 value() { return 42; }"
	const main = "module main; import answer as answer; fn i32 main() { return answer::value(); }"
	writeProjectRootFile(t, providerPath, provider)

	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	open := func(uri, text string) {
		t.Helper()
		out.Reset()
		send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
			"uri": uri, "version": 1, "languageId": "fairyc", "text": text,
		}})
	}
	open(providerURI, provider)
	open(mainURI, main)

	query := func(method string) json.RawMessage {
		t.Helper()
		out.Reset()
		send(t, &s, 2, "textDocument/"+method, map[string]any{
			"textDocument": documentID{URI: mainURI},
			"position":     bytePosition([]byte(main), strings.Index(main, "::value")+2),
			"context":      map[string]bool{"includeDeclaration": true},
		})
		return editorResult(t, &out)
	}
	hover := query("hover")
	var locations []Location
	references := query("references")
	if !bytes.Contains(hover, []byte("value")) || json.Unmarshal(references, &locations) != nil || len(locations) != 2 {
		t.Fatalf("fixture hover = %s, references = %s", hover, references)
	}

	open(scratchURI, "module scratch; fn i32 scratch() { return 0; }")
	diagnostics := s.docs[mainURI].View.diagnostics
	if scratch := diagnostics[scratchPath]; len(scratch) != 1 || scratch[0].Code != "editor-source-not-in-project" ||
		len(diagnostics[mainPath]) != 0 || len(diagnostics[providerPath]) != 0 ||
		!strings.Contains(out.String(), "editor-source-not-in-project") || !strings.Contains(out.String(), scratchURI) {
		t.Fatalf("expected a diagnostic only for scratch: %s", &out)
	}
	if got := query("hover"); !bytes.Equal(got, hover) {
		t.Errorf("hover with scratch open = %s, want %s", got, hover)
	}
	if got := query("references"); !bytes.Equal(got, references) {
		t.Errorf("references with scratch open = %s, want %s", got, references)
	}

	out.Reset()
	send(t, &s, 0, "textDocument/didChange", map[string]any{
		"textDocument":   documentID{URI: providerURI, Version: 2},
		"contentChanges": []map[string]string{{"text": strings.Replace(provider, "return 42", "return false", 1)}},
	})
	if len(s.docs[providerURI].View.result.Diagnostics) == 0 {
		t.Fatal("expected a compiler diagnostic in the dependency")
	}
	if got := query("references"); string(got) != "[]" {
		t.Fatalf("references with a broken dependency = %s, want []", got)
	}
}

func TestReferenceLimitRejectsInsteadOfTruncating(t *testing.T) {
	d := &document{Path: filepath.Join(t.TempDir(), "references.fy"), Text: bytes.Repeat([]byte("a "), maxReferenceLocations+2)}
	key := sourceKey{1, "references.fy"}
	d.View = &analysis{clean: true, files: map[sourceKey]sourceView{key: {Path: d.Path, Text: d.Text}}}
	declaration := hir.Span{Package: 1, Source: key.Path, EndByte: 1}
	target := semantic.SymbolCoordinate{Package: 1, Name: "a", Kind: "binding", Declaration: declaration}
	for i := 1; i <= maxReferenceLocations+1; i++ {
		use := hir.Span{Package: 1, Source: key.Path, StartByte: 2 * i, EndByte: 2*i + 1}
		d.View.result.Resolutions = append(d.View.result.Resolutions, semantic.Resolution{Use: use, Target: target})
	}
	s := &server{}
	if got, err := s.references(d, 2, false); got != nil || !errors.Is(err, errReferenceLimit) {
		t.Fatalf("oversized references returned %d results, error %v", len(got), err)
	}
}

func TestReferencesIncludeUnusedNamedDeclarations(t *testing.T) {
	root, filename := editorFixture(t)
	if err := os.WriteFile(filepath.Join(root, "src", "answer.fy"), []byte("module answer;"), 0644); err != nil {
		t.Fatal(err)
	}
	uri := fileURI(filename)
	var out bytes.Buffer
	s := server{out: &out, workspaceRoots: []string{root}, initialized: true, docs: map[string]*document{}}
	text := `module main;
import answer as unused_module;
struct Point { i32 x_component; }
resource Owned { i32 owned_value; cleanup {} }
enum Event { Ready { i32 ready_value; }; }
@contract_uid("editor.references")
state { cell u64 unused_cell; }
unsafe extern C fn i32 native_check(i32 native_arg);
fn i32 main() { return 0; }
`
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": text,
	}})
	d := s.docs[uri]
	if d == nil || d.View == nil || !d.View.clean {
		t.Fatalf("fixture did not check cleanly: %s", out.String())
	}
	for _, name := range []string{
		"unused_module", "Point", "x_component", "Owned", "owned_value", "Event", "Ready", "ready_value", "unused_cell", "native_arg",
	} {
		t.Run(name, func(t *testing.T) {
			start := strings.Index(text, name)
			got, err := s.references(d, start, true)
			want := []Location{{URI: uri, Range: textRange(d.Text, start, start+len(name))}}
			if err != nil || !slices.Equal(got, want) {
				t.Fatalf("references = %+v (%v), want %+v", got, err, want)
			}
			if got, err := s.references(d, start, false); err != nil || len(got) != 0 {
				t.Fatalf("references with declaration excluded = %+v (%v)", got, err)
			}
		})
	}
}
