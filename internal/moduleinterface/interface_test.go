package moduleinterface

import (
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestParseHashRejectsNoncanonicalText(t *testing.T) {
	t.Parallel()
	valid := HashPrefix + strings.Repeat("a", 64)
	parsed, err := ParseHash(valid)
	if err != nil || parsed.String() != valid {
		t.Fatalf("ParseHash(%q) = %v, %v", valid, parsed, err)
	}
	for _, value := range []string{
		strings.TrimPrefix(valid, HashPrefix),
		valid[:len(valid)-1],
		strings.ToUpper(valid),
		HashPrefix + strings.Repeat("g", 64),
	} {
		if _, err := ParseHash(value); err == nil {
			t.Fatalf("ParseHash(%q) accepted", value)
		}
	}
}

func TestExtractConsumesTypedSourceHIR(t *testing.T) {
	t.Parallel()
	self := testCoordinateID(t, "source-package")
	instances := map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: self}
	base := analyzeSource(t, `module api;
pub struct Packet { i32 value; }
pub fn i32 read(ref Packet packet) { return packet.value; }
fn i32 private_helper() { return 1; }
`)
	privateBodyChange := analyzeSource(t, `module api;
pub struct Packet { i32 value; }
pub fn i32 read(ref Packet packet) { return packet.value; }
fn i32 private_helper() { return 2; }
`)
	publicSignatureChange := analyzeSource(t, `module api;
pub struct Packet { i32 value; }
pub fn u32 read(ref Packet packet) { return 1; }
fn i32 private_helper() { return 1; }
`)

	baseHash := extractHash(t, base, hir.DefaultPackage, instances)
	if changed := extractHash(t, privateBodyChange, hir.DefaultPackage, instances); changed != baseHash {
		t.Fatalf("private source body invalidated interface: %s != %s", changed, baseHash)
	}
	if changed := extractHash(t, publicSignatureChange, hir.DefaultPackage, instances); changed == baseHash {
		t.Fatalf("public source signature did not invalidate interface: %s", changed)
	}
}

func TestExtractIgnoresPrivateItemsBodiesAndDeclarationOrder(t *testing.T) {
	t.Parallel()
	self := testCoordinateID(t, "self")
	instances := map[hir.PackageID]packagegraph.CoordinateID{1: self}
	first := hir.Package{Modules: []hir.Module{{
		Package: 1,
		Path:    "api",
		Functions: []hir.Function{
			{Name: "zeta", Public: true, Parameters: []hir.Parameter{{Type: hir.TypeI32}}, ResultType: hir.TypeI32, Body: hir.Block{Statements: []hir.Statement{{Kind: "first-body"}}}},
			{Name: "private", Public: false, ResultType: hir.TypeBool},
			{Name: "alpha", Public: true, ResultType: hir.TypeBool},
		},
	}}}
	second := hir.Package{Modules: []hir.Module{{
		Package: 1,
		Path:    "api",
		Functions: []hir.Function{
			{Name: "alpha", Public: true, ResultType: hir.TypeBool, Body: hir.Block{Statements: []hir.Statement{{Kind: "changed-body"}}}},
			{Name: "another-private", Public: false, ResultType: hir.TypeU32},
			{Name: "zeta", Public: true, Parameters: []hir.Parameter{{Name: "renamed", Mutable: true, Type: hir.TypeI32}}, ResultType: hir.TypeI32},
		},
	}}}

	firstHash := extractHash(t, first, 1, instances)
	secondHash := extractHash(t, second, 1, instances)
	if firstHash != secondHash {
		t.Fatalf("private/body/order-only change invalidated interface: %s != %s", firstHash, secondHash)
	}

	second.Modules[0].Functions[1].Public = true
	if changed := extractHash(t, second, 1, instances); changed == firstHash {
		t.Fatalf("new public function did not invalidate interface: %s", changed)
	}
}

