package mir

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerificationErrorsRetainDistinctDetailsAndSortDeterministically(t *testing.T) {
	expected := []VerificationError{
		{Code: CodeOperationShape, Entity: "one", Detail: "first"},
		{Code: CodeOperationShape, Entity: "one", Detail: "second"},
		{Code: CodeOperationType, Entity: "one", Detail: "first"},
	}
	input := []VerificationError{expected[2], expected[1], expected[0], expected[1]}
	for range 2 {
		if actual := canonicalVerificationErrors(slices.Clone(input)); !slices.Equal(actual, expected) {
			t.Fatalf("lost or reordered independent errors: %+v", actual)
		}
		slices.Reverse(input)
	}
}

func TestVerifyStorageFailureCategoriesRemainIndependent(t *testing.T) {
	program := Program{Functions: []Function{{
		Symbol: 1, Name: "main::main", ResultType: hir.TypeVoid, Entry: 0,
		Blocks: []Block{{ID: 0,
			Operations: []Operation{{Kind: OperationStorageTake, Entity: "take", DestinationLocal: NoLocal}},
			Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock},
		}},
	}}}
	for _, transfers := range []bool{false, true} {
		capabilities := admission.Capabilities{}
		if transfers {
			capabilities.Storage = admission.StorageTransfers
		}
		failures := VerifyWithCapabilities(program, capabilities)
		var codes []string
		for _, failure := range failures {
			if failure.Entity != "take" {
				t.Fatalf("unexpected entity: %+v", failures)
			}
			codes = append(codes, failure.Code)
		}
		expected := []string{CodeStorageAdmission, CodeStorageShape, CodeStorageProvenance}
		if transfers {
			expected = expected[1:]
		}
		if !slices.Equal(codes, expected) {
			t.Fatalf("transfers=%t: %+v", transfers, failures)
		}
	}
}
