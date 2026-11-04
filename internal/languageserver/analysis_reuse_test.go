package languageserver

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestUnchangedSavesReuseAnalysisAndInvalidateDiskInputs(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	var output bytes.Buffer
	s := server{out: &output, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	text := "module main; fn i32 main() { return 0; }"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": text}})
	d := s.docs[uri]
	first := d.View
	if first.snapshot == nil {
		t.Fatal("analysis has no freshness snapshot")
	}
	for range 3 {
		send(t, &s, 0, "textDocument/didSave", map[string]any{"textDocument": map[string]any{"uri": uri}})
		if d.View != first {
			t.Fatal("unchanged save repeated compilation")
		}
	}
	for _, mutate := range []func(){
		func() {
			if err := os.WriteFile(filepath.Join(root, "fairy.lock"), []byte("changed lock"), 0600); err != nil {
				t.Fatal(err)
			}
		},
		func() {
			if err := os.WriteFile(filepath.Join(root, "src", "closed.fy"), []byte("module closed; fn i32 helper() { return 1; }"), 0600); err != nil {
				t.Fatal(err)
			}
		},
		func() {
			if err := os.WriteFile(filepath.Join(root, "src", "closed.fy"), []byte("module closed; fn i32 helper() { return 2; }"), 0600); err != nil {
				t.Fatal(err)
			}
		},
		func() {
			if err := os.Remove(filepath.Join(root, "src", "closed.fy")); err != nil {
				t.Fatal(err)
			}
		},
		func() {
			path := filepath.Join(root, "fairy.toml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data = bytes.Replace(data, []byte("1.0.0"), []byte("1.0.1"), 1)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		before := d.View
		mutate()
		send(t, &s, 0, "textDocument/didSave", map[string]any{"textDocument": map[string]any{"uri": uri}})
		if d.View == before {
			t.Fatal("disk input change reused stale analysis")
		}
	}
	before := d.View
	send(t, &s, 0, "textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []any{map[string]any{"text": "module main; fn i32 main() { return 2; }"}}})
	if d.View == before {
		t.Fatal("changed overlay reused stale analysis")
	}
}

func TestStandaloneSaveReusesAnalysis(t *testing.T) {
	var output bytes.Buffer
	s := server{out: &output, docs: map[string]*document{}}
	uri := fileURI(filepath.Join(t.TempDir(), "main.fy"))
	send(t, &s, 1, "initialize", map[string]any{})
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": "module main; fn i32 main() { return 0; }"}})
	view := s.docs[uri].View
	send(t, &s, 0, "textDocument/didSave", map[string]any{"textDocument": map[string]any{"uri": uri}})
	if s.docs[uri].View != view {
		t.Fatal("unchanged standalone save repeated compilation")
	}
}
