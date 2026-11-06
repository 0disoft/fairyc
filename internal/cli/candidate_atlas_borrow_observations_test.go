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

func candidateAtlasBorrowObservationsMain() string {
	main := strings.SplitN(candidateAtlasTypesMain, " var usize source_work=0;", 2)[0]
	main = strings.Replace(main, "import project_atlas_types as types;", `import project_atlas_types as types;
import module_borrow_observations as observations;import project_atlas_declarations as declarations;
import module_artifact_borrow_observations as observation_work;
import module_unit_borrow_sites as sites;import project_compiler_source_mode as source_mode;
import module_project_artifact_encode as encoder;import module_project_artifact_decode as decoder;
import candidate_artifact_wire as payload;import module_program_artifact_encode as unit_encoder;
import module_unit_artifact_admission as unit_admit;import module_identified_ir as unit;
import module_project_identified_compile as units;import module_signatures as signatures;
import project_source_plan as source_plan;import generic_aggregate_types as aggregates;
import generic_source_plan as planned;import module_catalog as modules;`, 1)
	main = strings.Replace(main, "fn i32 main(){", "fn Result<i32,u32> execute(){", 1)
	main = strings.ReplaceAll(main, "return 7;", "return .ok(7);")
	return main + `
 let aggregate=switch(source_mode::aggregates(ref source,ref mut meter)){case .ok(p):{p}case .err(_):{return .ok(7);}};
 var usize source_work=0;var usize link_work=0;
 let complete=switch(if(aggregate){compiler::build_complete_aggregate_sources(input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut source_work,ref mut link_work)}else{compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)}){case .ok(p):{p}case .err(e):{discard io::write("fresh/");discard io::write_usize(e.code as usize);return .ok(7);}};
 discard verify_observation_controls(identified::borrows(compiler::checked_program(ref complete)))?;
 let unit_frames=verify_unit_sources(ref source,ref table,aggregate)?;if(process::arg_count()==2 && unit_frames<=identified::frame_count(compiler::checked_program(ref complete))){discard io::write("aliases/");discard io::write_usize(unit_frames);discard io::write("/");discard io::write_usize(identified::frame_count(compiler::checked_program(ref complete)));return .ok(7);}
 var usize encoding=0;let encoded=encoder::encode(compiler::checked_program(ref complete),ref mut encoding)?;
 var usize decoding=0;let saved=if(aggregate){decoder::decode_aggregates(payload::content(ref encoded),ref mut decoding)?}else{decoder::decode(payload::content(ref encoded),ref mut decoding)?};
 source_work=0;var usize artifact_work=0;link_work=0;
 let replay=switch(if(aggregate){compiler::restore_complete_aggregate_sources(saved,input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut source_work,ref mut artifact_work,ref mut link_work)}else{compiler::restore_complete_sources(saved,input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut source_work,ref mut artifact_work,ref mut link_work)}){case .ok(p):{p}case .err(e):{discard io::write("restore/");discard io::write_usize(e.code as usize);return .ok(7);}};
 var out=bytes::empty();discard w::raw(ref mut out,b"[",ref mut meter)?;
 discard render(compiler::checked_program(ref complete),ref source,ref mut out,ref mut meter)?;
 discard w::raw(ref mut out,b",",ref mut meter)?;discard render(compiler::checked_program(ref replay),ref source,ref mut out,ref mut meter)?;
 discard w::raw(ref mut out,b"]",ref mut meter)?;return .ok(if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4});
}
fn Result<bool,u32> render(ref identified::Program p,ref input::Input source,ref mut bytes::Buffer out,ref mut usize work){
 let rows=identified::borrows(p);let code=identified::ir_program(p);let cleaned=ir::cleanup_ir(code);
 if(!observations::ready(rows) || observations::count(rows)!=ir::count(code)){return .err(9);}discard w::raw(out,b"[",work)?;var usize frame=1;var usize written=0;
 while(frame<=ir::count(code)){let f=clean::frame_at(cleaned,frame);let spec=identified::identity(p,identified::frame_at(p,frame));
  if(f.kind==2 && spec.len==0){frame+=1;continue;}if(written>0){discard w::raw(out,b",",work)?;}written+=1;
  discard w::raw(out,b"{\"key\":",work)?;discard declarations::key(out,source,declarations::Row{.module_index=f.module_index,.declaration=f.declaration-1},work)?;
  discard w::raw(out,b",",work)?;discard w::pair(out,b"specialization",spec,work)?;discard w::raw(out,if(f.kind==2){b",\"cleanup\":true,\"sites\":["}else{b",\"cleanup\":false,\"sites\":["},work)?;
  var usize i=0;while(i<observations::site_count(rows,frame)){let s=observations::at(rows,frame,i);if(i>0){discard w::raw(out,b",",work)?;}
   discard w::raw(out,b"{\"span\":",work)?;discard declarations::span(out,source,f.module_index,s.start,s.end,work)?;discard w::raw(out,b",",work)?;
   discard w::pair(out,b"kind",if(s.mutable){b"mutable"}else{b"shared"},work)?;discard w::raw(out,if(s.escapes){b",\"escapes\":true}"}else{b",\"escapes\":false}"},work)?;i+=1;
  }discard w::raw(out,b"]}",work)?;frame+=1;
 }return w::raw(out,b"]",work);
}
fn Result<bool,u32> same_unit(ref unit::Program a,ref unit::Program b){
 if(!unit::has_borrows(a) || !unit::has_borrows(b) || unit::frame_count(a)!=unit::frame_count(b)){return .err(9);}let x=unit::borrows(a);let y=unit::borrows(b);var usize frame=1;
 while(frame<=unit::frame_count(a)){if(!bytes::equal(unit::identity(a,unit::frame_at(a,frame)),unit::identity(b,unit::frame_at(b,frame))) || observations::site_count(x,frame)!=observations::site_count(y,frame)){return .err(9);}var usize i=0;
  while(i<observations::site_count(x,frame)){let p=observations::at(x,frame,i);let q=observations::at(y,frame,i);if(p.start!=q.start || p.end!=q.end || p.mutable!=q.mutable || p.escapes!=q.escapes){return .err(9);}i+=1;}frame+=1;
 }return .ok(true);
}
fn Result<bool,u32> verify_observation_controls(ref observations::Table rows){
 var usize measured=0;let copied=observation_work::copy(rows,ref mut measured)?;if(measured==0 || observations::count(ref copied)!=observations::count(rows)){return .err(9);}
 var usize work=1073741824-measured;discard observation_work::copy(rows,ref mut work)?;if(work!=1073741824){return .err(9);}
 work=1073741824-measured+1;if(!switch(observation_work::copy(rows,ref mut work)){case .ok(_):{false}case .err(code):{code==3}}){return .err(9);}
 work=16777217;discard observation_work::copy(rows,ref mut work)?;if(work!=16777217+measured){return .err(9);}
 var slots=v::empty<sites::Range>();var seen=v::empty<u8>();work=0;discard observations::initialize(1,ref mut slots,ref mut seen,ref mut work)?;
 if(!switch(observations::seal(v::empty<sites::Site>(),slots,ref seen,ref mut work)){case .ok(_):{false}case .err(code):{code==9}}){return .err(9);}
 let absent=observations::absent();work=0;if(!switch(observations::copy(ref absent,ref mut work)){case .ok(_):{false}case .err(code):{code==9}}){return .err(9);}
 var usize i=1;while(i<=observations::count(rows)){if(observations::site_count(rows,i)>0){var data=v::empty<sites::Site>();var ranges=v::empty<sites::Range>();var marked=v::empty<u8>();work=0;discard observations::initialize(1,ref mut ranges,ref mut marked,ref mut work)?;
   discard observations::add_table(ref mut data,ref mut ranges,ref mut marked,1,rows,i,ref mut work)?;discard observations::add_table(ref mut data,ref mut ranges,ref mut marked,1,rows,i,ref mut work)?;
   var usize other=1;while(other<=observations::count(rows)){if(observations::site_count(rows,other)==0){if(!switch(observations::add_table(ref mut data,ref mut ranges,ref mut marked,1,rows,other,ref mut work)){case .ok(_):{false}case .err(code):{code==9}}){return .err(9);}break;}other+=1;}break;
  }i+=1;
 }return .ok(true);
}
fn Result<usize,u32> verify_units(ref input::Input source,ref signatures::Program metadata,ref planned::Plan plan){
 var usize work=0;let checked=switch(units::build_partitioned(input::sources(source),metadata,plan,1,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9);}};
 let scope=switch(modules::build_scoped(input::sources(source),input::owners(source),input::bindings(source))){case .ok(p):{p}case .err(_):{return .err(9);}};var usize i=0;var usize total=0;
 while(i<units::count(ref checked)){let original=units::at(ref checked,i);let range=units::range_at(ref checked,i);var usize encoding=0;let encoded=unit_encoder::encode(unit::ir_program(original),ref mut encoding)?;
  var usize source_work=0;var usize artifact_work=0;let restored=if(signatures::aggregate_types(metadata)){unit_admit::restore_aggregates(payload::content(ref encoded),input::sources(source),ref scope,metadata,plan,range.first,range.stop,ref mut source_work,ref mut artifact_work)?}else{unit_admit::restore(payload::content(ref encoded),input::sources(source),ref scope,metadata,plan,range.first,range.stop,ref mut source_work,ref mut artifact_work)?};
  discard same_unit(original,ref restored)?;total+=unit::frame_count(original);encoding=0;let repeated=unit_encoder::encode(unit::ir_program(ref restored),ref mut encoding)?;if(!bytes::equal(payload::content(ref encoded),payload::content(ref repeated))){return .err(9);}i+=1;
 }return .ok(total);
}
fn Result<usize,u32> verify_unit_sources(ref input::Input source,ref table::Coordinates table,bool aggregate){
 var usize work=0;
 if(aggregate){let plan=switch(source_plan::build_aggregate_sources(input::sources(source),input::owners(source),input::bindings(source),table,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9);}};
  let prepared=switch(aggregates::build(plan,input::sources(source),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9);}};let metadata=switch(signatures::build_aggregates(input::sources(source),prepared,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9);}};
  return verify_units(source,ref metadata,aggregates::source_plan(signatures::aggregate_source(ref metadata)));
 }
 let plan=switch(source_plan::build_sources(input::sources(source),input::owners(source),input::bindings(source),table,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9);}};
 let catalog=switch(modules::build_scoped(input::sources(source),input::owners(source),input::bindings(source))){case .ok(p):{p}case .err(_):{return .err(9);}};
 let metadata=switch(signatures::build_catalog(input::sources(source),catalog)){case .ok(p):{p}case .err(_):{return .err(9);}};return verify_units(source,ref metadata,ref plan);
}
@effects(filesystem,console) fn i32 main(){return switch(execute()){case .ok(p):{p}case .err(e):{discard io::write_usize(e as usize);7}};}
`
}