func TestExtractIsStableAcrossBuildLocalPackageNumbers(t *testing.T) {
	t.Parallel()
	self := testCoordinateID(t, "self")
	dependency := testCoordinateID(t, "dependency")
	left := interfacePackage(1, 2)
	right := interfacePackage(7, 9)
	leftHash := extractHash(t, left, 1, map[hir.PackageID]packagegraph.CoordinateID{1: self, 2: dependency})
	rightHash := extractHash(t, right, 7, map[hir.PackageID]packagegraph.CoordinateID{7: self, 9: dependency})
	if leftHash != rightHash {
		t.Fatalf("build-local numbering changed interface: %s != %s", leftHash, rightHash)
	}

	otherDependency := testCoordinateID(t, "other-dependency")
	changed := extractHash(t, right, 7, map[hir.PackageID]packagegraph.CoordinateID{7: self, 9: otherDependency})
	if changed == leftHash {
		t.Fatalf("dependency identity change did not invalidate interface: %s", changed)
	}
}

func TestExtractRejectsPublicSignatureUsingPrivateLocalType(t *testing.T) {
	t.Parallel()
	self := testCoordinateID(t, "self")
	value := hir.Package{Modules: []hir.Module{{
		Package: 1,
		Path:    "api",
		Structs: []hir.Struct{{Name: "Hidden", Public: false, Type: "$p1::api::Hidden"}},
		Functions: []hir.Function{{
			Name: "leak", Public: true, Parameters: []hir.Parameter{{Type: "$p1::api::Hidden"}}, ResultType: hir.TypeVoid,
		}},
	}}}
	_, err := Extract(value, 1, map[hir.PackageID]packagegraph.CoordinateID{1: self})
	if err == nil || !strings.Contains(err.Error(), "is not a public type") {
		t.Fatalf("Extract() error = %v", err)
	}
}

func TestExtractTracksPublicShapesEffectsAndForeignLinks(t *testing.T) {
	t.Parallel()
	self := testCoordinateID(t, "self")
	instances := map[hir.PackageID]packagegraph.CoordinateID{1: self}
	base := hir.Package{Modules: []hir.Module{{
		Package: 1,
		Path:    "api",
		Structs: []hir.Struct{{
			Name: "Pair", Public: true, Type: "$p1::api::Pair", ReprC: true,
			Fields: []hir.StructField{{Name: "left", Type: hir.TypeI32}, {Name: "right", Type: hir.TypeU32}},
		}},
		Enums: []hir.Enum{{
			Name: "Status", Public: true, Type: "$p1::api::Status",
			Variants: []hir.EnumVariant{{Name: "ready"}, {Name: "failed", Fields: []hir.EnumField{{Name: "code", Type: hir.TypeI32}}}},
		}},
		Resources: []hir.Resource{{
			Name: "Handle", Public: true, Type: "$p1::api::Handle", Fields: []hir.ResourceField{{Name: "private", Type: hir.TypeI32}},
			DropEffects: []hir.Effect{hir.EffectFilesystem},
		}},
		ForeignFunctions: []hir.ForeignFunction{{
			Name: "raw", Public: true, LinkName: "raw_v1", ABI: "C", Parameters: []hir.ForeignParameter{{Type: hir.TypeI32}}, ResultType: hir.TypeU32,
		}},
	}}}
	baseHash := extractHash(t, base, 1, instances)

	privateResourceChange := base
	privateResourceChange.Modules = slicesCloneModules(base.Modules)
	privateResourceChange.Modules[0].Resources[0].Fields = []hir.ResourceField{{Name: "changed", Type: hir.TypeU32}}
	if hash := extractHash(t, privateResourceChange, 1, instances); hash != baseHash {
		t.Fatalf("private resource representation invalidated source interface: %s != %s", hash, baseHash)
	}

	fieldOrderChange := clonePackage(base)
	fieldOrderChange.Modules[0].Structs[0].Fields[0], fieldOrderChange.Modules[0].Structs[0].Fields[1] = fieldOrderChange.Modules[0].Structs[0].Fields[1], fieldOrderChange.Modules[0].Structs[0].Fields[0]
	if hash := extractHash(t, fieldOrderChange, 1, instances); hash == baseHash {
		t.Fatalf("public field order did not invalidate interface: %s", hash)
	}

	effectChange := clonePackage(base)
	effectChange.Modules[0].Resources[0].DropEffects = []hir.Effect{hir.EffectConsole}
	if hash := extractHash(t, effectChange, 1, instances); hash == baseHash {
		t.Fatalf("resource effect change did not invalidate interface: %s", hash)
	}

	linkChange := clonePackage(base)
	linkChange.Modules[0].ForeignFunctions[0].LinkName = "raw_v2"
	if hash := extractHash(t, linkChange, 1, instances); hash == baseHash {
		t.Fatalf("foreign link change did not invalidate interface: %s", hash)
	}
}

