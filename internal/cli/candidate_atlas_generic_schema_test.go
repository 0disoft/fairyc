package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const atlasGenericSchemaSource = `module main;
import answer as a;
struct Pair{i32 value;}
struct Cell<T>{T value;}
struct Nest<T>{Cell<T> nested;a::Box<T> imported;Pair own;a::Pair remote;}
struct Duplex<T,U>{T left;U right;}
resource Empty<T>{T value;cleanup{}}
resource Token<T>{Cell<T> payload;cleanup{let T observed=finish<T>(this.payload.value);}}
pub fn T finish<T>(T value){return value;}
pub fn U choose<T,U>(T first,U second){discard first;return second;}
@effects(console,filesystem) unsafe fn void shapes<T>(ref T one,ref mut T[] many,ptr T raw,T[0x2] pair,Option<T> maybe,Result<T,i32> result,Cell<T> cell,a::Box<T> box,Pair local,a::Pair remote,ref u8[] bytes){}
unsafe fn void lengths<T>(T[4_097] larger,T[18446744073709551615] maximum){}
fn i32 main(){let c=Cell<i32>{.value=2};let token=Token<i32>{.payload=c};return choose<bool,i32>(true,finish<i32>(3));}
`

// The source-only aggregate projection cannot claim Complete: the current Fairy
// compiler rejects user aggregate templates before body verification. Keep that
// remaining runtime work separate from this original-declaration port.
func candidateAtlasGenericSchemaMain() string {
	main := candidateAtlasSchemaMain()
	start := strings.Index(main, " var usize source_work=0;")
	end := strings.Index(main, " var u8[64] argument=")
	main = main[:start] + main[end:]
	start = strings.Index(main, ` if(bytes::equal(mode,b"missing-coordinates"))`)
	end = strings.Index(main, ` if(bytes::equal(mode,b"budget"))`)
	main = main[:start] + main[end:]
	return strings.Replace(main, "schema::prepare(ref source,ref table,ref complete,ref mut meter)", "schema::prepare_declarations(ref source,ref mut meter)", 1)
}

func TestCandidateAtlasGenericSchema(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, atlasGenericSchemaSource)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", `module answer;pub struct Pair{i32 value;}pub struct Box<T>{T value;}pub fn i32 value(){return 42;}`)
	candidateAtlasGenericSchemaOracle(t, root, candidateAtlasGenericSchemaMain(), 10, false)
}

func TestCandidateAtlasGenericFunctionSchema(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, `module main;import answer as a;
 struct Pair{i32 value;}
 pub fn T identity<T>(T value){return value;}
 pub fn U choose<T,U>(T first,U second){discard first;return second;}
 @effects(console,filesystem) unsafe fn void shapes<T>(ref T one,ref mut T[] many,ptr T raw,T[0x2] pair,Option<T> maybe,Result<T,i32> result,Pair local,a::Pair remote,ref u8[] bytes){}
 unsafe fn void lengths<T>(T[4_097] larger,T[18446744073709551615] maximum){}
 fn i32 main(){return choose<bool,i32>(true,identity<i32>(a::value()));}`)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", `module answer;pub struct Pair{i32 value;}pub fn i32 value(){return 42;}`)
	candidateAtlasGenericSchemaOracle(t, root, candidateAtlasSchemaMain(), 4, true)
}

func candidateAtlasGenericSchemaOracle(t *testing.T, root, main string, templateCount int, complete bool) {
	t.Helper()
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("generic source declaration oracle: %d %s", code, &diagnostic)
	}
	want := map[string]string{}
	templates := 0
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
		for key, row := range atlasDeclarationRows(t, envelope.Map.Declarations) {
			want[key] = row
			if bytes.Contains([]byte(row), []byte(`"type_parameters":`)) {
				templates++
			}
		}
	}
	if templates != templateCount {
		t.Fatalf("original generic headers: got %d want %d", templates, templateCount)
	}
	candidateAtlasProbeSize(t, "project_atlas_schema", main, 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		before := remoteTree(t, root)
		code, out, stderr := fetchCLIProcess(t, image, root, nil, "")
		if code != 0 || stderr != "" {
			t.Fatalf("native source declaration projection: %d %s %s", code, out, stderr)
		}
		got := atlasDeclarationRows(t, []byte(out))
		if len(got) != len(want) {
			t.Fatalf("source header counts: got %d want %d", len(got), len(want))
		}
		for key, row := range want {
			if got[key] != row {
				t.Fatalf("source header %s differs\ngot %s\nwant %s", key, got[key], row)
			}
		}
		for _, bad := range []struct{ mode, text string }{{"budget", "3"}, {"past-module", "7"}, {"past-declaration", "7"}, {"unsupported-header", "101"}} {
			code, out, stderr := fetchCLIProcess(t, image, root, []string{bad.mode}, "")
			if code != 7 || out != bad.text || stderr != "" {
				t.Fatalf("%s: %d %q %q", bad.mode, code, out, stderr)
			}
		}
		if complete {
			for _, bad := range []string{"missing-coordinates", "stale-source"} {
				code, out, stderr := fetchCLIProcess(t, image, root, []string{bad}, "")
				if code != 7 || out != "7" || stderr != "" {
					t.Fatalf("%s: %d %q %q", bad, code, out, stderr)
				}
			}
		}
		if !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatal("generic header projection changed workspace")
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}
