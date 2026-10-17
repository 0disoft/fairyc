package semantic_test

import (
	"encoding/json"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func nominalBindingInput(t *testing.T) semantic.PackageInput {
	t.Helper()
	file := parsePackageFile(t, 1, "generic.fy", "module generic; fn i32 pass(i32 value) { return value; }")
	function := &file.Module.Functions[0]
	function.Parameters[0].Type.Kind = ast.TypeKindBoundNominal
	function.Parameters[0].Type.Path = "argument"
	function.ResultType.Kind = ast.TypeKindBoundNominal
	function.ResultType.Path = "argument"
	return semantic.PackageInput{Files: []semantic.File{file,
		parsePackageFile(t, 2, "owned.fy", "module owned; resource Token { i32 value; cleanup {} }"),
		parsePackageFile(t, 3, "owned.fy", "module owned; struct Token { i32 value; }"),
	}}
}

func TestPrivateNominalBindingPreservesExactOwner(t *testing.T) {
	for _, owner := range []hir.PackageID{2, 3} {
		input := nominalBindingInput(t)
		before, _ := json.Marshal(input)
		binding := semantic.NominalTypeBinding{Key: "argument", Package: owner, Module: "owned", Name: "Token"}
		result, err := semantic.AnalyzeBoundNominalPackages(input, []semantic.NominalTypeBinding{binding})
		if err != nil || result.CompletedThrough != semantic.PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
			t.Fatalf("bound type: %v %+v", err, result)
		}
		function := result.HIR.Modules[0].Functions[0]
		want := hir.ParseType(hir.QualifiedModulePath(owner, "owned") + "::Token")
		if function.ResultType != want || function.Parameters[0].Type != want {
			t.Fatalf("owner identity lost: %q %q, want %q", function.ResultType, function.Parameters[0].Type, want)
		}
		after, _ := json.Marshal(input)
		if string(before) != string(after) {
			t.Fatal("binding analysis mutated input")
		}
		ordinary, err := semantic.AnalyzePackages(input)
		if err != nil || ordinary.CompletedThrough == semantic.PhaseFlow {
			t.Fatal("ordinary analysis accepted private type binding")
		}
	}
}

func TestPrivateNominalBindingRejectsMissingOrMalformedAuthority(t *testing.T) {
	valid := semantic.NominalTypeBinding{Key: "argument", Package: 2, Module: "owned", Name: "Token"}
	for _, bindings := range [][]semantic.NominalTypeBinding{
		{valid, valid}, {{Key: "argument", Package: 4, Module: "owned", Name: "Token"}},
		{{Key: "argument", Package: 2, Module: "owned", Name: "Missing"}},
		{{Package: 2, Module: "owned", Name: "Token"}},
	} {
		if _, err := semantic.AnalyzeBoundNominalPackages(nominalBindingInput(t), bindings); err == nil {
			t.Fatal("invalid binding authority accepted")
		}
	}
	input := nominalBindingInput(t)
	missing, err := semantic.AnalyzeBoundNominalPackages(input, nil)
	if err != nil || missing.CompletedThrough == semantic.PhaseFlow {
		t.Fatal("unbound private type admitted")
	}
	input.Files[0].Module.Functions[0].Parameters[0].Type.Mutable = true
	malformed, err := semantic.AnalyzeBoundNominalPackages(input, []semantic.NominalTypeBinding{valid})
	if err != nil || malformed.CompletedThrough == semantic.PhaseFlow {
		t.Fatal("malformed private reference admitted")
	}
}

func TestPrivateNominalBindingDoesNotGrantFieldAccess(t *testing.T) {
	input := nominalBindingInput(t)
	file := parsePackageFile(t, 1, "generic.fy", "module generic; fn i32 read(ref i32 value) { return value.value; }")
	file.Module.Functions[0].Parameters[0].Type.Element.Kind = ast.TypeKindBoundNominal
	file.Module.Functions[0].Parameters[0].Type.Element.Path = "argument"
	input.Files[0] = file
	result, err := semantic.AnalyzeBoundNominalPackages(input, []semantic.NominalTypeBinding{{Key: "argument", Package: 2, Module: "owned", Name: "Token"}})
	if err != nil || !hasDiagnosticCode(result, "E0103") {
		t.Fatalf("private field accessible through binding: %v %+v", err, result.Diagnostics)
	}
}

func TestPrivateVectorCandidateRequiresExactConcreteSDKResource(t *testing.T) {
	input := semantic.PackageInput{Files: []semantic.File{
		parsePackageFile(t, 1, "std/vec.fy", "module std::vec; resource ConcreteVec { usize length; cleanup {} }"),
		parsePackageFile(t, 2, "std/vec.fy", "module std::vec; struct FakeVec { usize length; }"),
	}}
	valid := semantic.VectorTypeAdmission{Package: 1, Module: "std::vec", Name: "ConcreteVec"}
	accepted, err := semantic.AnalyzeBoundNominalVectorCandidatePackages(input, nil, []semantic.VectorTypeAdmission{valid})
	if err != nil || accepted.CompletedThrough != semantic.PhaseFlow {
		t.Fatalf("exact candidate resource rejected: %v %+v", err, accepted)
	}
	for _, vector := range []semantic.VectorTypeAdmission{
		{Package: 2, Module: "std::vec", Name: "FakeVec"},
		{Package: 1, Module: "other::vec", Name: "ConcreteVec"},
		{Package: 1, Module: "std::vec", Name: "Missing"},
	} {
		if _, err := semantic.AnalyzeBoundNominalVectorCandidatePackages(input, nil, []semantic.VectorTypeAdmission{vector}); err == nil {
			t.Fatalf("invalid vector candidate authority accepted: %+v", vector)
		}
	}
	if _, err := semantic.AnalyzeBoundNominalVectorCandidatePackages(input, nil,
		[]semantic.VectorTypeAdmission{valid, valid}); err == nil {
		t.Fatal("duplicate vector candidate authority accepted")
	}
}
