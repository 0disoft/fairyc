package contractstatemir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func BenchmarkStateBodySharedResourceGraph(b *testing.B) {
	for _, depth := range []int{8, 12, 16} {
		b.Run(fmt.Sprint(depth), func(b *testing.B) {
			input, root := sharedResourceGraphFixture(depth)
			base := stateBodyPackage(b, 1, 1)
			base.Modules[0].Structs = input.Modules[0].Structs
			base.Modules[0].Functions[0].Parameters[0].Type = root
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: {1}}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				hashes, err := ComputeHIRStateBodyHashes(base, []hir.PackageID{1}, coordinates)
				if err != nil || len(hashes) != 1 {
					b.Fatalf("hash depth-%d shared type graph: %v", depth, err)
				}
			}
		})
	}
}

func TestOwnerTypeGraphSharedBranchesAndCycles(t *testing.T) {
	resource := hir.ParseType("$p1::ledger::Handle")
	for _, tc := range []struct {
		name      string
		forbidden hir.Type
		check     func(hir.Package, hir.Type, map[hir.Type]bool) bool
	}{
		{"owned resource", resource, hirTypeContainsOwnedResource},
		{"raw pointer", hir.RawPointer(hir.TypeU8, false), hirTypeContainsRawPointer},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input, root := sharedResourceGraphFixture(12)
			input.Modules[0].Resources = []hir.Resource{{Type: resource}}
			if tc.check(input, root, make(map[hir.Type]bool)) {
				t.Fatal("safe shared graph was rejected")
			}
			// The forbidden type follows the repeated safe branch.
			input.Modules[0].Structs[0].Fields = append(input.Modules[0].Structs[0].Fields, hir.StructField{Type: tc.forbidden})
			if !tc.check(input, root, make(map[hir.Type]bool)) {
				t.Fatal("forbidden type after shared branches was missed")
			}
			second := hir.ParseType("$p1::ledger::Cycle")
			input.Modules[0].Structs = []hir.Struct{
				{Type: root, Fields: []hir.StructField{{Type: second}, {Type: tc.forbidden}}},
				{Type: second, Fields: []hir.StructField{{Type: root}}},
			}
			for _, start := range []hir.Type{root, second} {
				if !tc.check(input, start, make(map[hir.Type]bool)) {
					t.Fatalf("forbidden type reachable through cycle from %s was missed", start)
				}
			}
			input.Modules[0].Structs[0].Fields = input.Modules[0].Structs[0].Fields[:1]
			if tc.check(input, root, make(map[hir.Type]bool)) {
				t.Fatal("safe cycle was rejected")
			}
		})
	}
}

func sharedResourceGraphFixture(depth int) (hir.Package, hir.Type) {
	types := make([]hir.Type, depth+1)
	for i := range depth {
		types[i] = hir.ParseType(fmt.Sprintf("$p1::ledger::Node_%02d", i))
	}
	types[depth] = hir.TypeU32
	module := hir.Module{Package: 1, Path: "ledger"}
	for i := range depth {
		module.Structs = append(module.Structs, hir.Struct{
			Type: types[i], Name: fmt.Sprintf("Node_%02d", i),
			Fields: []hir.StructField{{Name: "left", Type: types[i+1]}, {Name: "right", Type: types[i+1]}},
		})
	}
	return hir.Package{Modules: []hir.Module{module}}, types[0]
}
