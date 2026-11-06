package cli

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

func atlasBorrowCollectorCases() []atlasBorrowOracleFixture {
	const prefix = "module main;fn i32 main(){return 0;}"
	return append(append(atlasBorrowSiteOracleCases(), atlasImplicitReceiverBorrowCases()...), []atlasBorrowOracleFixture{
		{name: "mutable-slice", source: prefix + "fn void exercise(){var i32[2] data=[1,2];let ref mut i32[] view=ref mut data[..];view[0]=3;}"},
		{name: "borrowed-element", source: prefix + "fn void exercise(){let i32[2] data=[1,2];let ref i32 item=ref data[0];discard item;}"},
		{name: "retained-subview", source: prefix + "fn void exercise(ref i32[] input){let ref i32[] view=input[..];discard view;}"},
		{name: "returned-subview", source: prefix + "fn ref i32[] exercise(ref i32[] input){return input[..];}"},
		{name: "tagged-return", source: prefix + "fn Option<ref i32> exercise(ref i32 input){return .some(input);}"},
		{name: "loop", source: prefix + "fn void exercise(bool flag){let i32 value=7;while(flag){let ref i32 view=ref value;discard view;}}"},
		{name: "two-borrows", source: prefix + "fn void exercise(){var i32 left=1;let i32 right=2;let ref i32 shared=ref right;discard shared;let ref mut i32 exclusive=ref mut left;*exclusive=3;}"},
		{name: "conditional-origins", source: prefix + "fn void exercise(bool flag){let i32 left=1;let i32 right=2;let ref i32 view=if(flag){ref left}else{ref right};discard view;}"},
		{name: "generic", source: "module main;fn i32 exercise<T>(T input){discard input;let i32 value=1;let ref i32 view=ref value;return *view;}fn i32 main(){return exercise<i32>(1)+exercise<bool>(true);}"},
		{name: "cleanup", source: "module main;resource Box<T>{T value;cleanup{let i32 value=1;let ref i32 view=ref value;discard view;}}fn i32 main(){let a=Box<i32>{.value=1};let b=Box<bool>{.value=true};discard a;discard b;return 0;}"},
	}...)
}

