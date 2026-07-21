package ast

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestLowerPreservesExplicitGrouping(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() {\n  let value = (1 + 2) * 3;\n  return value;\n}\n")
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

func TestLowerPreservesFixedArrayLiteralForm(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() { let i32[3] values = [1, 2, 3]; return values[1]; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Bindings) != 1 || module.Bindings[0].Type == nil || module.Bindings[0].Type.Text != "i32[3]" {
		t.Fatalf("bindings = %+v", module.Bindings)
	}
	typeReference := module.Bindings[0].Type
	if typeReference.Kind != TypeKindArray || typeReference.Element == nil || typeReference.Element.Kind != TypeKindNamed ||
		typeReference.Element.Path != "i32" || typeReference.Length == nil || typeReference.Length.Text != "3" {
		t.Fatalf("array type = %+v", typeReference)
	}
	if got, want := NormalizeExpression(module.Bindings[0].Expression), "(array 1 2 3)"; got != want {
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
	input := []byte("module main;\nfn i32 choose(bool flag) {\n  let i32 value = 1;\n  return value;\n}\n")
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

func TestLowerPreservesMigratedMutableParameterAsLocal(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 close(File input, u32 status) {\n    var File file = input; return 0; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Functions) != 1 || len(module.Functions[0].Parameters) != 2 {
		t.Fatalf("functions = %+v", module.Functions)
	}
	parameters := module.Functions[0].Parameters
	if parameters[0].Name != "input" || parameters[0].Mutable || parameters[0].Type.Text != "File" ||
		parameters[1].Name != "status" || parameters[1].Mutable || parameters[1].Type.Text != "u32" {
		t.Fatalf("parameters = %+v", parameters)
	}
	body := module.Functions[0].Body
	if body == nil || len(body.Statements) != 2 || body.Statements[0].Binding == nil ||
		body.Statements[0].Binding.Name != "file" || !body.Statements[0].Binding.Mutable {
		t.Fatalf("migrated mutable local = %+v", body)
	}
}

func TestLowerCapturesResourceFields(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nresource Pair { i32 left; i32 right; cleanup {} }\n")
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
	if module.Resources[0].Cleanup == nil || module.Resources[0].Cleanup.Body == nil || len(module.Resources[0].Cleanup.Body.Statements) != 0 || module.Resources[0].Cleanup.Body.Tail != nil {
		t.Fatalf("cleanup = %+v, want preserved empty body", module.Resources[0].Cleanup)
	}
}

func TestLowerPreservesNonEmptyResourceDropBody(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nresource Token { i32 value; cleanup { let i32 marker = 1; } }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Resources) != 1 || module.Resources[0].Cleanup == nil || module.Resources[0].Cleanup.Body == nil {
		t.Fatalf("resources = %+v", module.Resources)
	}
	body := module.Resources[0].Cleanup.Body
	if len(body.Statements) != 1 || body.Statements[0].Binding == nil || body.Statements[0].Binding.Name != "marker" {
		t.Fatalf("drop body = %+v", body)
	}
}

func TestLowerPreservesOwnerQualifiedFunction(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nstruct Counter { i32 value; }\nfn i32 Counter::double(i32 value) { return value * 2; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Functions) != 1 || module.Functions[0].Owner != "Counter" || module.Functions[0].Name != "double" {
		t.Fatalf("functions = %+v", module.Functions)
	}
	if module.Functions[0].OwnerSpan.EndByte <= module.Functions[0].OwnerSpan.StartByte {
		t.Fatalf("owner span = %+v", module.Functions[0].OwnerSpan)
	}
}

func TestLowerCapturesStructFields(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nstruct Holder { Region scratch; }\n")
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
	input := []byte("module main;\n@repr(C)\nstruct Packet { ptr u8 data; usize length; }\n")
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
	input := []byte("module main;\nenum Error { Failed; Os { i32 code; bool retryable; }; }\n")
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
	input := []byte("module main;\n@link_name(\"fairyc_fixture_add_\\x6fne\")\nunsafe extern C fn i32 add_one(i32 value) ;\nunsafe fn i32 wrapper(i32 value) { return add_one(value); }\nfn i32 main() { return unsafe { add_one(41) }; }\n")
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
	mainBody := module.Functions[1].Body
	if mainBody == nil || len(mainBody.Statements) != 1 || mainBody.Statements[0].Kind != syntax.KindReturn ||
		mainBody.Statements[0].Expression == nil || mainBody.Statements[0].Expression.Kind != syntax.KindUnsafe {
		t.Fatalf("main return = %+v", mainBody)
	}
}

func TestLowerPreservesCastTargetTypeSeparatelyFromOperand(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() { var i32 value = 0; let ptr mut i32 pointer = unsafe { ref mut value as ptr mut i32 }; return 0; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Functions) != 1 || module.Functions[0].Body == nil || len(module.Functions[0].Body.Statements) != 3 {
		t.Fatalf("functions = %+v", module.Functions)
	}
	unsafeExpression := module.Functions[0].Body.Statements[1].Binding.Expression
	if unsafeExpression == nil || unsafeExpression.Kind != syntax.KindUnsafe || len(unsafeExpression.Children) != 1 || unsafeExpression.Children[0].Block == nil {
		t.Fatalf("unsafe expression = %+v", unsafeExpression)
	}
	cast := unsafeExpression.Children[0].Block.Tail
	if cast == nil || cast.Kind != syntax.KindCast || cast.Type == nil || cast.Type.Text != "ptr mut i32" || len(cast.Children) != 1 || cast.Children[0].Kind != syntax.KindPrefix {
		t.Fatalf("cast = %+v", cast)
	}
	if cast.Type.Kind != TypeKindRawPointer || !cast.Type.Mutable || cast.Type.Element == nil || cast.Type.Element.Path != "i32" {
		t.Fatalf("cast target type = %+v", cast.Type)
	}
	if got, want := NormalizeExpression(cast), "(as (&mut value) ptr mut i32)"; got != want {
		t.Fatalf("NormalizeExpression() = %q, want %q", got, want)
	}
}

func TestLowerPreservesNestedTypeStructure(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn unsafe extern C fn u32(i32, ptr mut u8) inspect(Result<Option<ref mut u8[]>, ptr void> value) { return callback; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Functions) != 1 || len(module.Functions[0].Parameters) != 1 {
		t.Fatalf("functions = %+v", module.Functions)
	}
	parameter := module.Functions[0].Parameters[0].Type
	if parameter.Kind != TypeKindNamed || parameter.Path != "Result" || len(parameter.Arguments) != 2 {
		t.Fatalf("parameter type = %+v", parameter)
	}
	option := parameter.Arguments[0]
	if option.Kind != TypeKindNamed || option.Path != "Option" || len(option.Arguments) != 1 {
		t.Fatalf("option type = %+v", option)
	}
	reference := option.Arguments[0]
	if reference.Kind != TypeKindReference || !reference.Mutable || reference.Element == nil ||
		reference.Element.Kind != TypeKindSlice || reference.Element.Element == nil || reference.Element.Element.Path != "u8" {
		t.Fatalf("reference type = %+v", reference)
	}
	pointer := parameter.Arguments[1]
	if pointer.Kind != TypeKindRawPointer || pointer.Mutable || pointer.Element == nil || pointer.Element.Path != "void" {
		t.Fatalf("pointer type = %+v", pointer)
	}
	result := module.Functions[0].ResultType
	if result.Kind != TypeKindForeignFunction || result.ABI != "C" || len(result.Parameters) != 2 || result.Result == nil || result.Result.Path != "u32" ||
		result.Parameters[0].Path != "i32" || result.Parameters[1].Kind != TypeKindRawPointer || !result.Parameters[1].Mutable {
		t.Fatalf("function result type = %+v", result)
	}
}

func TestLowerPreservesCShapedTypeBinding(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn ptr u8[16] inspect(ref mut u8[] values, (ptr u8)[16] callbacks) { return callback; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Functions) != 1 || len(module.Functions[0].Parameters) != 2 {
		t.Fatalf("functions = %+v", module.Functions)
	}
	values := module.Functions[0].Parameters[0].Type
	if values.Kind != TypeKindReference || !values.Mutable || values.Element == nil || values.Element.Kind != TypeKindSlice ||
		values.Element.Element == nil || values.Element.Element.Path != "u8" {
		t.Fatalf("values type = %+v", values)
	}
	callbacks := module.Functions[0].Parameters[1].Type
	if callbacks.Kind != TypeKindArray || callbacks.Length == nil || callbacks.Length.Text != "16" || callbacks.Element == nil ||
		callbacks.Element.Kind != TypeKindGrouped || callbacks.Element.Element == nil || callbacks.Element.Element.Kind != TypeKindRawPointer {
		t.Fatalf("callbacks type = %+v", callbacks)
	}
	result := module.Functions[0].ResultType
	if result.Kind != TypeKindRawPointer || result.Element == nil || result.Element.Kind != TypeKindArray || result.Element.Length == nil || result.Element.Length.Text != "16" {
		t.Fatalf("result type = %+v", result)
	}
}

func TestLowerPreservesCShapedDeclarationOrder(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nstruct Pair { i32 left; i32 right; }\nfn i32 add(i32 left, i32 right) { let i32 sum = left + right; return sum; }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Structs) != 1 || len(module.Structs[0].Fields) != 2 || module.Structs[0].Fields[0].Name != "left" || module.Structs[0].Fields[0].Type.Path != "i32" {
		t.Fatalf("structs = %+v", module.Structs)
	}
	if len(module.Functions) != 1 || module.Functions[0].Name != "add" || module.Functions[0].ResultType.Path != "i32" || len(module.Functions[0].Parameters) != 2 {
		t.Fatalf("functions = %+v", module.Functions)
	}
	function := module.Functions[0]
	if function.Parameters[0].Name != "left" || function.Parameters[0].Type.Path != "i32" || function.Parameters[1].Name != "right" || function.Parameters[1].Type.Path != "i32" {
		t.Fatalf("parameters = %+v", function.Parameters)
	}
	if function.Body == nil || len(function.Body.Statements) != 2 || function.Body.Statements[0].Binding == nil || function.Body.Statements[0].Binding.Name != "sum" || function.Body.Statements[0].Binding.Type == nil || function.Body.Statements[0].Binding.Type.Path != "i32" {
		t.Fatalf("body = %+v", function.Body)
	}
}
