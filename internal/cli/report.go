package cli

import (
	"encoding/json"
	"io"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/syntax"
)

const parseReportSchema = "fairyc.experimental.parse/v0"

type parseReport struct {
	SchemaVersion string             `json:"schemaVersion"`
	SourcePath    string             `json:"sourcePath"`
	ByteLength    int                `json:"byteLength"`
	Truncated     bool               `json:"truncated"`
	Status        string             `json:"status"`
	Module        string             `json:"module"`
	Root          int                `json:"root"`
	ASTLowered    bool               `json:"astLowered"`
	Tokens        []tokenReport      `json:"tokens"`
	Nodes         []nodeReport       `json:"nodes"`
	Diagnostics   []diagnosticReport `json:"diagnostics"`
}

type tokenReport struct {
	Kind      string `json:"kind"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
}

type nodeReport struct {
	ID        int    `json:"id"`
	Kind      string `json:"kind"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
	Children  []int  `json:"children"`
}

type diagnosticReport struct {
	Code       string `json:"code"`
	StartByte  int    `json:"startByte"`
	EndByte    int    `json:"endByte"`
	Line       int    `json:"line"`
	ByteColumn int    `json:"byteColumn"`
	Repair     string `json:"repair"`
	Expected   string `json:"expected,omitempty"`
}

func buildParseReport(relativePath string, tree *syntax.Tree, astLowered bool) parseReport {
	status := "ok"
	if len(tree.Diagnostics) > 0 {
		status = "diagnostics"
	}
	report := parseReport{
		SchemaVersion: parseReportSchema,
		SourcePath:    relativePath,
		ByteLength:    len(tree.Source),
		Truncated:     false,
		Status:        status,
		Module:        tree.Module,
		Root:          int(tree.Root),
		ASTLowered:    astLowered,
		Tokens:        make([]tokenReport, 0, len(tree.Tokens)),
		Nodes:         make([]nodeReport, 0, len(tree.Nodes)),
		Diagnostics:   make([]diagnosticReport, 0, len(tree.Diagnostics)),
	}
	for _, token := range tree.Tokens {
		report.Tokens = append(report.Tokens, tokenReport{
			Kind:      tokenKindName(token.Kind),
			StartByte: token.Span.StartByte,
			EndByte:   token.Span.EndByte,
		})
	}
	for id, node := range tree.Nodes {
		children := make([]int, 0, len(node.Children))
		for _, child := range node.Children {
			children = append(children, int(child))
		}
		report.Nodes = append(report.Nodes, nodeReport{
			ID:        id,
			Kind:      string(node.Kind),
			StartByte: node.Span.StartByte,
			EndByte:   node.Span.EndByte,
			Children:  children,
		})
	}
	for _, value := range tree.Diagnostics {
		report.Diagnostics = append(report.Diagnostics, newDiagnosticReport(tree.Source, value))
	}
	return report
}

func buildTooLargeReport(relativePath string, limit int) parseReport {
	return parseReport{
		SchemaVersion: parseReportSchema,
		SourcePath:    relativePath,
		Status:        "diagnostics",
		Truncated:     true,
		Root:          int(syntax.NoNode),
		Tokens:        []tokenReport{},
		Nodes:         []nodeReport{},
		Diagnostics: []diagnosticReport{{
			Code:       "E0010",
			Line:       1,
			ByteColumn: 1,
			Repair:     string(diagnostic.RepairNone),
			Expected:   "",
		}},
		ByteLength: limit + 1,
	}
}

func writeParseReport(writer io.Writer, report parseReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func newDiagnosticReport(sourceBytes []byte, value diagnostic.Diagnostic) diagnosticReport {
	line, column := bytePosition(sourceBytes, value.Span.StartByte)
	report := diagnosticReport{
		Code:       value.Code,
		StartByte:  value.Span.StartByte,
		EndByte:    value.Span.EndByte,
		Line:       line,
		ByteColumn: column,
		Repair:     string(value.Repair),
	}
	if value.Repair == diagnostic.RepairInsert {
		report.Expected = value.Token
	}
	return report
}

func bytePosition(sourceBytes []byte, offset int) (line int, column int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(sourceBytes) {
		offset = len(sourceBytes)
	}
	line, column = 1, 1
	for _, value := range sourceBytes[:offset] {
		if value == '\n' {
			line, column = line+1, 1
			continue
		}
		column++
	}
	return line, column
}

func tokenKindName(kind lexer.Kind) string {
	switch kind {
	case lexer.Invalid:
		return "invalid"
	case lexer.EOF:
		return "eof"
	case lexer.Trivia:
		return "trivia"
	case lexer.Identifier:
		return "identifier"
	case lexer.Keyword:
		return "keyword"
	case lexer.Integer:
		return "integer"
	case lexer.Float:
		return "float"
	case lexer.String:
		return "string"
	case lexer.ByteString:
		return "byte-string"
	case lexer.CString:
		return "c-string"
	case lexer.Symbol:
		return "symbol"
	default:
		return "unknown"
	}
}
