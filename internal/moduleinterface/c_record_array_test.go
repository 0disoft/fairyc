package moduleinterface

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestCRecordArrayPublicIdentity(t *testing.T) {
	seen := map[Hash]bool{}
	for _, shape := range []struct {
		name, scalar string
		length       int
	}{
		{"Element", "f64", 3}, {"Element", "f32", 3}, {"Other", "f64", 3}, {"Element", "f64", 2}, {"Element", "c::double", 3},
	} {
		source := fmt.Sprintf("module api; @repr(C) pub struct %s{pub %s value;} @repr(C) pub struct Packet{pub %s[%d] values;}", shape.name, shape.scalar, shape.name, shape.length)
		value := analyzeSource(t, source)
		descriptor, err := Extract(value, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: testCoordinateID(t, "record-array")})
		if err != nil {
			t.Fatal(err)
		}
		hash, err := Compute(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		if seen[hash] {
			t.Fatalf("record element/field/length collision: %+v", shape)
		}
		seen[hash] = true
	}
}
