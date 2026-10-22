package moduleinterface

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestCEnumBaseChangesPublicInterface(t *testing.T) {
	value := analyzeSource(t, "module api; @repr(c::uint) pub enum State { Ready; Busy; }")
	descriptor, err := Extract(value, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: testCoordinateID(t, "c-enum")})
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Modules[0].Items[0].CBase != "c.uint" {
		t.Fatal("public enum base lost")
	}
	first, err := Compute(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	descriptor.Modules[0].Items[0].CBase = "c.ushort"
	second, err := Compute(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("C enum base did not invalidate public interface")
	}
}
