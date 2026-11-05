package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func candidateAtlasDropSitesMain() string {
	main := strings.SplitN(candidateAtlasGenericProduceMain(), " var packages=", 2)[0]
	main = strings.Replace(main, "import project_atlas_types as types;", "import project_atlas_types as types;import project_atlas_drop_sites as drops;import project_atlas_declarations as declarations;import project_compiler_source_mode as source_mode;import project_path as work;", 1)
	main = strings.Replace(main, " var usize source_work=0;", " let aggregate_mode=switch(source_mode::aggregates(ref source,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};var usize source_work=0;", 1)
	main = strings.Replace(main, "compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)", "if(aggregate_mode){compiler::build_complete_aggregate_sources(input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut source_work,ref mut link_work)}else{compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)}", 1)
	main = strings.Replace(main, "schema::prepare(ref source,ref table,ref complete,ref mut meter)", "if(aggregate_mode){schema::prepare_aggregates(ref source,ref table,ref complete,ref mut meter)}else{schema::prepare(ref source,ref table,ref complete,ref mut meter)}", 1)
	main = strings.Replace(main, "instances::collect(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)", "if(aggregate_mode){instances::collect_aggregates(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)}else{instances::collect(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)}", 1)
	main = strings.Replace(main, ` if(bytes::equal(mode,b"budget")){meter=16777216;}`, "", 1)
	return main + `
 var usize bulk_work=0;let all=drops::collect_all(p,65536,ref mut bulk_work)?;
 if(drops::frame_count(ref all)!=ir::count(p)){return 7;}
 var usize combined_work=0;let combined=drops::collect_observed_all(p,65536,6,ref mut combined_work)?;
 if(drops::frame_count(ref combined)!=ir::count(p)){return 7;}
 if(bytes::equal(mode,b"bulk-budget") || bytes::equal(mode,b"bulk-short-work") || bytes::equal(mode,b"bulk-exact-work") || bytes::equal(mode,b"bulk-more-work") || bytes::equal(mode,b"bulk-short-count") || bytes::equal(mode,b"bulk-exact-count") || bytes::equal(mode,b"bulk-more-count") || bytes::equal(mode,b"bulk-over-count")){
  var usize maximum=0;var usize i=0;while(i<drops::frame_count(ref all)){let r=drops::range(ref all,i);if(r.end-r.start>maximum){maximum=r.end-r.start;}i+=1;}
  var usize meter=0;if(bytes::equal(mode,b"bulk-budget")){meter=16777216;}else if(bytes::equal(mode,b"bulk-short-work")){meter=16777216-bulk_work+1;}else if(bytes::equal(mode,b"bulk-exact-work")){meter=16777216-bulk_work;}else if(bytes::equal(mode,b"bulk-more-work")){meter=16777216-bulk_work-1;}
  if(bytes::equal(mode,b"bulk-short-count")){maximum-=1;}else if(bytes::equal(mode,b"bulk-more-count")){maximum+=1;}else if(bytes::equal(mode,b"bulk-over-count")){maximum=65537;}
  return switch(drops::collect_all(p,maximum,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};
 }
 var out=bytes::empty();discard w::raw(ref mut out,b"[",ref mut meter)?;var usize frame=1;var usize written=0;
 while(frame<=ir::count(p)){
  let f=clean::frame_at(clean,frame);let spec=instances::frame_identity(ref observed,frame);
  if(f.kind==2 && spec.len==0){frame+=1;continue;}
  if(f.kind!=1 && f.kind!=2 && f.kind!=3){frame+=1;continue;}
  var usize measured=0;let sites=switch(drops::collect(p,frame,65536,ref mut measured)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
  let r=drops::range(ref all,frame-1);let cr=drops::range(ref combined,frame-1);if(r.end-r.start!=v::len<drops::Site>(ref sites) || cr.end-cr.start!=v::len<drops::Site>(ref sites)){return 7;}var usize n=0;while(n<v::len<drops::Site>(ref sites)){let a=drops::at(ref all,r.start+n);let b=*v::at<drops::Site>(ref sites,n);let c=drops::at(ref combined,cr.start+n);if(a.start!=b.start || a.end!=b.end || c.start!=b.start || c.end!=b.end){return 7;}n+=1;}
  if(bytes::equal(mode,b"zero") || bytes::equal(mode,b"past") || bytes::equal(mode,b"budget") || bytes::equal(mode,b"short-work") || bytes::equal(mode,b"exact-work") || bytes::equal(mode,b"more-work") || bytes::equal(mode,b"short-count") || bytes::equal(mode,b"exact-count") || bytes::equal(mode,b"more-count") || bytes::equal(mode,b"over-count")){
   if(bytes::equal(mode,b"short-count") && v::len<drops::Site>(ref sites)==0 || bytes::equal(mode,b"short-work") && measured==0){frame+=1;continue;}
   var usize selected=frame;var usize maximum=65536;var usize meter=0;
   if(bytes::equal(mode,b"zero")){selected=0;}else if(bytes::equal(mode,b"past")){selected=ir::count(p)+1;}else if(bytes::equal(mode,b"budget")){meter=16777216;}
   else if(bytes::equal(mode,b"short-work")){meter=16777216-measured+1;}else if(bytes::equal(mode,b"exact-work")){meter=16777216-measured;}else if(bytes::equal(mode,b"more-work")){meter=16777216-measured-1;}
   else if(bytes::equal(mode,b"short-count")){maximum=v::len<drops::Site>(ref sites)-1;}else if(bytes::equal(mode,b"exact-count")){maximum=v::len<drops::Site>(ref sites);}else if(bytes::equal(mode,b"more-count")){maximum=v::len<drops::Site>(ref sites)+1;}else if(bytes::equal(mode,b"over-count")){maximum=65537;}
   return switch(drops::collect(p,selected,maximum,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};
  }
  if(written>0){discard w::raw(ref mut out,b",",ref mut meter)?;}written+=1;discard w::raw(ref mut out,b"{\"key\":",ref mut meter)?;
  discard declarations::key(ref mut out,ref source,declarations::Row{.module_index=f.module_index,.declaration=f.declaration-1},ref mut meter)?;
  discard w::raw(ref mut out,b",",ref mut meter)?;discard w::pair(ref mut out,b"specialization",spec,ref mut meter)?;discard w::raw(ref mut out,if(f.kind==2){b",\"cleanup\":true,\"sites\":["}else{b",\"cleanup\":false,\"sites\":["},ref mut meter)?;
  var usize at=0;while(at<v::len<drops::Site>(ref sites)){let site=*v::at<drops::Site>(ref sites,at);if(at>0){discard w::raw(ref mut out,b",",ref mut meter)?;}discard declarations::span(ref mut out,ref source,f.module_index,site.start,site.end,ref mut meter)?;at+=1;}
  discard w::raw(ref mut out,b"]}",ref mut meter)?;frame+=1;
 }
 discard w::raw(ref mut out,b"]",ref mut meter)?;return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`
}

