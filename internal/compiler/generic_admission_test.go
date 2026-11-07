package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestGenericSignaturesAdmitDirectTransportScalars(t *testing.T) {
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: "generic.admission", VersionIdentity: "1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, scalar := range []string{"i8", "i16", "isize", "u16", "u64", "f32", "f64"} {
		t.Run(scalar, func(t *testing.T) {
			text := fmt.Sprintf("module main;unsafe extern C fn %s seed();fn T id<T>(T value){return value;}fn void main(){discard id<%s>(unsafe{seed()});}", scalar, scalar)
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			check := func(t *testing.T, result compiler.Result, err error) {
				t.Helper()
				if err != nil || !result.Verified {
					t.Fatalf("direct scalar signature: %v %+v", err, result.Semantic)
				}
			}
			t.Run("private", func(t *testing.T) {
				result, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
				check(t, result, err)
			})
			t.Run("explicit", func(t *testing.T) {
				result, err := compiler.CheckExplicitFiles([]semantic.File{file})
				check(t, result, err)
			})
			t.Run("package", func(t *testing.T) {
				result, _, err := compiler.CheckGenericPackages(semantic.PackageInput{Files: []semantic.File{file}},
					map[hir.PackageID]packagegraph.CoordinateID{file.Package: coordinate}, genericplan.DefaultPlanLimits())
				check(t, result, err)
			})
		})
	}
}

func TestGenericSignatureAdmissionKeepsOrdinaryTransportAndPhantomArguments(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
fn u16 copy(u16 value){return value;}
fn i32 constant<T>(){return 42;}
fn i32 main(){return constant<u16>();}`)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("ordinary transport or unused argument rejected: %v %+v", err, result.Semantic)
	}
}

func TestGenericSignatureAdmissionPreservesCompositeSpellingAndResolution(t *testing.T) {
	for _, scalar := range []string{"i8", "i16", "isize", "u16", "u64", "f32", "f64"} {
		for _, shape := range []string{"ref T", "ptr T", "Result<T,i32>", "T[2]", "Option<Option<T>>", "ref Option<T>", "Option<ref T>", "Result<Option<T>,i32>", "Option<Result<T,i32>>", "ref T[]"} {
			t.Run(scalar+"/"+shape, func(t *testing.T) {
				// Typed forwarding has valid call arity and argument types: the two
				// rejections must belong to the specialized signature, not a bad call.
				concrete := strings.ReplaceAll(shape, "T", scalar)
				text := fmt.Sprintf("module main;fn %[1]s id<T>(%[1]s value){return value;}fn %[2]s forward(%[2]s value){discard id<%[3]s>(value);return id<%[3]s>(value);}", shape, concrete, scalar)
				file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
				result, err := compiler.CheckExplicitFiles([]semantic.File{file})
				if err != nil || result.Verified || len(result.Semantic.Unsupported) != 2 || len(result.Semantic.Diagnostics) != 0 {
					t.Fatalf("composite signature: %v %+v", err, result.Semantic)
				}
				for _, row := range result.Semantic.Unsupported {
					if row.Feature != "type "+shape || text[row.Span.StartByte:row.Span.EndByte] != shape {
						t.Fatalf("changed composite spelling: %+v", row)
					}
				}
				file = parseCompilerFile(t, hir.DefaultPackage, "main.fy", strings.Replace(text, "return value;", "return missing;", 1))
				result, err = compiler.CheckExplicitFiles([]semantic.File{file})
				if err != nil || result.Semantic.FailedPhase != semantic.PhaseResolve || len(result.Semantic.Diagnostics) == 0 {
					t.Fatalf("signature check hid name-resolution failure: %v %+v", err, result.Semantic)
				}
			})
		}
	}
}

func TestGenericSignatureAdmissionChecksForwardedPackageTypes(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	input.Files[0] = parseCompilerFile(t, 1, "main.fy", "module main;import util as u;unsafe extern C fn u16 seed();fn void main(){discard u::wrap<u16>(unsafe{seed()});discard u::wrap<u16>(unsafe{seed()});}")
	result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("forwarded direct scalar: %v %+v", err, result.Semantic)
	}
}