func TestCandidateAtlasBorrowObservations(t *testing.T) {
	type fixture struct {
		name, root string
		want       map[string][]semanticsnapshot.Borrow
	}
	var fixtures []fixture
	cases := append(atlasBorrowCollectorCases(), atlasBorrowOracleFixture{name: "cross-package-aliases", source: "module main;import answer as a;fn i32 local<T>(T input){return a::observe<T>(input);}fn i32 main(){let left=a::make();let right=a::make();discard left;discard right;let i32 value=3;let ref i32 view=a::relay(ref value);discard view;return local<i32>(1)+a::observe<i32>(1)+local<bool>(true);}"})
	for _, f := range cases {
		root := writeWorkspaceSessionFixture(t, f.source)
		if f.name == "cross-package-aliases" {
			writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", "module answer;pub resource Box<T>{T value;cleanup{let i32 value=9;let ref i32 view=ref value;discard view;}}pub fn Box<i32> make(){return Box<i32>{.value=1};}pub fn i32 observe<T>(T input){discard input;let i32 value=7;let ref i32 view=ref value;return *view;}pub fn ref i32 relay(ref i32 input){return input;}")
		}
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("borrow owner oracle %s: %d %s", f.name, code, &diagnostic)
		}
		want := map[string][]semanticsnapshot.Borrow{}
		for _, p := range prepared.partitions {
			for _, row := range p.Functions() {
				want[atlasDropKey(row.Function, "", false)] = row.Borrows
			}
			for _, row := range p.Specializations() {
				if row.Aggregate == nil {
					want[atlasDropKey(row.Declaration, row.ID, false)] = row.Function.Borrows
				}
				if row.Cleanup != nil {
					want[atlasDropKey(row.Declaration, row.ID, true)] = row.Cleanup.Borrows
				}
			}
		}
		fixtures = append(fixtures, fixture{f.name, root, want})
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateAtlasBorrowObservationsMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				var args []string
				if f.name == "cross-package-aliases" {
					args = []string{"expect-merged-aliases"}
				}
				code, out, stderr := candidateAtlasSDKCLIProcess(t, image, f.root, args)
				if code != 0 || stderr != "" {
					t.Fatalf("borrow Complete/restore: %d %.3000s %.500s", code, out, stderr)
				}
				var views [][]struct {
					Key struct {
						Package            string `json:"package_instance"`
						Module, Kind, Name string
					}
					Specialization string
					Cleanup        bool
					Sites          []struct {
						Span struct {
							Package string `json:"package_instance"`
							Source  string
							Start   int `json:"start_byte"`
							End     int `json:"end_byte"`
						}
						Kind    string
						Escapes bool
					}
				}
				if err := json.Unmarshal([]byte(out), &views); err != nil || len(views) != 2 {
					t.Fatalf("borrow views: %v %s", err, out)
				}
				for i, rows := range views {
					seen := map[string]bool{}
					for _, row := range rows {
						key := atlasDropKey(semanticsnapshot.DeclarationKey{PackageInstance: row.Key.Package, Module: row.Key.Module, Kind: semanticsnapshot.DeclarationKind(row.Key.Kind), Name: row.Key.Name}, row.Specialization, row.Cleanup)
						want, ok := f.want[key]
						if !ok || seen[key] {
							t.Fatalf("unknown/duplicate borrow owner %s", key)
						}
						seen[key] = true
						var got []semanticsnapshot.Borrow
						for _, b := range row.Sites {
							got = append(got, semanticsnapshot.Borrow{Span: semanticsnapshot.Span{PackageInstance: b.Span.Package, Source: b.Span.Source, StartByte: b.Span.Start, EndByte: b.Span.End}, Kind: b.Kind, Escapes: b.Escapes})
						}
						if !slices.Equal(got, want) {
							t.Fatalf("borrow view %d owner %s\n got %+v\nwant %+v", i, key, got, want)
						}
					}
					if len(seen) != len(f.want) {
						t.Fatalf("borrow view %d coverage: %d/%d", i, len(seen), len(f.want))
					}
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("borrow restore changed workspace")
				}
			})
			if t.Failed() {
				return
			}
		}
	}, "module_project_artifact_encode", "module_project_artifact_decode", "module_unit_artifact_admission", "project_compiler_source_mode")
}

