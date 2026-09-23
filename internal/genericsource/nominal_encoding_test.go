package genericsource

import (
	"encoding/json"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
)

func TestNominalContextRetainsLegacyEncoding(t *testing.T) {
	provider := &expander{owner: "sdk", module: &ast.Module{Path: "std::vec", Functions: []ast.Function{{Name: "load"}}}}
	consumer := &expander{owner: "client", module: &ast.Module{Path: "main", SourceSize: 7, Literals: []ast.Literal{{Raw: `"a<&>"`, Cooked: "a<&>"}}}, imports: map[string]*expander{"v": provider}}
	modules := []*expander{provider, consumer}
	if err := bindNominalContext(modules); err != nil {
		t.Fatal(err)
	}
	type legacyEntry struct {
		Owner   genericplan.PackageID
		Module  *ast.Module
		Imports map[string]genericplan.PackageID
	}
	legacy, err := json.Marshal([]legacyEntry{
		{Owner: consumer.owner, Module: consumer.module, Imports: map[string]genericplan.PackageID{"v": provider.owner}},
		{Owner: provider.owner, Module: provider.module, Imports: map[string]genericplan.PackageID{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := genericplan.DigestText("source-nominal-closed-graph-v0", string(legacy)).String()
	for _, module := range modules {
		if module.nominalContext != want {
			t.Fatalf("nominal context for %s = %q, want %q", module.module.Path, module.nominalContext, want)
		}
		encoded, err := json.Marshal(module.module)
		if err != nil {
			t.Fatal(err)
		}
		if string(module.nominalModuleJSON) != string(encoded) {
			t.Fatalf("nominal module encoding changed for %s", module.module.Path)
		}
	}
}
