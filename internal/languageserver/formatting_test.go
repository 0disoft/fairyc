package languageserver

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"strings"
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

func TestRangeFormattingKeepsTextOutsideSelection(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	if !bytes.Contains(editorResult(t, &out), []byte(`"documentRangeFormattingProvider":true`)) {
		t.Fatal("range formatting capability missing")
	}
	input := "module main;  \r\nfn i32 main() { /* 😀 */ return 0; }  \r\n// keep \r\n"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": input,
	}})
	start := strings.Index(input, "}  ") + 1
	end := strings.Index(input, "// keep")
	selection := textRange([]byte(input), start, end)
	query := func(r Range) {
		t.Helper()
		out.Reset()
		send(t, &s, 2, "textDocument/rangeFormatting", map[string]any{
			"textDocument": documentID{URI: uri},
			"range":        r,
			"options":      map[string]any{"tabSize": 4, "insertSpaces": true},
		})
	}
	query(selection)
	var edits []textEdit
	if err := json.Unmarshal(editorResult(t, &out), &edits); err != nil ||
		!slices.Equal(edits, []textEdit{{Range: selection, NewText: "\n"}}) {
		t.Fatalf("range edit = %+v (%v)", edits, err)
	}
	if got := input[:start] + edits[0].NewText + input[end:]; got !=
		"module main;  \r\nfn i32 main() { /* 😀 */ return 0; }\n// keep \r\n" {
		t.Fatalf("unexpected formatted source %q", got)
	}
	if d := s.docs[uri]; string(d.Text) != input || d.Version != 1 {
		t.Fatal("range formatting changed the open text or version")
	}
	query(Range{Start: selection.Start, End: selection.Start})
	if got := string(editorResult(t, &out)); got != "[]" {
		t.Fatalf("empty selection returned %s", got)
	}
	midSurrogate := bytePosition([]byte(input), strings.Index(input, "😀"))
	midSurrogate.Character++
	query(Range{Start: midSurrogate, End: selection.End})
	if !strings.Contains(out.String(), `"code":-32602`) {
		t.Fatal("range formatting accepted the middle of a surrogate pair")
	}
}
