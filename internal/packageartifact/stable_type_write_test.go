package packageartifact

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableTypeWritePreservesExactSpellingAndFailures(t *testing.T) {
	coordinate := testCoordinateID(t, "writer")
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: coordinate}
	nominal := hir.ParseType("$p1::model::Packet")
	stable := "pkg[" + coordinate.String() + "]::model::Packet"
	for _, test := range []struct {
		value hir.Type
		want  string
	}{
		{hir.TypeI32, "i32"},
		{nominal, stable},
		{hir.Slice(nominal), stable + "[]"},
		{hir.SharedBorrow(nominal), "ref " + stable},
		{hir.MutableBorrow(nominal), "ref mut " + stable},
		{hir.RawPointer(nominal, false), "ptr " + stable},
		{hir.RawPointer(nominal, true), "ptr mut " + stable},
		{hir.FixedArray(nominal, 32), stable + "[32]"},
		{hir.Option(nominal), "Option<" + stable + ">"},
		{hir.Result(nominal, hir.TypeI32), "Result<" + stable + ", i32>"},
		{hir.Result(hir.Option(hir.SharedBorrow(hir.Slice(nominal))), hir.FixedArray(hir.TypeU8, 3)), "Result<Option<ref " + stable + "[]>, u8[3]>"},
	} {
		if got, err := stableType(test.value, coordinates); err != nil || got != test.want {
			t.Fatalf("stable spelling: got %q, %v; want %q", got, err, test.want)
		}
	}
	missing := hir.ParseType("$p9::missing::Value")
	for _, value := range []hir.Type{
		hir.TypeInvalid, hir.Option(hir.TypeInvalid), missing,
		hir.SharedBorrow(missing), hir.RawPointer(missing, true), hir.Slice(missing), hir.FixedArray(missing, 4),
		hir.Option(missing), hir.Result(missing, hir.TypeI32), hir.Result(hir.TypeI32, missing),
	} {
		if got, err := stableType(value, coordinates); err == nil || got != "" {
			t.Fatalf("partial or invalid stable type escaped: %q, %v", got, err)
		}
	}
	coordinates[1] = testCoordinateID(t, "new-writer-owner")
	if got, err := stableType(nominal, coordinates); err != nil || got == stable {
		t.Fatalf("stale package coordinate used: %q, %v", got, err)
	}
}

func BenchmarkStableTypeWrite(b *testing.B) {
	for _, depth := range []int{0, 16, 256, 1024} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			value := hir.TypeI32
			for range depth {
				value = hir.RawPointer(value, false)
			}
			want := strings.Repeat("ptr ", depth) + "i32"
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if got, err := stableType(value, nil); err != nil || got != want {
					b.Fatalf("type serialization changed: %v", err)
				}
			}
		})
	}
}

func BenchmarkDescribeContextNestedPointerTypes(b *testing.B) {
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: "stable-writer", VersionIdentity: "1"})
	if err != nil {
		b.Fatal(err)
	}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: coordinate}
	value := hir.TypeI32
	for range 64 {
		value = hir.RawPointer(value, false)
	}
	program := mir.Program{}
	for i := range 256 {
		program.ForeignFunctions = append(program.ForeignFunctions, mir.ForeignFunction{
			Symbol: hir.SymbolID(i + 1), Name: fmt.Sprintf("main::foreign%d", i), LinkName: fmt.Sprintf("foreign%d", i), ABI: "C",
			ParameterTypes: []hir.Type{value}, ResultType: hir.TypeVoid, Span: hir.Span{Package: hir.DefaultPackage},
		})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		context, err := DescribeContext(program, coordinates, nil)
		if err != nil || len(context.ForeignFunctions) != 256 {
			b.Fatalf("context serialization: %v", err)
		}
	}
}
