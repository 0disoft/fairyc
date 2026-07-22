package symbolidentity_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

func TestFromSnapshotIgnoresBodySpanVersionInstanceAndBuildNumbers(t *testing.T) {
	t.Parallel()
	base := onePackageSymbols(t, 1, "example.identity", "1.0.0", "main.fy", `module main;
pub fn i32 answer() { return 1; }
`)
	bodyAndSpanChanged := onePackageSymbols(t, 77, "example.identity", "1.0.0", "main.fy", `module main;

pub fn i32 answer() {
    return 2;
}
`)
	versionChanged := onePackageSymbols(t, 9, "example.identity", "2.0.0", "main.fy", `module main;
pub fn i32 answer() { return 1; }
`)
	if len(base) != 1 || len(bodyAndSpanChanged) != 1 || len(versionChanged) != 1 {
		t.Fatalf("symbol counts = %d, %d, %d", len(base), len(bodyAndSpanChanged), len(versionChanged))
	}
	if base[0].ID != bodyAndSpanChanged[0].ID || base[0].ID != versionChanged[0].ID {
		t.Fatalf("stable declaration identity drifted: base=%+v body=%+v version=%+v", base[0].ID, bodyAndSpanChanged[0].ID, versionChanged[0].ID)
	}
}

func TestFromSnapshotChangesIdentityForSemanticCoordinatesAndSignature(t *testing.T) {
	t.Parallel()
	base := onlyID(t, onePackageSymbols(t, 1, "example.identity", "1.0.0", "main.fy", `module main;
pub fn i32 answer() { return 1; }
`))
	tests := []struct {
		name   string
		uid    string
		path   string
		source string
	}{
		{name: "package uid", uid: "example.other", path: "main.fy", source: "module main;\npub fn i32 answer() { return 1; }\n"},
		{name: "module", uid: "example.identity", path: "other.fy", source: "module other;\npub fn i32 answer() { return 1; }\n"},
		{name: "declaration path", uid: "example.identity", path: "main.fy", source: "module main;\npub fn i32 changed() { return 1; }\n"},
		{name: "return type", uid: "example.identity", path: "main.fy", source: "module main;\npub fn u32 answer() { return 1; }\n"},
		{name: "unsafe", uid: "example.identity", path: "main.fy", source: "module main;\npub unsafe fn i32 answer() { return 1; }\n"},
		{name: "parameter", uid: "example.identity", path: "main.fy", source: "module main;\npub fn i32 answer(i32 value) { return value; }\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := onlyID(t, onePackageSymbols(t, 42, test.uid, "1.0.0", test.path, test.source))
			if candidate == base {
				t.Fatalf("semantic change retained SymbolId %+v", candidate)
			}
		})
	}
}

func TestFromSnapshotExcludesVisibilityAndParameterBindingNames(t *testing.T) {
	t.Parallel()
	base := onlyID(t, onePackageSymbols(t, 1, "example.binding", "1.0.0", "main.fy", `module main;
pub fn i32 echo(i32 left) { return left; }
`))
	visibilityChanged := onlyID(t, onePackageSymbols(t, 2, "example.binding", "1.0.0", "main.fy", `module main;
fn i32 echo(i32 left) { return left; }
`))
	parameterRenamed := onlyID(t, onePackageSymbols(t, 3, "example.binding", "1.0.0", "main.fy", `module main;
pub fn i32 echo(i32 right) { return right; }
`))
	if base != visibilityChanged || base != parameterRenamed {
		t.Fatalf("non-signature facts changed SymbolId: base=%+v visibility=%+v parameter=%+v", base, visibilityChanged, parameterRenamed)
	}
}

func TestFromSnapshotIncludesAggregateAndForeignContractShape(t *testing.T) {
	t.Parallel()
	structLeft := onlyID(t, onePackageSymbols(t, 1, "example.shape", "1.0.0", "shape.fy", `module shape;
pub struct Pair { i32 left; u32 right; }
`))
	structRight := onlyID(t, onePackageSymbols(t, 2, "example.shape", "1.0.0", "shape.fy", `module shape;
pub struct Pair { u32 right; i32 left; }
`))
	if structLeft == structRight {
		t.Fatalf("aggregate field order did not change SymbolId: %+v", structLeft)
	}

	foreignLeft := onlyID(t, onePackageSymbols(t, 3, "example.foreign", "1.0.0", "ffi.fy", `module ffi;
@link_name("left_answer")
unsafe extern C fn i32 answer();
`))
	foreignRight := onlyID(t, onePackageSymbols(t, 4, "example.foreign", "1.0.0", "ffi.fy", `module ffi;
@link_name("right_answer")
unsafe extern C fn i32 answer();
`))
	if foreignLeft == foreignRight {
		t.Fatalf("foreign link contract did not change SymbolId: %+v", foreignLeft)
	}
}

func TestFromSnapshotUsesPackageUIDForNominalSignatureTypes(t *testing.T) {
	t.Parallel()
	left := nominalSymbols(t, 3, "1.0.0")
	right := nominalSymbols(t, 99, "2.0.0")
	if !reflect.DeepEqual(idsByName(left), idsByName(right)) {
		t.Fatalf("package-coordinate version leaked into SymbolId:\nleft=%+v\nright=%+v", idsByName(left), idsByName(right))
	}
}

