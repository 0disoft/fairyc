package packageartifact

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestForeignContextIndexPreservesBindingsAndRejectsDrift(t *testing.T) {
	program, stable, coordinates := foreignIndexFixture(t, 8)
	available, err := indexForeignContext(program, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range stable.ForeignFunctions {
		local := ContextDescriptor{ForeignFunctions: []ContextForeignFunction{declaration}}
		bindings, err := available.symbolsFor(local)
		if err != nil || len(bindings) != 1 {
			t.Fatalf("resolve local context: %v", err)
		}
		var matched mir.ForeignFunction
		for _, foreign := range program.ForeignFunctions {
			if foreign.Symbol == bindings[0] {
				matched = foreign
			}
		}
		if matched.LinkName != declaration.LinkName || coordinates[matched.Span.Package] != declaration.Package {
			t.Fatalf("resolved the wrong package or declaration: %+v", matched)
		}
	}
	for _, change := range []string{"link name", "ABI", "parameters", "result", "missing"} {
		t.Run(change, func(t *testing.T) {
			changed := cloneContextDescriptor(stable)
			switch change {
			case "link name":
				changed.ForeignFunctions[0].LinkName += "_changed"
			case "ABI":
				changed.ForeignFunctions[0].ABI = "other"
			case "parameters":
				changed.ForeignFunctions[0].ParameterTypes = []string{"i32"}
			case "result":
				changed.ForeignFunctions[0].ResultType = "usize"
			case "missing":
				changed.ForeignFunctions[0].Name += "_missing"
			}
			if _, err := available.symbolsFor(changed); err == nil {
				t.Fatal("accepted missing or conflicting declaration")
			}
		})
	}
	// The index owns copied declarations; matching must not depend on source order.
	// A later reassembly gets fresh symbols rather than reusing a global cache.
	program.ForeignFunctions[0].Symbol += 50000
	fresh, err := indexForeignContext(program, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh) != len(available) {
		t.Fatal("fresh index lost declarations")
	}
	for key, binding := range fresh {
		if binding.declaration.LinkName == program.ForeignFunctions[0].LinkName && binding.symbol != program.ForeignFunctions[0].Symbol {
			t.Fatalf("reused stale symbol for %s", key)
		}
	}
}

func TestForeignContextIndexRejectsAmbiguousAndInvalidMergedDeclarations(t *testing.T) {
	for _, change := range []string{"symbol", "signature", "owner"} {
		t.Run(change, func(t *testing.T) {
			program, _, coordinates := foreignIndexFixture(t, 2)
			duplicate := program.ForeignFunctions[0]
			switch change {
			case "symbol":
				duplicate.Symbol += hir.SymbolID(1000)
			case "signature":
				duplicate.ResultType = hir.TypeUSize
			case "owner":
				delete(coordinates, duplicate.Span.Package)
			}
			program.ForeignFunctions = append(program.ForeignFunctions, duplicate)
			if _, err := indexForeignContext(program, coordinates); err == nil {
				t.Fatal("accepted ambiguous or invalid merged declarations")
			}
		})
	}
}

// Measure the foreign-binding phase for many independently projected contexts,
// each requiring one declaration from the same merged process context.
func BenchmarkForeignBindingsAcrossContexts(b *testing.B) {
	for _, count := range []int{64, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			program, stable, coordinates := foreignIndexFixture(b, count)
			contexts := make([]ContextDescriptor, count)
			for i := range count {
				contexts[i].ForeignFunctions = stable.ForeignFunctions[i : i+1]
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				available, err := indexForeignContext(program, coordinates)
				if err != nil {
					b.Fatal(err)
				}
				for _, context := range contexts {
					bindings, err := available.symbolsFor(context)
					if err != nil || len(bindings) != 1 {
						b.Fatalf("resolve foreign context: %v", err)
					}
				}
			}
		})
	}
}