func TestCandidateAtlasDropSites(t *testing.T) {
	type fixture struct {
		name, root string
		want       map[string][]semanticsnapshot.Span
	}
	var fixtures []fixture
	resource := "module main;import answer as a;resource Token{i32 value;cleanup{}}\n"
	for _, item := range []struct{ name, source string }{
		{"none", "module main;fn i32 main(){return 0;}"},
		{"scope", resource + "fn i32 main(){let a=Token{.value=1};let b=Token{.value=2};return 0;}"},
		{"branch-return", resource + "fn i32 choose(bool flag){let t=Token{.value=1};if(flag){return 1;}return 2;}fn i32 main(){return choose(true);}"},
		{"nested-scope", resource + "fn i32 main(){let a=Token{.value=1};{let b=Token{.value=2};}return 0;}"},
		{"discard", resource + "fn i32 main(){discard Token{.value=1};return 0;}"},
		{"transfer", resource + "fn void take(Token t){}fn i32 main(){let t=Token{.value=1};take(t);return 0;}"},
		{"loop-exits", resource + "fn i32 main(){var i32 i=0;while(i<3){let t=Token{.value=i};i+=1;if(i==1){continue;}break;}return 0;}"},
		{"trap", resource + "fn i32 main(){let t=Token{.value=1};assert(false);return 0;}"},
		{"grouped-trap", resource + "fn i32 main(){let t=Token{.value=1};assert((false));return 0;}"},
		{"branch-trap", resource + "fn i32 choose(bool flag){let t=Token{.value=1};if(flag){assert(false);}return 0;}fn i32 main(){return choose(true);}"},
		{"nested-owner", resource + "resource Parent{Token inner;cleanup{}}fn i32 main(){let t=Parent{.inner=Token{.value=1}};return 0;}"},
		{"owning-tag", resource + "fn i32 main(){let Option<Token> t=.some(Token{.value=1});return 0;}"},
		{"consuming-method", resource + "fn void Token::finish(Token this){}fn i32 main(){let t=Token{.value=1};t.finish();return 0;}"},
		{"generic", "module main;resource Token<T>{T value;cleanup{}}fn i32 use<T>(T value){let t=Token<T>{.value=value};return 0;}fn i32 main(){discard use<i32>(1);return use<u8>(2);}"},
		{"generic-cleanup", resource + "resource Wrap<T>{Token inner;T marker;cleanup{let t=Token{.value=1};}}fn i32 main(){let t=Wrap<i32>{.inner=Token{.value=2},.marker=3};return 0;}"},
	} {
		root := writeWorkspaceSessionFixture(t, item.source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("drop oracle %s: %d %s", item.name, code, &diagnostic)
		}
		want := map[string][]semanticsnapshot.Span{}
		for _, part := range prepared.partitions {
			for _, f := range part.Functions() {
				want[atlasDropKey(f.Function, "", false)] = f.DropSites
			}
			for _, s := range part.Specializations() {
				if s.Aggregate == nil {
					want[atlasDropKey(s.Declaration, s.ID, false)] = s.Function.DropSites
				}
				if s.Cleanup != nil {
					want[atlasDropKey(s.Declaration, s.ID, true)] = s.Cleanup.DropSites
				}
			}
		}
		fixtures = append(fixtures, fixture{item.name, root, want})
	}
	// The development main uses ? only for test-owned JSON assembly. Keep its
	// typed failure handler outside the compiler source under test.
	main := candidateAtlasDropSitesMain()
	main = strings.Replace(main, "@effects(filesystem,console,environment) fn i32 main(){", "@effects(filesystem,console,environment) fn Result<i32,u32> execute(){", 1)
	// Wrap all existing explicit i32 exits in Result without changing the source
	// fixture, compiler implementation or independent Go expectations.
	main = strings.ReplaceAll(main, "return 7;", "return .ok(7);")
	main = strings.ReplaceAll(main, "return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};", "return .ok(if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4});")
	main = strings.ReplaceAll(main, "return switch(drops::collect", "return .ok(switch(drops::collect")
	main = strings.ReplaceAll(main, "io::write_usize(code as usize);7}};", "io::write_usize(code as usize);7}});")
	main += "\n@effects(filesystem,console,environment) fn i32 main(){return switch(execute()){case .ok(p):{p}case .err(_):{7}};}"
	candidateAtlasProbeSize(t, "project_atlas_source", main, 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				code, out, stderr := fetchCLIProcess(t, image, f.root, nil, "")
				if code != 0 || stderr != "" {
					t.Fatalf("drop collector: %d %.3000s %.500s", code, out, stderr)
				}
				var rows []struct {
					Key struct {
						Package            string `json:"package_instance"`
						Module, Kind, Name string
					}
					Specialization string
					Cleanup        bool
					Sites          []struct {
						Package string `json:"package_instance"`
						Source  string
						Start   int `json:"start_byte"`
						End     int `json:"end_byte"`
					}
				}
				if err := json.Unmarshal([]byte(out), &rows); err != nil {
					t.Fatal(err, out)
				}
				seen := map[string]bool{}
				for _, row := range rows {
					key := atlasDropKey(semanticsnapshot.DeclarationKey{PackageInstance: row.Key.Package, Module: row.Key.Module, Kind: semanticsnapshot.DeclarationKind(row.Key.Kind), Name: row.Key.Name}, row.Specialization, row.Cleanup)
					want, ok := f.want[key]
					if !ok || seen[key] {
						t.Fatalf("unknown/duplicate drop owner %s", key)
					}
					seen[key] = true
					var actual []semanticsnapshot.Span
					for _, s := range row.Sites {
						actual = append(actual, semanticsnapshot.Span{PackageInstance: s.Package, Source: s.Source, StartByte: s.Start, EndByte: s.End})
					}
					if !slices.Equal(actual, want) {
						t.Fatalf("drop sites %s\n got %#v\nwant %#v", key, actual, want)
					}
				}
				if len(seen) != len(f.want) {
					t.Fatalf("drop owner coverage: %d/%d", len(seen), len(f.want))
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("collector changed workspace")
				}
			})
			if t.Failed() {
				return
			}
		}
		for _, mode := range []string{"zero", "past", "budget", "short-work", "exact-work", "more-work", "short-count", "exact-count", "more-count", "over-count", "bulk-budget", "bulk-short-work", "bulk-exact-work", "bulk-more-work", "bulk-short-count", "bulk-exact-count", "bulk-more-count", "bulk-over-count"} {
			code, out, stderr := fetchCLIProcess(t, image, fixtures[1].root, []string{mode}, "")
			want := 0
			failure := ""
			if mode == "zero" || mode == "past" || mode == "over-count" || mode == "bulk-over-count" {
				want, failure = 7, "9"
			} else if mode == "budget" || mode == "short-work" || mode == "short-count" || mode == "bulk-budget" || mode == "bulk-short-work" || mode == "bulk-short-count" {
				want, failure = 7, "3"
			}
			if code != want || out != failure || stderr != "" {
				t.Fatalf("drop control %s: %d/%d %q/%q %q", mode, code, want, out, failure, stderr)
			}
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph", "project_atlas_drop_sites", "project_compiler_source_mode")
}

func atlasDropKey(key semanticsnapshot.DeclarationKey, specialization string, cleanup bool) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%t", key.PackageInstance, key.Module, key.Kind, key.Name, specialization, cleanup)
}
