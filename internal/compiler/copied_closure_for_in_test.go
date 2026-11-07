package compiler_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

func copiedClosureForInFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "cli", "testdata", "copied_closure_for_in", name+".fy"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestCopiedClosureViewForIn(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", copiedClosureForInFixture(t, "view"))
	before, _ := json.Marshal(file.Module)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("view closure: %v %+v", err, result.Semantic)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("source AST mutated")
	}
	out, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || out.Value != int32(42) {
		t.Fatalf("view execution: %+v %v", out, err)
	}
	runExpandedLanguageNative(t, result, "main::main", "")
}

func TestCopiedClosureVectorForIn(t *testing.T) {
	input, coordinates := vectorSDKInputWithCandidate(t, copiedClosureForInFixture(t, "vector"), true)
	before, _ := json.Marshal(input)
	analysis, err := compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{3})
	if err != nil || len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
		t.Fatalf("vector closure: %v %+v", err, analysis.Analysis.Semantic)
	}
	result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !result.Verified {
		t.Fatalf("vector MIR: %v %+v", err, result.Semantic)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("SDK/source AST mutated")
	}
	runVectorStorageMIRNativeExit(t, result.MIR, hir.QualifiedModulePath(1, "main")+"::main", "", true, 42)
}

func TestCopiedClosureForInRejectsEscapesAndItemCalls(t *testing.T) {
	const prefix = `module main;fn i32 main(){let i32[1] data=[1];let ref i32[] view=ref data[..];let add=fn [] i32(i32 n){return n;};`
	for name, body := range map[string]string{
		"item outside":         `for(let i32 item in view){discard add(item);}return item;`,
		"item shadows callee":  `for(let i32 add in view){discard add(1);}return 0;`,
		"loop closure binding": `for(let i32 item in view){let next=fn [] i32(i32 n){return n;};discard next(item);}return 0;`,
		"copy":                 `for(let i32 item in view){let copy=add;discard copy(item);}return 0;`,
		"return":               `for(let i32 item in view){return add;}return 0;`,
		"store":                `for(let i32 item in view){var i32 n=0;n=add;}return 0;`,
	} {
		t.Run(name, func(t *testing.T) {
			result, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", prefix+body+"}")})
			if err == nil && result.Verified {
				t.Fatal("closure/item scope escaped")
			}
		})
	}
	for name, body := range map[string]string{
		"owner mutation":      `for(let ref i32 item in ref items){assert(v::push<i32>(ref mut items,add(*item)).is_ok);}`,
		"item outside":        `for(let ref i32 item in ref items){discard add(*item);}discard *item;`,
		"item shadows callee": `for(let ref i32 add in ref items){discard add(1);}`,
	} {
		t.Run("vector/"+name, func(t *testing.T) {
			text := copiedClosureForInFixture(t, "vector")
			start := strings.Index(text, " for(let ref i32 item")
			text = text[:start] + body + "return 0;}"
			input, coordinates := vectorSDKInputWithCandidate(t, text, true)
			result, err := compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{3})
			if err == nil && len(result.Analysis.Semantic.Diagnostics) == 0 && len(result.Analysis.Semantic.Unsupported) == 0 {
				t.Fatal("vector closure relaxed owner/item rules")
			}
		})
	}
}
