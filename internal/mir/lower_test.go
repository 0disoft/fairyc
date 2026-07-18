package mir

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestLowerPreservesResourceDefinitionAndFieldTypes(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Pair {
    left: i32;
    ready: bool;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let pair: Pair = Pair { left: 42, ready: true };
    pair.left
}
`)
	if len(program.ResourceTypes) != 1 {
		t.Fatalf("resource types = %+v", program.ResourceTypes)
	}
	definition := program.ResourceTypes[0]
	if definition.Type != hir.Type("main::Pair") || definition.Entity != "resource-type:main-pair" || definition.DropKind != hir.ResourceDropEmpty || len(definition.Fields) != 2 {
		t.Fatalf("resource definition = %+v", definition)
	}
	if definition.Fields[0].Name != "left" || definition.Fields[0].Type != hir.TypeI32 || definition.Fields[1].Name != "ready" || definition.Fields[1].Type != hir.TypeBool {
		t.Fatalf("resource fields = %+v", definition.Fields)
	}

	var construct, read *Operation
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			operations := program.Functions[functionIndex].Blocks[blockIndex].Operations
			for operationIndex := range operations {
				operation := &operations[operationIndex]
				switch operation.Kind {
				case OperationResourceConstruct:
					construct = operation
				case OperationFieldRead:
					read = operation
				}
			}
		}
	}
	if construct == nil || construct.Type != definition.Type || !slices.Equal(construct.Fields, []string{"left", "ready"}) {
		t.Fatalf("resource construction = %+v", construct)
	}
	if read == nil || read.OwnerType != definition.Type || read.Type != hir.TypeI32 || read.Field != "left" {
		t.Fatalf("field read = %+v", read)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerRejectsNonEmptyResourceDropKind(t *testing.T) {
	t.Parallel()
	_, err := Lower(hir.Package{Modules: []hir.Module{{
		Path: "main",
		Resources: []hir.Resource{{
			Name: "Token", Type: hir.Type("main::Token"), DropKind: hir.ResourceDropNonEmpty,
			Fields: []hir.ResourceField{{Name: "value", Type: hir.TypeI32}},
		}},
	}}})
	if err == nil || !strings.Contains(err.Error(), "unsupported drop kind") {
		t.Fatalf("Lower() error = %v", err)
	}
}

func TestCloneDeepCopiesResourceDefinitions(t *testing.T) {
	t.Parallel()
	original := Program{ResourceTypes: []ResourceType{{
		Type: hir.Type("main::Token"), DropKind: hir.ResourceDropEmpty,
		Fields: []ResourceField{{Name: "value", Type: hir.TypeI32}},
	}}}
	cloned := Clone(original)
	cloned.ResourceTypes[0].Fields[0].Name = "changed"
	if original.ResourceTypes[0].Fields[0].Name != "value" {
		t.Fatalf("Clone() aliased resource fields: original = %+v, clone = %+v", original, cloned)
	}
}

func lowerTestSource(t *testing.T, text string) Program {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil {
		t.Fatalf("AnalyzeSyntax() error = %v", err)
	}
	if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
		t.Fatalf("AnalyzeSyntax() diagnostics = %+v, AST = %+v", parsed.Tree.Diagnostics, parsed.AST)
	}
	result := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if result.CompletedThrough != semantic.PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("Analyze() result = %+v", result)
	}
	program, err := Lower(result.HIR)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	return program
}