func TestFromSnapshotReturnsCanonicalOrderAndBothCollisionForms(t *testing.T) {
	t.Parallel()
	symbols := onePackageSymbols(t, 1, "example.order", "1.0.0", "main.fy", `module main;
pub fn i32 zed() { return 1; }
pub struct Packet { i32 value; }
pub fn i32 alpha() { return 2; }
`)
	if len(symbols) != 3 {
		t.Fatalf("symbols = %+v", symbols)
	}
	if !slices.IsSortedFunc(symbols, func(left, right symbolidentity.Symbol) int {
		return slices.Compare(left.ID.Full[:], right.ID.Full[:])
	}) {
		t.Fatalf("symbols are not full-digest sorted: %+v", symbols)
	}
	for _, symbol := range symbols {
		if symbol.ID.Visible.String() == "" || symbol.ID.Full.String() == "" {
			t.Fatalf("symbol lacks collision-detecting forms: %+v", symbol)
		}
	}
}

func onePackageSymbols(t *testing.T, packageID hir.PackageID, uid, version, path, text string) []symbolidentity.Symbol {
	t.Helper()
	result := compilePackage(t, []semantic.File{{Package: packageID, Path: path, Module: analyzeSyntax(t, path, text)}}, nil)
	input := snapshotPackage(t, packageID, uid, version, []packagecontent.File{{Path: path, Bytes: []byte(text)}})
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatalf("semanticsnapshot.Build() error = %v", err)
	}
	symbols, err := symbolidentity.FromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("FromSnapshot() error = %v", err)
	}
	return symbols
}

func nominalSymbols(t *testing.T, packageID hir.PackageID, version string) []symbolidentity.Symbol {
	t.Helper()
	text := `module api;
pub struct Packet { i32 value; }
pub fn Packet identity(Packet value) { return value; }
`
	result := compilePackage(t, []semantic.File{{Package: packageID, Path: "api.fy", Module: analyzeSyntax(t, "api.fy", text)}}, nil)
	input := snapshotPackage(t, packageID, "example.nominal", version, []packagecontent.File{{Path: "api.fy", Bytes: []byte(text)}})
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatalf("semanticsnapshot.Build() error = %v", err)
	}
	symbols, err := symbolidentity.FromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("FromSnapshot() error = %v", err)
	}
	return symbols
}

func compilePackage(t *testing.T, files []semantic.File, imports []semantic.PackageImport) compiler.Result {
	t.Helper()
	result, err := compiler.CheckPackages(semantic.PackageInput{Files: files, Imports: imports})
	if err != nil || !result.Verified || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 0 {
		t.Fatalf("CheckPackages() = verified %t error %v diagnostics %+v unsupported %+v", result.Verified, err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	return result
}

func analyzeSyntax(t *testing.T, path, text string) *ast.Module {
	t.Helper()
	result, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
	if err != nil || result.AST == nil || len(result.Tree.Diagnostics) != 0 {
		t.Fatalf("AnalyzeSyntax(%s) = error %v diagnostics %+v", path, err, result.Tree.Diagnostics)
	}
	return result.AST
}

func snapshotPackage(t *testing.T, packageID hir.PackageID, uid, version string, files []packagecontent.File) semanticsnapshot.PackageInput {
	t.Helper()
	digest, err := packagecontent.Compute(files)
	if err != nil {
		t.Fatal(err)
	}
	identity := packagegraph.Identity{PackageUID: uid, SourceIdentity: digest.String(), VersionIdentity: version}
	instance, err := packagegraph.ComputeInstanceID(identity)
	if err != nil {
		t.Fatal(err)
	}
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.CoordinateOf(identity))
	if err != nil {
		t.Fatal(err)
	}
	fileInputs := make([]semanticsnapshot.FileInput, 0, len(files))
	for _, file := range files {
		fileDigest, lineMap := packagecontent.ComputeFileIdentities(file.Bytes)
		fileInputs = append(fileInputs, semanticsnapshot.FileInput{
			Path: file.Path, Digest: fileDigest, LineMap: lineMap, Kind: semanticsnapshot.FileKindSource,
		})
	}
	return semanticsnapshot.PackageInput{
		BuildID: packageID, UID: uid, Name: "fixture", Version: version, Language: "fairyc-language/0",
		Instance: instance, Coordinate: coordinate, Source: digest, SourceKind: "local", Role: "root",
		Files: fileInputs,
	}
}

func onlyID(t *testing.T, symbols []symbolidentity.Symbol) symbolidentity.ID {
	t.Helper()
	if len(symbols) != 1 {
		t.Fatalf("symbols = %+v", symbols)
	}
	return symbols[0].ID
}

func idsByName(symbols []symbolidentity.Symbol) map[string]symbolidentity.ID {
	result := make(map[string]symbolidentity.ID, len(symbols))
	for _, symbol := range symbols {
		result[symbol.Declaration.Name] = symbol.ID
	}
	return result
}
