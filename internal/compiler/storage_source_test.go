package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const storageSource = `module main;
struct Padded { u8 tag; u32 value; }
resource Owner { Padded payload; ptr mut void handle; cleanup {} }
fn usize size<T>() { return storage_size<T>(); }
fn usize alignment<T>() { return storage_alignment<T>(); }
fn i32 main() {
    return if (size<bool>() == 4 && alignment<bool>() == 4 && size<Padded>() == 8 && alignment<Padded>() == 4 && size<Owner>() == 16 && alignment<Owner>() == 8) { 42 } else { 0 };
}`

func TestPrivateStorageQuerySource(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", storageSource)
	before, _ := json.Marshal(file.Module)
	result, plan, err := compiler.CheckStorageGenericFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified || len(plan.Specializations) != 6 {
		t.Fatalf("source queries: %v %+v %+v", err, result.Semantic, plan)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("source AST mutated")
	}
	count := 0
	for _, function := range result.MIR.Functions {
		for _, block := range function.Blocks {
			for _, op := range block.Operations {
				if op.Kind == mir.OperationStorageLayout {
					count++
					if len(op.Operands) != 0 || op.OwnerType == hir.ParseType("") || op.Span.Source == "" {
						t.Fatal("query lost source/type or acquired value operand")
					}
				}
			}
		}
	}
	if count != 6 || len(mir.Verify(result.MIR)) == 0 {
		t.Fatal("source query bypassed symbolic/private boundary")
	}
	if _, err := interpreter.Execute(result.MIR, "main::main"); err == nil {
		t.Fatal("targetless interpreter accepted query")
	}
	if _, err := mircodec.Encode(result.MIR, 1<<20); err == nil {
		t.Fatal("ordinary artifact accepted query")
	}
	ordinary, err := compiler.Check([]semantic.File{file})
	if err == nil && ordinary.Verified {
		t.Fatal("ordinary source entry admitted private query")
	}
	nominal, _, err := compiler.CheckNominalGenericFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	if err == nil && nominal.Verified {
		t.Fatal("ordinary nominal entry admitted private query")
	}
	checkStorageQueryNative(t, result.MIR)
}

func TestPrivateStorageQuerySourceRejections(t *testing.T) {
	uninvoked := strings.Replace(storageSource, "storage_size<T>()", "storage_size<T>", 1)
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(uninvoked)), source.DefaultLimits())
	if err != nil || len(parsed.Tree.Diagnostics) == 0 || parsed.AST != nil {
		t.Fatal("uninvoked specialization must be rejected by the parser")
	}
	for name, source := range map[string]string{
		"value argument":    strings.Replace(storageSource, "storage_size<T>()", "storage_size<T>(1)", 1),
		"type arity":        strings.Replace(storageSource, "storage_size<T>()", "storage_size<T, T>()", 1),
		"unknown type":      strings.Replace(storageSource, "storage_size<T>()", "storage_size<Missing>()", 1),
		"unsupported type":  strings.Replace(storageSource, "size<bool>()", "size<ref mut i32[]>()", 1),
		"reserved function": strings.Replace(storageSource, "fn usize size<T>()", "fn usize storage_size<T>()", 1),
		"reserved binding":  strings.Replace(storageSource, "return storage_size<T>();", "let usize storage_size = 1; return storage_size<T>();", 1),
	} {
		t.Run(name, func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)
			result, _, err := compiler.CheckStorageGenericFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("invalid query admitted")
			}
		})
	}
}
