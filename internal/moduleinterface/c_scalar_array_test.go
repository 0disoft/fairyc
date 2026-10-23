package moduleinterface

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestCScalarArrayPublicFieldIdentity(t *testing.T) {
	seen := map[Hash]string{}
	for _, typ := range []string{"c::int[2]", "c::uint[2]", "c::long[2]", "c::int[3]", "i32[2]", "c::float[2]", "f32[2]", "c::double[2]", "f64[2]", "c::bool[2]"} {
		value := analyzeSource(t, fmt.Sprintf("module api; @repr(C) pub struct Packet{%[1]s values;} pub fn %[1]s read(Packet p){return p.values;}", typ))
		descriptor, err := Extract(value, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: testCoordinateID(t, "c-array")})
		if err != nil {
			t.Fatal(err)
		}
		hash, err := Compute(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		if other, found := seen[hash]; found {
			t.Fatalf("%s and %s share a public interface identity", typ, other)
		}
		seen[hash] = typ
	}
}
