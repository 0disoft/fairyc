package languageserver

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeActionInsertsVerifiedTokenWithDocumentVersion(t *testing.T) {
	root, _ := editorFixture(t)
	path := filepath.Join(root, "src", "nested", "repair.fy")
	uri := fileURI(path)
	writeProjectRootFile(t, path, "module nested::repair; fn i32 value() { return 0; }")
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"rootUri": fileURI(root),
		"capabilities": map[string]any{
			"workspace":    map[string]any{"workspaceEdit": map[string]bool{"documentChanges": true}},
			"textDocument": map[string]any{"codeAction": map[string]any{"codeActionLiteralSupport": map[string]any{}}},
		},
	})
	if !bytes.Contains(editorResult(t, &out), []byte(`"codeActionProvider":{"codeActionKinds":["quickfix"]}`)) {
		t.Fatal("quick fix capability missing")
	}
	const text = "module nested::repair; fn i32 value() { /* 😀 */ return 42 }"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 3, "languageId": "fairyc", "text": text,
	}})
	d := s.docs[uri]
	if d == nil || d.View == nil || len(d.View.diagnostics[path]) != 1 {
		t.Fatalf("expected a single missing-token diagnostic: %s", &out)
	}
	diagnostics := d.View.diagnostics[path]
	query := func(only []string) []codeAction {
		t.Helper()
		out.Reset()
		send(t, &s, 2, "textDocument/codeAction", map[string]any{
			"textDocument": documentID{URI: uri},
			"range":        diagnostics[0].Range,
			"context":      codeActionContext{Diagnostics: diagnostics, Only: only},
		})
		var actions []codeAction
		if err := json.Unmarshal(editorResult(t, &out), &actions); err != nil || actions == nil {
			t.Fatalf("code actions: %s (%v)", &out, err)
		}
		return actions
	}
	if actions := query([]string{"refactor"}); len(actions) != 0 {
		t.Fatalf("quick fix returned for refactor filter: %+v", actions)
	}
	actions := query([]string{"quickfix"})
	if len(actions) != 1 || actions[0].Kind != "quickfix" || len(actions[0].Edit.DocumentChanges) != 1 {
		t.Fatalf("quick fix missing: %+v", actions)
	}
	edit := actions[0].Edit.DocumentChanges[0]
	offset := strings.Index(text, "42") + 2
	wantRange := textRange(d.Text, offset, offset)
	if edit.TextDocument != (documentID{URI: uri, Version: 3}) || len(edit.Edits) != 1 || edit.Edits[0] != (textEdit{Range: wantRange, NewText: ";"}) {
		t.Fatalf("unexpected version or edit: %+v", edit)
	}
	if string(d.Text) != text || d.Version != 3 {
		t.Fatal("query applied the edit to the live document")
	}
	send(t, &s, 0, "textDocument/didChange", map[string]any{
		"textDocument":   documentID{URI: uri, Version: 4},
		"contentChanges": []map[string]string{{"text": text[:offset] + ";" + text[offset:]}},
	})
	if actions := query(nil); len(actions) != 0 || !d.View.clean {
		t.Fatalf("stale diagnostic returned actions after correction: %+v", actions)
	}
}
