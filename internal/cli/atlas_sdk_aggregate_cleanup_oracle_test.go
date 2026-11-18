package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

// The SDK is used inside a concrete user cleanup, rather than only beside a
// resource in main. A generic function also returns a concrete user struct.
const atlasSDKAggregateCleanupSource = `module main;import std::vec as v;
pub struct Cell<T>{T value;}
pub resource Check<T>{T value;cleanup{
var values=v::empty<i32>();assert(v::push<i32>(ref mut values,42).is_ok);
assert(*v::at<i32>(ref values,0)==42);}}
pub fn Cell<T> cell<T>(T value){return Cell<T>{.value=value};}
fn i32 leaf(){let check=Check<bool>{.value=true};return cell<i32>(42).value;}
fn i32 main(){return leaf();}`

func atlasSDKAggregateCleanupOracle(t *testing.T) (string, preparedAtlasWorkspace) {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, atlasSDKAggregateCleanupSource)
	var diagnostic bytes.Buffer
	p, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		candidateAtlasSDKUserAggregateOracleFailure(t, root)
		t.Fatalf("cleanup SDK aggregate Go oracle: %d %s", code, &diagnostic)
	}
	return root, p
}

func TestAtlasSDKAggregateCleanupOracle(t *testing.T) {
	_, p := atlasSDKAggregateCleanupOracle(t)
	counts := map[string]int{}
	var sdk atlasmap.Partition
	for _, partition := range p.partitions {
		h := partition.Header()
		t.Logf("Go source %s: provenance=%s declarations=%d functions=%d implementations=%d relations=%d", h.PackageUID, h.PackageSourceKind, len(partition.Declarations()), len(partition.Functions()), len(partition.Specializations()), h.Counts.Relations)
		if h.PackageUID == "fairyc.std" {
			sdk = partition
		}
		for _, s := range partition.Specializations() {
			if len(s.TypeArguments) != 1 {
				t.Fatalf("non-scalar source family: %+v", s)
			}
			counts[s.Declaration.Module+"::"+s.Declaration.Name+"<"+s.TypeArguments[0].Name+">"]++
			if s.Declaration.Module == "main" && s.Declaration.Name == "Check" {
				if s.Aggregate == nil || len(s.Aggregate.Fields) != 1 || s.Aggregate.Fields[0].Type.Name != "bool" || s.Cleanup == nil {
					t.Fatalf("missing user resource shape/cleanup: %+v", s)
				}
				calls := map[string]int{}
				for _, call := range s.Cleanup.Calls {
					calls[call.Target.Module+"::"+call.Target.Name]++
					written := map[string]string{
						"empty": "v::empty<i32>()", "push": "v::push<i32>(ref mut values,42)", "at": "v::at<i32>(ref values,0)",
					}[call.Target.Name]
					start := strings.Index(atlasSDKAggregateCleanupSource, written)
					if written == "" || call.Specialization == "" || call.Span.Source != "main.fy" || call.Span.StartByte != start || call.Span.EndByte != start+len(written) {
						t.Fatalf("cleanup SDK endpoint lost: %+v", call)
					}
				}
				if !reflect.DeepEqual(calls, map[string]int{"std::vec::empty": 1, "std::vec::push": 1, "std::vec::at": 1}) {
					t.Fatalf("cleanup SDK calls: %v", calls)
				}
			}
		}
	}
	want := map[string]int{"main::Cell<i32>": 1, "main::Check<bool>": 1, "main::cell<i32>": 1}
	for _, scalar := range []string{"u8", "i64", "i32", "u32", "usize"} {
		want["std::vec::Vec<"+scalar+">"] = 1
		want["std::vec::take_slot<"+scalar+">"] = 1
	}
	for _, name := range []string{"empty", "push", "at"} {
		want["std::vec::"+name+"<i32>"] = 1
	}
	if !reflect.DeepEqual(counts, want) {
		t.Fatalf("combined source closure: got %v want %v", counts, want)
	}
	// Root demand must preserve the original pinned SDK declaration inventory,
	// interface, and source provenance; only its concrete observations grow.
	unusedRoot := writeWorkspaceSessionFixture(t, candidateAtlasSDKVectorUnusedSource)
	var diagnostic bytes.Buffer
	unused, code := prepareAtlasWorkspaceSchema(unusedRoot, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("unused SDK oracle: %d %s", code, &diagnostic)
	}
	for _, baseline := range unused.partitions {
		if baseline.Header().PackageUID == "fairyc.std" && (!reflect.DeepEqual(baseline.Declarations(), sdk.Declarations()) || baseline.Header().GenericInterfaceHash != sdk.Header().GenericInterfaceHash || baseline.Header().PackageSource != sdk.Header().PackageSource || baseline.Header().PackageSourceKind != sdk.Header().PackageSourceKind) {
			t.Fatal("user cleanup demand changed original SDK authority")
		}
	}
}

func TestAtlasSDKAggregateCleanupRefusals(t *testing.T) {
	for _, denied := range []struct{ name, appended, diagnostic string }{
		{"client storage", `fn usize forged(){return unsafe{storage_size<i32>()};}`, "E0101"},
		{"nominal user argument", `struct Thing{i32 value;}fn i32 nominal(){let item=Cell<Thing>{.value=Thing{.value=1}};return 1;}`, "E4502"},
	} {
		t.Run(denied.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, atlasSDKAggregateCleanupSource+denied.appended)
			workspace, _, err := loadWorkspaceWithLock(root, false)
			if err != nil {
				t.Fatal(err)
			}
			checked, err := workspacecompiler.CheckGeneric(workspace)
			found := false
			for _, diagnostic := range checked.Semantic.Diagnostics {
				found = found || diagnostic.Code == denied.diagnostic && diagnostic.Primary.Source == "main.fy"
			}
			if err != nil || checked.Verified || !found {
				t.Fatalf("missing %s refusal: %v %+v", denied.diagnostic, err, checked.Semantic.Diagnostics)
			}
		})
	}
}
