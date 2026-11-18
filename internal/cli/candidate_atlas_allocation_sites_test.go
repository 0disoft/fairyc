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

func candidateAtlasObservedSitesMain(t *testing.T, collector string) string {
	t.Helper()
	main := strings.Replace(candidateAtlasMoveSitesMain(t), "import project_atlas_move_sites as drops;", "import "+collector+" as drops;", 1)
	// Compare standalone/batch APIs for every selected root frame. The batch
	// still publishes all SDK frame observations against the independent oracle;
	// do not repeat whole-program verification for each unrelated SDK helper.
	call := "drops::collect(p,frame,65536,ref mut measured)"
	if strings.Count(main, call) != 1 {
		t.Fatal("missing allocation single/batch comparison")
	}
	main = strings.Replace(main, call, "if(input::origin(ref source,f.module_index).owner==0){drops::collect(p,frame,65536,ref mut measured)}else{retained_sites(ref all,frame)}", 1)
	return main + `
fn Result<v::Vec<drops::Site>,u32> retained_sites(ref drops::Collection all,usize frame){
 let r=drops::range(all,frame-1);var rows=v::empty<drops::Site>();var usize i=r.start;
 while(i<r.end){if(!v::push<drops::Site>(ref mut rows,drops::at(all,i)).is_ok){return .err(2);}i+=1;}return .ok(rows);
}`
}

func candidateAtlasAllocationSitesMain(t *testing.T) string {
	t.Helper()
	main := candidateAtlasObservedSitesMain(t, "project_atlas_allocation_sites")
	const frameCountCheck = " if(drops::frame_count(ref all)!=ir::count(p)){return 7;}"
	main = strings.Replace(main, frameCountCheck, frameCountCheck+`
 if(bytes::equal(mode,b"allocation-controls")){
  var usize frame=1;
  while(frame<=ir::count(p)){
   let f=clean::frame_at(clean,frame);let r=drops::range(ref all,frame-1);
   if((f.kind==1 || f.kind==3) && input::origin(ref source,f.module_index).owner==0 && r.end>r.start){
    return allocation_controls(p,ref all,frame,r.end-r.start,bulk_work);
   }
   frame+=1;
  }
  return 7;
 }`, 1)
	return main + `
fn u32 allocation_one(ref ir::Program p,usize frame,usize count,usize initial){
 var usize meter=initial;
 let result=drops::collect(p,frame,count,ref mut meter);
 return switch(result){case .ok(_):{0}case .err(code):{code}};
}
fn u32 allocation_all(ref ir::Program p,usize count,usize initial){
 var usize meter=initial;
 let result=drops::collect_all(p,count,ref mut meter);
 return switch(result){case .ok(_):{0}case .err(code):{code}};
}
@effects(console) fn i32 allocation_controls(ref ir::Program p,ref drops::Collection all,usize frame,usize count,usize bulk_work){
 var usize measured=0;
 if(!drops::collect(p,frame,65536,ref mut measured).is_ok){return 8;}
 var usize maximum=0;var usize i=0;
 while(i<drops::frame_count(all)){
  let r=drops::range(all,i);if(r.end-r.start>maximum){maximum=r.end-r.start;}i+=1;
 }
 let usize limit=16777216;
 let u32[19] results=[
  allocation_all(p,0,0),
  allocation_one(p,0,65536,0),allocation_one(p,ir::count(p)+1,65536,0),
  allocation_one(p,frame,65536,limit),
  allocation_one(p,frame,65536,limit-measured+1),allocation_one(p,frame,65536,limit-measured),allocation_one(p,frame,65536,limit-measured-1),
  allocation_one(p,frame,count-1,0),allocation_one(p,frame,count,0),allocation_one(p,frame,count+1,0),allocation_one(p,frame,65537,0),
  allocation_all(p,maximum,limit),
  allocation_all(p,maximum,limit-bulk_work+1),allocation_all(p,maximum,limit-bulk_work),allocation_all(p,maximum,limit-bulk_work-1),
  allocation_all(p,maximum-1,0),allocation_all(p,maximum,0),allocation_all(p,maximum+1,0),allocation_all(p,65537,0)];
 i=0;while(i<19){
  if(!io::write_usize(results[i] as usize).is_ok || !io::write("\n").is_ok){return 4;}i+=1;
 }
 return 0;
}`
}

