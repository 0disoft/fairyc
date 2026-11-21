package languageserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestBorrowMoveDiagnosticPublishesCurrentRelatedLocations(t *testing.T) {
	const text = "module main;\nresource R { cleanup {} }\nfn void observe(ref R value) {}\nfn void take(R value) {}\nfn i32 main() {\n" +
		"    /*😀*/ let R owner = R {};\n" +
		"    /*😀*/ let ref R view = ref owner;\n" +
		"    /*😀*/ take(owner);\n" +
		"    /*😀*/ observe(view);\n" +
		"    return 0;\n}\n"
	for _, name := range []string{"undeclared", "disabled", "enabled"} {
		t.Run(name, func(t *testing.T) {
			root, filename := editorFixture(t)
			uri := fileURI(filename)
			var output bytes.Buffer
			s := server{out: &output, docs: map[string]*document{}}
			init := map[string]any{"rootUri": fileURI(root)}
			if name != "undeclared" {
				init["capabilities"] = map[string]any{
					"textDocument": map[string]any{
						"publishDiagnostics": map[string]bool{"relatedInformation": name == "enabled"},
					},
				}
			}
			send(t, &s, 1, "initialize", init)
			output.Reset()
			send(t, &s, 0, "textDocument/didOpen", map[string]any{
				"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": text},
			})
			published, err := readMessage(bufio.NewReader(&output))
			if err != nil || published.Method != "textDocument/publishDiagnostics" {
				t.Fatalf("notification = %+v, %v", published, err)
			}
			var params struct {
				URI         string          `json:"uri"`
				Version     int             `json:"version"`
				Diagnostics []lspDiagnostic `json:"diagnostics"`
			}
			if err := json.Unmarshal(published.Params, &params); err != nil {
				t.Fatal(err)
			}
			if params.URI != uri || params.Version != 1 || len(params.Diagnostics) != 1 {
				t.Fatalf("unexpected diagnostics: %s", published.Params)
			}

			// The comment prefix occupies 11 UTF-16 units and 13 UTF-8 bytes.
			at := func(line, start, end int) Range {
				return Range{Start: Position{line, start}, End: Position{line, end}}
			}
			diag := params.Diagnostics[0]
			if diag.Code != "E2105" || diag.Range != at(7, 16, 21) {
				t.Fatalf("move diagnostic: %+v", diag)
			}
			if name != "enabled" {
				if bytes.Contains(published.Params, []byte("\"relatedInformation\"")) {
					t.Fatalf("related locations sent to a client without support: %s", published.Params)
				}
				return
			}
			want := []diagnosticRelatedInfo{
				{Location{uri, at(8, 19, 23)}, "Borrow last used here"},
				{Location{uri, at(6, 28, 37)}, "Borrow created here"},
				{Location{uri, at(5, 17, 22)}, "Owner declared here"},
			}
			if !slices.Equal(diag.RelatedInformation, want) {
				t.Fatalf("related locations: got %+v, want %+v", diag.RelatedInformation, want)
			}

			fixed := strings.Replace(text, "take(owner);\n    /*😀*/ observe(view);", "observe(view);\n    /*😀*/ take(owner);", 1)
			output.Reset()
			send(t, &s, 0, "textDocument/didChange", map[string]any{
				"textDocument":   map[string]any{"uri": uri, "version": 2},
				"contentChanges": []map[string]string{{"text": fixed}},
			})
			published, err = readMessage(bufio.NewReader(&output))
			if err != nil || published.Method != "textDocument/publishDiagnostics" {
				t.Fatalf("notification after fix = %+v, %v", published, err)
			}
			if err := json.Unmarshal(published.Params, &params); err != nil || params.Version != 2 || len(params.Diagnostics) != 0 {
				t.Fatalf("diagnostic persisted after the fix: %s, %v", published.Params, err)
			}
		})
	}
}

func TestBorrowMoveRelatedInformationSkipsUnavailableSpans(t *testing.T) {
	_, path := editorFixture(t)
	move := hir.Span{Package: 1, Source: "main.fy", StartByte: 2, EndByte: 4}
	a := analysis{files: map[sourceKey]sourceView{
		{1, "main.fy"}: {Path: path, Text: []byte("abcdef")},
	}}
	got := a.borrowMoveRelatedInfo(semantic.Diagnostic{
		Code: "E2105", Primary: move,
		Related: []semantic.RelatedSpan{
			{Role: "owner", Span: hir.Span{Package: 2, Source: "main.fy", StartByte: 1, EndByte: 2}},
			{Role: "loan", Span: hir.Span{Package: 1, Source: "main.fy", StartByte: 0, EndByte: 7}},
			{Role: "last-use", Span: move},
		},
	})
	if len(got) != 0 {
		t.Fatalf("unavailable or synthetic positions became links: %+v", got)
	}
}
