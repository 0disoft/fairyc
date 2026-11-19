package languageserver

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRenameVersionedEditsKeepExactBindings(t *testing.T) {
	root, path := editorFixture(t)
	uri := fileURI(path)
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"rootUri":      fileURI(root),
		"capabilities": map[string]any{"workspace": map[string]any{"workspaceEdit": map[string]bool{"documentChanges": true}}},
	})
	if !bytes.Contains(editorResult(t, &out), []byte(`"renameProvider":{"prepareProvider":true}`)) {
		t.Fatal("rename capability missing")
	}
	const text = "module main;\nfn i32 helper(i32 base) { let i32 chosen = 1; if (true) { let i32 nested = 2; /* 😀 */ return chosen + nested; } return chosen + base; }\nfn i32 main() { return helper(3); }"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 7, "languageId": "fairyc", "text": text,
	}})
	doc := s.docs[uri]
	if doc == nil || doc.View == nil || !doc.View.clean {
		t.Fatalf("fixture did not check cleanly: %s", &out)
	}
	call := func(method, needle, newName string) {
		t.Helper()
		out.Reset()
		send(t, &s, 2, method, map[string]any{
			"textDocument": documentID{URI: uri},
			"position":     bytePosition(doc.Text, strings.Index(text, needle)),
			"newName":      newName,
		})
	}
	call("textDocument/prepareRename", "chosen + nested", "")
	var prep struct {
		Range       Range
		Placeholder string
	}
	if err := json.Unmarshal(editorResult(t, &out), &prep); err != nil || prep.Placeholder != "chosen" ||
		prep.Range.Start != bytePosition(doc.Text, strings.Index(text, "chosen + nested")) {
		t.Fatalf("prepareRename result = %+v (%v)", prep, err)
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		needle, oldName, newName string
		count                    int
	}{
		{"chosen + nested", "chosen", "answer", 3},
		{"helper(3)", "helper", "compute_value", 2},
	} {
		call("textDocument/rename", scenario.needle, scenario.newName)
		var result struct{ DocumentChanges []documentEdit }
		if err := json.Unmarshal(editorResult(t, &out), &result); err != nil || len(result.DocumentChanges) != 1 {
			t.Fatalf("rename result = %+v (%v)", result, err)
		}
		edit := result.DocumentChanges[0]
		if edit.TextDocument != (documentID{URI: uri, Version: 7}) || len(edit.Edits) != scenario.count {
			t.Fatalf("wrong versioned edit = %+v", edit)
		}
		for _, item := range edit.Edits {
			start, ok := positionByte(doc.Text, item.Range.Start)
			end, valid := positionByte(doc.Text, item.Range.End)
			if !ok || !valid || string(doc.Text[start:end]) != scenario.oldName || item.NewText != scenario.newName {
				t.Fatalf("unexpected rename edit %+v", item)
			}
		}
	}
	// This candidate still typechecks, but its first return would resolve
	// to nested's inner declaration instead of the renamed outer binding.
	call("textDocument/rename", "chosen + nested", "nested")
	if !strings.Contains(out.String(), errRenameChangesResolution.Error()) {
		t.Fatalf("capture-changing rename was accepted: %s", &out)
	}
	call("textDocument/prepareRename", "main()", "")
	if got := string(editorResult(t, &out)); got != "null" {
		t.Fatalf("prepareRename allowed entry point: %s", got)
	}
	if current, err := os.ReadFile(path); err != nil || !bytes.Equal(current, disk) || string(doc.Text) != text || doc.Version != 7 {
		t.Fatal("rename changed files or open document state")
	}
}