func TestCandidateAtlasAllocationSites(t *testing.T) {
	items := atlasAllocationCollectorCases()
	emptyRoot := writeWorkspaceSessionFixture(t, items[0].source)
	controlRoot := writeWorkspaceSessionFixture(t, items[2].source)
	// The native probe loads in locked mode. Only the corpus is prepared by
	// the Go oracle, so create the two control lockfiles here.
	for _, control := range []string{emptyRoot, controlRoot} {
		if _, _, err := loadWorkspaceWithLock(control, false); err != nil {
			t.Fatalf("prepare allocation control lockfile: %v", err)
		}
	}
	root := writeWorkspaceSessionFixture(t, items[0].source)
	modules := make(map[string]bool, len(items)-1)
	for _, item := range items[1:] {
		module := "case_" + strings.ReplaceAll(item.name, "-", "_")
		source := strings.Replace(item.source, "module main;", "module "+module+";", 1)
		source = strings.Replace(source, "fn i32 main()", "fn i32 fixture_entry()", 1)
		writeWorkspaceFile(t, root, "src/"+module+".fy", source)
		modules[module] = false
	}
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("allocation corpus oracle: %d %s", code, &diagnostic)
	}
	want := map[string][]semanticsnapshot.Span{}
	for _, part := range prepared.partitions {
		for _, f := range part.Functions() {
			want[atlasDropKey(f.Function, "", false)] = f.AllocationSites
			if f.Function.Name == "fixture_entry" {
				modules[f.Function.Module] = true
			}
		}
		for _, s := range part.Specializations() {
			if s.Aggregate == nil {
				want[atlasDropKey(s.Declaration, s.ID, false)] = s.Function.AllocationSites
			}
			if s.Cleanup != nil {
				want[atlasDropKey(s.Declaration, s.ID, true)] = s.Cleanup.AllocationSites
			}
		}
	}
	for module, seen := range modules {
		if !seen {
			t.Fatalf("allocation corpus is missing fixture %s", module)
		}
	}
	// The development main uses ? only for test-owned JSON assembly. Keep its
	// typed failure handler outside the compiler source under test.
	main := candidateAtlasAllocationSitesMain(t)
	main = strings.Replace(main, "@effects(filesystem,console,environment) fn i32 main(){", "@effects(filesystem,console,environment) fn Result<i32,u32> execute(){", 1)
	// Wrap all existing explicit i32 exits in Result without changing the source
	// fixture, compiler implementation or independent Go expectations.
	main = strings.ReplaceAll(main, "return 7;", "return .ok(7);")
	main = strings.ReplaceAll(main, "return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};", "return .ok(if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4});")
	main = strings.ReplaceAll(main, "return switch(drops::collect", "return .ok(switch(drops::collect")
	main = strings.Replace(main, "return allocation_controls(p,ref all,frame,r.end-r.start,bulk_work);", "return .ok(allocation_controls(p,ref all,frame,r.end-r.start,bulk_work));", 1)
	main = strings.ReplaceAll(main, "io::write_usize(code as usize);7}};", "io::write_usize(code as usize);7}});")
	main += "\n@effects(filesystem,console,environment) fn i32 main(){return switch(execute()){case .ok(p):{p}case .err(_):{7}};}"
	candidateAtlasProbeSize(t, "project_atlas_source", main, 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		t.Run("corpus", func(t *testing.T) {
			before := remoteTree(t, root)
			code, out, stderr := candidateAtlasSDKCLIProcess(t, image, root, nil)
			if code != 0 || stderr != "" {
				t.Fatalf("allocation collector: %d %.3000s %.500s", code, out, stderr)
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
				expected, ok := want[key]
				if !ok || seen[key] {
					t.Fatalf("unknown/duplicate allocation owner %s", key)
				}
				seen[key] = true
				var actual []semanticsnapshot.Span
				for _, s := range row.Sites {
					actual = append(actual, semanticsnapshot.Span{PackageInstance: s.Package, Source: s.Source, StartByte: s.Start, EndByte: s.End})
				}
				if !slices.Equal(actual, expected) {
					t.Fatalf("allocation sites %s\n got %#v\nwant %#v", key, actual, expected)
				}
			}
			if len(seen) != len(want) {
				t.Fatalf("allocation owner coverage: %d/%d", len(seen), len(want))
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("collector changed workspace")
			}
		})
		if t.Failed() {
			return
		}
		t.Run("controls", func(t *testing.T) {
			t.Run("bulk-zero-count-empty", func(t *testing.T) {
				code, out, stderr := candidateAtlasSDKCLIProcess(t, image, emptyRoot, []string{"bulk-zero-count"})
				if code != 0 || out != "" || stderr != "" {
					t.Fatalf("allocation empty zero cap: %d %q %q", code, out, stderr)
				}
			})
			controls := []struct{ name, want string }{
				{"bulk-zero-count-nonempty", "3"},
				{"zero", "9"}, {"past", "9"}, {"budget", "3"},
				{"short-work", "3"}, {"exact-work", "0"}, {"more-work", "0"},
				{"short-count", "3"}, {"exact-count", "0"}, {"more-count", "0"}, {"over-count", "9"},
				{"bulk-budget", "3"}, {"bulk-short-work", "3"}, {"bulk-exact-work", "0"}, {"bulk-more-work", "0"},
				{"bulk-short-count", "3"}, {"bulk-exact-count", "0"}, {"bulk-more-count", "0"}, {"bulk-over-count", "9"},
			}
			code, out, stderr := candidateAtlasSDKCLIProcess(t, image, controlRoot, []string{"allocation-controls"})
			rows, terminated := strings.CutSuffix(out, "\n")
			results := strings.Split(rows, "\n")
			if code != 0 || stderr != "" || !terminated || len(results) != len(controls) {
				t.Fatalf("allocation controls: %d %q %q", code, out, stderr)
			}
			for i, control := range controls {
				t.Run(control.name, func(t *testing.T) {
					if results[i] != control.want {
						t.Fatalf("allocation control: got %q, want %q", results[i], control.want)
					}
				})
			}
		})
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph", "project_atlas_allocation_sites", "project_compiler_source_mode")
}

