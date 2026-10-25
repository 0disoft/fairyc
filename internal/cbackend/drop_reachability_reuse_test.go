package cbackend

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestRepeatedDropsKeepNestedGlueAndExcludeUnusedGlue(t *testing.T) {
	program, bySymbol, entry := repeatedDropProgram(t, 8, 8)
	unit, err := Emit(program, "main::main", Stage0Profile())
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		reachable, _, err := reachableFunctionIndices(program, program.Functions, bySymbol, entry)
		if err != nil {
			t.Fatal(err)
		}
		for _, definition := range program.ResourceTypes {
			index, found := bySymbol[definition.DropSymbol]
			if !found {
				t.Fatal("fixture drop glue missing")
			}
			_, kept := reachable[index]
			if want := definition.Type != hir.ParseType("main::Unused"); kept != want {
				t.Fatalf("glue for %s: kept=%v, want %v", definition.Type, kept, want)
			}
			if slices.Contains(unit.Functions, program.Functions[index].Name) != kept {
				t.Fatalf("emitted functions disagree with reachable glue for %s", definition.Type)
			}
		}
	}
}

func TestRepeatedDropReachabilityStillRejectsMissingGlue(t *testing.T) {
	program, bySymbol, entry := repeatedDropProgram(t, 8, 8)
	for _, definition := range program.ResourceTypes {
		if definition.Type == hir.ParseType("main::Leaf") {
			delete(bySymbol, definition.DropSymbol)
		}
	}
	if _, _, err := reachableFunctionIndices(program, program.Functions, bySymbol, entry); err == nil || !strings.Contains(err.Error(), "missing drop symbol") {
		t.Fatalf("missing nested drop glue: %v", err)
	}
}

func TestRepeatedDropReachabilityStillChecksEachLocal(t *testing.T) {
	program, bySymbol, entry := repeatedDropProgram(t, 8, 8)
	for index := range program.Functions[entry].Blocks {
		block := &program.Functions[entry].Blocks[index]
		for _, operation := range block.Operations {
			if operation.Kind != mir.OperationDrop {
				continue
			}
			block.Operations = append(block.Operations, mir.Operation{Kind: mir.OperationDrop, Symbol: hir.NoSymbol})
			if _, _, err := reachableFunctionIndices(program, program.Functions, bySymbol, entry); err == nil || !strings.Contains(err.Error(), "undeclared resource local") {
				t.Fatalf("invalid drop after a valid drop: %v", err)
			}
			return
		}
	}
	t.Fatal("fixture has no drop operation")
}

func BenchmarkRepeatedDropReachability(b *testing.B) {
	for _, count := range []int{64, 1024} {
		b.Run(fmt.Sprintf("drops-%d", count), func(b *testing.B) {
			program, bySymbol, entry := repeatedDropProgram(b, count, 64)
			b.ReportAllocs()
			for b.Loop() {
				reachable, _, err := reachableFunctionIndices(program, program.Functions, bySymbol, entry)
				if err != nil || len(reachable) != len(program.Functions)-1 {
					b.Fatalf("reachable=%d, err=%v", len(reachable), err)
				}
			}
		})
	}
}

// Build through syntax, semantic analysis, lowering and verification so the
// benchmark measures reachable code from valid resource ownership, not double drops.
func repeatedDropProgram(tb testing.TB, drops, fields int) (mir.Program, map[hir.SymbolID]int, int) {
	tb.Helper()
	var text strings.Builder
	text.WriteString("module main;\nfn void audit() {}\nresource Leaf { i32 value; cleanup { audit(); } }\nresource Unused { i32 value; cleanup { audit(); } }\nresource Group {\n")
	for index := range fields {
		fmt.Fprintf(&text, "Leaf field_%d;\n", index)
	}
	text.WriteString("cleanup { audit(); }\n}\nfn Group make() { return Group {\n")
	for index := range fields {
		fmt.Fprintf(&text, ".field_%d = Leaf { .value = %d },\n", index, index)
	}
	text.WriteString("}; }\nfn i32 main() {\n")
	for index := range drops {
		fmt.Fprintf(&text, "{ let Group local_%d = make(); }\n", index)
	}
	text.WriteString("return 42; }\n")
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text.String())), source.DefaultLimits())
	if err != nil {
		tb.Fatal(err)
	}
	if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
		tb.Fatalf("syntax: %+v", parsed.Tree.Diagnostics)
	}
	analyzed := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if len(analyzed.Diagnostics) != 0 || len(analyzed.Unsupported) != 0 {
		tb.Fatalf("semantic: %+v, unsupported: %+v", analyzed.Diagnostics, analyzed.Unsupported)
	}
	program, err := mir.Lower(analyzed.HIR)
	if err != nil {
		tb.Fatal(err)
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		tb.Fatalf("MIR: %+v", failures)
	}
	bySymbol, entry, err := indexFunctions(program.Functions, "main::main")
	if err != nil {
		tb.Fatal(err)
	}
	return program, bySymbol, entry
}