func candidateAtlasBorrowSitesMain(t *testing.T) string {
	t.Helper()
	main := strings.SplitN(candidateAtlasTypesMain, " var usize source_work=0;", 2)[0]
	main = strings.Replace(main, "import project_atlas_types as types;", `import project_atlas_types as types;
import module_unit_checked_bodies as checking;import module_unit_borrow_sites as sites;
import module_unit_values as values;import module_unit_value_model as model;
import module_catalog as modules;import module_signatures as signatures;
import project_atlas_borrow_sites as collect;import project_atlas_declarations as declarations;
import project_source_plan as source_plan;import generic_aggregate_types as aggregate_types;
import project_compiler_source_mode as source_mode;`, 1)
	main = strings.Replace(main, "fn i32 main(){", "fn Result<i32,u32> execute(){", 1)
	main = strings.ReplaceAll(main, "return 7;", "return .ok(7);")
	return main + `
 let aggregate=switch(source_mode::aggregates(ref source,ref mut meter)){case .ok(p):{p}case .err(_):{return .ok(7);}};
 let Result<signatures::Program,u32> built=if(aggregate){aggregate_metadata(ref source,ref table,ref mut meter)}else{ordinary_metadata(ref source)};
 let metadata=switch(built){case .ok(p):{p}case .err(_):{return .ok(7);}};
 let checked=switch(checking::build(input::sources(ref source),ref metadata,0,input::count(ref source))){case .ok(p):{p}case .err(e):{discard io::write_usize(e.stage as usize);discard io::write("/");discard io::write_usize(e.code as usize);return .ok(7);}};
 let typed=checking::checked_values(ref checked);let retained=checking::checked_borrows(ref checked);
 if(sites::frame_count(retained)!=values::count(typed)){return .ok(7);}
 var u8[64] argument=[0;64];let usize n=if(process::arg_count()==2){switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{return .ok(7);}}}else{0};let ref u8[] raw=ref argument;let mode=raw[..n];
 var usize bulk_work=0;let all=collect::collect_all(ref checked,65536,ref mut bulk_work)?;
 var usize largest=0;var usize frame=1;while(frame<=sites::frame_count(ref all)){let r=sites::range(ref all,frame-1);if(r.end-r.start>largest){largest=r.end-r.start;}frame+=1;}
 if(bytes::equal(mode,b"bulk-budget") || bytes::equal(mode,b"bulk-short-work") || bytes::equal(mode,b"bulk-exact-work") || bytes::equal(mode,b"bulk-more-work") || bytes::equal(mode,b"bulk-short-count") || bytes::equal(mode,b"bulk-exact-count") || bytes::equal(mode,b"bulk-more-count") || bytes::equal(mode,b"bulk-zero-count") || bytes::equal(mode,b"bulk-over-count")){
  var usize measured=0;var usize maximum=largest;
  if(bytes::equal(mode,b"bulk-budget")){measured=16777216;}else if(bytes::equal(mode,b"bulk-short-work")){measured=16777216-bulk_work+1;}else if(bytes::equal(mode,b"bulk-exact-work")){measured=16777216-bulk_work;}else if(bytes::equal(mode,b"bulk-more-work")){measured=16777216-bulk_work-1;}
  if(bytes::equal(mode,b"bulk-short-count")){maximum-=1;}else if(bytes::equal(mode,b"bulk-more-count")){maximum+=1;}else if(bytes::equal(mode,b"bulk-zero-count")){maximum=0;}else if(bytes::equal(mode,b"bulk-over-count")){maximum=65537;}
  return .ok(switch(collect::collect_all(ref checked,maximum,ref mut measured)){case .ok(_):{0}case .err(e):{discard io::write_usize(e as usize);7}});
 }
 var out=bytes::empty();discard w::raw(ref mut out,b"[",ref mut meter)?;var usize written=0;frame=1;
 while(frame<=values::count(typed)){
  let f=model::frame(values::at(typed,frame-1));var usize measured=0;let rows=collect::collect(ref checked,frame,65536,ref mut measured)?;let r=sites::range(ref all,frame-1);
  if(r.end-r.start!=v::len<sites::Site>(ref rows)){return .ok(7);}var usize j=0;
  while(j<v::len<sites::Site>(ref rows)){let a=*v::at<sites::Site>(ref rows,j);let b=sites::at(ref all,r.start+j);if(a.start!=b.start || a.end!=b.end || a.mutable!=b.mutable || a.escapes!=b.escapes){return .ok(7);}j+=1;}
  if(n>0){
   if(v::len<sites::Site>(ref rows)==0){frame+=1;continue;}var usize selected=frame;var usize maximum=v::len<sites::Site>(ref rows);var usize work=0;
   if(bytes::equal(mode,b"zero")){selected=0;}else if(bytes::equal(mode,b"past")){selected=values::count(typed)+1;}else if(bytes::equal(mode,b"budget")){work=16777216;}
   else if(bytes::equal(mode,b"short-work")){work=16777216-measured+1;}else if(bytes::equal(mode,b"exact-work")){work=16777216-measured;}else if(bytes::equal(mode,b"more-work")){work=16777216-measured-1;}
   else if(bytes::equal(mode,b"short-count")){maximum-=1;}else if(bytes::equal(mode,b"more-count")){maximum+=1;}else if(bytes::equal(mode,b"over-count")){maximum=65537;}
   return .ok(switch(collect::collect(ref checked,selected,maximum,ref mut work)){case .ok(_):{0}case .err(e):{discard io::write_usize(e as usize);7}});
  }
  if(written>0){discard w::raw(ref mut out,b",",ref mut meter)?;}written+=1;discard w::raw(ref mut out,b"{\"key\":",ref mut meter)?;
  discard declarations::key(ref mut out,ref source,declarations::Row{.module_index=f.module_index,.declaration=f.declaration-1},ref mut meter)?;
  discard w::raw(ref mut out,if(f.kind==2){b",\"cleanup\":true,\"sites\":["}else{b",\"cleanup\":false,\"sites\":["},ref mut meter)?;
  j=0;while(j<v::len<sites::Site>(ref rows)){let row=*v::at<sites::Site>(ref rows,j);if(j>0){discard w::raw(ref mut out,b",",ref mut meter)?;}discard w::raw(ref mut out,b"{\"span\":",ref mut meter)?;
   discard declarations::span(ref mut out,ref source,f.module_index,row.start,row.end,ref mut meter)?;discard w::raw(ref mut out,b",",ref mut meter)?;discard w::pair(ref mut out,b"kind",if(row.mutable){b"mutable"}else{b"shared"},ref mut meter)?;
   discard w::raw(ref mut out,if(row.escapes){b",\"escapes\":true}"}else{b",\"escapes\":false}"},ref mut meter)?;j+=1;
  }discard w::raw(ref mut out,b"]}",ref mut meter)?;frame+=1;
 }discard w::raw(ref mut out,b"]",ref mut meter)?;return .ok(if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4});
}
fn Result<signatures::Program,u32> aggregate_metadata(ref input::Input source,ref table::Coordinates table,ref mut usize work){
 let planned=switch(source_plan::build_aggregate_sources(input::sources(source),input::owners(source),input::bindings(source),table,work)){case .ok(p):{p}case .err(_):{return .err(9);}};
 let prepared=switch(aggregate_types::build(planned,input::sources(source),work)){case .ok(p):{p}case .err(_):{return .err(9);}};
 return switch(signatures::build_aggregates(input::sources(source),prepared,work)){case .ok(p):{.ok(p)}case .err(_):{.err(9)}};
}
fn Result<signatures::Program,u32> ordinary_metadata(ref input::Input source){
 let catalog=switch(modules::build_scoped(input::sources(source),input::owners(source),input::bindings(source))){case .ok(p):{p}case .err(_):{return .err(9);}};
 return switch(signatures::build_catalog(input::sources(source),catalog)){case .ok(p):{.ok(p)}case .err(_):{.err(9)}};
}
@effects(filesystem,console) fn i32 main(){return switch(execute()){case .ok(p):{p}case .err(e):{discard io::write_usize(e as usize);7}};}
`
}

