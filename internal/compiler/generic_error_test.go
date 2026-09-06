package compiler_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestGenericErrorsRetainOriginalSourceAndKernelCategory(t *testing.T) {
	for _, test := range []struct {
		source, primary, kind string
		kernel                genericplan.Code
	}{
		{"fn T id<T>(T v) { return v; } fn i32 main() { return id<i32,bool>(42); }", "id<i32,bool>", genericsource.FailureRequest, genericplan.CodeTypeArgumentArity},
		{"fn i32 main() { return missing<i32>(42); }", "missing<i32>", genericsource.FailureUnknown, ""},
		{"fn T id<T>(T v) { return v; } fn i32 main() { return id<Thing>(42); }", "id<Thing>", genericsource.FailureRequest, ""},
		{"fn T id<T>(T v) { return id<T>(v); } fn i32 main() { return id<i32>(42); }", "id<i32>", genericsource.FailureExpansion, genericplan.CodeSpecializationCycle},
		{"fn T id<T,T>(T v) { return v; } fn i32 main() { return 0; }", "fn T id<T,T>(T v) { return v; }", genericsource.FailureDeclaration, genericplan.CodeInvalidDeclaration},
	} {
		t.Run(test.primary, func(t *testing.T) {
			text := "module main; " + test.source
			file := parseCompilerFile(t, hir.PackageID(7), "main.fy", text)
			_, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
			var source *genericsource.SourceError
			if !errors.As(err, &source) {
				t.Fatalf("no source provenance: %v", err)
			}
			if source.Kind != test.kind || source.Owner != "package_7" || source.Module != "main" || !source.Span.Valid(len(text)) || text[source.Span.StartByte:source.Span.EndByte] != test.primary || genericplan.ErrorCode(err) != test.kernel {
				t.Fatalf("wrong provenance/category: %+v %v", source, err)
			}
		})
	}
}

func TestGenericPackagePlanningFailureKeepsRootOriginAcrossReordering(t *testing.T) {
	input, _ := genericPackageInput(t)
	limits := genericplan.DefaultPlanLimits()
	limits.MaxExpansionDepth = 1
	err := genericPackageSourceError(input, limits)
	var original *genericsource.SourceError
	var request *genericplan.RequestError
	if !errors.As(err, &original) || !errors.As(err, &request) || genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
		t.Fatalf("unlocated planning error: %v", err)
	}
	if original.Module != "main" || original.Kind != genericsource.FailureExpansion || !strings.HasSuffix(string(request.Request.Declaration), "::util::wrap") {
		t.Fatalf("wrong root: %+v %+v", original, request)
	}
	text := "module main; import util as u; fn i32 main() { return u::wrap<i32>(42); }"
	if text[original.Span.StartByte:original.Span.EndByte] != "u::wrap<i32>" {
		t.Fatal("planning error did not identify root source request")
	}
	slices.Reverse(input.Files)
	slices.Reverse(input.Imports)
	err = genericPackageSourceError(input, limits)
	var reordered *genericsource.SourceError
	if !errors.As(err, &reordered) || original.Owner != reordered.Owner || original.Module != reordered.Module || original.Span != reordered.Span {
		t.Fatalf("reordered origin differs: %v", err)
	}
	// Invalid caller budgets are not source mistakes and must remain unlocated.
	err = genericPackageSourceError(input, genericplan.PlanLimits{})
	var invalid *genericsource.SourceError
	if err == nil || errors.As(err, &invalid) {
		t.Fatalf("invalid configuration mislabeled as source: %v", err)
	}
}

func TestGenericPrivateAccessOriginBelongsToConsumer(t *testing.T) {
	input, _ := genericPackageInput(t)
	input.Files[1].Module.Functions[0].Public = false
	err := genericPackageSourceError(input, genericplan.DefaultPlanLimits())
	var source *genericsource.SourceError
	if !errors.As(err, &source) || source.Module != "main" || source.Kind != genericsource.FailurePrivate {
		t.Fatalf("private use origin: %v", err)
	}
}

// The raw source adapter retains planner causes; the package compiler maps
// those causes to diagnostics and is covered separately.
func genericPackageSourceError(input semantic.PackageInput, limits genericplan.PlanLimits) error {
	modules := make([]genericsource.PackageModule, len(input.Files))
	owner := func(id hir.PackageID) genericplan.PackageID {
		return genericplan.PackageID(fmt.Sprintf("test_package_%d", id))
	}
	for i, file := range input.Files {
		imports := map[string]genericplan.PackageID{}
		for _, imported := range input.Imports {
			if imported.FromPackage == file.Package {
				imports[imported.ImportPath] = owner(imported.ToPackage)
			}
		}
		modules[i] = genericsource.PackageModule{Module: file.Module, Owner: owner(file.Package), Imports: imports}
	}
	_, err := genericsource.PreparePackages(modules, limits)
	return err
}
