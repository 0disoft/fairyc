package cli

import (
	"bytes"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type candidateAtlasSDKUserAggregateFixture struct {
	name, source, aggregate string
}

func candidateAtlasSDKUserAggregateFixtures() []candidateAtlasSDKUserAggregateFixture {
	return []candidateAtlasSDKUserAggregateFixture{
		{"core SDK and struct", `module main;import std::num as n;
pub struct Cell<T>{T value;}
fn i32 main(){let cell=Cell<i32>{.value=42};let parsed=n::parse_i64_decimal(b"42");assert(parsed.is_ok);return cell.value;}`, "Cell"},
		{"SDK Vec and struct", `module main;import std::vec as v;
pub struct Cell<T>{T value;}
fn i32 main(){let cell=Cell<i32>{.value=42};var values=v::empty<i32>();assert(v::push<i32>(ref mut values,cell.value).is_ok);return *v::at<i32>(ref values,0);}`, "Cell"},
		{"SDK Vec and resource", `module main;import std::vec as v;
pub resource Flag<T>{T value;cleanup{assert(this.value);}}
fn i32 main(){let flag=Flag<bool>{.value=true};var values=v::empty<bool>();assert(v::push<bool>(ref mut values,flag.value).is_ok);return if(*v::at<bool>(ref values,0)){42}else{0};}`, "Flag"},
		{"SDK Vec and same-name struct", `module main;import std::vec as v;
pub struct Vec<T>{T value;}
fn i32 main(){let cell=Vec<i32>{.value=42};var values=v::empty<i32>();assert(v::push<i32>(ref mut values,cell.value).is_ok);return *v::at<i32>(ref values,0);}`, "Vec"},
		{"SDK Vec pop and struct", `module main;import std::vec as v;
pub struct Cell<T>{T value;}
fn i32 main(){let cell=Cell<i32>{.value=42};var values=v::empty<i32>();assert(v::push<i32>(ref mut values,cell.value).is_ok);return switch(v::pop<i32>(ref mut values)){case .some(value):{value}case .none:{0}};}`, "Cell"},
	}
}

func TestAtlasSDKUserAggregateOracle(t *testing.T) {
	for _, fixture := range candidateAtlasSDKUserAggregateFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, fixture.source)
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				candidateAtlasSDKUserAggregateOracleFailure(t, root)
				t.Fatalf("SDK/user aggregate oracle: %d %s", code, &diagnostic)
			}
			var sdk, aggregate bool
			for _, partition := range prepared.partitions {
				encoded, err := atlasmap.EncodeV3(partition, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				roundtrip, err := atlasmap.EncodeV3(decoded, 4<<20)
				if err != nil || !bytes.Equal(encoded, roundtrip) {
					t.Fatalf("SDK/user aggregate canonical v3 roundtrip: %v", err)
				}
				sdk = sdk || partition.Header().PackageUID == "fairyc.std"
				if partition.Header().PackageUID != "example.session.app" {
					continue
				}
				for _, instance := range partition.Specializations() {
					if instance.Declaration.Module == "main" && instance.Declaration.Name == fixture.aggregate && instance.Aggregate != nil {
						if len(instance.TypeArguments) != 1 || len(instance.Aggregate.Fields) != 1 || (instance.Cleanup != nil) != (fixture.aggregate == "Flag") {
							t.Fatalf("missing scalar arguments/field/cleanup: %+v", instance)
						}
						aggregate = true
					}
				}
			}
			if !sdk || !aggregate {
				t.Fatalf("missing simultaneous SDK/user aggregate facts: sdk=%v aggregate=%v", sdk, aggregate)
			}
			if _, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 8192}); err != nil {
				t.Fatal("SDK aggregate current join", err)
			}
			stable, err := workspacecompiler.Check(prepared.workspace)
			if err != nil || stable.Verified || len(stable.Semantic.Diagnostics)+len(stable.Semantic.Unsupported) == 0 {
				t.Fatalf("private aggregate opt-in escaped stable source gate: %v %+v", err, stable)
			}
			for _, denied := range []struct{ source, diagnostic string }{
				{fixture.source + `fn usize forged(){return unsafe{storage_size<i32>()};}`, "E0101"},
				{fixture.source + `struct Thing{i32 value;}fn i32 nominal(){let value=` + fixture.aggregate + `<Thing>{.value=Thing{.value=1}};return 1;}`, "E4502"},
			} {
				deniedRoot := writeWorkspaceSessionFixture(t, denied.source)
				workspace, err := workspaceloader.Load(deniedRoot)
				if err != nil {
					t.Fatal(err)
				}
				checked, err := workspacecompiler.CheckGeneric(workspace)
				found := false
				for _, diagnostic := range checked.Semantic.Diagnostics {
					found = found || diagnostic.Code == denied.diagnostic && diagnostic.Primary.Source == "main.fy"
				}
				if err != nil || checked.Verified || !found {
					t.Fatalf("SDK composition must reject client authority/nominal arguments with %s: %v diagnostics=%+v", denied.diagnostic, err, checked.Semantic.Diagnostics)
				}
			}
		})
	}
}

func candidateAtlasSDKUserAggregateOracleFailure(t *testing.T, root string) {
	t.Helper()
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := workspacecompiler.CheckGeneric(workspace)
	if err != nil || !compiled.Verified {
		t.Fatalf("SDK aggregate compilation: %v diagnostics=%+v unsupported=%+v", err, compiled.Semantic.Diagnostics, compiled.Semantic.Unsupported)
	}
	build, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := atlasmap.PrepareWorkspaceV3(workspace, compiled, build); err != nil {
		t.Fatalf("SDK aggregate Atlas snapshot: %v", err)
	}
	if _, err := atlasmap.CurrentWorkspaceExpectationsV3(workspace, compiled, build); err != nil {
		t.Fatalf("SDK aggregate Atlas expectations: %v", err)
	}
}
