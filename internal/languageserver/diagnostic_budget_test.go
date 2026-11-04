package languageserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestSemanticDiagnosticBudgetBoundsPublishedNotifications(t *testing.T) {
	root := t.TempDir()
	uri := fileURI(filepath.Join(root, "main.fy"))
	var text strings.Builder
	text.WriteString("module main; fn i32 main() {\n")
	for i := range 3000 {
		fmt.Fprintf(&text, "missing_%d();\n", i)
	}
	text.WriteString("return 0; }")
	var output bytes.Buffer
	s := server{out: &output, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	output.Reset()
	send(t, &s, 0, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": text.String()},
	})
	published, err := readMessage(bufio.NewReader(&output))
	if err != nil || published.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("published notification: %+v, %v", published, err)
	}
	var params struct {
		URI         string          `json:"uri"`
		Diagnostics []lspDiagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(published.Params, &params); err != nil {
		t.Fatal(err)
	}
	if params.URI != uri || len(params.Diagnostics) != 257 {
		t.Fatalf("expected 256 admitted diagnostics plus truncation marker, got %d for %s", len(params.Diagnostics), params.URI)
	}
	markers := 0
	for _, diagnostic := range params.Diagnostics {
		if diagnostic.Code == "E0011" {
			markers++
		}
	}
	if markers != 1 || s.docs[uri].View.clean || s.docs[uri].LastClean != nil {
		t.Fatalf("truncation must publish one marker and stay unsuccessful: markers=%d", markers)
	}
}
