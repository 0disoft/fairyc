package languageserver

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSemanticTokensUseCurrentUTF16Positions(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	if !bytes.Contains(editorResult(t, &out), []byte(`"semanticTokensProvider":{"full":true`)) {
		t.Fatal("semantic token capability missing")
	}
	input := "module main;\r\nconst i32 ANSWER = 42;\r\nfn i32 read(i32 value) { /* 😀 */ return value + ANSWER; }\r\n"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": input,
	}})
	query := func() []int {
		t.Helper()
		out.Reset()
		send(t, &s, 2, "textDocument/semanticTokens/full", map[string]any{"textDocument": documentID{URI: uri}})
		var tokens semanticTokens
		if err := json.Unmarshal(editorResult(t, &out), &tokens); err != nil || tokens.Data == nil || len(tokens.Data)%5 != 0 {
			t.Fatalf("invalid token response: %s (%v)", &out, err)
		}
		return tokens.Data
	}
	data := query()
	position := Position{}
	got := map[Position][3]int{}
	for i := 0; i < len(data); i += 5 {
		if data[i] != 0 {
			position.Character = 0
		}
		position.Line += data[i]
		position.Character += data[i+1]
		got[position] = [3]int{data[i+2], data[i+3], data[i+4]}
	}
	for _, want := range []struct {
		word string
		last bool
		kind tokenKind
		mods int
	}{
		{"ANSWER", false, tokenVariable, tokenDeclaration | tokenReadonly},
		{"read", false, tokenFunction, tokenDeclaration},
		{"value", false, tokenParameter, tokenDeclaration},
		{"value", true, tokenParameter, 0},
		{"ANSWER", true, tokenVariable, tokenReadonly},
	} {
		offset := strings.Index(input, want.word)
		if want.last {
			offset = strings.LastIndex(input, want.word)
		}
		pos := bytePosition([]byte(input), offset)
		if token := got[pos]; token != [3]int{len(want.word), int(want.kind), want.mods} {
			t.Fatalf("%q at %+v has token %+v", want.word, pos, token)
		}
	}
	if len(got) != 5 {
		t.Fatalf("unexpected tokens: %+v", got)
	}
	send(t, &s, 0, "textDocument/didChange", map[string]any{
		"textDocument":   documentID{URI: uri, Version: 2},
		"contentChanges": []map[string]string{{"text": strings.Replace(input, "return value", "return missing", 1)}},
	})
	if tokens := query(); len(tokens) != 0 {
		t.Fatalf("stale tokens after unsaved edit: %v", tokens)
	}
}
