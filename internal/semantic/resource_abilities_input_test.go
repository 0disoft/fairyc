package semantic

import (
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
)

func TestCloneResourceAbilityInputsPreservesCallerOwnedAST(t *testing.T) {
	t.Parallel()

	module := &ast.Module{
		Path: "main",
		Resources: []ast.Resource{{
			Name: "Coin",
			Attributes: []ast.Attribute{{
				Name: resourceAbilitiesAttribute,
			}},
		}},
	}
	files := []File{{Path: "main.fy", Module: module}}

	cloned := cloneResourceAbilityInputs(files)
	if len(cloned) != 1 || cloned[0].Module == nil || cloned[0].Module == module {
		t.Fatalf("clone did not create one private module: %+v", cloned)
	}
	if len(cloned[0].Module.Resources) != 1 || len(cloned[0].Module.Resources[0].Attributes) != 1 {
		t.Fatalf("clone lost resource ability input: %+v", cloned[0].Module.Resources)
	}

	cloned[0].Module.Resources[0].Attributes[0].Name = "effects"
	cloned[0].Module.Resources[0].Attributes = nil

	if len(module.Resources) != 1 || len(module.Resources[0].Attributes) != 1 ||
		module.Resources[0].Attributes[0].Name != resourceAbilitiesAttribute {
		t.Fatalf("caller-owned AST was mutated: %+v", module.Resources)
	}
}
