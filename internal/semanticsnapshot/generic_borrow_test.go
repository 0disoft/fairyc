package semanticsnapshot

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestGenericBorrowFactsJoinByFunctionNotSharedSourceSpan(t *testing.T) {
	text := `module main;
fn T id<T>(T value) { let ref T view = ref value; return *view; }
fn i32 main() { assert(id<i32>(42) == 42); assert(id<u32>(7) == 7); return 0; }
`
	syntax, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || syntax.AST == nil || len(syntax.Tree.Diagnostics) != 0 {
		t.Fatalf("syntax: %+v %v", syntax, err)
	}
	compiled, err := compiler.CheckExplicitFiles([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: syntax.AST}})
	if err != nil || !compiled.Verified {
		t.Fatalf("compile: %+v %v", compiled.Semantic, err)
	}
	facts := compiled.Semantic.Borrows
	if len(facts) != 2 || facts[0].Borrow != facts[1].Borrow || facts[0].Function == hir.NoSymbol || facts[0].Function == facts[1].Function {
		t.Fatalf("missing distinct specialization owners: %+v", facts)
	}
	encoded, err := json.Marshal(facts)
	if err != nil || strings.Contains(string(encoded), "Function") {
		t.Fatalf("build-local owner leaked into JSON: %s %v", encoded, err)
	}
	// Test the declaration-to-MIR join directly: a full generic snapshot also
	// needs original-declaration and documentation joins, which are separate work.
	metadata := map[hir.PackageID]packageMetadata{hir.DefaultPackage: {instance: "fixture-instance", coord: "fixture-coordinate"}}
	_, _, symbols, foreign, private, bounds, err := buildDeclarations(compiled.Semantic.HIR, metadata)
	if err != nil {
		t.Fatal(err)
	}
	build := func(borrows []semantic.BorrowFact) ([]FunctionSummary, error) {
		values, _, _, err := buildFunctionSummaries(compiled.MIR, borrows, metadata, symbols, foreign, private, bounds)
		return values, err
	}
	values, err := build(facts)
	if err != nil || len(values) != 3 {
		t.Fatalf("summaries: %+v %v", values, err)
	}
	for _, fact := range facts {
		key := symbols[fact.Function]
		index := slices.IndexFunc(values, func(value FunctionSummary) bool { return value.Function == key })
		if index < 0 || len(values[index].Borrows) != 1 || values[index].Borrows[0].Span.StartByte != fact.Borrow.StartByte {
			t.Fatalf("borrow assigned to wrong specialization: %+v %+v", fact, values)
		}
	}
	mutated := slices.Clone(facts)
	mutated[0].Function = hir.NoSymbol
	if _, err := build(mutated); err == nil || !strings.Contains(err.Error(), "multiple functions") {
		t.Fatalf("ambiguous legacy source-span join: %v", err)
	}
	var mainSymbol hir.SymbolID
	for symbol, key := range symbols {
		if key.Name == "main" {
			mainSymbol = symbol
		}
	}
	mutated[0].Function = mainSymbol
	if _, err := build(mutated); err == nil || !strings.Contains(err.Error(), "matching function identity") {
		t.Fatalf("owner/span mismatch accepted: %v", err)
	}
}
