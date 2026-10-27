package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func cleanupEffectCacheProgram(functions, depth int) Program {
	program := Program{}
	child := hir.TypeI32
	for i := range depth {
		typ := hir.ParseType(fmt.Sprintf("main::Cleanup%d", i))
		program.ResourceTypes = append(program.ResourceTypes, ResourceType{
			Type: typ, DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol,
			Fields: []ResourceField{{Name: "inner", Type: child}},
		})
		child = typ
	}
	for i := range functions {
		program.Functions = append(program.Functions, Function{
			Symbol: hir.SymbolID(i + 100), Name: fmt.Sprintf("main::consume%d", i),
			Parameters: []LocalID{2}, ParameterTypes: []hir.Type{child}, ResultType: hir.TypeVoid, Entry: 0,
			Resources: []ResourceLocal{{Symbol: 2, Type: child, Parameter: true, Entity: "resource:value"}},
			Blocks: []Block{{ID: 0,
				Operations: []Operation{{Kind: OperationDrop, Symbol: 2, Type: child, DestinationLocal: NoLocal, Entity: "drop:value", FlagClearedFirst: true}},
				Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock},
			}},
		})
	}
	return program
}

func BenchmarkVerifyCleanupEffectsAcrossFunctions(b *testing.B) {
	for _, count := range []int{1, 256} {
		b.Run(fmt.Sprintf("functions=%d", count), func(b *testing.B) {
			program := cleanupEffectCacheProgram(count, 128)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if failures := Verify(program); len(failures) != 0 {
					b.Fatal(failures)
				}
			}
		})
	}
}

func TestCleanupEffectCacheKeepsAuthorityPerFunctionAndRefreshesDefinitions(t *testing.T) {
	program := lowerTestSource(t, cleanupEffectSource+`
@effects(console) fn void allowed(Box value) {}
@effects(console) fn void denied(Box value) {}`)
	// Distinct entities keep canonical diagnostic deduplication from collapsing
	// the two otherwise identical cleanup failures in this fixture.
	for i := range program.Functions {
		function := &program.Functions[i]
		for j := range function.Blocks {
			for k := range function.Blocks[j].Operations {
				operation := &function.Blocks[j].Operations[k]
				if operation.Kind == OperationDrop {
					operation.Entity = fmt.Sprintf("%s.cleanup.%d.%d", function.Name, j, k)
				}
			}
		}
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("authorized functions rejected: %+v", failures)
	}
	for _, name := range []string{"main::denied", "main::allowed"} {
		for i := range program.Functions {
			if program.Functions[i].Name == name {
				program.Functions[i].Effects = nil
			}
		}
		failures := Verify(program)
		want := 1
		if name == "main::allowed" {
			want = 2
		}
		if len(failures) != want {
			t.Fatalf("per-function authority: got %+v, want %d failures", failures, want)
		}
		for _, failure := range failures {
			if failure.Code != CodeEffectAuthority {
				t.Fatalf("unexpected rejection: %+v", failure)
			}
		}
	}
	for i := range program.ResourceTypes {
		program.ResourceTypes[i].DropEffects = nil
	}
	for i := range program.Functions {
		program.Functions[i].Effects = nil
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("stale required effects retained between verifications: %+v", failures)
	}
	for i := range program.ResourceTypes {
		resource := &program.ResourceTypes[i]
		if resource.DropKind == hir.ResourceDropNonEmpty {
			resource.DropEffects = []hir.Effect{hir.EffectConsole}
			for j := range program.Functions {
				if program.Functions[j].Symbol == resource.DropSymbol {
					program.Functions[j].Effects = resource.DropEffects
				}
			}
		}
	}
	if failures := Verify(program); len(failures) != 2 || failures[0].Code != CodeEffectAuthority || failures[1].Code != CodeEffectAuthority {
		t.Fatalf("stale empty effects accepted after definition change: %+v", failures)
	}
}
