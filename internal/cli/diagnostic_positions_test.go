package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/syntax"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestDiagnosticPositionIndexMatchesOriginalByteColumns(t *testing.T) {
	for _, contents := range [][]byte{nil, []byte("a\r\n한🧚\r\xff\n"), bytes.Repeat([]byte{'\n'}, 16385), bytes.Repeat([]byte{'x'}, 16385), bytes.Repeat([]byte("한\r\n🧚"), 2048)} {
		positions := diagnosticPositionIndex{source: contents}
		offsets := []int{len(contents) + 1, -1, 0, 1, 4095, 4096, 4097, 8192, len(contents), 4, 3, 2}
		for i := range 128 {
			offsets = append(offsets, i*7919%(len(contents)+1))
		}
		for _, offset := range offsets {
			line, column := positions.position(offset)
			wantLine, wantColumn := bytePosition(contents, offset)
			if line != wantLine || column != wantColumn {
				t.Fatalf("length=%d offset=%d: got %d:%d want %d:%d", len(contents), offset, line, column, wantLine, wantColumn)
			}
		}
		if len(positions.checkpoints) > len(contents)/diagnosticPositionChunk+1 {
			t.Fatal("position storage exceeded sparse checkpoint bound")
		}
	}
}

func TestDiagnosticOutputPositionsPreserveFormatting(t *testing.T) {
	contents := bytes.Repeat([]byte("한\r\n🧚\n"), 2048)
	values := []diagnostic.Diagnostic{
		{Code: "E0007", Span: diagnostic.Span{StartByte: len(contents), EndByte: len(contents)}, Repair: diagnostic.RepairInsert, Token: ";"},
		{Code: "E0008", Span: diagnostic.Span{StartByte: 2, EndByte: 3}, Repair: diagnostic.RepairNone},
	}
	var got, want bytes.Buffer
	if err := writeDiagnostics(&got, "main.fy", contents, values); err != nil {
		t.Fatal(err)
	}
	report := buildParseReport("main.fy", &syntax.Tree{Source: contents, Diagnostics: values}, false)
	for i, value := range values {
		line, column := bytePosition(contents, value.Span.StartByte)
		fmt.Fprintf(&want, "main.fy:%d:%d: %s: %s [bytes %d..%d, repair=%s]\n", line, column, value.Code, diagnosticSummary(value), value.Span.StartByte, value.Span.EndByte, value.Repair)
		entry := report.Diagnostics[i]
		if entry.Line != line || entry.ByteColumn != column || entry.Code != value.Code || entry.Repair != string(value.Repair) || entry.Expected != value.Token {
			t.Fatalf("JSON report diagnostic changed: %+v", entry)
		}
	}
	if got.String() != want.String() {
		t.Fatal("frontend diagnostic formatting changed")
	}
}

type diagnosticPositionFailWriter struct{ err error }

