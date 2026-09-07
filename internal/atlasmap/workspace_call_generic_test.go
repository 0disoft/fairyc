package atlasmap

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestWorkspaceCallV2JoinsConcreteDependencyImplementations(t *testing.T) {
	partitions, expectations := genericCallPartitions(t)
	budget := WorkspaceCallBudget{MaxPackages: 2, MaxCalls: 4}
	if _, err := NewWorkspaceCallIndex(partitions, expectations, budget); err == nil {
		t.Fatal("v1 join admitted v2")
	}
	index, err := NewWorkspaceCallIndexV2(partitions, expectations, budget)
	if err != nil {
		t.Fatal(err)
	}
	if index.Usage() != (WorkspaceCallUsage{Packages: 2, Calls: 4, CrossPackageCalls: 2}) {
		t.Fatalf("usage: %+v", index.Usage())
	}
	concrete := map[string]bool{}
	for _, call := range index.Calls() {
		if call.Source.PackageInstance == call.Target.PackageInstance {
			continue
		}
		if call.SourceSpecialization == "" || call.TargetSpecialization == "" {
			t.Fatal("cross-package concrete endpoint erased")
		}
		concrete[call.TargetSpecialization] = true
		incoming, err := index.IncomingCalls(call.Target.PackageInstance, call.Target.Symbol)
		if err != nil || len(incoming) != 2 {
			t.Fatalf("incoming: %v %+v", err, incoming)
		}
		outgoing, err := index.OutgoingCalls(call.Source.PackageInstance, call.Source.Symbol)
		if err != nil || !reflect.DeepEqual(incoming, outgoing) {
			t.Fatal("directions disagree")
		}
		incoming[0].TargetSpecialization = "changed"
		fresh, _ := index.IncomingCalls(call.Target.PackageInstance, call.Target.Symbol)
		if fresh[0].TargetSpecialization == "changed" {
			t.Fatal("result aliases ledger")
		}
	}
	if len(concrete) != 2 {
		t.Fatal("distinct implementations collapsed")
	}
	slices.Reverse(partitions)
	again, err := NewWorkspaceCallIndexV2(partitions, expectations, budget)
	if err != nil || !reflect.DeepEqual(index.Calls(), again.Calls()) {
		t.Fatalf("ordering: %v", err)
	}
	budget.MaxCalls--
	if _, err := NewWorkspaceCallIndexV2(partitions, expectations, budget); !errors.Is(err, ErrWorkspaceJoinBudget) {
		t.Fatalf("budget: %v", err)
	}
}

func TestWorkspaceCallV2RejectsMissingDependencyImplementation(t *testing.T) {
	for _, id := range []string{"", "g0_" + strings.Repeat("0", 64)} {
		partitions, expectations := genericCallPartitions(t)
		for i := range partitions {
			if len(partitions[i].Header().Dependencies) == 0 {
				continue
			}
			for j := range partitions[i].data.specializations {
				for k := range partitions[i].data.specializations[j].Function.Calls {
					partitions[i].data.specializations[j].Function.Calls[k].Specialization = id
				}
			}
		}
		if _, err := NewWorkspaceCallIndexV2(partitions, expectations, WorkspaceCallBudget{MaxPackages: 2, MaxCalls: 4}); err == nil {
			t.Fatal("invalid dependency implementation admitted")
		}
	}
}

func genericCallPartitions(t *testing.T) ([]Partition, map[string]LoadExpectation) {
	t.Helper()
	root := t.TempDir()
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.join.PACKAGE\"\nname = \"PACKAGE\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/14\"\nsource = \"src\"\n"
	files := map[string]string{
		"fairy.toml":     strings.ReplaceAll(manifest, "PACKAGE", "app") + "[workspace]\nmembers = [\"lib\"]\n[dependencies.lib]\npath = \"lib\"\nversion = \"1.0.0\"\nfeatures = []\n",
		"lib/fairy.toml": strings.ReplaceAll(manifest, "PACKAGE", "lib"),
		"src/main.fy":    "module main; import lib as lib; fn T pass<T>(T x) { return lib::id<T>(x); } fn i32 main() { assert(pass<i32>(42) == 42); assert(pass<u32>(7) == 7); return 0; }",
		"lib/src/lib.fy": "module lib; pub fn T id<T>(T x) { return x; }",
	}
	for name, contents := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	_, partitions, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectationsV2(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	return partitions, expectations
}
