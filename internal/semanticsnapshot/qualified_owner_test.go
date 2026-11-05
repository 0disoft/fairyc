package semanticsnapshot

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestFunctionOwnerQualificationIsAppliedOnce(t *testing.T) {
	metadata := map[hir.PackageID]packageMetadata{hir.DefaultPackage: {instance: "instance", coord: "coordinate"}}
	module := hir.Module{Package: hir.DefaultPackage, Path: "main"}
	for _, name := range []string{"get", "Cell::get"} {
		function := hir.Function{Name: name, ResultType: hir.TypeI32, OwnerType: hir.ParseType("main::Cell"), Receiver: "shared", Span: hir.Span{Package: hir.DefaultPackage, Source: "main.fy", EndByte: 20}}
		got, err := functionDeclaration(module, function, metadata)
		if err != nil || got.Key.Name != "Cell::get" || got.Owner == nil || got.Owner.Name != "main::Cell" || got.Receiver != "shared" {
			t.Fatalf("owner qualification %q: %+v %v", name, got, err)
		}
		if function.Name != name {
			t.Fatal("owner normalization mutated HIR")
		}
	}
}
