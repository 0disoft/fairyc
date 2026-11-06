package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

// Inspect the production source loader's *admitted* maps, then invoke its
// actual impact command. No test replacement selects aggregate mode or builds
// a Complete; both observations use project_atlas_source::load.
const atlasSDKAggregateCleanupMain = `module main;
import project_atlas_cli as cli;import project_atlas_source as source;
import project_atlas_options as options;import project_atlas_join as join;
import project_atlas_validate as validate;import project_json_tree as json;
import std::process as process;import std::io as io;
@effects(filesystem,console,environment)fn i32 main(){
 if(process::arg_count()!=1){return cli::arguments();}
 let options=switch(options::selected(b"example.session.app",b"fysym1-sha256-128-11111111111111111111111111111111",8,8192,8,64,4194304)){case .ok(p):{p}case .err(_):{return 4;}};
 let loaded=switch(source::load(b".",ref options)){case .ok(p):{p}case .err(e):{discard io::write_usize(e.stage as usize);discard io::write("/");discard io::write_usize(e.code as usize);return 7;}};
 let graph=source::admitted_graph(ref loaded);discard io::write("[");var usize i=0;
 while(i<join::package_count(graph)){if(i>0){discard io::write(",");}discard io::write_bytes(json::source(validate::tree(join::partition(graph,join::package_at(graph,i)))));i+=1;}
 discard io::write("]");return 0;
}`

func TestCandidateAtlasSDKAggregateCleanupSource(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for the full SDK/aggregate source replay")
	}
	root, prepared := atlasSDKAggregateCleanupOracle(t)
	want := map[string]atlasmap.Partition{}
	for _, p := range prepared.partitions {
		encoded, err := atlasmap.EncodeV3(p, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		want[p.Header().PackageInstance] = decoded
	}
	global := candidateAtlasSDKVectorGlobalOracle(t, want)
	index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 8192})
	if err != nil {
		t.Fatal(err)
	}
	candidateAtlasProbeSize(t, "project_atlas_source", atlasSDKAggregateCleanupMain, 2<<20, func(t *testing.T, image string) {
		imageBytes, err := os.ReadFile(image)
		if err != nil {
			t.Fatal(err)
		}
		compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(imageBytes))
		t.Setenv("PATH", "")
		before := remoteTree(t, root)
		code, out, stderr := candidateAtlasSDKCLIProcess(t, image, root, nil)
		if code != 0 || stderr != "" {
			t.Fatalf("production source admission: %d %s %s", code, out, stderr)
		}
		var documents []json.RawMessage
		if err := json.Unmarshal([]byte(out), &documents); err != nil || len(documents) != len(want) {
			t.Fatalf("admitted source partitions: %v %d want %d", err, len(documents), len(want))
		}
		seen := map[string]bool{}
		for _, raw := range documents {
			got, err := atlasmap.DecodeV3(raw, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			instance := got.Header().PackageInstance
			expected, ok := want[instance]
			if !ok || seen[instance] {
				t.Fatalf("unexpected/duplicate partition: %s", instance)
			}
			seen[instance] = true
			verifyCandidateAtlasSDKVectorPartition(t, got, expected)
			verifyCandidateAtlasSDKVectorCompleteness(t, got, expected, global)
		}
		queries := 0
		for _, p := range prepared.partitions {
			for _, symbol := range p.Symbols() {
				if !(symbol.Declaration.Module == "std::vec" && symbol.Declaration.Name == "at" || symbol.Declaration.Module == "main" && symbol.Declaration.Name == "Check") {
					continue
				}
				queries++
				args := append([]string{"experimental", "atlas-impact"}, impactArgs(p.Header().PackageUID, symbol.Full)...)
				args = replaceWorkspaceSessionArgument(args, "--max-calls", strconv.Itoa(len(index.Calls())))
				code, out, diagnostic := candidateAtlasSDKCLIProcess(t, image, root, args)
				if code != 0 || diagnostic != "" {
					t.Fatalf("combined impact: %d %s %s", code, out, diagnostic)
				}
				var got atlasmap.ImpactResult
				if err := json.Unmarshal([]byte(out), &got); err != nil {
					t.Fatal(err)
				}
				expected, err := index.Impact(p.Header().PackageInstance, symbol.Full, 8, 64)
				if err != nil {
					t.Fatal(err)
				}
				compareCandidateAtlasAggregateImpact(t, got, expected, compiler)
			}
		}
		if queries != 2 || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatal("missing SDK/cleanup query or changed workspace", queries)
		}
	}, "project_atlas_cli")
}