func atlasAllocationCollectorCases() []atlasAllocationOracleFixture {
	const prefix = "module main;import std::alloc as alloc;"
	const entry = "fn i32 main(){return 0;}"
	return append(atlasAllocationSiteOracleCases(), []atlasAllocationOracleFixture{
		{name: "after-direct-trap", source: prefix + "fn void exercise(Region r){assert(false);discard r.place(7);}" + entry},
		{name: "nested-placements", source: prefix + "fn void exercise(Region r){discard r.place({discard r.place(7);8});}" + entry},
		{name: "result-propagation", source: prefix + "fn Result<i32,alloc::AllocError> exercise(Region r){let ref i32 n=r.place(7)?;return .ok(*n);}" + entry},
		{name: "loop-exits", source: prefix + "fn void exercise(Region r){var i32 i=0;while(i<3){discard r.place(i);i+=1;if(i<2){continue;}break;}}" + entry},
		{name: "generic", source: prefix + "fn void put<T>(Region r,T n){discard r.place(n);}fn void exercise(Region r){put<i32>(r,7);put<u8>(r,8);}" + entry},
		{name: "cleanup", source: prefix + "resource Wrap<T>{T marker;cleanup{var alloc::Arena arena=switch(alloc::create()){case .ok(a):{a}case .err(_):{assert(false);return;}};region r in ref mut arena{discard r.place(7);}}}fn i32 main(){let value=Wrap<i32>{.marker=1};return 0;}"},
	}...)
}

func TestAtlasAllocationCollectionSources(t *testing.T) {
	for _, f := range atlasAllocationCollectorCases() {
		t.Run(f.name, func(t *testing.T) {
			var diagnostic bytes.Buffer
			if _, code := prepareAtlasWorkspaceSchema(writeWorkspaceSessionFixture(t, f.source), false, &diagnostic, atlasmap.LogicalSchemaV3); code != ExitSuccess {
				t.Fatalf("allocation collection source: %d %s", code, &diagnostic)
			}
		})
	}
}
