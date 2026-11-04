package languageserver

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

func TestDocumentAliasesCannotReplaceAnOpenOverlay(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	aliases := []string{strings.Replace(uri, "file:///", "file://localhost/", 1), strings.Replace(uri, "main.fy", "%6dain.fy", 1)}
	if runtime.GOOS == "windows" {
		aliases = append(aliases, strings.Replace(uri, "main.fy", "MAIN.fy", 1))
	}
	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			var output bytes.Buffer
			s := server{out: &output, docs: map[string]*document{}}
			send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
			good := "module main; fn i32 main() { return 0; }"
			send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": good}})
			original := s.docs[uri]
			view := original.View
			output.Reset()
			send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": alias, "languageId": "fairyc", "version": 99, "text": "invalid alternate buffer"}})
			if len(s.docs) != 1 || s.docs[uri] != original || original.View != view || string(original.Text) != good || original.Version != 1 {
				t.Fatal("alias changed the original document or its analysis")
			}
			if !strings.Contains(output.String(), "document already open") {
				t.Fatalf("alias rejection = %s", output.String())
			}
		})
	}
}
