package ast_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func lowerClosureValueType(t *testing.T, text string) *ast.Module {
	t.Helper()
	tree := syntax.Parse(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if !tree.ErrorFree() {
		t.Fatal(tree.Diagnostics)
	}
	module, err := ast.Lower(tree)
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func TestClosureValueTypeASTStructuralLoweringAndSpans(t *testing.T) {
	signature := "fn /* 한글 */ Result<u32,u16>(i8, ref u8[], fn bool(u64,f32),)"
	text := "module main; // UTF-8 한글\r\nstruct Job{" + signature + " action;}\r\n"
	module := lowerClosureValueType(t, text)
	typ := module.Structs[0].Fields[0].Type
	if typ.Kind != ast.TypeKindClosureFunction || typ.ABI != "" || typ.Path != "" || typ.Result == nil || typ.Result.Kind != ast.TypeKindNamed || typ.Result.Path != "Result" || len(typ.Result.Arguments) != 2 || len(typ.Parameters) != 3 {
		t.Fatalf("signature structure lost: %+v", typ)
	}
	if typ.Result.Arguments[0].Path != "u32" || typ.Result.Arguments[1].Path != "u16" || typ.Parameters[0].Path != "i8" || typ.Parameters[1].Kind != ast.TypeKindReference || typ.Parameters[1].Element == nil || typ.Parameters[1].Element.Kind != ast.TypeKindSlice {
		t.Fatal("signature parameter/result order lost")
	}
	inner := typ.Parameters[2]
	if inner.Kind != ast.TypeKindClosureFunction || inner.Result == nil || inner.Result.Path != "bool" || len(inner.Parameters) != 2 || inner.Parameters[0].Path != "u64" || inner.Parameters[1].Path != "f32" {
		t.Fatal("nested safe signature lost")
	}
	for _, tc := range []struct {
		typ  ast.TypeReference
		want string
	}{{typ, signature}, {*typ.Result, "Result<u32,u16>"}, {typ.Parameters[0], "i8"}, {typ.Parameters[1], "ref u8[]"}, {inner, "fn bool(u64,f32)"}, {*inner.Result, "bool"}, {inner.Parameters[0], "u64"}, {inner.Parameters[1], "f32"}} {
		if tc.typ.Text != tc.want || text[tc.typ.Span.StartByte:tc.typ.Span.EndByte] != tc.want {
			t.Fatalf("type source origin lost: %+v want %q", tc.typ, tc.want)
		}
	}
	if typ.Span.StartByte != strings.Index(text, signature) {
		t.Fatal("byte origin changed after UTF-8/CRLF")
	}
	raw, err := json.Marshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	var restored ast.TypeReference
	if err := json.Unmarshal(raw, &restored); err != nil || !reflect.DeepEqual(restored, typ) {
		t.Fatalf("AST signature round trip: %v", err)
	}
}

func TestClosureValueTypeASTNeverReparsesRawText(t *testing.T) {
	text := `module main;struct Job{fn u32(i8,u64) action;}`
	tree := syntax.Parse(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if !tree.ErrorFree() {
		t.Fatal(tree.Diagnostics)
	}
	for i := range tree.Nodes {
		if tree.Nodes[i].Kind == syntax.KindType {
			tree.Nodes[i].Text = "unsafe extern C fn void()"
		}
	}
	module, err := ast.Lower(tree)
	if err != nil {
		t.Fatal(err)
	}
	typ := module.Structs[0].Fields[0].Type
	if typ.Kind != ast.TypeKindClosureFunction || typ.ABI != "" || typ.Result == nil || typ.Result.Path != "u32" || len(typ.Parameters) != 2 || typ.Parameters[0].Path != "i8" || typ.Parameters[1].Path != "u64" {
		t.Fatalf("type shape inferred from raw Text: %+v", typ)
	}
}

func TestClosureValueTypeASTTypedBindingAndCallbackDistinction(t *testing.T) {
	text := `module main;struct Job{unsafe extern C fn i32(i32,ptr mut void) callback;fn i32(i32) action;}fn fn i32(i32) make(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset+x;};return action;}`
	module := lowerClosureValueType(t, text)
	fields := module.Structs[0].Fields
	callback, safe := fields[0].Type, fields[1].Type
	if callback.Kind != ast.TypeKindForeignFunction || callback.ABI != "C" || len(callback.Parameters) != 2 || callback.Result == nil || callback.Result.Path != "i32" || callback.Parameters[1].Kind != ast.TypeKindRawPointer || !callback.Parameters[1].Mutable || safe.Kind != ast.TypeKindClosureFunction || safe.ABI != "" {
		t.Fatal("safe function and existing foreign callback aliases")
	}
	fn := module.Functions[0]
	binding := fn.Body.Statements[0].Binding
	if fn.ResultType.Kind != ast.TypeKindClosureFunction || binding.Type == nil || binding.Type.Kind != ast.TypeKindClosureFunction || binding.Mutable || binding.Expression.Closure == nil || len(binding.Expression.Closure.Captures) != 1 {
		t.Fatal("typed closure binding/literal edges lost")
	}
}

func TestClosureValueTypeASTPreservesLegacyTypeBytes(t *testing.T) {
	callback := "unsafe extern C fn i32(i32,ptr mut void)"
	text := "module main;struct Job{" + callback + " callback;i32 count;}"
	module := lowerClosureValueType(t, text)
	start := strings.Index(text, callback)
	result := ast.NamedTypeReference("i32", diagnostic.Span{StartByte: start + len("unsafe extern C fn "), EndByte: start + len("unsafe extern C fn i32")})
	parameterStart := strings.Index(text, "(i32,") + 1
	first := ast.NamedTypeReference("i32", diagnostic.Span{StartByte: parameterStart, EndByte: parameterStart + 3})
	pointerStart := strings.Index(text, "ptr mut void")
	element := ast.NamedTypeReference("void", diagnostic.Span{StartByte: pointerStart + len("ptr mut "), EndByte: pointerStart + len("ptr mut void")})
	pointer := ast.TypeReference{Kind: ast.TypeKindRawPointer, Text: "ptr mut void", Span: diagnostic.Span{StartByte: pointerStart, EndByte: pointerStart + len("ptr mut void")}, Mutable: true, Element: &element}
	countStart := strings.Index(text, "i32 count")
	for i, want := range []ast.TypeReference{
		{Kind: ast.TypeKindForeignFunction, Text: callback, Span: diagnostic.Span{StartByte: start, EndByte: start + len(callback)}, Parameters: []ast.TypeReference{first, pointer}, Result: &result, ABI: "C"},
		ast.NamedTypeReference("i32", diagnostic.Span{StartByte: countStart, EndByte: countStart + 3}),
	} {
		gotJSON, err := json.Marshal(module.Structs[0].Fields[i].Type)
		if err != nil {
			t.Fatal(err)
		}
		wantJSON, err := json.Marshal(want)
		if err != nil || string(gotJSON) != string(wantJSON) {
			t.Fatalf("existing type JSON bytes changed: %s want %s (%v)", gotJSON, wantJSON, err)
		}
	}
}

func TestClosureValueTypeKeepsUnsupportedShapesClosed(t *testing.T) {
	for _, text := range []string{
		`module main;fn void accept(fn bool(i32) action){discard action;}`,
		`module main;fn void accept(Option<fn i32(i32)> action){discard action;}`,
		`module main;fn void accept(Result<fn i32(i32),u32> action){discard action;}`,
		`module main;fn void accept(ref fn i32(i32) action){discard action;}`,
		`module main;fn void accept(ptr fn i32(i32) action){discard action;}`,
		`module main;fn void accept((fn i32(i32))[2] action){discard action;}`,
		`module main;unsafe extern C fn void foreign(fn i32(i32) action);`,
		`module main;fn void accept<T>(fn T(T) action){discard action;}`,
	} {
		module := lowerClosureValueType(t, text)
		before, _ := json.Marshal(module)
		result := semantic.Analyze([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: module}})
		if len(result.Unsupported) == 0 && len(result.Diagnostics) == 0 {
			t.Fatalf("unsupported closure shape admitted: unsupported=%+v diagnostics=%+v", result.Unsupported, result.Diagnostics)
		}
		after, _ := json.Marshal(module)
		if string(before) != string(after) {
			t.Fatal("unsupported execution analysis mutated written AST")
		}
	}
}

func TestClosureValueTypeStageKeepsGenericArgumentsClosed(t *testing.T) {
	module := lowerClosureValueType(t, `module main;fn T id<T>(T value){return value;}fn void main(){discard id<fn i32(i32)>(0);}`)
	before, _ := json.Marshal(module)
	if _, _, err := genericsource.Expand(module, "owner", genericplan.DefaultPlanLimits()); err == nil {
		t.Fatal("structural closure type gained generic argument admission")
	}
	after, _ := json.Marshal(module)
	if string(before) != string(after) {
		t.Fatal("rejected generic closure argument mutated written AST")
	}
}
