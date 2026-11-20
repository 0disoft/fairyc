package languageserver

import (
	"slices"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

type syntaxRepair struct {
	path, relative string
	diagnostic     diagnostic.Diagnostic
	replacement    string
}

func singleSyntaxRepair(path, relative string, diagnostics []diagnostic.Diagnostic) *syntaxRepair {
	if relative == "" || len(diagnostics) != 1 {
		return nil
	}
	diag := diagnostics[0]
	switch diag.Token {
	case ";", ",", ")", "]", "}", ">":
	default:
		return nil
	}
	switch {
	case diag.Code == "E0007" && diag.Repair == diagnostic.RepairInsert && diag.Span.StartByte == diag.Span.EndByte:
		return &syntaxRepair{path, relative, diag, diag.Token}
	case diag.Code == "E0008" && diag.Repair == diagnostic.RepairSkip && diag.Span.EndByte-diag.Span.StartByte == len(diag.Token):
		return &syntaxRepair{path, relative, diag, ""}
	default:
		return nil
	}
}

// Removing punctuation can join neighboring lexemes or change which comments
// are documentation. Keep every token except the skipped symbol and whitespace.
func removalPreservesTokens(before, after lexer.Result, remove diagnostic.Span) bool {
	if len(before.Diagnostics) != 0 {
		return false
	}
	next := 0
	removed := false
	for _, token := range before.Tokens {
		if token.Trivia == lexer.TriviaWhitespace {
			continue
		}
		if token.Kind == lexer.Symbol && token.Span == remove {
			removed = true
			continue
		}
		for next < len(after.Tokens) && after.Tokens[next].Trivia == lexer.TriviaWhitespace {
			next++
		}
		if next == len(after.Tokens) {
			return false
		}
		other := after.Tokens[next]
		if token.Kind != other.Kind || token.Trivia != other.Trivia ||
			token.Cooked != other.Cooked || token.Text(before.Source) != other.Text(after.Source) {
			return false
		}
		next++
	}
	return removed && next == len(after.Tokens)
}

type codeActionContext struct {
	Diagnostics []lspDiagnostic `json:"diagnostics"`
	Only        []string        `json:"only"`
}

type editableDocumentID struct {
	URI     string `json:"uri"`
	Version *int   `json:"version"`
}

type documentEdit struct {
	TextDocument editableDocumentID `json:"textDocument"`
	Edits        []textEdit         `json:"edits"`
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
	if d.View == nil || d.View.repair == nil {
		return empty, nil
	}
	fix := d.View.repair
	span := fix.diagnostic.Span
	size := len(d.Text) + len(fix.replacement) - (span.EndByte - span.StartByte)
	if !sameDocumentPath(fix.path, d.Path) || !span.Valid(len(d.Text)) ||
		span.EndByte < start || span.StartByte > end || size > s.documentByteLimit(d.URI) {
		return empty, nil
	}
	if fix.replacement == "" && string(d.Text[span.StartByte:span.EndByte]) != fix.diagnostic.Token {
		return empty, nil
	}
	r := textRange(d.Text, span.StartByte, span.EndByte)
	matches := func(diag lspDiagnostic) bool {
		return diag.Code == fix.diagnostic.Code && diag.Range == r
	}
	if !slices.ContainsFunc(requested.Diagnostics, matches) {
		return empty, nil
	}
	text := make([]byte, 0, size)
	text = append(text, d.Text[:span.StartByte]...)
	text = append(text, fix.replacement...)
	text = append(text, d.Text[span.EndByte:]...)
	checked, err := frontend.AnalyzeSyntax(source.NewFile(fix.relative, text), source.DefaultLimits())
	if s.ctx != nil && s.ctx.Err() != nil {
		return nil, s.ctx.Err()
	}
	if err != nil || checked.AST == nil {
		return empty, nil
	}
	title := "Insert missing " + fix.diagnostic.Token
	if fix.replacement == "" {
		before := lexer.Lex(source.NewFile(fix.relative, d.Text), source.NewBudget(source.DefaultLimits(), len(d.Text)))
		after := lexer.Result{Source: checked.Tree.Source, Tokens: checked.Tree.Tokens}
		if !removalPreservesTokens(before, after, span) {
			return empty, nil
		}
		title = "Remove unexpected " + fix.diagnostic.Token
	}
	action := codeAction{Title: title, Kind: "quickfix"}
	for _, diag := range d.View.diagnostics[fix.path] {
		if matches(diag) {
			action.Diagnostics = []lspDiagnostic{diag}
			break
		}
	}
	if len(action.Diagnostics) == 0 {
		return empty, nil
	}
	action.Edit.DocumentChanges = []documentEdit{{
		TextDocument: editableDocumentID{URI: d.URI, Version: new(d.Version)},
		Edits:        []textEdit{{Range: r, NewText: fix.replacement}},
	}}
	return []codeAction{action}, nil
}
