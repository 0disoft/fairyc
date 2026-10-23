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

func TestGenericSignaturesKeepTransportScalarsUnadmitted(t *testing.T) {
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: "generic.admission", VersionIdentity: "1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, scalar := range []string{"i8", "i16", "isize", "u16", "u64", "f32", "f64"} {
		t.Run(scalar, func(t *testing.T) {
			text := fmt.Sprintf("module main;fn T id<T>(T value){return value;}fn void main(){discard id<%s>(42);}", scalar)
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			check := func(t *testing.T, result compiler.Result, err error) {
				t.Helper()
				if err != nil || result.Verified || len(result.MIR.Functions) != 0 || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 2 {
					t.Fatalf("signature admission: err=%v verified=%v semantic=%+v", err, result.Verified, result.Semantic)
				}
				for _, rejected := range result.Semantic.Unsupported {
					if rejected.Feature != "type "+scalar || text[rejected.Span.StartByte:rejected.Span.EndByte] != "T" {
						t.Fatalf("lost original signature span: %+v", rejected)
					}
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
	for _, shape := range []string{"ref T", "ptr T", "Option<T>", "T[2]"} {
		t.Run(shape, func(t *testing.T) {
			text := fmt.Sprintf("module main;fn %s id<T>(%s value){return value;}fn void main(){discard id<u16>(42);discard id<u16>(43);}", shape, shape)
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

func TestGenericSignatureAdmissionChecksForwardedPackageTypes(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	input.Files[0] = parseCompilerFile(t, 1, "main.fy", "module main;import util as u;fn void main(){discard u::wrap<u16>(42);discard u::wrap<u16>(43);}")
	result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || result.Verified || len(result.Semantic.Unsupported) != 4 || len(result.Semantic.Diagnostics) != 0 {
		t.Fatalf("forwarded signature admission: %v %+v", err, result.Semantic)
	}
	for _, row := range result.Semantic.Unsupported {
		owner, name := row.Span.Package, row.Span.Source
		if row.Feature != "type u16" || (owner != 2 || name != "util.fy") && (owner != 3 || name != "core.fy") {
			t.Fatalf("lost package-owned signature: %+v", row)
		}
	}
}
