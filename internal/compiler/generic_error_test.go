package compiler_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
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
	input, coordinates := genericPackageInput(t)
	limits := genericplan.DefaultPlanLimits()
	limits.MaxExpansionDepth = 1
	_, _, err := compiler.CheckGenericPackages(input, coordinates, limits)
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
	_, _, err = compiler.CheckGenericPackages(input, coordinates, limits)
	var reordered *genericsource.SourceError
	if !errors.As(err, &reordered) || original.Owner != reordered.Owner || original.Module != reordered.Module || original.Span != reordered.Span {
		t.Fatalf("reordered origin differs: %v", err)
	}
	// Invalid caller budgets are not source mistakes and must remain unlocated.
	_, _, err = compiler.CheckGenericPackages(input, coordinates, genericplan.PlanLimits{})
	var invalid *genericsource.SourceError
	if err == nil || errors.As(err, &invalid) {
		t.Fatalf("invalid configuration mislabeled as source: %v", err)
	}
}

func TestGenericPrivateAccessOriginBelongsToConsumer(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	input.Files[1].Module.Functions[0].Public = false
	_, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	var source *genericsource.SourceError
	if !errors.As(err, &source) || source.Module != "main" || source.Kind != genericsource.FailurePrivate {
		t.Fatalf("private use origin: %v", err)
	}
}
