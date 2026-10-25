package moduleinterface

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func batchInterfaceInput(tb testing.TB, count int) (hir.Package, []hir.PackageID, map[hir.PackageID]packagegraph.CoordinateID) {
	tb.Helper()
	value := hir.Package{}
	ids := make([]hir.PackageID, count)
	coordinates := make(map[hir.PackageID]packagegraph.CoordinateID, count)
	for index := range count {
		id := hir.PackageID(index + 1)
		ids[index] = id
		coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: fmt.Sprintf("package.%d", id), VersionIdentity: "1"})
		if err != nil {
			tb.Fatal(err)
		}
		coordinates[id] = coordinate
		if index%8 == 0 {
			value.StateLayouts = append(value.StateLayouts, hir.StateLayout{
				Package: id, ContractUID: fmt.Sprintf("state.%d", id),
				Fields: []hir.StateField{{Name: "count", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()}},
			})
		}
	}
	// Modules from different owners are interleaved, not grouped or sorted.
	for module := 3; module >= 0; module-- {
		for _, id := range ids {
			value.Modules = append(value.Modules, hir.Module{
				Package: id, Path: fmt.Sprintf("m%d", module),
				Functions: []hir.Function{{Name: "read", Public: true, ResultType: hir.TypeU32}},
			})
		}
	}
	return value, ids, coordinates
}

func TestComputePackagesBatchMatchesIndependentExtraction(t *testing.T) {
	value, ids, coordinates := batchInterfaceInput(t, 16)
	// Include a public local type and a signature referring to another owner.
	value.Modules = append(value.Modules, interfacePackage(1, 2).Modules...)
	before := clonePackage(value)
	want := make(map[hir.PackageID]Hash, len(ids))
	for _, id := range ids {
		want[id] = extractHash(t, value, id, coordinates)
	}
	slices.Reverse(ids)
	got, err := ComputePackages(value, ids, coordinates)
	if err != nil || !maps.Equal(got, want) {
		t.Fatalf("batch hashes differ: error=%v got=%v want=%v", err, got, want)
	}
	if !reflect.DeepEqual(value, before) || ids[0] != 16 {
		t.Fatal("batch extraction mutated its inputs")
	}
	// A subset must ignore malformed modules and state layouts from other owners.
	value.Modules[0].Path = ""
	value.StateLayouts[0].ContractUID = ""
	got, err = ComputePackages(value, []hir.PackageID{2}, coordinates)
	if err != nil || len(got) != 1 || got[2] != want[2] {
		t.Fatalf("unrequested package affected subset: hashes=%v error=%v", got, err)
	}
}

func TestComputePackagesBatchPreservesExtractionFailures(t *testing.T) {
	for _, mode := range []string{"duplicate module", "missing module", "bad layout", "duplicate layout"} {
		t.Run(mode, func(t *testing.T) {
			value, _, coordinates := batchInterfaceInput(t, 2)
			switch mode {
			case "duplicate module":
				value.Modules = append(value.Modules, value.Modules[0])
			case "missing module":
				value.Modules = slices.DeleteFunc(value.Modules, func(m hir.Module) bool { return m.Package == 1 })
			case "bad layout":
				value.StateLayouts[0].ContractUID = ""
			case "duplicate layout":
				value.StateLayouts = append(value.StateLayouts, value.StateLayouts[0])
			}
			_, want := Extract(value, 1, coordinates)
			if want == nil {
				t.Fatal("expected independent extraction failure")
			}
			_, got := ComputePackages(value, []hir.PackageID{2, 1}, coordinates)
			if got == nil || got.Error() != "extract public interface for package 1: "+want.Error() {
				t.Fatalf("batch error=%v, independent error=%v", got, want)
			}
		})
	}
}

func BenchmarkComputePackagesBatch(b *testing.B) {
	for _, count := range []int{16, 1024} {
		b.Run(fmt.Sprintf("packages_%d", count), func(b *testing.B) {
			value, ids, coordinates := batchInterfaceInput(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				hashes, err := ComputePackages(value, ids, coordinates)
				if err != nil || len(hashes) != count {
					b.Fatalf("hashes=%d error=%v", len(hashes), err)
				}
			}
		})
	}
}
