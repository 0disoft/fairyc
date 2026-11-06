package cli

import "testing"

// Use the actual graph Complete constructor: checked_program intentionally
// asserts for this mode, so each ordinary consumer must reject it first.
const stateGraphCompleteGuardsProbe = `module main;
import project_compiler_core as core;
import project_compiler_state_graph as graph_complete;
import module_project_c_owner as c_owner;
import module_project_complete_c as complete_c;
import project_cache_specializations as cache;
import project_cache_workspace as cache_workspace;
import module_project_interfaces as interfaces;
import project_atlas_schema as atlas;
import project_atlas_source_version as version;
import module_unit_interpreter as interpreter;
import project_test_options as options;
import project_workspace_lock as loader;
import project_source_set as sources;
import project_module_visibility as visibility;
import project_coordinate_graph as coordinates;
import module_public_type_text as table;
import project_compiler_input as input;
import module_signatures as signatures;
import std::vec as v;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

@effects(console) fn core::Complete built(ref table::Coordinates coords){
 let actual=b"module ledger;@contract_uid(\"fixture.complete-guards\")state{cell bool flag;}@effects(storage_write)fn void transaction(){storage::commit();}";
 var usize work=0;
 return switch(core::build_complete_state_graph(actual,coords,0,ref mut work)){
  case .ok(p):{assert(core::is_state_graph(ref p));assert(graph_complete::owner(core::checked_state_graph(ref p))==0);assert(interfaces::count(core::public_interfaces(ref p))==1);assert(work>0 && work<=16777216);p}
  case .err(e):{discard io::write_stderr("graph Complete construction: ");discard io::write_stderr_u32(e.phase);discard io::write_stderr("/");discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);assert(false);built(coords)}
 };
}
@effects(console) fn void observed(ref str name,u32 code){discard io::write(name);discard io::write("=");discard io::write_u32(code);discard io::write("\n");}
@effects(filesystem,console) fn i32 main(){
 var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
 let loaded=switch(loader::load(b".",true,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{return 10;}};
 let resolved=loader::workspace(ref loaded);
 let coords=switch(coordinates::collect(visibility::workspace_graph(resolved),ref mut meter)){case .ok(p):{p}case .err(_):{return 11;}};
 let source=switch(input::project(resolved,ref mut meter)){case .ok(p):{p}case .err(_):{return 12;}};
 let complete=built(ref coords);
 if(process::arg_count()==2){
  let argv=v::empty<bytes::Buffer>();let selected=switch(options::parse_test_options(ref argv)){case .ok(p):{p}case .err(_):{return 13;}};
  let tests=v::empty<options::TestIdentity>();let chosen=v::empty<usize>();
  return interpreter::tests(ref complete,ref selected,ref tests,ref chosen,ref source,0);
 }
 // Every consuming call receives a fresh checked owner; a prior refusal cannot
 // leave the next gate testing a moved or fabricated Complete.
 observed("c-owner",switch(c_owner::complete(built(ref coords))){case .ok(_):{0}case .err(e):{e.code}});
 observed("partition",switch(complete_c::partition(built(ref coords),1,1)){case .ok(_):{0}case .err(e):{e.code}});
 observed("partition-entry",switch(complete_c::partition_entry(built(ref coords),1,1)){case .ok(_):{0}case .err(e):{e.code}});
 observed("partition-test",switch(complete_c::partition_test(built(ref coords),1,1)){case .ok(_):{0}case .err(e):{e.code}});
 observed("emit-test",switch(complete_c::emit_test(ref complete,1,1)){case .ok(_):{0}case .err(e):{e.code}});
 observed("prepare-tests",switch(complete_c::prepare_tests(built(ref coords))){case .ok(_):{0}case .err(e):{e.code}});
 var usize work=0;
 observed("cache-workspace",switch(cache_workspace::collect(resolved,ref complete,ref mut work)){case .ok(_):{0}case .err(code):{code}});
 observed("cache",switch(cache::collect(ref complete,0,ref mut work)){case .ok(_):{0}case .err(code):{code}});
 observed("cache-aggregates",switch(cache::collect_aggregates(ref complete,0,ref mut work)){case .ok(_):{0}case .err(code):{code}});
 assert(work==0);
 observed("atlas",switch(atlas::prepare(ref source,ref coords,ref complete,ref mut work)){case .ok(_):{0}case .err(code):{code}});
 observed("atlas-aggregates",switch(atlas::prepare_aggregates(ref source,ref coords,ref complete,ref mut work)){case .ok(_):{0}case .err(code):{code}});
 let metadata=switch(signatures::build(input::sources(ref source))){case .ok(p):{p}case .err(_):{return 14;}};
 observed("atlas-version",switch(version::admit(ref source,ref coords,ref complete,ref metadata,3,ref mut work)){case .ok(_):{0}case .err(code):{code}});
 assert(work==0);return 0;
}`

func TestCandidateStateGraphCompleteGuards(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, "module main;fn i32 main(){return 0;}")
	if code, _, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess {
		t.Fatalf("prepare locked input fixture: exit=%d diagnostics=%s", code, diagnostic)
	}
	const want = "c-owner=1001\npartition=1001\npartition-entry=1001\npartition-test=1001\nemit-test=1001\nprepare-tests=1001\ncache-workspace=9\ncache=9\ncache-aggregates=9\natlas=7\natlas-aggregates=7\natlas-version=7\n"
	candidateAtlasProbeSize(t, "project_compiler_core", stateGraphCompleteGuardsProbe, 512<<10, func(t *testing.T, image string) {
		t.Run("ordinary-consumers", func(t *testing.T) {
			code, out, stderr := fetchCLIProcess(t, image, root, nil, "")
			if code != 0 || out != want || stderr != "" {
				t.Fatalf("graph Complete guards: exit=%d stdout=%q stderr=%q, want %q", code, out, stderr, want)
			}
		})
		t.Run("interpreter", func(t *testing.T) {
			code, out, stderr := fetchCLIProcess(t, image, root, []string{"interpreter"}, "")
			if code != 2 || out != "" || stderr != "fairyc: state graph semantic execution is not connected\n" {
				t.Fatalf("graph Complete interpreter refusal: exit=%d stdout=%q stderr=%q", code, out, stderr)
			}
		})
	}, "module_project_complete_c", "project_cache_specializations", "project_cache_workspace", "project_atlas_schema", "project_atlas_source_version", "module_unit_interpreter", "project_workspace_lock", "project_coordinate_graph")
}
