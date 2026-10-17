package mir

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestInlineNominalArrayFieldBorrowGuard(t *testing.T) {
	token := hir.ParseType("main::Token")
	owner := hir.ParseType("main::Owner")
	nominal := hir.FixedArray(token, 2)
	scalar := hir.FixedArray(hir.TypeI64, 2)
	lookup := func(typ hir.Type) *StructType {
		switch typ {
		case token:
			return &StructType{Type: token, Fields: []StructField{{Name: "x", Type: hir.TypeI64}}}
		case owner:
			return &StructType{Type: owner, Fields: []StructField{{Name: "tokens", Type: nominal}, {Name: "scalars", Type: scalar}}}
		default:
			return nil
		}
	}
	for _, borrow := range []func(hir.Type) hir.Type{hir.SharedBorrow, hir.MutableBorrow} {
		if !validMutableFieldBorrow(Operation{OwnerType: borrow(owner), Type: borrow(nominal)}, lookup) {
			t.Fatal("valid nominal array field address rejected")
		}
		for _, invalid := range []hir.Type{hir.FixedArray(hir.ParseType("main::Missing"), 2), hir.FixedArray(token, 4097)} {
			if validMutableFieldBorrow(Operation{OwnerType: borrow(owner), Type: borrow(invalid)}, lookup) {
				t.Fatal("invalid nominal array address admitted")
			}
		}
		if !validMutableFieldBorrow(Operation{OwnerType: borrow(owner), Type: borrow(scalar)}, lookup) {
			t.Fatal("existing scalar array field reference rejected")
		}
	}
}
