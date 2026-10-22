package moduleinterface

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestIntegerTransportPublicFieldIdentity(t *testing.T) {
	seen := map[Hash]bool{}
	for _, typ := range []string{"i8", "i16", "u16", "u64", "isize", "c::longlong", "i8[2]", "i16[2]", "u16[2]", "u64[2]", "isize[2]", "i8[3]"} {
		value := analyzeSource(t, fmt.Sprintf("module api; @repr(C) pub struct Point{%[1]s x;} pub fn %[1]s read(Point p){return p.x;}", typ))
		descriptor, err := Extract(value, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: testCoordinateID(t, "integer")})
		if err != nil {
			t.Fatal(err)
		}
		hash, err := Compute(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		if seen[hash] {
			t.Fatalf("%s shares a public interface identity with another integer type", typ)
		}
		seen[hash] = true
	}
}
