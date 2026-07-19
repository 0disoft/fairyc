package ast

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestLowerPreservesExplicitGrouping(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> i32 {\n  let value = (1 + 2) * 3;\n  value\n}\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Bindings) != 1 {
		t.Fatalf("bindings = %d, want 1", len(module.Bindings))
	}
	if got, want := NormalizeExpression(module.Bindings[0].Expression), "(* (group (+ 1 2)) 3)"; got != want {
		t.Fatalf("NormalizeExpression() = %q, want %q", got, want)
	}
}

func TestLowerRejectsDiagnosticTree(t *testing.T) {
	t.Parallel()
	input := []byte("module wrong;")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if _, err := Lower(tree); err == nil {
		t.Fatal("Lower() accepted diagnostic syntax tree")
	}
}

func TestLowerPreservesFunctionSemanticStructure(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn choose(flag: bool) -> i32 {\n  let value: i32 = 1;\n  return value;\n}\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Functions) != 1 {
		t.Fatalf("functions = %+v", module.Functions)
	}
	function := module.Functions[0]
	if function.Name != "choose" || function.ResultType.Text != "i32" || len(function.Parameters) != 1 || function.Body == nil {
		t.Fatalf("function = %+v", function)
	}
	if function.Parameters[0].Name != "flag" || function.Parameters[0].Type.Text != "bool" {
		t.Fatalf("parameters = %+v", function.Parameters)
	}
	if len(function.Body.Statements) != 2 || function.Body.Statements[0].Binding == nil || function.Body.Statements[1].Kind != syntax.KindReturn {
		t.Fatalf("body = %+v", function.Body)
	}
}

func TestLowerCapturesResourceFields(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nresource Pair { left: i32; right: i32; drop fn(self: &mut Self) {} }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Resources) != 1 || module.Resources[0].Name != "Pair" || len(module.Resources[0].Fields) != 2 {
		t.Fatalf("resources = %+v", module.Resources)
	}
	if module.Resources[0].Fields[0].Name != "left" || module.Resources[0].Fields[0].Type.Text != "i32" {
		t.Fatalf("first field = %+v", module.Resources[0].Fields[0])
	}
	if module.Resources[0].Drop == nil || module.Resources[0].Drop.Body == nil || len(module.Resources[0].Drop.Body.Statements) != 0 || module.Resources[0].Drop.Body.Tail != nil {
		t.Fatalf("drop = %+v, want preserved empty body", module.Resources[0].Drop)
	}
}

func TestLowerPreservesNonEmptyResourceDropBody(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nresource Token { value: i32; drop fn(self: &mut Self) { let marker: i32 = 1; } }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Resources) != 1 || module.Resources[0].Drop == nil || module.Resources[0].Drop.Body == nil {
		t.Fatalf("resources = %+v", module.Resources)
	}
	body := module.Resources[0].Drop.Body
	if len(body.Statements) != 1 || body.Statements[0].Binding == nil || body.Statements[0].Binding.Name != "marker" {
		t.Fatalf("drop body = %+v", body)
	}
}

func TestLowerCapturesStructFields(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nstruct Holder { scratch: Region; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Structs) != 1 || module.Structs[0].Name != "Holder" || len(module.Structs[0].Fields) != 1 {
		t.Fatalf("structs = %+v", module.Structs)
	}
	if module.Structs[0].Fields[0].Name != "scratch" || module.Structs[0].Fields[0].Type.Text != "Region" {
		t.Fatalf("field = %+v", module.Structs[0].Fields[0])
	}
}

func TestLowerPreservesStructAttributesAndArgumentOrder(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\n@repr(C)\nstruct Packet { data: *u8; length: usize; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Structs) != 1 || len(module.Structs[0].Attributes) != 1 {
		t.Fatalf("structs = %+v", module.Structs)
	}
	attribute := module.Structs[0].Attributes[0]
	if attribute.Name != "repr" || len(attribute.Arguments) != 1 || attribute.Arguments[0].Raw != "C" || attribute.Arguments[0].Cooked != "" {
		t.Fatalf("attribute = %+v", attribute)
	}
	if len(module.Structs[0].Fields) != 2 || module.Structs[0].Fields[0].Name != "data" || module.Structs[0].Fields[1].Name != "length" {
		t.Fatalf("fields = %+v", module.Structs[0].Fields)
	}
}