func interfacePackage(self, dependency hir.PackageID) hir.Package {
	selfPrefix := "$p" + strconv.FormatUint(uint64(self), 10) + "::"
	dependencyPrefix := "$p" + strconv.FormatUint(uint64(dependency), 10) + "::"
	return hir.Package{Modules: []hir.Module{{
		Package:   self,
		Path:      "api",
		Structs:   []hir.Struct{{Name: "Packet", Public: true, Type: hir.Type(selfPrefix + "api::Packet"), Fields: []hir.StructField{{Name: "token", Type: hir.Type(dependencyPrefix + "dep::Token")}}}},
		Functions: []hir.Function{{Name: "read", Public: true, Parameters: []hir.Parameter{{Type: hir.SharedBorrow(hir.Type(selfPrefix + "api::Packet"))}}, ResultType: hir.Type(dependencyPrefix + "dep::Token")}},
	}}}
}

func extractHash(t *testing.T, value hir.Package, packageID hir.PackageID, instances map[hir.PackageID]packagegraph.CoordinateID) Hash {
	t.Helper()
	descriptor, err := Extract(value, packageID, instances)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	hash, err := Compute(descriptor)
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	return hash
}

func testCoordinateID(t *testing.T, name string) packagegraph.CoordinateID {
	t.Helper()
	id, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: name, VersionIdentity: "1"})
	if err != nil {
		t.Fatalf("ComputeCoordinateID() error = %v", err)
	}
	return id
}

func analyzeSource(t *testing.T, text string) hir.Package {
	t.Helper()
	frontendResult, err := frontend.AnalyzeSyntax(source.NewFile("api.fy", []byte(text)), source.DefaultLimits())
	if err != nil {
		t.Fatalf("AnalyzeSyntax() error = %v", err)
	}
	if len(frontendResult.Tree.Diagnostics) != 0 || frontendResult.AST == nil {
		t.Fatalf("AnalyzeSyntax() diagnostics = %+v, AST = %+v", frontendResult.Tree.Diagnostics, frontendResult.AST)
	}
	result := semantic.Analyze([]semantic.File{{Path: "api.fy", Module: frontendResult.AST}})
	if result.CompletedThrough != semantic.PhaseFlow || result.FailedPhase != semantic.PhaseNone || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("semantic result = %+v", result)
	}
	return result.HIR
}

func clonePackage(value hir.Package) hir.Package {
	result := value
	result.Modules = slicesCloneModules(value.Modules)
	return result
}

func slicesCloneModules(values []hir.Module) []hir.Module {
	result := append([]hir.Module(nil), values...)
	for index := range result {
		result[index].Structs = append([]hir.Struct(nil), values[index].Structs...)
		for structIndex := range result[index].Structs {
			result[index].Structs[structIndex].Fields = append([]hir.StructField(nil), values[index].Structs[structIndex].Fields...)
		}
		result[index].Enums = append([]hir.Enum(nil), values[index].Enums...)
		result[index].Resources = append([]hir.Resource(nil), values[index].Resources...)
		result[index].ForeignFunctions = append([]hir.ForeignFunction(nil), values[index].ForeignFunctions...)
		result[index].Functions = append([]hir.Function(nil), values[index].Functions...)
	}
	return result
}
