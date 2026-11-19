package languageserver

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestDocumentFormattingReturnsVerifiedOverlayEdit(t *testing.T) {
	root, filename := editorFixture(t)
	disk, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	uri := fileURI(filename)
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	var init struct {
		Capabilities struct {
			Formatting bool `json:"documentFormattingProvider"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(editorResult(t, &out), &init); err != nil || !init.Capabilities.Formatting {
		t.Fatal("formatting capability missing", err)
	}
	input := "module main;  \r\n// 😀\r\nfn i32 main() { return 0; }  "
	formatted := "module main;\n// 😀\nfn i32 main() { return 0; }\n"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": input,
	}})
	query := func() []textEdit {
		t.Helper()
		out.Reset()
		send(t, &s, 2, "textDocument/formatting", map[string]any{
			"textDocument": documentID{URI: uri},
			"options":      map[string]any{"tabSize": 4, "insertSpaces": true},
		})
		var edits []textEdit
		if err := json.Unmarshal(editorResult(t, &out), &edits); err != nil || edits == nil {
			t.Fatalf("formatting result must be an array: %s (%v)", out.Bytes(), err)
		}
		return edits
	}
	want := []textEdit{{Range: textRange([]byte(input), 0, len(input)), NewText: formatted}}
	if edits := query(); !slices.Equal(edits, want) {
		t.Fatalf("edit = %+v, want %+v", edits, want)
	}
	if string(s.docs[uri].Text) != input {
		t.Fatal("formatting changed the server's open document")
	}
	if current, err := os.ReadFile(filename); err != nil || !bytes.Equal(current, disk) {
		t.Fatalf("formatting changed the disk file: %v", err)
	}

	for i, text := range []string{formatted, "module main; fn i32 main() { return"} {
		send(t, &s, 0, "textDocument/didChange", map[string]any{
			"textDocument":   documentID{URI: uri, Version: i + 2},
			"contentChanges": []map[string]string{{"text": text}},
		})
		if edits := query(); len(edits) != 0 {
			t.Fatalf("unchanged or malformed source returned edits: %+v", edits)
		}
	}
}
