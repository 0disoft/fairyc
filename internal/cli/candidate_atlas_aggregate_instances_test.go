package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func candidateAtlasAggregateInstancesMain() string {
	main := strings.SplitN(candidateAtlasTypesMain, " if(process::arg_count()==2)", 2)[0]
	main = strings.Replace(main, "import project_atlas_types as types;", `import project_atlas_types as types;
import project_atlas_instances as instances;import project_atlas_schema as schema;`, 1)
	main = strings.Replace(main, "compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)", "compiler::build_complete_aggregate_sources(input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut source_work,ref mut link_work)", 1)
	return main + `
 let metadata=switch(schema::prepare_aggregates(ref source,ref table,ref complete,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
 var u8[64] raw=[0;64];let usize n=if(process::arg_count()>1){switch(process::arg(1,ref mut raw)){case .ok(n):{n}case .err(_):{return 7;}}}else{0};let ref u8[] argument=ref raw;let mode=argument[..n];
 let absent=table::assemble(bytes::empty(),v::empty<table::Coordinate>());
 if(bytes::equal(mode,b"budget")){meter=16777216;}
 if(bytes::equal(mode,b"ordinary")){return switch(instances::collect(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};}
 if(bytes::equal(mode,b"stale")){var u8[4096] path=[0;4096];let n=switch(process::arg(2,ref mut path)){case .ok(n):{n}case .err(_):{return 7;}};let ref u8[] root=ref path;var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;let changed=switch(visibility::load(root[..n],ref mut budget,ref mut work)){case .ok(p):{p}case .err(_):{return 4;}};let projected=switch(input::project(ref changed,ref mut work)){case .ok(p):{p}case .err(_):{return 4;}};return switch(schema::prepare_aggregates(ref projected,ref table,ref complete,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};}
 let observed=switch(if(bytes::equal(mode,b"missing-coordinate")){instances::collect_aggregates(ref source,ref absent,compiler::checked_program(ref complete),ref metadata,ref mut meter)}else{instances::collect_aggregates(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)}){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
 if(!instances::aggregate_types(ref observed)){return 4;}
 var usize type=1;while(type<=clean::type_count(clean)){if(clean::type_at(clean,type).kind==22 && types::render(ref source,ref table,clean,type,ref mut meter).is_ok){return 4;}type+=1;}
 var out=bytes::empty();if(!w::raw(ref mut out,b"[",ref mut meter).is_ok){return 4;}var usize i=0;
 while(i<instances::count(ref observed)){let r=instances::at(ref observed,i);if(i>0 && !w::raw(ref mut out,b",",ref mut meter).is_ok){return 4;}
  if(!w::raw(ref mut out,b"{",ref mut meter).is_ok || !w::pair(ref mut out,b"id",instances::identity(ref observed,r),ref mut meter).is_ok || !w::raw(ref mut out,b",\"declaration\":",ref mut meter).is_ok || !schema::key(ref mut out,ref source,r.declaration,ref mut meter).is_ok || !w::raw(ref mut out,b",\"type_arguments\":",ref mut meter).is_ok || !instances::arguments(ref mut out,ref observed,r,ref mut meter).is_ok || !w::raw(ref mut out,b",\"type\":",ref mut meter).is_ok){return 4;}
  if(r.type==0){if(!w::raw(ref mut out,b"null",ref mut meter).is_ok){return 4;}}else{let t=switch(types::render(ref source,ref table,clean,r.type,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};if(!w::raw(ref mut out,bytes::view(ref t),ref mut meter).is_ok){return 4;}}
  if(!w::raw(ref mut out,if(r.type!=0 && r.frame!=0){b",\"cleanup\":true,\"calls\":["}else{b",\"cleanup\":false,\"calls\":["},ref mut meter).is_ok){return 4;}var usize j=0;if(r.frame!=0){let body=ir::at(p,r.frame);while(j<ir::call_count(body)){if(j>0 && !w::raw(ref mut out,b",",ref mut meter).is_ok){return 4;}let target=instances::call_frame(ref observed,r.frame,j);if(!w::string(ref mut out,instances::frame_identity(ref observed,target),ref mut meter).is_ok){return 4;}j+=1;}}
  if(!w::raw(ref mut out,b"]}",ref mut meter).is_ok){return 4;}i+=1;
 }if(!w::raw(ref mut out,b"]",ref mut meter).is_ok){return 4;}return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`
}

