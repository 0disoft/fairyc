package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func candidateAtlasProduceMain() string {
	main := strings.SplitN(candidateAtlasTypesMain, " if(process::arg_count()==2)", 2)[0]
	main = strings.Replace(main, "@effects(filesystem,console)", "@effects(filesystem,console,environment)", 1)
	main = strings.Replace(main, "import project_atlas_types as types;", `import project_atlas_types as types;
import project_atlas_produce as produce;import project_atlas_schema as schema;
import project_atlas_facts as facts;import project_lock_document as locks;
import project_workspace_graph as graph;import project_cache_environment as environment;
import project_compiler_image_hash as image;`, 1)
	return main + `
 let lock=switch(locks::project(resolved,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 let metadata=switch(schema::prepare(ref source,ref table,ref complete,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 var rows=switch(schema::type_rows(ref metadata,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 var usize image_work=0;let compiler_image=switch(environment::compiler_current(ref mut image_work)){case .ok(p):{p}case .err(_):{return 7;}};
 var packages=v::empty<produce::Package>();var usize owner=0;var usize total=0;var usize external=0;
 while(owner<graph::node_count(graph)){
  let f=switch(facts::collect(graph,ref lock,ref table,compiler::public_interfaces(ref complete),p,image::value(ref compiler_image),owner,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
  let part=switch(produce::package(f,ref source,p,ref table,ref metadata,ref mut rows,owner,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
  total+=produce::call_count(ref part);external+=produce::external_count(ref part);if(!v::push<produce::Package>(ref mut packages,part).is_ok){return 7;}owner+=1;
 }
 var out=bytes::empty();if(!w::raw(ref mut out,b"[",ref mut meter).is_ok){return 7;}owner=0;
 while(owner<v::len<produce::Package>(ref packages)){
  let encoded=switch(produce::encode(v::at<produce::Package>(ref packages,owner),total-external,external,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
  if(owner>0 && !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::raw(ref mut out,bytes::view(ref encoded),ref mut meter).is_ok){return 7;}owner+=1;
 }
 if(!w::raw(ref mut out,b"]",ref mut meter).is_ok){return 7;}return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`
}

func TestCandidateAtlasProduction(t *testing.T) {
	root := candidateAtlasFamilyWorkspace(t)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("production oracle: %d %s", code, &diagnostic)
	}
	want := map[string]map[string]string{}
	wantFunctions := map[string][]semanticsnapshot.FunctionSummary{}
	for _, p := range prepared.partitions {
		data, err := atlasmap.EncodeV3(p, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		var envelope struct {
			Map struct{ Declarations json.RawMessage }
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			t.Fatal(err)
		}
		want[p.Header().PackageInstance] = atlasDeclarationRows(t, envelope.Map.Declarations)
		wantFunctions[p.Header().PackageInstance] = p.Functions()
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateAtlasProduceMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		before := remoteTree(t, root)
		code, out, stderr := fetchCLIProcess(t, image, root, nil, "")
		if code != 0 || stderr != "" {
			t.Fatalf("native production: %d %s %s", code, out, stderr)
		}
		var documents []json.RawMessage
		if err := json.Unmarshal([]byte(out), &documents); err != nil {
			t.Fatalf("native maps: %v %s", err, out)
		}
		if len(documents) != len(want) {
			t.Fatal("partition count", len(documents))
		}
		seen := map[string]bool{}
		for _, data := range documents {
			partition, err := atlasmap.DecodeV3(data, 4<<20)
			if err != nil {
				t.Fatalf("Go rejected Fairy map: %v", err)
			}
			instance := partition.Header().PackageInstance
			if _, exists := want[instance]; !exists || seen[instance] {
				t.Fatal("unexpected or duplicate partition", instance)
			}
			seen[instance] = true
			var envelope struct {
				Map struct{ Declarations json.RawMessage }
			}
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatal(err)
			}
			got := atlasDeclarationRows(t, envelope.Map.Declarations)
			if !reflect.DeepEqual(got, want[partition.Header().PackageInstance]) {
				t.Fatal("produced declarations differ", partition.Header().PackageUID)
			}
			functions := partition.Functions()
			expected := wantFunctions[instance]
			if len(functions) != len(expected) {
				t.Fatal("produced function count differs", instance)
			}
			for i, f := range functions {
				w := expected[i]
				if f.Function != w.Function || f.Provenance != w.Provenance || !slices.Equal(f.Calls, w.Calls) {
					t.Fatalf("produced calls differ: got %+v, want %+v", f, w)
				}
			}
		}
		if !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatal("production changed workspace")
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}
