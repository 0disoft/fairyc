package languageserver

import (
	"slices"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

type syntaxInsertion struct {
	path, relative string
	offset         int
	token          string
}

func singleSyntaxInsertion(path, relative string, diagnostics []diagnostic.Diagnostic) *syntaxInsertion {
	if relative == "" || len(diagnostics) != 1 {
		return nil
	}
	diag := diagnostics[0]
	if diag.Code != "E0007" || diag.Repair != diagnostic.RepairInsert || diag.Span.StartByte != diag.Span.EndByte {
		return nil
	}
	switch diag.Token {
	case ";", ",", ")", "]", "}", ">":
		return &syntaxInsertion{path, relative, diag.Span.StartByte, diag.Token}
	default:
		return nil
	}
}

type codeActionContext struct {
	Diagnostics []lspDiagnostic `json:"diagnostics"`
	Only        []string        `json:"only"`
}

type documentEdit struct {
	TextDocument documentID `json:"textDocument"`
	Edits        []textEdit `json:"edits"`
}

type codeAction struct {
	Title       string          `json:"title"`
	Kind        string          `json:"kind"`
	Diagnostics []lspDiagnostic `json:"diagnostics"`
	Edit        struct {
		DocumentChanges []documentEdit `json:"documentChanges"`
	} `json:"edit"`
}

func (s *server) codeActions(d *document, start, end int, requested codeActionContext) ([]codeAction, error) {
	empty := []codeAction{}
	if !s.versionedEdits || len(requested.Only) > 0 && !slices.Contains(requested.Only, "quickfix") && !slices.Contains(requested.Only, "") {
		return empty, nil
	}
	if err := s.ensureAnalysis(d); err != nil {
		return nil, err
	}
	if d.View == nil || d.View.insertion == nil {
		return empty, nil
	}
	fix := d.View.insertion
	if !sameDocumentPath(fix.path, d.Path) || fix.offset < start || fix.offset > end || len(d.Text) >= s.documentByteLimit(d.URI) {
		return empty, nil
	}
	r := textRange(d.Text, fix.offset, fix.offset)
	if !slices.ContainsFunc(requested.Diagnostics, func(diag lspDiagnostic) bool {
		return diag.Code == "E0007" && diag.Range == r
	}) {
		return empty, nil
	}
	text := make([]byte, 0, len(d.Text)+1)
	text = append(text, d.Text[:fix.offset]...)
	text = append(text, fix.token...)
	text = append(text, d.Text[fix.offset:]...)
	checked, err := frontend.AnalyzeSyntax(source.NewFile(fix.relative, text), source.DefaultLimits())
	if s.ctx != nil && s.ctx.Err() != nil {
		return nil, s.ctx.Err()
	}
	if err != nil || checked.AST == nil {
		return empty, nil
	}
	action := codeAction{Title: "Insert missing " + fix.token, Kind: "quickfix"}
	for _, diag := range d.View.diagnostics[fix.path] {
		if diag.Code == "E0007" && diag.Range == r {
			action.Diagnostics = []lspDiagnostic{diag}
			break
		}
	}
	if len(action.Diagnostics) == 0 {
		return empty, nil
	}
	action.Edit.DocumentChanges = []documentEdit{{
		TextDocument: documentID{URI: d.URI, Version: d.Version},
		Edits:        []textEdit{{Range: r, NewText: fix.token}},
	}}
	return []codeAction{action}, nil
}
