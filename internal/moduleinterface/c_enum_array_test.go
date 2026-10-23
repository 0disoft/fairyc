package moduleinterface

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestCEnumArrayPublicIdentity(t *testing.T) {
	seen := map[Hash]bool{}
	for _, shape := range []struct{ base, name, length string }{
		{"c::uint", "Status", "3"}, {"c::int", "Status", "3"}, {"c::long", "Status", "3"},
		{"c::uint", "Other", "3"}, {"c::uint", "Status", "2"},
	} {
		value := analyzeSource(t, fmt.Sprintf("module api; @repr(%s) pub enum %s{Ready;Busy;} @repr(C) pub struct Packet{%s[%s] values;}", shape.base, shape.name, shape.name, shape.length))
		descriptor, err := Extract(value, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: testCoordinateID(t, "enum-array")})
		if err != nil {
			t.Fatal(err)
		}
		hash, err := Compute(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		if seen[hash] {
			t.Fatalf("enum base/name/array length collision: %+v", shape)
		}
		seen[hash] = true
	}
}