func (writer diagnosticPositionFailWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestDiagnosticPositionsKeepWorkspaceSourcesSeparate(t *testing.T) {
	first, second := []byte("a\n한"), []byte("xyz\r\nq")
	workspace := diagnosticPositionWorkspace(first, second)
	values := []semantic.Diagnostic{{Code: "E0101", Primary: hir.Span{Package: 1, Source: "main.fy", StartByte: 2, EndByte: 5}, Related: []semantic.RelatedSpan{
		{Role: "other", Span: hir.Span{Package: 2, Source: "main.fy", StartByte: 2, EndByte: 3}},
		{Role: "earlier", Span: hir.Span{Package: 1, Source: "main.fy", StartByte: 0, EndByte: 1}},
	}}}
	var got bytes.Buffer
	if err := writeWorkspaceDiagnostics(&got, workspace, values); err != nil {
		t.Fatal(err)
	}
	want := "pkg-0/main.fy:2:1: E0101: unresolved name or path [bytes 2..5]\n  other: pkg-1/main.fy:1:3 [bytes 2..3]\n  earlier: pkg-0/main.fy:1:1 [bytes 0..1]\n"
	if got.String() != want {
		t.Fatalf("got %q want %q", got.String(), want)
	}
	sentinel := errors.New("output stopped")
	if err := writeWorkspaceDiagnostics(diagnosticPositionFailWriter{sentinel}, workspace, values); !errors.Is(err, sentinel) {
		t.Fatalf("writer failure lost: %v", err)
	}
	workspace.Packages[0].Sources[0].Bytes = []byte("abcde")
	got.Reset()
	if err := writeWorkspaceDiagnostics(&got, workspace, values); err != nil || !bytes.Contains(got.Bytes(), []byte("pkg-0/main.fy:1:3:")) {
		t.Fatal("new report reused positions from an earlier source version")
	}
	values[0].Related[0].Span.Source = "missing.fy"
	if err := writeWorkspaceDiagnostics(io.Discard, workspace, values); err == nil || err.Error() != "resolve related workspace diagnostic source" {
		t.Fatalf("missing source failure changed: %v", err)
	}
	values[0].Primary.Package = 3
	if err := writeWorkspaceDiagnostics(io.Discard, workspace, values); err == nil || err.Error() != "resolve workspace diagnostic source" {
		t.Fatalf("missing primary source failure changed: %v", err)
	}
}

func TestDiagnosticPositionsSemanticRelatedPaths(t *testing.T) {
	sources := map[string][]byte{"a.fy": []byte("a\n한"), "b.fy": []byte("xyz\r\nq")}
	values := []semantic.Diagnostic{{Code: "E0101", Primary: hir.Span{Source: "a.fy", StartByte: 2, EndByte: 5}, Related: []semantic.RelatedSpan{
		{Role: "other", Span: hir.Span{Source: "b.fy", StartByte: 2, EndByte: 3}},
		{Role: "earlier", Span: hir.Span{Source: "a.fy", StartByte: 0, EndByte: 1}},
	}}}
	var got bytes.Buffer
	if err := writeSemanticDiagnostics(&got, sources, values); err != nil {
		t.Fatal(err)
	}
	want := "a.fy:2:1: E0101: unresolved name or path [bytes 2..5]\n  other: b.fy:1:3 [bytes 2..3]\n  earlier: a.fy:1:1 [bytes 0..1]\n"
	if got.String() != want {
		t.Fatalf("got %q want %q", got.String(), want)
	}
	sentinel := errors.New("output stopped")
	if err := writeSemanticDiagnostics(diagnosticPositionFailWriter{sentinel}, sources, values); !errors.Is(err, sentinel) {
		t.Fatalf("writer failure lost: %v", err)
	}
	if err := writeDiagnostics(diagnosticPositionFailWriter{sentinel}, "a.fy", sources["a.fy"], []diagnostic.Diagnostic{{Code: "E0008"}}); !errors.Is(err, sentinel) {
		t.Fatalf("frontend writer failure lost: %v", err)
	}
}

func diagnosticPositionWorkspace(sources ...[]byte) workspaceloader.Workspace {
	var workspace workspaceloader.Workspace
	for i, contents := range sources {
		path := fmt.Sprintf("pkg-%d", i)
		workspace.Packages = append(workspace.Packages, workspaceloader.Package{Path: path, Sources: []workspaceloader.SourceFile{{Path: "main.fy", Bytes: contents}}})
		workspace.Instances = append(workspace.Instances, workspaceloader.Instance{NodeID: path, PackagePath: path})
		workspace.Resolution.InstanceClasses = append(workspace.Resolution.InstanceClasses, packagegraph.InstanceClass{Members: []string{path}})
	}
	return workspace
}

func BenchmarkWorkspaceDiagnosticPositions(b *testing.B) {
	contents := bytes.Repeat([]byte("var x = 0;\r\n"), (4<<20)/12)
	workspace := diagnosticPositionWorkspace(contents)
	values := make([]semantic.Diagnostic, 256)
	for i := range values {
		offset := len(contents) - i - 1
		values[i] = semantic.Diagnostic{Code: "E0101", Primary: hir.Span{Package: 1, Source: "main.fy", StartByte: offset, EndByte: offset + 1}}
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := writeWorkspaceDiagnostics(io.Discard, workspace, values); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiagnosticOutputPositions(b *testing.B) {
	for _, size := range []int{1 << 20, 4 << 20} {
		contents := bytes.Repeat([]byte("var x = 0;\r\n"), size/12)
		values := make([]diagnostic.Diagnostic, 256)
		semanticValues := make([]semantic.Diagnostic, len(values))
		for i := range values {
			offset := len(contents) - i - 1
			values[i] = diagnostic.Diagnostic{Code: "E0008", Span: diagnostic.Span{StartByte: offset, EndByte: offset + 1}, Repair: diagnostic.RepairNone}
			semanticValues[i] = semantic.Diagnostic{Code: "E0101", Primary: hir.Span{Source: "main.fy", StartByte: offset, EndByte: offset + 1}}
		}
		sources := map[string][]byte{"main.fy": contents}
		tree := &syntax.Tree{Source: contents, Diagnostics: values}
		for _, mode := range []string{"frontend", "semantic", "json-report"} {
			b.Run(fmt.Sprintf("bytes=%d/%s", len(contents), mode), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					switch mode {
					case "frontend":
						if err := writeDiagnostics(io.Discard, "main.fy", contents, values); err != nil {
							b.Fatal(err)
						}
					case "semantic":
						if err := writeSemanticDiagnostics(io.Discard, sources, semanticValues); err != nil {
							b.Fatal(err)
						}
					case "json-report":
						if got := buildParseReport("main.fy", tree, false); len(got.Diagnostics) != len(values) {
							b.Fatal("diagnostics missing")
						}
					}
				}
			})
		}
	}
}
