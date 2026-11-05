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

func candidateAtlasTrapsMain() string {
	main := strings.SplitN(candidateAtlasDropSitesMain(), " var usize bulk_work=", 2)[0]
	main = strings.Replace(main, "import project_atlas_drop_sites as drops;", "import project_atlas_drop_sites as drops;import project_atlas_traps as traps;", 1)
	main += `
 var usize bulk_work=0;let all=traps::collect_all(p,6,ref mut bulk_work)?;
 if(traps::frame_count(ref all)!=ir::count(p)){return 7;}
 if(bytes::equal(mode,b"bulk-budget") || bytes::equal(mode,b"bulk-short-work") || bytes::equal(mode,b"bulk-exact-work") || bytes::equal(mode,b"bulk-more-work") || bytes::equal(mode,b"bulk-short-count") || bytes::equal(mode,b"bulk-exact-count") || bytes::equal(mode,b"bulk-more-count") || bytes::equal(mode,b"bulk-over-count")){
  var usize maximum=0;var usize i=0;while(i<traps::frame_count(ref all)){var usize count=0;var usize k=0;while(k<6){if(traps::contains(traps::at(ref all,i),k)){count+=1;}k+=1;}if(count>maximum){maximum=count;}i+=1;}
  var usize meter=0;if(bytes::equal(mode,b"bulk-budget")){meter=16777216;}else if(bytes::equal(mode,b"bulk-short-work")){meter=16777216-bulk_work+1;}else if(bytes::equal(mode,b"bulk-exact-work")){meter=16777216-bulk_work;}else if(bytes::equal(mode,b"bulk-more-work")){meter=16777216-bulk_work-1;}
  if(bytes::equal(mode,b"bulk-short-count")){maximum-=1;}else if(bytes::equal(mode,b"bulk-more-count")){maximum+=1;}else if(bytes::equal(mode,b"bulk-over-count")){maximum=7;}
  return switch(traps::collect_all(p,maximum,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};
 }
 var out=bytes::empty();discard w::raw(ref mut out,b"[",ref mut meter)?;var usize frame=1;var usize written=0;
 while(frame<=ir::count(p)){
  let f=clean::frame_at(clean,frame);let spec=instances::frame_identity(ref observed,frame);
  if(f.kind==2 && spec.len==0){frame+=1;continue;}if(f.kind!=1 && f.kind!=2 && f.kind!=3){frame+=1;continue;}
  var usize measured=0;let mask=traps::collect(p,frame,6,ref mut measured)?;
  if(traps::at(ref all,frame-1)!=mask){return 7;}var usize count=0;var usize k=0;while(k<6){if(traps::contains(mask,k)){count+=1;}k+=1;}
  if(bytes::equal(mode,b"zero") || bytes::equal(mode,b"past") || bytes::equal(mode,b"budget") || bytes::equal(mode,b"short-work") || bytes::equal(mode,b"exact-work") || bytes::equal(mode,b"more-work") || bytes::equal(mode,b"short-count") || bytes::equal(mode,b"exact-count") || bytes::equal(mode,b"more-count") || bytes::equal(mode,b"over-count")){
   if(count==0){frame+=1;continue;}var usize selected=frame;var usize maximum=6;var usize meter=0;
   if(bytes::equal(mode,b"zero")){selected=0;}else if(bytes::equal(mode,b"past")){selected=ir::count(p)+1;}else if(bytes::equal(mode,b"budget")){meter=16777216;}
   else if(bytes::equal(mode,b"short-work")){meter=16777216-measured+1;}else if(bytes::equal(mode,b"exact-work")){meter=16777216-measured;}else if(bytes::equal(mode,b"more-work")){meter=16777216-measured-1;}
   else if(bytes::equal(mode,b"short-count")){maximum=count-1;}else if(bytes::equal(mode,b"exact-count")){maximum=count;}else if(bytes::equal(mode,b"more-count")){maximum=count+1;}else if(bytes::equal(mode,b"over-count")){maximum=7;}
   return switch(traps::collect(p,selected,maximum,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};
  }
  if(written>0){discard w::raw(ref mut out,b",",ref mut meter)?;}written+=1;discard w::raw(ref mut out,b"{\"key\":",ref mut meter)?;
  discard declarations::key(ref mut out,ref source,declarations::Row{.module_index=f.module_index,.declaration=f.declaration-1},ref mut meter)?;
  discard w::raw(ref mut out,b",",ref mut meter)?;discard w::pair(ref mut out,b"specialization",spec,ref mut meter)?;
  discard w::raw(ref mut out,if(f.kind==2){b",\"cleanup\":true,\"categories\":["}else{b",\"cleanup\":false,\"categories\":["},ref mut meter)?;
  k=0;var usize n=0;while(k<6){if(traps::contains(mask,k)){if(n>0){discard w::raw(ref mut out,b",",ref mut meter)?;}discard w::string(ref mut out,traps::category(k),ref mut meter)?;n+=1;}k+=1;}
  discard w::raw(ref mut out,b"]}",ref mut meter)?;frame+=1;
 }
 discard w::raw(ref mut out,b"]",ref mut meter)?;return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`
	main = strings.Replace(main, "@effects(filesystem,console,environment) fn i32 main(){", "@effects(filesystem,console,environment) fn Result<i32,u32> execute(){", 1)
	main = strings.ReplaceAll(main, "return 7;", "return .ok(7);")
	main = strings.ReplaceAll(main, "return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};", "return .ok(if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4});")
	main = strings.ReplaceAll(main, "return switch(traps::collect", "return .ok(switch(traps::collect")
	main = strings.ReplaceAll(main, "io::write_usize(code as usize);7}};", "io::write_usize(code as usize);7}});")
	return main + "\n@effects(filesystem,console,environment) fn i32 main(){return switch(execute()){case .ok(p):{p}case .err(_):{7}};}"
}

