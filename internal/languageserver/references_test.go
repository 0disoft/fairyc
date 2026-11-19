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
