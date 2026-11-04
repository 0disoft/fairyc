package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const atlasTypeSource = `module main;
import answer as a;
@repr(C) struct Pair{i32 value;}
resource Token{Pair value;cleanup{}}
enum State{ready;done;}
fn void scalars(i32 a,i64 b,u8 c,u32 d,usize e,bool f){}
fn void text(ref str a,ref CStr b){}
fn void views(ref i32[] a,ref mut u32[] b,ref u8[] c,ref mut u8[] d){}
fn void nominals(Pair a,Token b,State c){}
fn void borrows(ref Pair a,ref mut Pair b){}
fn void arrays(Pair[2] a,u8[4] b){}
fn void tags(Option<Pair> a,Result<i32,Token> b){discard a;discard b;}
fn void nominal_views(ref Pair[] a,ref mut Pair[] b){}
unsafe fn void pointers(ptr Pair a,ptr mut Pair b,ptr void c,ptr mut u8 d,ptr mut ptr mut void e){}
fn void imported(a::Pair p){}
fn void c1(c::bool a,c::char b,c::double c,c::float d){}
fn void c2(c::int a,c::long b,c::longlong c,c::schar d){}
fn void c3(c::short a,c::size_t b,c::ssize_t c,c::uchar d){}
fn void c4(c::uint a,c::ulong b,c::ulonglong c,c::ushort d){}
fn i32 main(){return a::value();}
`

const candidateAtlasTypesMain = `module main;
import project_atlas_types as types;import project_workspace_lock as loader;
import project_source_set as sources;import project_module_visibility as visibility;
import project_coordinate_graph as coordinates;import module_public_type_text as table;
import project_compiler_input as input;import project_compiler_core as compiler;
import module_project_identified_ir as identified;import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;import project_atlas_write as w;
import syntax as syntax;import std::bytes as bytes;import std::vec as v;
import std::io as io;import std::process as process;
@effects(filesystem,console) fn i32 main(){
 var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
 let loaded=switch(loader::load(b".",true,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 let resolved=loader::workspace(ref loaded);let graph=visibility::workspace_graph(resolved);
 let table=switch(coordinates::collect(graph,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 let source=switch(input::project(resolved,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 var usize source_work=0;var usize link_work=0;
 let complete=switch(compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(p):{discard io::write_usize(p.stage as usize);discard io::write("/");discard io::write_usize(p.code as usize);return 7;}};
 let p=identified::ir_program(compiler::checked_program(ref complete));let clean=ir::cleanup_ir(p);
 if(process::arg_count()==2){
  var u8[64] raw=[0;64];let n=switch(process::arg(1,ref mut raw)){case .ok(n):{n}case .err(_):{return 7;}};let ref u8[] mode=ref raw;
  var usize id=1;if(bytes::equal(mode[..n],b"zero")){id=0;}else if(bytes::equal(mode[..n],b"past")){id=clean::type_count(clean)+1;}else if(bytes::equal(mode[..n],b"budget")){meter=16777216;}
  let absent=table::assemble(bytes::empty(),v::empty<table::Coordinate>());
  if(bytes::equal(mode[..n],b"missing-coordinate")){id=1;while(id<=clean::type_count(clean) && clean::type_at(clean,id).kind!=2){id+=1;}}
  let result=if(bytes::equal(mode[..n],b"missing-coordinate")){types::render(ref source,ref absent,clean,id,ref mut meter)}else{types::render(ref source,ref table,clean,id,ref mut meter)};
  return switch(result){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};
 }
 var out=bytes::empty();if(!w::raw(ref mut out,b"[",ref mut meter).is_ok){return 7;}var usize at=1;var usize written=0;
 while(at<=ir::count(p)){
  let c=clean::frame_at(clean,at);let body=ir::at(p,at);if(c.kind!=1 && c.kind!=3){at+=1;continue;}
  if(written!=0 && !w::raw(ref mut out,b",",ref mut meter).is_ok){return 7;}written+=1;
  if(!w::raw(ref mut out,b"{",ref mut meter).is_ok || !w::pair(ref mut out,b"file",input::file(ref source,c.module_index),ref mut meter).is_ok || !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::pair(ref mut out,b"name",syntax::declaration_name(input::source(ref source,c.module_index),c.declaration),ref mut meter).is_ok || !w::raw(ref mut out,b",\"type\":",ref mut meter).is_ok){return 7;}
  let result=switch(types::render(ref source,ref table,clean,c.result,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
  if(!w::raw(ref mut out,bytes::view(ref result),ref mut meter).is_ok || !w::raw(ref mut out,b",\"parameters\":[",ref mut meter).is_ok){return 7;}
  var usize arg=1;var usize parameters=0;while(arg<=ir::local_count(body)){let local=ir::local_at(body,arg);arg+=1;if(local.kind!=1){continue;}
   let parameter=switch(types::render(ref source,ref table,clean,local.type,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
   if(parameters!=0 && !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::raw(ref mut out,bytes::view(ref parameter),ref mut meter).is_ok){return 7;}parameters+=1;}
  if(!w::raw(ref mut out,b"]}",ref mut meter).is_ok){return 7;}at+=1;
 }
 if(!w::raw(ref mut out,b"]",ref mut meter).is_ok){return 7;}return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`