func TestLowerCapturesEnumType(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nenum Error { Failed; Os { code: i32; retryable: bool; }; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Enums) != 1 || module.Enums[0].Name != "Error" || len(module.Enums[0].Variants) != 2 {
		t.Fatalf("enums = %+v", module.Enums)
	}
	if module.Enums[0].Variants[0].Name != "Failed" || len(module.Enums[0].Variants[0].Fields) != 0 {
		t.Fatalf("payloadless variant = %+v", module.Enums[0].Variants[0])
	}
	variant := module.Enums[0].Variants[1]
	if variant.Name != "Os" || len(variant.Fields) != 2 || variant.Fields[0].Name != "code" || variant.Fields[0].Type.Text != "i32" || variant.Fields[1].Name != "retryable" || variant.Fields[1].Type.Text != "bool" {
		t.Fatalf("payload variant = %+v", variant)
	}
}

func TestLowerPreservesForeignDeclarationAndUnsafeBoundaries(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\n@link_name(\"fairyc_fixture_add_\\x6fne\")\nunsafe extern C fn add_one(value: i32) -> i32;\nunsafe fn wrapper(value: i32) -> i32 { add_one(value) }\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.ForeignFunctions) != 1 {
		t.Fatalf("foreign functions = %+v", module.ForeignFunctions)
	}
	foreign := module.ForeignFunctions[0]
	if foreign.Name != "add_one" || foreign.ABI != "C" || len(foreign.Parameters) != 1 || foreign.Parameters[0].Type.Text != "i32" || foreign.ResultType.Text != "i32" || len(foreign.Attributes) != 1 {
		t.Fatalf("foreign function = %+v", foreign)
	}
	attribute := foreign.Attributes[0]
	if attribute.Name != "link_name" || len(attribute.Arguments) != 1 || attribute.Arguments[0].Raw != "\"fairyc_fixture_add_\\x6fne\"" || attribute.Arguments[0].Cooked != "fairyc_fixture_add_one" {
		t.Fatalf("foreign attribute = %+v", attribute)
	}
	if len(module.Functions) != 2 || !module.Functions[0].Unsafe || module.Functions[1].Unsafe {
		t.Fatalf("functions = %+v", module.Functions)
	}
	if module.Functions[1].Body == nil || module.Functions[1].Body.Tail == nil || module.Functions[1].Body.Tail.Kind != syntax.KindUnsafe {
		t.Fatalf("main tail = %+v", module.Functions[1].Body)
	}
}

func TestLowerPreservesCastTargetTypeSeparatelyFromOperand(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> i32 { var value: i32 = 0; let pointer: *mut i32 = unsafe { &mut value as *mut i32 }; 0 }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Functions) != 1 || module.Functions[0].Body == nil || len(module.Functions[0].Body.Statements) != 2 {
		t.Fatalf("functions = %+v", module.Functions)
	}
	unsafeExpression := module.Functions[0].Body.Statements[1].Binding.Expression
	if unsafeExpression == nil || unsafeExpression.Kind != syntax.KindUnsafe || len(unsafeExpression.Children) != 1 || unsafeExpression.Children[0].Block == nil {
		t.Fatalf("unsafe expression = %+v", unsafeExpression)
	}
	cast := unsafeExpression.Children[0].Block.Tail
	if cast == nil || cast.Kind != syntax.KindCast || cast.Type == nil || cast.Type.Text != "*mut i32" || len(cast.Children) != 1 || cast.Children[0].Kind != syntax.KindPrefix {
		t.Fatalf("cast = %+v", cast)
	}
	if got, want := NormalizeExpression(cast), "(as (&mut value) *mut i32)"; got != want {
		t.Fatalf("NormalizeExpression() = %q, want %q", got, want)
	}
}