func TestCandidateAtlasAggregateInstances(t *testing.T) {
	type declaration struct {
		PackageInstance string `json:"package_instance"`
		Module          string `json:"module"`
		Kind            string `json:"kind"`
		Name            string `json:"name"`
	}
	type fact struct {
		ID            string               `json:"id"`
		Declaration   declaration          `json:"declaration"`
		TypeArguments []atlasCandidateType `json:"type_arguments"`
		Type          *atlasCandidateType  `json:"type"`
		Cleanup       bool                 `json:"cleanup"`
		Calls         []string             `json:"calls"`
	}
	type fixture struct {
		name, root, changed string
		want                []fact
	}
	var fixtures []fixture
	cases := []candidateAggregateLayoutFixture{
		{name: "ordered tuples and imported resources", root: "module root;import maker as m;fn i32 main(){let a=m::Cell<i32>{.value=42};return a.value;}"},
		{name: "unused templates", root: "module root;import maker as m;fn i32 main(){return m::answer();}", maker: "module maker;import helper as h;pub struct Cell<T>{T value;}pub resource Token<T>{T value;cleanup{discard h::finish<T>(this.value);}}pub fn i32 answer(){return 42;}"},
		{name: "two scalar instances", root: "module root;import maker as m;fn i32 main(){let a=m::Cell<i32>{.value=42};let b=m::Cell<bool>{.value=true};assert(b.value);return a.value;}"},
	}
	for _, execution := range candidateAggregateExecutions()[3:] {
		cases = append(cases, execution.fixture)
	}
	for _, source := range cases {
		root := candidateAggregateCacheWorkspace(t, source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			candidateAtlasAggregateOracleFailure(t, root)
			t.Fatalf("aggregate instance oracle %s: %d %s", source.name, code, &diagnostic)
		}
		f := fixture{name: source.name, root: root, want: []fact{}}
		for _, part := range prepared.partitions {
			for _, s := range part.Specializations() {
				r := fact{ID: s.ID, Declaration: declaration{s.Declaration.PackageInstance, s.Declaration.Module, string(s.Declaration.Kind), s.Declaration.Name}, TypeArguments: []atlasCandidateType{}, Cleanup: s.Cleanup != nil, Calls: []string{}}
				for _, argument := range s.TypeArguments {
					r.TypeArguments = append(r.TypeArguments, candidateTypeFromGo(argument))
				}
				body := s.Function
				if s.Aggregate != nil {
					typ := candidateTypeFromGo(s.Aggregate.Type)
					r.Type = &typ
					if s.Cleanup != nil {
						body = *s.Cleanup
					}
				}
				for _, call := range body.Calls {
					r.Calls = append(r.Calls, call.Specialization)
				}
				f.want = append(f.want, r)
			}
		}
		// Instance rows are globally sorted by their original specialization ID.
		slices.SortFunc(f.want, func(a, b fact) int { return strings.Compare(a.ID, b.ID) })
		changed := source
		changed.root += "\nfn i32 changed(){return 1;}"
		f.changed = candidateAggregateCacheWorkspace(t, changed)
		fixtures = append(fixtures, f)
	}
	candidateAtlasProbeSize(t, "project_atlas_instances", candidateAtlasAggregateInstancesMain(), 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				code, out, stderr := fetchCLIProcess(t, image, f.root, nil, "")
				if code != 0 || stderr != "" {
					t.Fatalf("aggregate instances: %d %s %s", code, out, stderr)
				}
				var got []fact
				if err := json.Unmarshal([]byte(out), &got); err != nil || !reflect.DeepEqual(got, f.want) {
					t.Fatalf("aggregate facts differ: %v\ngot %s\nwant %+v", err, out, f.want)
				}
				for _, bad := range []struct{ mode, text string }{{"budget", "3"}, {"missing-coordinate", "7"}, {"ordinary", "7"}, {"stale", "7"}} {
					args := []string{bad.mode}
					if bad.mode == "stale" {
						args = append(args, f.changed)
					}
					code, out, stderr := fetchCLIProcess(t, image, f.root, args, "")
					if code != 7 || out != bad.text || stderr != "" {
						t.Fatalf("%s: %d %q %q", bad.mode, code, out, stderr)
					}
				}
			})
		}
	}, "project_atlas_schema", "project_workspace_lock", "project_coordinate_graph")
}

func candidateAtlasAggregateOracleFailure(t *testing.T, root string) {
	t.Helper()
	w, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := w.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := w.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	compiled, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !compiled.Verified {
		t.Fatalf("aggregate Go compilation: %v %+v", err, compiled.Semantic)
	}
	build, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := atlasmap.PrepareWorkspaceV3(w, compiled, build); err != nil {
		t.Fatalf("aggregate Go Atlas preparation: %v", err)
	}
	if _, err := atlasmap.CurrentWorkspaceExpectationsV3(w, compiled, build); err != nil {
		t.Fatalf("aggregate Go Atlas expectations: %v", err)
	}
}
