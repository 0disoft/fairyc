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

// Each generated cleanup drops the next nested resource. Those distinct drop
// roots share progressively shorter suffixes of the same resource graph.
func overlappingDropProgram(tb testing.TB, count int) (mir.Program, map[hir.SymbolID]int, int) {
	tb.Helper()
	var text strings.Builder
	text.WriteString("module main;\nfn void audit() {}\nresource Unused { i32 value; cleanup { audit(); } }\n")
	for index := range count {
		fmt.Fprintf(&text, "resource Node%d { ", index)
		if index+1 < count {
			fmt.Fprintf(&text, "Node%d child; ", index+1)
		} else {
			text.WriteString("i32 value; ")
		}
		text.WriteString("cleanup { audit(); } }\n")
		fmt.Fprintf(&text, "fn Node%d make%d() { return Node%d { ", index, index, index)
		if index+1 < count {
			fmt.Fprintf(&text, ".child = make%d()", index+1)
		} else {
			text.WriteString(".value = 42")
		}
		text.WriteString(" }; }\n")
	}
	text.WriteString("fn i32 main() {\n")
	for index := range count {
		fmt.Fprintf(&text, "{ let Node%d local%d = make%d(); }\n", index, index, index)
	}
	text.WriteString("return 42; }\n")
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text.String())), source.DefaultLimits())
	if err != nil {
		tb.Fatal(err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
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

func TestOverlappingDropsPreserveNestedGlueAndFreshness(t *testing.T) {
	program, bySymbol, entry := overlappingDropProgram(t, 16)
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
				t.Fatalf("emitted glue disagrees with reachability for %s", definition.Type)
			}
		}
	}
	for _, definition := range program.ResourceTypes {
		if definition.Type == hir.ParseType("main::Node15") {
			delete(bySymbol, definition.DropSymbol)
		}
	}
	if _, _, err := reachableFunctionIndices(program, program.Functions, bySymbol, entry); err == nil || !strings.Contains(err.Error(), "missing drop symbol") {
		t.Fatalf("removed nested glue was reused: %v", err)
	}
}

func BenchmarkOverlappingDropReachability(b *testing.B) {
	for _, count := range []int{64, 256} {
		b.Run(fmt.Sprintf("resources-%d", count), func(b *testing.B) {
			program, bySymbol, entry := overlappingDropProgram(b, count)
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
