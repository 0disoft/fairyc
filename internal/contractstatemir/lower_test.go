package contractstatemir

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestLowerHIRLayoutsIsPackageOrderedAndCanonical(t *testing.T) {
	input := hir.Package{StateLayouts: []hir.StateLayout{
		{Package: 9, ContractUID: "example.wallet", Fields: []hir.StateField{
			{Name: "owners", Kind: hir.StateFieldMap, KeyTypeID: contractstate.Byte20.TypeID(), ValueTypeID: contractstate.Byte20.TypeID()},
		}},
		{Package: 2, ContractUID: "example.token", Fields: []hir.StateField{
			{Name: "total_supply", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()},
			{Name: "balances", Kind: hir.StateFieldMap, KeyTypeID: contractstate.Byte20.TypeID(), ValueTypeID: contractstate.U64.TypeID()},
		}},
	}}
	got, err := LowerHIRLayouts(input)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(got, []PackageLayout{{Package: 2}, {Package: 9}}, func(left, right PackageLayout) bool {
		return left.Package == right.Package
	}) {
		t.Fatalf("package order = %+v", got)
	}
	expected, err := BuildLayout("example.token", []Declaration{
		{Name: "total_supply", Kind: contractstate.FieldCell, ValueTypeID: contractstate.U64.TypeID()},
		{Name: "balances", Kind: contractstate.FieldMap, KeyTypeID: contractstate.Byte20.TypeID(), ValueTypeID: contractstate.U64.TypeID()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Layout.Hash != expected.Hash || got[0].Layout.ContractUID != expected.ContractUID {
		t.Fatalf("lowered token layout = %+v, want hash %s", got[0], expected.Hash)
	}
}

func TestLowerHIRLayoutsRejectsDuplicatePackageAndUnknownKind(t *testing.T) {
	duplicate := hir.Package{StateLayouts: []hir.StateLayout{
		{Package: 1, ContractUID: "example.one"},
		{Package: 1, ContractUID: "example.two"},
	}}
	if _, err := LowerHIRLayouts(duplicate); err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("duplicate error = %v", err)
	}

	unknown := hir.Package{StateLayouts: []hir.StateLayout{{
		Package: 1, ContractUID: "example.one",
		Fields: []hir.StateField{{Name: "value", Kind: "dynamic", ValueTypeID: contractstate.U64.TypeID()}},
	}}}
	if _, err := LowerHIRLayouts(unknown); err == nil || !strings.Contains(err.Error(), "unsupported field kind") {
		t.Fatalf("unknown kind error = %v", err)
	}
}