// Whole-project metadata joins separately bounded units. A collector's batch
// limit must not become a new cumulative source-compilation restriction.
func TestCandidateAtlasBorrowObservationCapacity(t *testing.T) {
	const main = `module main;
import module_borrow_observations as observations;import module_unit_borrow_sites as sites;
import std::vec as v;import std::io as io;
fn Result<bool,u32> verify(usize total){
 var rows=v::empty<sites::Site>();var ranges=v::empty<sites::Range>();var seen=v::empty<u8>();var usize i=0;
 while(i<total){if(!v::push<sites::Site>(ref mut rows,sites::Site{.start=i,.end=i+1,.mutable=i%2==0,.escapes=i%3==0}).is_ok){return .err(2);}i+=1;}
 i=0;while(i<total){let end=if(total-i>16384){i+16384}else{total};if(!v::push<sites::Range>(ref mut ranges,sites::Range{.start=i,.end}).is_ok || !v::push<u8>(ref mut seen,1).is_ok){return .err(2);}i=end;}
 var usize work=0;let original=observations::seal(rows,ranges,ref seen,ref mut work)?;let copy=observations::copy(ref original,ref mut work)?;
 if(observations::count(ref original)!=observations::count(ref copy)){return .err(9);}var usize frame=1;var usize checked=0;
 while(frame<=observations::count(ref copy)){if(observations::site_count(ref original,frame)!=observations::site_count(ref copy,frame)){return .err(9);}i=0;
  while(i<observations::site_count(ref copy,frame)){let s=observations::at(ref copy,frame,i);if(s.start!=checked || s.end!=checked+1 || s.mutable!=(checked%2==0) || s.escapes!=(checked%3==0)){return .err(9);}checked+=1;i+=1;}frame+=1;
 }return if(checked==total){.ok(true)}else{.err(9)};
}
@effects(console) fn i32 main(){var usize total=65535;while(total<=65537){switch(verify(total)){case .ok(_):{}case .err(e):{discard io::write_usize(total);discard io::write("/");discard io::write_usize(e as usize);return 7;}};total+=1;}return 0;}
`
	candidateAtlasProbeSize(t, "module_borrow_observations", main, 32<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, out := candidateAtlasProcess(t, image, nil); code != 0 || out != "" {
			t.Fatalf("whole-project borrow count boundary: exit=%d stdout=%q", code, out)
		}
	})
}
