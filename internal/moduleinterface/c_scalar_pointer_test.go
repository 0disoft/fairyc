package moduleinterface

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestCScalarPointerPublicIdentity(t *testing.T) {
	seen := map[Hash]string{}
	for _, typ := range []string{"ptr c::int", "ptr mut c::int", "ptr mut ptr mut c::int", "ptr c::long", "ptr c::uint", "ptr i32", "ptr c::float", "ptr c::double", "ptr c::bool"} {
		value := analyzeSource(t, fmt.Sprintf("module api; @repr(C) pub struct Packet{%[1]s value;} pub fn %[1]s identity(%[1]s p){return p;}", typ))
		descriptor, err := Extract(value, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: testCoordinateID(t, "c-scalar-pointer")})
		if err != nil {
			t.Fatal(err)
		}
		hash, err := Compute(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		if other, found := seen[hash]; found {
			t.Fatalf("%s and %s share identity", typ, other)
		}
		seen[hash] = typ
	}
}
