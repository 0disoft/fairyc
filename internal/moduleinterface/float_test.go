package moduleinterface

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestFloatingPublicFieldIdentity(t *testing.T) {
	value := analyzeSource(t, "module api; @repr(C) pub struct Point{f32 x;f64 y;} pub fn f32 read(Point p){return p.x;}")
	descriptor, err := Extract(value, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: testCoordinateID(t, "float")})
	if err != nil {
		t.Fatal(err)
	}
	first, err := Compute(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	for mi := range descriptor.Modules {
		for ii := range descriptor.Modules[mi].Items {
			item := &descriptor.Modules[mi].Items[ii]
			if item.Kind == "struct" {
				if item.Fields[0].Type != "f32" || item.Fields[1].Type != "f64" {
					t.Fatal("floating field type lost")
				}
				item.Fields[0].Type = "c::float"
			}
		}
	}
	second, err := Compute(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("Fairy and C float field types share an interface identity")
	}
}
