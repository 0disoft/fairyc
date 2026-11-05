package cli

import (
	"reflect"
	"testing"
)

const candidateAtlasSDKVectorAuthorityMain = `module main;
import project_atlas_sdk_vector as vector;import project_atlas_declarations as declarations;
import project_workspace_lock as loader;import project_source_set as sources;
import project_module_visibility as visibility;import project_compiler_input as input;
import project_edit_body as editing;import syntax as syntax;
import std::bytes as bytes;import std::vec as v;import std::io as io;import std::process as process;
@effects(console) fn i32 failure(u32 code){discard io::write_usize(code as usize);return 7;}
@effects(filesystem,console) fn i32 main(){
 var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
 let loaded=switch(loader::load(b".",true,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{return failure(20);}};
 let source=switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(p):{p}case .err(_):{return failure(21);}};
 var u8[64] raw=[0;64];let usize n=if(process::arg_count()==2){switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return failure(22);}}}else{0};let ref u8[] mode=ref raw;
 var usize selected=input::count(ref source);var usize i=0;
 while(i<input::count(ref source)){let admitted=switch(vector::source(ref source,i,ref mut meter)){case .ok(p):{p}case .err(code):{return failure(code);}};if(admitted){if(selected!=input::count(ref source)){return failure(23);}selected=i;}i+=1;}
 if(selected==input::count(ref source)){return failure(24);}
 if(bytes::equal(mode[..n],b"leaf") || bytes::equal(mode[..n],b"leaf-budget")){
  if(bytes::equal(mode[..n],b"leaf-budget")){meter=16777216;}
  let leaf=switch(vector::take_leaf(ref source,selected,ref mut meter)){case .ok(p):{p}case .err(code):{return failure(code);}};
  let d=syntax::declaration_at(input::source(ref source,selected),leaf-1);let valid=d.kind==3 && !d.public && bytes::equal(syntax::declaration_name(input::source(ref source,selected),leaf),b"take_slot");discard io::write_usize(if(valid){1}else{0});return if(valid){0}else{7};
 }
 if(bytes::equal(mode[..n],b"budget")){meter=16777216;return switch(vector::source(ref source,selected,ref mut meter)){case .ok(_):{0}case .err(code):{failure(code)}};}
 if(bytes::equal(mode[..n],b"past")){return switch(vector::source(ref source,input::count(ref source),ref mut meter)){case .ok(_):{0}case .err(code):{failure(code)}};}
 let owner=input::origin(ref source,selected).owner;
 if(bytes::equal(mode[..n],b"ordinary")){return switch(declarations::collect_generics(ref source,owner,ref mut meter)){case .ok(_):{0}case .err(code):{failure(code)}};}
 if(bytes::equal(mode[..n],b"changed") || bytes::equal(mode[..n],b"changed-leaf")){
  var changed=switch(bytes::copy(syntax::source_view(input::source(ref source,selected)))){case .ok(p):{p}case .err(_):{return failure(25);}};
  if(!bytes::append(ref mut changed,b"\n").is_ok){return failure(26);}
  let parsed=switch(editing::parse(changed)){case .ok(p):{p}case .err(_):{return failure(27);}};
  let overlay=switch(input::with_source(ref source,selected,ref parsed,ref mut meter)){case .ok(p):{p}case .err(_):{return failure(28);}};
  let admitted=switch(vector::source(ref overlay,selected,ref mut meter)){case .ok(p):{p}case .err(code):{return failure(code);}};
  if(admitted){return failure(29);}
  if(bytes::equal(mode[..n],b"changed-leaf")){return switch(vector::take_leaf(ref overlay,selected,ref mut meter)){case .ok(_):{0}case .err(code):{failure(code)}};}
  return switch(declarations::collect_sdk_vectors(ref overlay,owner,ref mut meter)){case .ok(_):{0}case .err(code):{failure(code)}};
 }
 let rows=switch(declarations::collect_sdk_vectors(ref source,owner,ref mut meter)){case .ok(p):{p}case .err(code):{return failure(code);}};
 var usize found=0;i=0;while(i<v::len<declarations::Row>(ref rows)){let row=*v::at<declarations::Row>(ref rows,i);let p=input::source(ref source,row.module_index);let d=syntax::declaration_at(p,row.declaration);
  if(d.kind==6 && d.generic_parameters!=0){let admitted=switch(vector::declaration(ref source,row.module_index,row.declaration+1,ref mut meter)){case .ok(p):{p}case .err(code):{return failure(code);}};if(!admitted){return failure(30);}found+=1;}i+=1;}
 discard io::write_usize(found);return if(found==1){0}else{7};
}`

func TestCandidateAtlasSDKVectorAuthority(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, `module main;import std::vec as v;fn i32 main(){return 0;}`)
	if code, _, diagnostic := runForTest([]string{"check"}, root); code != 0 {
		t.Fatalf("vector authority fixture: %d %s", code, diagnostic)
	}
	candidateAtlasProbeSize(t, "project_atlas_sdk_vector", candidateAtlasSDKVectorAuthorityMain, 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		before := remoteTree(t, root)
		for _, check := range []struct {
			mode string
			code int
			out  string
		}{{"", 0, "1"}, {"leaf", 0, "1"}, {"leaf-budget", 7, "3"}, {"ordinary", 7, "101"}, {"changed", 7, "101"}, {"changed-leaf", 7, "7"}, {"budget", 7, "3"}, {"past", 7, "7"}} {
			var arguments []string
			if check.mode != "" {
				arguments = []string{check.mode}
			}
			code, out, diagnostic := fetchCLIProcess(t, image, root, arguments, "")
			if code != check.code || out != check.out || diagnostic != "" {
				t.Fatalf("vector authority %q: %d %q %q", check.mode, code, out, diagnostic)
			}
		}
		if !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatal("vector authority checks changed workspace")
		}
	}, "project_workspace_lock", "project_compiler_input", "project_edit_body", "project_atlas_declarations")
}
