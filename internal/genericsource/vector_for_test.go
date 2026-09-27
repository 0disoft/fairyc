package genericsource

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func vectorForSourceModule(t *testing.T) *ast.Module {
	t.Helper()
	input := []byte("module main; import std::vec as items; " +
		"fn void show(ref items::Vec<i32> values) { if (true) { for (let ref i32 item in values) {} } }")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	module, err := ast.Lower(tree)
	if err != nil || len(tree.Diagnostics) != 0 {
		t.Fatalf("vector source: %+v, %v", tree.Diagnostics, err)
	}
	return module
}

func TestSDKVectorLoopPlansOnlyExactProviderSpecializations(t *testing.T) {
	client := vectorForSourceModule(t)
	providerSource := []byte(`module std::vec;
pub resource Vec<T> { usize length; cleanup {} }
pub fn usize len<T>(ref Vec<T> value) { return value.length; }
pub fn ref T at<T>(ref Vec<T> value, usize index) { return null; }`)
	tree := syntax.Parse(source.NewFile("std/vec.fy", providerSource), source.DefaultLimits())
	provider, err := ast.Lower(tree)
	if err != nil || len(tree.Diagnostics) != 0 {
		t.Fatalf("SDK fixture: %+v, %v", tree.Diagnostics, err)
	}
	inputs := []PackageModule{
		{Module: client, Owner: "client", Imports: map[string]genericplan.PackageID{"std::vec": "sdk"}},
		{Module: provider, Owner: "sdk"},
	}
	ordinary, err := PrepareOwnedStorageTransferPackages(inputs, genericplan.DefaultPlanLimits(), []genericplan.PackageID{"sdk"})
	if err != nil || vectorForNestedStatement(ordinary.Modules[0]).Expression != nil {
		t.Fatalf("ordinary owner lane activated vector source: %v", err)
	}
	expansion, err := PrepareSDKStoragePackages(inputs, genericplan.DefaultPlanLimits(), []genericplan.PackageID{"sdk"})
	if err != nil {
		t.Fatal(err)
	}
	if vectorForNestedStatement(client).Expression != nil ||
		vectorForNestedStatement(client).ByteFor.PreparedVector != nil ||
		vectorForNestedStatement(expansion.Modules[0]).Expression == nil {
		t.Fatal("SDK source mutated or vector loop not prepared")
	}
	seen := map[string]bool{}
	for _, specialization := range expansion.Plan.Specializations {
		if strings.HasPrefix(string(specialization.Declaration), "sdk::std::vec::") {
			seen[strings.TrimPrefix(string(specialization.Declaration), "sdk::std::vec::")] = true
		}
	}
	if !seen["len"] || !seen["at"] {
		t.Fatalf("prepared loop did not plan exact SDK calls: %v", seen)
	}
}

func vectorForNestedStatement(module *ast.Module) *ast.Statement {
	return &module.Functions[0].Body.Statements[0].Expression.Children[1].Block.Statements[0]
}

func TestAuthenticatedSDKVectorLoopActivatesPlannerCallsWithoutChangingSource(t *testing.T) {
	original := vectorForSourceModule(t)
	provider := &expander{module: &ast.Module{Path: "std::vec"}, allowStorage: true}
	client := &expander{module: original, imports: map[string]*expander{"items": provider}}
	if err := client.activateSDKVectorLoops(); err != nil {
		t.Fatal(err)
	}
	if client.module == original || vectorForNestedStatement(original).Expression != nil ||
		vectorForNestedStatement(original).ByteFor.PreparedVector != nil {
		t.Fatal("SDK activation changed the original source")
	}
	active := vectorForNestedStatement(client.module)
	if active.Expression == nil || active.Expression != active.ByteFor.PreparedVector {
		t.Fatalf("prepared loop was not activated: %+v", active)
	}
	requests := 0
	_, err := (transform{expression: func(expression *ast.Expression) error {
		if expression.Kind == syntax.KindSpecialize {
			requests++
		}
		return nil
	}}).function(client.module.Functions[0])
	if err != nil || requests != 2 {
		t.Fatalf("planned SDK len/at calls = %d, error = %v", requests, err)
	}
}

func TestAuthenticatedSDKVectorLoopInClassicForInitializer(t *testing.T) {
	input := []byte(`module main; import std::vec as items;
fn void show(ref items::Vec<i32> values) {
    for (var i32 index = { for (let ref i32 item in values) {} 0 }; index < 1; index += 1) {}
}`)
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	original, err := ast.Lower(tree)
	if err != nil || len(tree.Diagnostics) != 0 {
		t.Fatalf("classic for initializer: %+v, %v", tree.Diagnostics, err)
	}
	loop := func(module *ast.Module) *ast.Statement {
		return &module.Functions[0].Body.Statements[0].For.Initializer.Binding.Expression.Block.Statements[0]
	}
	provider := &expander{module: &ast.Module{Path: "std::vec"}, allowStorage: true}
	client := &expander{module: original, imports: map[string]*expander{"items": provider}}
	if err := client.activateSDKVectorLoops(); err != nil {
		t.Fatal(err)
	}
	if client.module == original || loop(original).Expression != nil ||
		loop(original).ByteFor.PreparedVector != nil {
		t.Fatal("SDK activation changed the original initializer")
	}
	active := loop(client.module)
	if active.ByteFor.PreparedVector == nil || active.Expression != active.ByteFor.PreparedVector {
		t.Fatal("nested vector loop was discovered but not prepared")
	}
	requests := 0
	_, err = (transform{expression: func(expression *ast.Expression) error {
		if expression.Kind == syntax.KindSpecialize {
			requests++
		}
		return nil
	}}).function(client.module.Functions[0])
	if err != nil || requests != 2 {
		t.Fatalf("planned SDK len/at calls = %d, error = %v", requests, err)
	}
}

func TestVectorLoopDoesNotActivateForUnownedOrWrongProvider(t *testing.T) {
	for _, sample := range []struct {
		name     string
		provider *expander
	}{
		{name: "unowned SDK spelling", provider: &expander{module: &ast.Module{Path: "std::vec"}}},
		{name: "owned wrong module", provider: &expander{module: &ast.Module{Path: "other::vec"}, allowStorage: true}},
		{name: "unbound import"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			original := vectorForSourceModule(t)
			client := &expander{module: original, imports: map[string]*expander{"items": sample.provider}}
			if err := client.activateSDKVectorLoops(); err != nil {
				t.Fatal(err)
			}
			if client.module != original || vectorForNestedStatement(original).Expression != nil {
				t.Fatal("unauthenticated vector candidate became executable")
			}
		})
	}
}
