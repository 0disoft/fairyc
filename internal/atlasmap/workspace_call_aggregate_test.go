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
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestWorkspaceCallV3JoinsAggregateCleanupAcrossPackages(t *testing.T) {
	partitions, expectations := aggregateCallPartitions(t)
	budget := WorkspaceCallBudget{MaxPackages: 2, MaxCalls: 4}
	if _, err := NewWorkspaceCallIndex(partitions, expectations, budget); err == nil {
		t.Fatal("v1 admitted v3")
	}
	if _, err := NewWorkspaceCallIndexV2(partitions, expectations, budget); err == nil {
		t.Fatal("v2 admitted v3")
	}
	index, err := NewWorkspaceCallIndexV3(partitions, expectations, budget)
	if err != nil {
		t.Fatal(err)
	}
	if index.Usage() != (WorkspaceCallUsage{Packages: 2, Calls: 4, CrossPackageCalls: 2}) {
		t.Fatalf("usage: %+v", index.Usage())
	}
	implementations := map[string]bool{}
	for _, call := range index.Calls() {
		if call.SourceBody == "" {
			if call.SourceSpecialization != "" || call.Source.PackageInstance != call.Target.PackageInstance {
				t.Fatal("ordinary main call acquired cleanup identity")
			}
			continue
		}
		if call.SourceBody != "cleanup" || call.SourceSpecialization == "" || call.TargetSpecialization == "" || call.Source.PackageInstance == call.Target.PackageInstance {
			t.Fatalf("cleanup identity lost: %+v", call)
		}
		symbol := workspaceTestSymbol(t, partitions, call.Source.PackageInstance, "main", "Token")
		if call.Source.Symbol != symbol.Visible {
			t.Fatal("cleanup not owned by original resource")
		}
		implementations[call.SourceSpecialization] = true
		incoming, err := index.IncomingCalls(call.Target.PackageInstance, call.Target.Symbol)
		if err != nil || len(incoming) != 2 {
			t.Fatalf("incoming: %v %+v", err, incoming)
		}
		outgoing, err := index.OutgoingCalls(call.Source.PackageInstance, symbol.Full)
		if err != nil || !reflect.DeepEqual(outgoing, incoming) {
			t.Fatal("directions disagree")
		}
		incoming[0].SourceBody = "corrupt"
		fresh, _ := index.OutgoingCalls(call.Source.PackageInstance, symbol.Visible)
		if fresh[0].SourceBody != "cleanup" {
			t.Fatal("caller mutated ledger")
		}
		context, err := index.PackageContext(call.Source.PackageInstance)
		if err != nil || context.LogicalSchema != LogicalSchemaV3 {
			t.Fatal("v3 context lost")
		}
		unknown := false
		for _, fact := range context.Completeness {
			if fact.Relation == "generic-template-bodies" && fact.Status == semanticsnapshot.CompletenessUnavailable {
				unknown = true
			}
		}
		if !unknown {
			t.Fatal("join claimed complete template bodies")
		}
	}
	if len(implementations) != 2 {
		t.Fatal("cleanup implementations collapsed")
	}
	slices.Reverse(partitions)
	again, err := NewWorkspaceCallIndexV3(partitions, expectations, budget)
	if err != nil || !reflect.DeepEqual(index.Calls(), again.Calls()) {
		t.Fatalf("partition ordering: %v", err)
	}
	budget.MaxCalls--
	if _, err := NewWorkspaceCallIndexV3(partitions, expectations, budget); !errors.Is(err, ErrWorkspaceJoinBudget) {
		t.Fatalf("budget: %v", err)
	}
}

func TestWorkspaceCallV3RejectsMissingCleanupTargetAndStaleMaps(t *testing.T) {
	for _, id := range []string{"", "g0_" + strings.Repeat("0", 64)} {
		partitions, expectations := aggregateCallPartitions(t)
		for i := range partitions {
			for j := range partitions[i].data.specializations {
				cleanup := partitions[i].data.specializations[j].Cleanup
				if cleanup != nil {
					cleanup.Calls[0].Specialization = id
				}
			}
		}
		if _, err := NewWorkspaceCallIndexV3(partitions, expectations, WorkspaceCallBudget{MaxPackages: 2, MaxCalls: 4}); err == nil {
			t.Fatal("invalid cleanup target admitted")
		}
	}
	partitions, expectations := aggregateCallPartitions(t)
	instance := partitions[0].Header().PackageInstance
	changed := expectations[instance]
	changed.Current.Source[0] ^= 1
	expectations[instance] = changed
	if _, err := NewWorkspaceCallIndexV3(partitions, expectations, WorkspaceCallBudget{MaxPackages: 2, MaxCalls: 4}); !errors.Is(err, ErrStaleMap) {
		t.Fatalf("stale: %v", err)
	}
}

func aggregateCallPartitions(t *testing.T) ([]Partition, map[string]LoadExpectation) {
	t.Helper()
	root := t.TempDir()
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.cleanup.join.PACKAGE\"\nname = \"PACKAGE\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/17\"\nsource = \"src\"\n"
	files := map[string]string{
		"fairy.toml":     strings.ReplaceAll(manifest, "PACKAGE", "app") + "[workspace]\nmembers = [\"lib\"]\n[dependencies.lib]\npath = \"lib\"\nversion = \"1.0.0\"\nfeatures = []\n",
		"lib/fairy.toml": strings.ReplaceAll(manifest, "PACKAGE", "lib"),
		"src/main.fy": `module main; import lib as lib;
resource Token<T> { T value; cleanup { let T observed = lib::id<T>(this.value); } }
fn Token<T> make<T>(T value) { return Token<T> { .value = value }; }
fn i32 main() { let Token<i32> a = make<i32>(42); let Token<u32> b = make<u32>(7); return a.value; }`,
		"lib/src/lib.fy": "module lib; pub fn T id<T>(T value) { return value; }",
	}
	for name, contents := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
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
	_, partitions, err := PrepareWorkspaceV3(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectationsV3(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	for i := range partitions {
		encoded, err := EncodeV3(partitions[i], testMapLimit)
		if err != nil {
			t.Fatal(err)
		}
		partitions[i], err = DecodeV3(encoded, testMapLimit)
		if err != nil {
			t.Fatal(err)
		}
	}
	return partitions, expectations
}
