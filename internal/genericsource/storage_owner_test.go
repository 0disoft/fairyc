package genericsource_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/source"
)

func ownedStorageInputs(t *testing.T, client string) []genericsource.PackageModule {
	t.Helper()
	texts := []string{client, `module provider; pub fn usize size<T>() { return storage_size<T>(); }`}
	owners := []genericplan.PackageID{"client", "sdk"}
	paths := []string{"main.fy", "provider.fy"}
	inputs := make([]genericsource.PackageModule, 2)
	for i, text := range texts {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(paths[i], []byte(text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
			t.Fatalf("parse owner fixture: %v %+v", err, parsed.Tree)
		}
		inputs[i] = genericsource.PackageModule{Module: parsed.AST, Owner: owners[i], Imports: map[string]genericplan.PackageID{"provider": "sdk"}}
	}
	return inputs
}

func TestOwnedStorageProviderCallPreservesSource(t *testing.T) {
	inputs := ownedStorageInputs(t, `module main; import provider as p; fn usize main() { return p::size<i32>(); }`)
	before, _ := json.Marshal(inputs)
	result, err := genericsource.PrepareOwnedStorageTransferPackages(inputs, genericplan.DefaultPlanLimits(), []genericplan.PackageID{"sdk"})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(inputs)
	if string(before) != string(after) {
		t.Fatal("owner-scoped preparation mutated source")
	}
	encoded, _ := json.Marshal(result.Modules)
	if !strings.Contains(string(encoded), "private-storage-layout") {
		t.Fatal("selected provider storage query did not expand")
	}
}

func TestOwnedStorageRejectsConsumerIntrinsics(t *testing.T) {
	for _, operation := range []string{"storage_size<i32>()", "storage_alignment<i32>()", "storage_take<i32>(null)", "storage_put<i32>(null, 1)", "storage_borrow<i32>(null, null)", "storage_borrow_mut<i32>(null, null)"} {
		t.Run(operation, func(t *testing.T) {
			inputs := ownedStorageInputs(t, "module main; import provider as p; fn usize main() { return "+operation+"; }")
			if _, err := genericsource.PrepareOwnedStorageTransferPackages(inputs, genericplan.DefaultPlanLimits(), []genericplan.PackageID{"sdk"}); err == nil {
				t.Fatal("consumer private intrinsic expanded")
			}
		})
	}
}

func TestOwnedStorageOwnerSetMustBeExact(t *testing.T) {
	inputs := ownedStorageInputs(t, `module main; import provider as p; fn usize main() { return p::size<i32>(); }`)
	for _, owners := range [][]genericplan.PackageID{nil, {}, {""}, {"unknown"}, {"sdk", "sdk"}, {"client"}} {
		if _, err := genericsource.PrepareOwnedStorageTransferPackages(inputs, genericplan.DefaultPlanLimits(), owners); err == nil {
			t.Fatalf("invalid or non-provider owner set admitted: %v", owners)
		}
	}
}