func TestCandidateAtlasBorrowSites(t *testing.T) {
	type observation struct {
		frames int
		sites  []semanticsnapshot.Borrow
	}
	type fixture struct {
		name, root string
		want       map[string]observation
	}
	key := func(k semanticsnapshot.DeclarationKey, cleanup bool) string {
		return fmt.Sprintf("%s/%s/%s/%s/%v", k.PackageInstance, k.Module, k.Kind, k.Name, cleanup)
	}
	compare := func(a, b semanticsnapshot.Borrow) int {
		if n := cmp.Compare(a.Span.StartByte, b.Span.StartByte); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Span.EndByte, b.Span.EndByte); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Kind, b.Kind); n != 0 {
			return n
		}
		if a.Escapes == b.Escapes {
			return 0
		}
		if a.Escapes {
			return 1
		}
		return -1
	}
	canonical := func(rows []semanticsnapshot.Borrow) { slices.SortFunc(rows, compare) }
	var fixtures []fixture
	cases := append(atlasBorrowCollectorCases(), atlasBorrowOracleFixture{name: "implicit-private-cleanup", source: `module main;struct Cell{i32 value;}fn ref Cell Cell::view(ref Cell this){return this;}resource Token{i32 value;cleanup{let cell=Cell{.value=42};let view=cell.view();assert(view.value==42);}}fn i32 main(){let token=Token{.value=1};return 0;}`})
	for _, f := range cases {
		root := writeWorkspaceSessionFixture(t, f.source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != ExitSuccess {
			t.Fatalf("borrow source oracle %s: %d %s", f.name, code, &diagnostic)
		}
		want := map[string]observation{}
		// Logical Atlas deliberately omits nongeneric private drop bodies. The
		// checked-body collector includes them, so use the independent Go
		// semantic facts for those owners instead of inventing empty observations.
		checked, err := workspacecompiler.CheckGeneric(prepared.workspace)
		if err != nil {
			t.Fatal(err)
		}
		packages, err := prepared.workspace.SemanticPackages()
		if err != nil {
			t.Fatal(err)
		}
		add := func(k semanticsnapshot.DeclarationKey, cleanup bool, sites []semanticsnapshot.Borrow) {
			name := key(k, cleanup)
			r := want[name]
			r.frames++
			r.sites = append(r.sites, sites...)
			want[name] = r
		}
		for _, part := range prepared.partitions {
			for _, declaration := range part.Declarations() {
				if declaration.Key.Kind != semanticsnapshot.DeclarationResource || len(declaration.TypeParameters) != 0 {
					continue
				}
				var sites []semanticsnapshot.Borrow
				for _, fact := range checked.Semantic.Borrows {
					span := fact.Borrow
					for _, pkg := range packages {
						if pkg.ID == span.Package && pkg.Instance.String() == declaration.Key.PackageInstance && span.Source == declaration.Span.Source && span.StartByte >= declaration.Span.StartByte && span.EndByte <= declaration.Span.EndByte {
							sites = append(sites, semanticsnapshot.Borrow{Span: semanticsnapshot.Span{PackageInstance: declaration.Key.PackageInstance, Source: span.Source, StartByte: span.StartByte, EndByte: span.EndByte}, Kind: fact.Kind, Escapes: fact.Escapes})
						}
					}
				}
				add(declaration.Key, true, sites)
			}
			for _, f := range part.Functions() {
				add(f.Function, false, f.Borrows)
			}
			for _, s := range part.Specializations() {
				if s.Aggregate == nil {
					add(s.Declaration, false, s.Function.Borrows)
				}
				if s.Cleanup != nil {
					add(s.Declaration, true, s.Cleanup.Borrows)
				}
			}
		}
		for _, r := range want {
			canonical(r.sites)
		}
		if f.name == "implicit-private-cleanup" {
			count := 0
			for name, r := range want {
				if strings.HasSuffix(name, "/true") {
					count += len(r.sites)
				}
			}
			if count == 0 {
				t.Fatal("private cleanup Go oracle contains no borrow observations")
			}
		}
		fixtures = append(fixtures, fixture{f.name, root, want})
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateAtlasBorrowSitesMain(t), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				code, out, stderr := candidateAtlasSDKCLIProcess(t, image, f.root, nil)
				if code != 0 || stderr != "" {
					t.Fatalf("borrow collector: %d %.3000s %.500s", code, out, stderr)
				}
				var rows []struct {
					Key struct {
						Package            string `json:"package_instance"`
						Module, Kind, Name string
					}
					Cleanup bool
					Sites   []struct {
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
				if err := json.Unmarshal([]byte(out), &rows); err != nil {
					t.Fatal(err, out)
				}
				got := map[string]observation{}
				for _, row := range rows {
					name := key(semanticsnapshot.DeclarationKey{PackageInstance: row.Key.Package, Module: row.Key.Module, Kind: semanticsnapshot.DeclarationKind(row.Key.Kind), Name: row.Key.Name}, row.Cleanup)
					r := got[name]
					r.frames++
					var sites []semanticsnapshot.Borrow
					for _, b := range row.Sites {
						sites = append(sites, semanticsnapshot.Borrow{Span: semanticsnapshot.Span{PackageInstance: b.Span.Package, Source: b.Span.Source, StartByte: b.Span.Start, EndByte: b.Span.End}, Kind: b.Kind, Escapes: b.Escapes})
					}
					if !slices.IsSortedFunc(sites, compare) {
						t.Fatalf("borrow sites not in canonical order: %+v", sites)
					}
					r.sites = append(r.sites, sites...)
					got[name] = r
				}
				for _, r := range got {
					canonical(r.sites)
				}
				if !reflect.DeepEqual(got, f.want) {
					t.Fatalf("borrow owner/site inventory\n got %#v\nwant %#v", got, f.want)
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("collector changed workspace")
				}
			})
			if t.Failed() {
				return
			}
		}
		t.Run("controls", func(t *testing.T) {
			for _, mode := range []string{"zero", "past", "budget", "short-work", "exact-work", "more-work", "short-count", "exact-count", "more-count", "over-count", "bulk-budget", "bulk-short-work", "bulk-exact-work", "bulk-more-work", "bulk-short-count", "bulk-exact-count", "bulk-more-count", "bulk-over-count"} {
				t.Run(mode, func(t *testing.T) {
					code, out, stderr := candidateAtlasSDKCLIProcess(t, image, fixtures[1].root, []string{mode})
					want, failure := 0, ""
					if mode == "zero" || mode == "past" || mode == "over-count" || mode == "bulk-over-count" {
						want, failure = 7, "9"
					} else if mode == "budget" || strings.Contains(mode, "short-") || mode == "bulk-budget" {
						want, failure = 7, "3"
					}
					if code != want || out != failure || stderr != "" {
						t.Fatalf("borrow control: %d/%d %q/%q %q", code, want, out, failure, stderr)
					}
				})
			}
			for _, i := range []int{0, 1} {
				code, out, stderr := candidateAtlasSDKCLIProcess(t, image, fixtures[i].root, []string{"bulk-zero-count"})
				if i == 0 && (code != 0 || out != "") || i == 1 && (code != 7 || out != "3") || stderr != "" {
					t.Fatalf("borrow zero count %d: %d %q %q", i, code, out, stderr)
				}
			}
		})
	}, "project_compiler_source_mode", "project_atlas_declarations", "project_atlas_borrow_sites")
}
