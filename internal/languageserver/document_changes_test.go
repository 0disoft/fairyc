package languageserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestIncrementalDocumentChangesUseUpdatedUTF16Coordinates(t *testing.T) {
	original := []byte("한😀x\r\nold\rnext\n")
	changes := json.RawMessage(`[
		{"range":{"start":{"line":0,"character":1},"end":{"line":0,"character":3}},"text":"a\nnew"},
		{"range":{"start":{"line":1,"character":3},"end":{"line":2,"character":3}},"text":"!"},
		{"range":{"start":{"line":2,"character":0},"end":{"line":2,"character":4}},"text":"끝"}
	]`)
	got, err := applyDocumentChanges(original, changes, 100, false)
	if err != nil || string(got) != "한a\nnew!\r끝\n" {
		t.Fatalf("changed text: %q (%v)", got, err)
	}
	if string(original) != "한😀x\r\nold\rnext\n" {
		t.Fatal("changed the input snapshot")
	}
}

func TestIncrementalDocumentChangesRejectInvalidBatch(t *testing.T) {
	for _, tc := range []struct {
		name, rejected, problem string
	}{
		{"surrogate", `{"start":{"line":0,"character":1},"end":{"line":0,"character":2}}`, "invalid UTF-16 change range"},
		{"outside", `{"start":{"line":0,"character":0},"end":{"line":1,"character":0}}`, "invalid UTF-16 change range"},
		{"inverted", `{"start":{"line":0,"character":2},"end":{"line":0,"character":0}}`, "invalid UTF-16 change range"},
		{"missing", `{"start":{"line":0},"end":{"line":0,"character":0}}`, "invalid UTF-16 change range"},
		{"null", `null`, "invalid UTF-16 change range"},
		{"size", `{"start":{"line":0,"character":0},"end":{"line":0,"character":0}}`, "editor document limit exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := []byte("😀x")
			changes := json.RawMessage(`[
				{"range":{"start":{"line":0,"character":2},"end":{"line":0,"character":3}},"text":"y"},
				{"range":` + tc.rejected + `,"text":"z"}
			]`)
			got, err := applyDocumentChanges(original, changes, len(original), false)
			if err == nil || err.Error() != tc.problem || got != nil || string(original) != "😀x" {
				t.Fatalf("got text %q, original %q, err %v", got, original, err)
			}
		})
	}
}

func TestIncrementalDocumentChangeOverlayAndRecovery(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	if !strings.Contains(out.String(), `"textDocumentSync":{"change":2,`) {
		t.Fatal("incremental synchronization was not advertised")
	}
	initial := "module main;\nfn i32 main() { /*한😀*/ return 7; }\n"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": initial,
	}})
	d := s.docs[uri]
	value := bytePosition(d.Text, strings.Index(initial, "7"))
	change := func(version int, changes ...any) {
		t.Helper()
		send(t, &s, 0, "textDocument/didChange", map[string]any{
			"textDocument": documentID{URI: uri, Version: version}, "contentChanges": changes,
		})
	}
	edit := func(width int, text string) any {
		return map[string]any{
			"range": Range{Start: value, End: Position{Line: value.Line, Character: value.Character + width}},
			"text":  text,
		}
	}
	// A superseding notification may cancel analysis, but never the accepted text delta.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.ctx = ctx
	raw, err := json.Marshal(map[string]any{
		"textDocument": documentID{URI: uri, Version: 2}, "contentChanges": []any{edit(1, "false")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.handle(message{Method: "textDocument/didChange", Params: raw}); !errors.Is(err, context.Canceled) || d.Version != 2 {
		t.Fatalf("canceled recheck did not retain accepted delta: version %d, %v", d.Version, err)
	}
	s.ctx = context.Background()
	change(3, edit(5, "false"))
	if d.Version != 3 || d.View.clean || len(d.View.diagnostics[filename]) == 0 {
		t.Fatal("second delta did not use the accepted unsaved text for diagnostics")
	}
	change(3, edit(5, "7"))
	if d.Version != 3 || !bytes.Contains(d.Text, []byte("return false")) {
		t.Fatal("stale version changed the overlay")
	}
	change(4, edit(5, "7"), edit(200, "broken"))
	change(5, edit(5, "7"))
	if d.Version != 3 || !d.needsFullText || !bytes.Contains(d.Text, []byte("return false")) {
		t.Fatal("rejected batch committed text or allowed an update on an unknown base")
	}
	out.Reset()
	send(t, &s, 2, "textDocument/formatting", map[string]any{"textDocument": documentID{URI: uri}})
	if !strings.Contains(out.String(), `"code":-32801`) {
		t.Fatal("returned formatting for an unsynchronized document")
	}
	change(6, map[string]string{"text": initial}, edit(1, "9"))
	if d.Version != 6 || d.needsFullText || !d.View.clean || !bytes.Contains(d.Text, []byte("return 9")) {
		t.Fatal("full update did not restore incremental synchronization")
	}
	disk, err := os.ReadFile(filename)
	if err != nil || string(disk) != "module main; fn i32 main() { return 0; }" {
		t.Fatalf("unexpected on-disk source: %q (%v)", disk, err)
	}
}
