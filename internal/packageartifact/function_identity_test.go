package packageartifact

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestCheckedInWorkspaceFunctionsHaveUniqueStableIdentities(t *testing.T) {
	t.Parallel()
	workspace, err := workspaceloader.Load(filepath.Join("..", "..", "examples", "workspace_quickstart"))
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	identities, err := workspace.PackageIdentities()
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified || len(result.MIR.Functions) == 0 {
		t.Fatalf("workspace result is not verified or has no functions: %+v", result)
	}
	seen := make(map[FunctionID]string, len(result.MIR.Functions))
	for _, function := range result.MIR.Functions {
		descriptor, describeErr := DescribeFunction(function, identities)
		if describeErr != nil {
			t.Fatalf("DescribeFunction(%q): %v", function.Name, describeErr)
		}
		id := mustFunctionID(t, descriptor)
		if previous, duplicate := seen[id]; duplicate {
			t.Fatalf("functions %q and %q share FunctionID %s", previous, function.Name, id)
		}
		seen[id] = function.Name
	}
}

func TestFunctionIdentityIsStableAcrossBuildLocalPackageNumbers(t *testing.T) {
	t.Parallel()
	owner := testInstanceID(t, "owner")
	dependency := testInstanceID(t, "dependency")
	left := mir.Function{
		Name: "$p1::api::Reader::read", Span: hir.Span{Package: 1},
		ParameterTypes: []hir.Type{hir.SharedBorrow("$p1::api::Reader")},
		ResultType:     hir.Result("$p2::model::Packet", hir.TypeI32),
		Effects:        []hir.Effect{hir.EffectFilesystem},
	}
	right := left
	right.Name = "$p7::api::Reader::read"
	right.Span.Package = 7
	right.ParameterTypes = []hir.Type{hir.SharedBorrow("$p7::api::Reader")}
	right.ResultType = hir.Result("$p9::model::Packet", hir.TypeI32)

	leftDescriptor, err := DescribeFunction(left, map[hir.PackageID]packagegraph.InstanceID{1: owner, 2: dependency})
	if err != nil {
		t.Fatal(err)
	}
	rightDescriptor, err := DescribeFunction(right, map[hir.PackageID]packagegraph.InstanceID{7: owner, 9: dependency})
	if err != nil {
		t.Fatal(err)
	}
	leftID := mustFunctionID(t, leftDescriptor)
	rightID := mustFunctionID(t, rightDescriptor)
	if leftID != rightID {
		t.Fatalf("build-local package numbers changed FunctionID: %s != %s", leftID, rightID)
	}
	if leftDescriptor.Name != "api::Reader::read" || leftDescriptor.ParameterTypes[0] != "ref pkg["+owner.String()+"]::api::Reader" {
		t.Fatalf("descriptor = %+v", leftDescriptor)
	}
}

func TestFunctionIdentityTracksPackageSignatureEffectsAndDropGlue(t *testing.T) {
	t.Parallel()
	owner := testInstanceID(t, "owner")
	base := FunctionDescriptor{Package: owner, Name: "api::read", ParameterTypes: []string{"i32"}, ResultType: "i32"}
	baseID := mustFunctionID(t, base)
	changes := []FunctionDescriptor{
		{Package: testInstanceID(t, "other"), Name: base.Name, ParameterTypes: base.ParameterTypes, ResultType: base.ResultType},
		{Package: owner, Name: "api::write", ParameterTypes: base.ParameterTypes, ResultType: base.ResultType},
		{Package: owner, Name: base.Name, ParameterTypes: []string{"u32"}, ResultType: base.ResultType},
		{Package: owner, Name: base.Name, ParameterTypes: base.ParameterTypes, ResultType: "u32"},
		{Package: owner, Name: base.Name, ParameterTypes: base.ParameterTypes, ResultType: base.ResultType, Effects: []string{"filesystem"}},
		{Package: owner, Name: base.Name, ParameterTypes: base.ParameterTypes, ResultType: base.ResultType, DropGlueFor: "pkg[" + owner.String() + "]::api::Handle"},
	}
	for index, changed := range changes {
		if id := mustFunctionID(t, changed); id == baseID {
			t.Fatalf("change %d did not invalidate FunctionID %s", index, id)
		}
	}
}

func TestFunctionIdentityRejectsIncompleteOrBuildLocalInputs(t *testing.T) {
	t.Parallel()
	owner := testInstanceID(t, "owner")
	if _, err := DescribeFunction(mir.Function{Name: "$p2::api::read", Span: hir.Span{Package: 1}, ResultType: hir.TypeVoid}, map[hir.PackageID]packagegraph.InstanceID{1: owner}); err == nil {
		t.Fatal("DescribeFunction accepted mismatched function owner prefix")
	}
	if _, err := DescribeFunction(mir.Function{Name: "$p1::api::read", Span: hir.Span{Package: 1}, ParameterTypes: []hir.Type{"$p2::dep::Token"}, ResultType: hir.TypeVoid}, map[hir.PackageID]packagegraph.InstanceID{1: owner}); err == nil {
		t.Fatal("DescribeFunction accepted an unmapped nominal type")
	}
	if _, err := ComputeFunctionID(FunctionDescriptor{Package: owner, Name: "$p1::api::read", ResultType: "void"}); err == nil {
		t.Fatal("ComputeFunctionID accepted a build-local logical name")
	}
	if _, err := ComputeFunctionID(FunctionDescriptor{Package: owner, Name: "api::read", ParameterTypes: []string{"$p1::api::Token"}, ResultType: "void"}); err == nil {
		t.Fatal("ComputeFunctionID accepted a build-local parameter type")
	}
	if _, err := ComputeFunctionID(FunctionDescriptor{Package: owner, Name: "api::read", ParameterTypes: []string{"not a canonical type"}, ResultType: "void"}); err == nil {
		t.Fatal("ComputeFunctionID accepted an arbitrary parameter type")
	}
	if _, err := ComputeFunctionID(FunctionDescriptor{Package: owner, Name: "api::read", ResultType: "void", Effects: []string{"network", "filesystem"}}); err == nil {
		t.Fatal("ComputeFunctionID accepted noncanonical effects")
	}
}

func TestParseFunctionIDRejectsNoncanonicalText(t *testing.T) {
	t.Parallel()
	valid := FunctionIDPrefix + strings.Repeat("a", 64)
	parsed, err := ParseFunctionID(valid)
	if err != nil || parsed.String() != valid {
		t.Fatalf("ParseFunctionID(%q) = %v, %v", valid, parsed, err)
	}
	for _, value := range []string{
		strings.TrimPrefix(valid, FunctionIDPrefix),
		valid[:len(valid)-1],
		strings.ToUpper(valid),
		FunctionIDPrefix + strings.Repeat("g", 64),
	} {
		if _, err := ParseFunctionID(value); err == nil {
			t.Fatalf("ParseFunctionID(%q) accepted", value)
		}
	}
}

func mustFunctionID(t *testing.T, descriptor FunctionDescriptor) FunctionID {
	t.Helper()
	id, err := ComputeFunctionID(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func testInstanceID(t *testing.T, name string) packagegraph.InstanceID {
	t.Helper()
	id, err := packagegraph.ComputeInstanceID(packagegraph.Identity{PackageUID: name, SourceIdentity: "source-" + name, VersionIdentity: "1"})
	if err != nil {
		t.Fatal(err)
	}
	return id
}
