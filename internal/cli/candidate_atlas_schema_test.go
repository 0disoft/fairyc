package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const atlasSchemaExtra = `
struct Container{Pair[2] pairs;Option<Pair> maybe;ptr mut Pair raw;}
enum Payload{none;data{Pair pair;i32 count;};}
resource Empty{cleanup{}}
@effects(console) resource Loud{i32 value;cleanup{discard 1;}}
@effects(filesystem,console,clock) fn void permitted(){}
unsafe extern C fn c::int unused_external(c::int value);
@link_name("fairyc_atlas_other") unsafe extern C fn i32 renamed_external(i32 value);
`

func candidateAtlasSchemaMain() string {
	main := strings.SplitN(candidateAtlasTypesMain, " if(process::arg_count()==2)", 2)[0]
	main = strings.Replace(main, "import project_atlas_types as types;", "import project_atlas_types as types;import project_atlas_schema as schema;import project_atlas_declarations as declarations;import scanner as scanner;", 1)
	return main + `
 var u8[64] argument=[0;64];var usize argument_size=0;
 if(process::arg_count()==2){argument_size=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{return 7;}};}
 let ref u8[] argument_view=ref argument;let mode=argument_view[..argument_size];
 if(bytes::equal(mode,b"missing-coordinates")){
  let absent=table::assemble(bytes::empty(),v::empty<table::Coordinate>());
  return switch(schema::prepare(ref source,ref absent,ref complete,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};
 }
 if(bytes::equal(mode,b"stale-source")){
  var changed=bytes::empty();discard bytes::append(ref mut changed,syntax::source_view(input::source(ref source,0)));discard bytes::append(ref mut changed,b"\n");
  let tokens=switch(scanner::collect(changed,false,8388608,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return 9;}};
  let parsed=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{return 9;}};
  let overlay=switch(input::with_source(ref source,0,ref parsed,ref mut meter)){case .ok(p):{p}case .err(_):{return 9;}};
  return switch(schema::prepare(ref overlay,ref table,ref complete,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};
 }
 if(bytes::equal(mode,b"budget")){meter=16777216;}
 let metadata=switch(schema::prepare(ref source,ref table,ref complete,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
 var rows=switch(schema::type_rows(ref metadata,ref mut meter)){case .ok(p):{p}case .err(code):{return 7;}};
 if(argument_size!=0){
  let row=if(bytes::equal(mode,b"past-module")){declarations::Row{.module_index=input::count(ref source),.declaration=0}}else if(bytes::equal(mode,b"past-declaration")){declarations::Row{.module_index=0,.declaration=syntax::declaration_count(input::source(ref source,0))}}else{declarations::Row{.module_index=0,.declaration=0}};
  return switch(schema::declaration(ref source,ref table,ref metadata,ref mut rows,row,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};
 }
 var out=bytes::empty();if(!w::raw(ref mut out,b"[",ref mut meter).is_ok){return 7;}var usize module_index=0;var usize written=0;
 while(module_index<input::count(ref source)){
  let p=input::source(ref source,module_index);var usize at=0;
  while(at<syntax::declaration_count(p)){let d=syntax::declaration_at(p,at);let row=declarations::Row{.module_index,.declaration=at};at+=1;
   if(d.kind==1 || d.kind==2){continue;}let declaration=switch(schema::declaration(ref source,ref table,ref metadata,ref mut rows,row,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
   if(written!=0 && !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::raw(ref mut out,bytes::view(ref declaration),ref mut meter).is_ok){return 7;}written+=1;
  }module_index+=1;
 }
 if(!w::raw(ref mut out,b"]",ref mut meter).is_ok){return 7;}return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`
}

func atlasDeclarationRows(t *testing.T, data []byte) map[string]string {
	t.Helper()
	var rows []json.RawMessage
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("declaration array: %v: %s", err, data)
	}
	result := map[string]string{}
	for _, row := range rows {
		var d struct {
			Key struct {
				Instance string `json:"package_instance"`
				Module   string `json:"module"`
				Kind     string `json:"kind"`
				Name     string `json:"name"`
			} `json:"key"`
		}
		if err := json.Unmarshal(row, &d); err != nil {
			t.Fatal(err)
		}
		key := d.Key.Instance + "/" + d.Key.Module + "/" + d.Key.Kind + "/" + d.Key.Name
		if _, exists := result[key]; exists {
			t.Fatal("duplicate declaration", key)
		}
		result[key] = string(row)
	}
	return result
}

func TestCandidateAtlasSchema(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, atlasTypeSource+atlasSchemaExtra)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", "module answer;pub struct Pair{i32 value;}pub fn i32 value(){return 42;}")
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("Go declaration oracle: %d %s", code, &diagnostic)
	}
	want := map[string]string{}
	for _, p := range prepared.partitions {
		data, err := atlasmap.EncodeV3(p, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		var envelope struct {
			Map struct {
				Declarations json.RawMessage `json:"declarations"`
			} `json:"map"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			t.Fatal(err)
		}
		for key, value := range atlasDeclarationRows(t, envelope.Map.Declarations) {
			if _, exists := want[key]; exists {
				t.Fatal("duplicate oracle declaration", key)
			}
			want[key] = value
		}
	}
	if len(want) != 27 {
		t.Fatalf("oracle declarations=%d", len(want))
	}
	candidateAtlasProbeSize(t, "project_atlas_schema", candidateAtlasSchemaMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		before := remoteTree(t, root)
		code, out, stderr := fetchCLIProcess(t, image, root, nil, "")
		if code != 0 || stderr != "" {
			t.Fatalf("source schema: %d %s %s", code, out, stderr)
		}
		got := atlasDeclarationRows(t, []byte(out))
		if len(got) != len(want) {
			t.Fatalf("declaration counts: got %d, want %d", len(got), len(want))
		}
		for key, value := range want {
			if got[key] != value {
				t.Fatalf("%s differs\ngot  %s\nwant %s", key, got[key], value)
			}
		}
		for _, bad := range []struct{ mode, text string }{{"missing-coordinates", "7"}, {"stale-source", "7"}, {"budget", "3"}, {"past-module", "7"}, {"past-declaration", "7"}, {"unsupported-header", "101"}} {
			code, out, stderr := fetchCLIProcess(t, image, root, []string{bad.mode}, "")
			if code != 7 || out != bad.text || stderr != "" {
				t.Fatalf("%s: %d %q %q", bad.mode, code, out, stderr)
			}
		}
		if !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatal("schema projection changed workspace")
		}
		if strings.Contains(out, root) {
			t.Fatal("schema leaked host path")
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}