type atlasCandidateType struct {
	Kind       string               `json:"kind"`
	Name       string               `json:"name"`
	Coordinate string               `json:"package_coordinate"`
	Mutable    bool                 `json:"mutable"`
	Length     uint64               `json:"length"`
	Arguments  []atlasCandidateType `json:"arguments"`
}

func candidateTypeFromGo(t semanticsnapshot.TypeRef) atlasCandidateType {
	args := make([]atlasCandidateType, len(t.Arguments))
	for i, a := range t.Arguments {
		args[i] = candidateTypeFromGo(a)
	}
	return atlasCandidateType{string(t.Kind), t.Name, t.PackageCoordinate, t.Mutable, t.Length, args}
}

type atlasCandidateCallable struct {
	File       string               `json:"file"`
	Name       string               `json:"name"`
	Type       atlasCandidateType   `json:"type"`
	Parameters []atlasCandidateType `json:"parameters"`
}

func TestCandidateAtlasTypes(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, atlasTypeSource)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", "module answer;pub struct Pair{i32 value;}pub fn i32 value(){return 42;}")
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("Go type oracle: %d %s", code, &diagnostic)
	}
	want := map[string]atlasCandidateCallable{}
	for _, p := range prepared.partitions {
		for _, d := range p.Declarations() {
			if d.Key.Kind != semanticsnapshot.DeclarationFunction {
				continue
			}
			item := atlasCandidateCallable{File: d.Span.Source, Name: d.Key.Name, Type: candidateTypeFromGo(d.Type), Parameters: make([]atlasCandidateType, len(d.Parameters))}
			for i, a := range d.Parameters {
				item.Parameters[i] = candidateTypeFromGo(a.Type)
			}
			want[item.File+"::"+item.Name] = item
		}
	}
	if len(want) != 16 {
		t.Fatalf("oracle callable count=%d", len(want))
	}
	candidateAtlasProbeSize(t, "project_atlas_types", candidateAtlasTypesMain, 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		before := remoteTree(t, root)
		code, out, stderr := fetchCLIProcess(t, image, root, nil, "")
		var rows []atlasCandidateCallable
		if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &rows) != nil {
			t.Fatalf("native type projection: %d %s %s", code, out, stderr)
		}
		got := map[string]atlasCandidateCallable{}
		for _, row := range rows {
			key := row.File + "::" + row.Name
			if _, exists := got[key]; exists {
				t.Fatal("duplicate callable", key)
			}
			got[key] = row
		}
		if !reflect.DeepEqual(got, want) {
			data, _ := json.MarshalIndent(got, "", "  ")
			golden, _ := json.MarshalIndent(want, "", "  ")
			t.Fatalf("type facts differ\ngot %s\nwant %s", data, golden)
		}
		for _, bad := range []struct{ mode, text string }{{"zero", "7"}, {"past", "7"}, {"budget", "3"}, {"missing-coordinate", "7"}} {
			code, out, stderr := fetchCLIProcess(t, image, root, []string{bad.mode}, "")
			if code != 7 || out != bad.text || stderr != "" {
				t.Fatalf("%s: %d %q %q", bad.mode, code, out, stderr)
			}
		}
		if !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatal("type facts changed workspace")
		}
		for _, row := range rows {
			data, _ := json.Marshal(row)
			if strings.Contains(string(data), root) {
				t.Fatal(fmt.Sprintf("host path leaked in %s", row.Name))
			}
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}
