package languageserver

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func editorResult(t *testing.T, out *bytes.Buffer) json.RawMessage {
	t.Helper()
	_, body, ok := bytes.Cut(out.Bytes(), []byte("\r\n\r\n"))
	var response struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if !ok || json.Unmarshal(body, &response) != nil || response.Result == nil || response.Error != nil {
		t.Fatalf("unexpected LSP response: %s", out.Bytes())
	}
	return response.Result
}

func TestHoverUsesCurrentResolvedSnapshot(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	providerURI := fileURI(filepath.Join(root, "src", "answer.fy"))
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	var init struct {
		Capabilities struct {
			HoverProvider bool `json:"hoverProvider"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(editorResult(t, &out), &init); err != nil || !init.Capabilities.HoverProvider {
		t.Fatal("hover capability missing", err)
	}
	open := func(uri, text string) {
		send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
			"uri": uri, "version": 1, "languageId": "fairyc", "text": text,
		}})
	}
	query := func(uri, text string, offset int) *hoverResult {
		t.Helper()
		out.Reset()
		send(t, &s, 2, "textDocument/hover", documentParams{documentID{URI: uri}, bytePosition([]byte(text), offset)})
		var got *hoverResult
		if err := json.Unmarshal(editorResult(t, &out), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}

	provider := "module answer;\npub fn i32 value(i32 input) { return input; }\nfn i32 unused() { return 0; }\n"
	open(providerURI, provider)
	text := "module main; import answer as answer; fn i32 main() { let i32 selected = 42; /* 😀 */ return answer::value(selected); }"
	open(uri, text)
	start := strings.Index(text, "answer::value")
	wantRange := textRange([]byte(text), start, start+len("answer::value"))
	got := query(uri, text, start+len("answer::"))
	if got == nil || got.Contents != (hoverContents{"plaintext", "pub fn i32 value(i32 input)"}) || got.Range != wantRange {
		t.Fatalf("call hover = %+v, want range %+v", got, wantRange)
	}
	if got := query(uri, text, strings.LastIndex(text, "selected")); got == nil || !strings.HasSuffix(got.Contents.Value, " selected: i32") {
		t.Fatalf("local hover = %+v", got)
	}
	if got := query(providerURI, provider, strings.Index(provider, "unused")); got == nil || got.Contents.Value != "fn i32 unused()" {
		t.Fatalf("unused declaration hover = %+v", got)
	}
	if got := query(uri, text, strings.Index(text, "/*")); got != nil {
		t.Fatalf("comment hover = %+v", got)
	}

	broken := strings.Replace(provider, "return input;", "return }", 1)
	send(t, &s, 0, "textDocument/didChange", map[string]any{
		"textDocument":   documentID{URI: providerURI, Version: 2},
		"contentChanges": []map[string]string{{"text": broken}},
	})
	if got := query(uri, text, start); got != nil {
		t.Fatalf("stale hover = %+v", got)
	}
}