func TestCandidateAtlasTraps(t *testing.T) {
	type fixture struct {
		name, root string
		want       map[string][]string
	}
	var fixtures []fixture
	for _, item := range []struct{ name, source string }{
		{"none", "module main;fn i32 main(){return 0;}"},
		{"signed-division", "module main;fn i32 div(i32 a,i32 b){return a/b;}fn i32 main(){return div(10,2);}"},
		{"signed-remainder", "module main;fn i64 rem(i64 a,i64 b){return a%b;}fn i32 main(){discard rem(-4,2);return 0;}"},
		{"unsigned-division", "module main;fn usize div(usize a,usize b){return a/b;}fn i32 main(){discard div(10,2);return 0;}"},
		{"arithmetic", "module main;fn i32 f(i32 a,i32 b){return -(a+b)-a*b;}fn i32 main(){return f(2,3);}"},
		{"left-shift", "module main;fn u32 shift(u32 a,u32 b){return a<<b;}fn i32 main(){discard shift(1,2);return 0;}"},
		{"right-shift", "module main;fn i32 shift(i32 a,i32 b){return a>>b;}fn i32 main(){return shift(8,1);}"},
		{"minimum-literal", "module main;fn i32 main(){return -2147483648;}"},
		{"assert-true", "module main;fn i32 main(){assert(true);return 0;}"},
		{"assert-terminal", "module main;fn i32 main(){assert(false);return 1/0;}"},
		{"assert-grouped-terminal", "module main;fn i32 main(){assert((false));return 1<<32;}"},
		{"assert-constant-terminal", "module main;const bool STOP=false;fn i32 main(){assert(STOP);return 1/0;}"},
		{"branch", "module main;fn i32 f(bool flag){if(flag){assert(false);}return 1/2;}fn i32 main(){return f(false);}"},
		{"short-circuit", "module main;fn bool f(bool a,i32 b){return a && b/2>0;}fn i32 main(){discard f(false,1);return 0;}"},
		{"index", "module main;fn u8 get(usize i){let u8[2] values=[1,2];return values[i];}fn i32 main(){discard get(0);return 0;}"},
		{"indexed-store", "module main;fn i32 main(){var u8[2] values=[1,2];let usize i=0;values[i]=3;return 0;}"},
		{"slice", "module main;fn i32 main(){let data=b\"abc\";let part=data[1..2];discard part;return 0;}"},
		{"compound", "module main;fn i32 f(i32 a,i32 b){var i32 x=a;x/=b;x<<=1;return x;}fn i32 main(){return f(8,2);}"},
		{"all-six", "module main;fn i32 f(bool flag,i32 a,usize i){let u8[1] data=[1];if(flag){assert(false);}discard data[i];return (a/2)<<1;}fn i32 main(){return f(false,8,0);}"},
		{"bitwise-and-borrow", "module main;fn i32 f(ref i32 a){return (*a & 3)|1;}fn i32 main(){let i32 a=2;return f(ref a);}"},
		{"loop", "module main;fn i32 main(){var i32 i=0;while(i<3){i+=1;if(i==1){continue;}break;}return i;}"},
		{"byte-iteration", "module main;fn i32 main(){let data=b\"ab\";for(let u8 byte in data){discard byte;}return 0;}"},
		{"empty-iteration", "module main;fn i32 main(){let data=b\"\";for(let u8 byte in data){discard byte;}return 0;}"},
		{"iteration-trap", "module main;fn i32 main(){let data=b\"ab\";for(let u8 byte in data){assert(false);}return 0;}"},
		{"defer", "module main;fn void consume(i32 x){}fn i32 main(){defer consume(1/2);return 0;}"},
		{"cleanup", "module main;resource Token{i32 value;cleanup{var i32 x=2;x+=1;assert(x>0);}}fn i32 main(){let t=Token{.value=2};return 0;}"},
		{"generic", "module main;fn T div<T>(T a,T b){return a/b;}fn i32 main(){discard div<i32>(4,2);discard div<usize>(4,2);return 0;}"},
		{"generic-cleanup", "module main;resource Token<T>{T value;cleanup{var i32 x=2;x/=1;}}fn i32 main(){let t=Token<i32>{.value=2};return 0;}"},
	} {
		root := writeWorkspaceSessionFixture(t, item.source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("trap oracle %s: %d %s", item.name, code, &diagnostic)
		}
		want := map[string][]string{}
		for _, part := range prepared.partitions {
			for _, f := range part.Functions() {
				want[atlasDropKey(f.Function, "", false)] = f.TrapCategories
			}
			for _, s := range part.Specializations() {
				if s.Aggregate == nil {
					want[atlasDropKey(s.Declaration, s.ID, false)] = s.Function.TrapCategories
				}
				if s.Cleanup != nil {
					want[atlasDropKey(s.Declaration, s.ID, true)] = s.Cleanup.TrapCategories
				}
			}
		}
		fixtures = append(fixtures, fixture{item.name, root, want})
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateAtlasTrapsMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				code, out, stderr := fetchCLIProcess(t, image, f.root, nil, "")
				if code != 0 || stderr != "" {
					t.Fatalf("trap collector: %d %.3000s %.500s", code, out, stderr)
				}
				var rows []struct {
					Key struct {
						Package            string `json:"package_instance"`
						Module, Kind, Name string
					}
					Specialization string   `json:"specialization"`
					Cleanup        bool     `json:"cleanup"`
					Categories     []string `json:"categories"`
				}
				if err := json.Unmarshal([]byte(out), &rows); err != nil {
					t.Fatal(err, out)
				}
				seen := map[string]bool{}
				for _, row := range rows {
					key := atlasDropKey(semanticsnapshot.DeclarationKey{PackageInstance: row.Key.Package, Module: row.Key.Module, Kind: semanticsnapshot.DeclarationKind(row.Key.Kind), Name: row.Key.Name}, row.Specialization, row.Cleanup)
					want, ok := f.want[key]
					if !ok || seen[key] || !slices.Equal(row.Categories, want) {
						t.Fatalf("trap categories %s: known=%v duplicate=%v\n got %q\nwant %q", key, ok, seen[key], row.Categories, want)
					}
					seen[key] = true
				}
				if len(seen) != len(f.want) {
					t.Fatalf("trap owner coverage: %d/%d", len(seen), len(f.want))
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
			want, failure := 0, ""
			if mode == "zero" || mode == "past" || mode == "over-count" || mode == "bulk-over-count" {
				want, failure = 7, "9"
			} else if mode == "budget" || mode == "short-work" || mode == "short-count" || mode == "bulk-budget" || mode == "bulk-short-work" || mode == "bulk-short-count" {
				want, failure = 7, "3"
			}
			if code != want || out != failure || stderr != "" {
				t.Fatalf("trap control %s: %d/%d %q/%q %q", mode, code, want, out, failure, stderr)
			}
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph", "project_atlas_traps", "project_compiler_source_mode")
}
