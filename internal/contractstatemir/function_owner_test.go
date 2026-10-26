package contractstatemir

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestTransactionOwnerIndexPreservesLookupScope(t *testing.T) {
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: {1}}
	baseline, err := ComputeHIRStateBodyHashes(stateFunctionLookupFixture(t, 4, 2), []hir.PackageID{1}, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		packageID hir.PackageID
		function  int
		wantError string
	}{
		{"unrelated duplicate", 1, 0, ""},
		{"same symbol in another package", 2, 3, ""},
		{"used duplicate across modules", 1, 3, "function symbol 4 is duplicated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := stateFunctionLookupFixture(t, 4, 2)
			input.Modules = append(input.Modules, hir.Module{
				Package: tc.packageID, Path: "other",
				Functions: []hir.Function{input.Modules[0].Functions[tc.function]},
			})
			hashes, err := ComputeHIRStateBodyHashes(input, []hir.PackageID{1}, coordinates)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("owner ambiguity error = %v", err)
				}
				return
			}
			if err != nil || hashes[1] != baseline[1] {
				t.Fatalf("unrelated owner changed the stable body hash: %v", err)
			}
		})
	}
}

func TestTransactionOwnerIndexIsOperationLocal(t *testing.T) {
	input := stateFunctionLookupFixture(t, 4, 2)
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: {1}}
	before, err := ComputeHIRStateBodyHashes(input, []hir.PackageID{1}, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	input.Modules[0].Functions[3].Name = "renamed"
	after, err := ComputeHIRStateBodyHashes(input, []hir.PackageID{1}, coordinates)
	if err != nil || after[1] == before[1] {
		t.Fatalf("later operation reused stale owner metadata: %v", err)
	}
}
