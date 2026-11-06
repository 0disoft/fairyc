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

const atlasGenericProductionUsedSource = `module main;import answer as a;
pub fn T wrap<T>(T value){return a::id<T>(value);}
pub fn U choose<T,U>(T first,U second){discard first;return second;}
pub fn T unused<T>(T value){return value;}
fn i32 main(){discard wrap<u8>(3);discard choose<u8,i32>(2,3);return choose<bool,i32>(true,wrap<i32>(a::other<i32>(4)));}`

func candidateAtlasGenericProductionWorkspace(t *testing.T, source string) string {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, source)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", `module answer;
pub fn T id<T>(T value){return value;}
pub fn T other<T>(T value){return value;}
pub fn T unused<T>(T value){return value;}
pub fn i32 value(){return 42;}`)
	return root
}

func candidateAtlasGenericProduceMain() string {
	main := candidateAtlasProduceMain()
	main = strings.Replace(main, "import project_atlas_produce as produce;", `import project_atlas_produce as produce;
import project_atlas_instances as instances;import project_atlas_current as current;
import project_atlas_join as join;`, 1)
	main = strings.Replace(main, " var packages=", `
 var u8[64] argument=[0;64];let usize length=if(process::arg_count()==2){switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{return 7;}}}else{0};let ref u8[] raw=ref argument;let mode=raw[..length];
 if(bytes::equal(mode,b"budget")){meter=16777216;}
 let absent=table::assemble(bytes::empty(),v::empty<table::Coordinate>());
 let observed=switch(if(bytes::equal(mode,b"missing-coordinate")){instances::collect(ref source,ref absent,compiler::checked_program(ref complete),ref metadata,ref mut meter)}else{instances::collect(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)}){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
 var packages=`, 1)
	main = strings.Replace(main, "produce::package(f,ref source,p,ref table,ref metadata,ref mut rows,owner,ref mut meter)", "produce::package_generics(f,ref source,p,ref table,ref metadata,ref mut rows,ref observed,owner,ref mut meter)", 1)
	main = strings.Replace(main, " var out=bytes::empty();", " var admitted=v::empty<current::Ready>();var out=bytes::empty();", 1)
	main = strings.Replace(main, "  if(owner>0", `
  let ready=switch(produce::ready(v::at<produce::Package>(ref packages,owner),total-external,external,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write("ready/");discard io::write_usize(code as usize);return 7;}};
  if(!v::push<current::Ready>(ref mut admitted,ready).is_ok){return 7;}
  if(owner>0`, 1)
	main = strings.Replace(main, ` if(!w::raw(ref mut out,b"]"`, `
 let joined=switch(join::join(admitted,8,128,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write("join/");discard io::write_usize(code as usize);return 7;}};
 if(join::package_count(ref joined)!=graph::node_count(graph)){return 7;}
 if(!w::raw(ref mut out,b"]"`, 1)
	return main
}

func TestCandidateAtlasGenericProduction(t *testing.T) {
	type fixture struct {
		name, root string
		partitions map[string]atlasmap.Partition
	}
	var fixtures []fixture
	for _, source := range []struct{ name, main string }{
		{"used", atlasGenericProductionUsedSource},
		{"unused", `module main;import answer as a;
pub fn T unused<T>(T value){return value;}
fn i32 main(){return a::value();}`},
	} {
		root := candidateAtlasGenericProductionWorkspace(t, source.main)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("generic production oracle %s: %d %s", source.name, code, &diagnostic)
		}
		f := fixture{name: source.name, root: root, partitions: map[string]atlasmap.Partition{}}
		specializations := 0
		for _, p := range prepared.partitions {
			encoded, err := atlasmap.EncodeV3(p, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			p, err = atlasmap.DecodeV3(encoded, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			f.partitions[p.Header().PackageInstance] = p
			specializations += len(p.Specializations())
		}
		if source.name == "used" && specializations != 7 || source.name == "unused" && specializations != 0 {
			t.Fatalf("oracle coverage %s: %d implementations", source.name, specializations)
		}
		fixtures = append(fixtures, f)
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateAtlasGenericProduceMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				code, out, stderr := fetchCLIProcess(t, image, f.root, nil, "")
				if code != 0 || stderr != "" {
					t.Fatalf("native generic production: %d %s %s", code, out, stderr)
				}
				var documents []json.RawMessage
				if err := json.Unmarshal([]byte(out), &documents); err != nil {
					t.Fatalf("generic documents: %v %s", err, out)
				}
				if len(documents) != len(f.partitions) {
					t.Fatal("generic partition count", len(documents))
				}
				seen := map[string]bool{}
				for _, raw := range documents {
					got, err := atlasmap.DecodeV3(raw, 4<<20)
					if err != nil {
						t.Fatalf("Go rejected produced generic map: %v", err)
					}
					h := got.Header()
					want, ok := f.partitions[h.PackageInstance]
					if !ok || seen[h.PackageInstance] {
						t.Fatal("unexpected or duplicate generic partition", h.PackageInstance)
					}
					seen[h.PackageInstance] = true
					if !reflect.DeepEqual(got.Declarations(), want.Declarations()) {
						t.Fatalf("original generic declarations differ %s: got %+v want %+v", h.PackageUID, got.Declarations(), want.Declarations())
					}
					if !reflect.DeepEqual(got.Symbols(), want.Symbols()) || h.InterfaceHash != want.Header().InterfaceHash || h.GenericInterfaceHash != want.Header().GenericInterfaceHash {
						t.Fatalf("generic identities differ %s: interfaces %s/%s want %s/%s; symbols got %+v want %+v", h.PackageUID, h.InterfaceHash, h.GenericInterfaceHash, want.Header().InterfaceHash, want.Header().GenericInterfaceHash, got.Symbols(), want.Symbols())
					}
					compareAtlasProducedFunctions(t, got.Functions(), want.Functions())
					actual, expected := got.Specializations(), want.Specializations()
					if len(actual) != len(expected) || h.Counts.Specializations != len(expected) {
						t.Fatal("generic implementation count differs", h.PackageUID, len(actual), len(expected))
					}
					for i, a := range actual {
						w := expected[i]
						if a.ID != w.ID || a.Declaration != w.Declaration || !reflect.DeepEqual(a.TypeArguments, w.TypeArguments) || a.Aggregate != nil || a.Cleanup != nil {
							t.Fatalf("generic implementation differs: got %+v want %+v", a, w)
						}
						compareAtlasProducedFunctions(t, []semanticsnapshot.FunctionSummary{a.Function}, []semanticsnapshot.FunctionSummary{w.Function})
					}
					for _, c := range got.Completeness() {
						if c.Relation == "generic-specializations" && (c.Status != "exact" || c.Unknown != 0) || c.Relation == "generic-template-bodies" && (c.Status != "unavailable" || c.Unknown == 0) {
							t.Fatalf("incorrect generic completeness: %+v", c)
						}
					}
				}
				for _, bad := range []struct{ mode, text string }{{"budget", "3"}, {"missing-coordinate", "7"}} {
					code, out, stderr := fetchCLIProcess(t, image, f.root, []string{bad.mode}, "")
					if code != 7 || out != bad.text || stderr != "" {
						t.Fatalf("%s: %d %q %q", bad.mode, code, out, stderr)
					}
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("generic production mutated workspace")
				}
			})
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}

func compareAtlasProducedFunctions(t *testing.T, got, want []semanticsnapshot.FunctionSummary) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("function count: got %d want %d", len(got), len(want))
	}
	for i, f := range got {
		w := want[i]
		if f.Function != w.Function || f.Provenance != w.Provenance || !slices.Equal(f.Calls, w.Calls) || !slices.Equal(f.DropSites, w.DropSites) || !slices.Equal(f.TrapCategories, w.TrapCategories) || !slices.Equal(f.UnsafeSites, w.UnsafeSites) || !slices.Equal(f.MoveSites, w.MoveSites) {
			t.Fatalf("produced calls/drops/traps/unsafe/moves differ: got %+v want %+v", f, w)
		}
	}
}

// Count only the currently collected families from independent Go source facts.
// In particular, do not use candidate map counts as the expected inventory.
func candidateAtlasProducedRelationCount(p atlasmap.Partition) int {
	count := 0
	add := func(f semanticsnapshot.FunctionSummary) {
		count += len(f.Calls) + len(f.DropSites) + len(f.TrapCategories) + len(f.UnsafeSites) + len(f.MoveSites)
	}
	for _, f := range p.Functions() {
		add(f)
	}
	for _, s := range p.Specializations() {
		if s.Aggregate == nil {
			add(s.Function)
		}
		if s.Cleanup != nil {
			add(*s.Cleanup)
		}
	}
	return count
}
