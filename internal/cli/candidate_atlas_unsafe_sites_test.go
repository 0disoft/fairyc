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

func candidateAtlasUnsafeSitesMain(t *testing.T) string {
	t.Helper()
	main := candidateAtlasDropSitesMain()
	replace := func(old, next string) {
		t.Helper()
		if strings.Count(main, old) != 1 {
			t.Fatalf("unsafe collector requires one exact probe port: %q", old)
		}
		main = strings.Replace(main, old, next, 1)
	}
	replace("import project_atlas_drop_sites as drops;", "import project_atlas_unsafe_sites as drops;")
	start := strings.Index(main, " var usize combined_work=")
	if start < 0 {
		t.Fatal("missing combined collector probe start")
	}
	stop := strings.Index(main[start:], " if(bytes::equal(mode,b\"bulk-budget\")")
	if stop < 0 {
		t.Fatal("missing combined collector probe boundary")
	}
	end := start + stop
	main = main[:start] + main[end:]
	start = strings.Index(main, "  let r=drops::range(ref all,frame-1);")
	if start < 0 {
		t.Fatal("missing combined collector comparison start")
	}
	stop = strings.Index(main[start:], "\n  if(bytes::equal(mode,")
	if stop < 0 {
		t.Fatal("missing combined collector comparison boundary")
	}
	end = start + stop
	main = main[:start] + `  let r=drops::range(ref all,frame-1);if(r.end-r.start!=v::len<drops::Site>(ref sites)){return 7;}var usize n=0;while(n<v::len<drops::Site>(ref sites)){let a=drops::at(ref all,r.start+n);let b=*v::at<drops::Site>(ref sites,n);if(a.start!=b.start || a.end!=b.end){return 7;}n+=1;}` + main[end:]
	replace(`var usize measured=0;let sites=switch(drops::collect(p,frame,65536,ref mut measured))`, `var usize measured=0;let sites=switch(if(bytes::equal(mode,b"batch")){batch_sites(ref all,drops::range(ref all,frame-1))}else{drops::collect(p,frame,65536,ref mut measured)})`)
	replace(`|| bytes::equal(mode,b"bulk-over-count")){`, `|| bytes::equal(mode,b"bulk-over-count") || bytes::equal(mode,b"bulk-zero-count")){`)
	replace(`else if(bytes::equal(mode,b"bulk-over-count")){maximum=65537;}`, `else if(bytes::equal(mode,b"bulk-over-count")){maximum=65537;}else if(bytes::equal(mode,b"bulk-zero-count")){maximum=0;}`)
	replace(` discard w::raw(ref mut out,b"]",ref mut meter)?;return`, `
 var usize inline_index=0;while(inline_index<instances::count(ref observed)){
  let row=instances::at(ref observed,inline_index);inline_index+=1;if(!instances::inline_leaf(ref observed,row)){continue;}
  let site=drops::leaf(ref all,row.declaration.module_index,ref mut meter)?;
  if(written>0){discard w::raw(ref mut out,b",",ref mut meter)?;}written+=1;discard w::raw(ref mut out,b"{\"key\":",ref mut meter)?;
  discard declarations::key(ref mut out,ref source,row.declaration,ref mut meter)?;discard w::raw(ref mut out,b",",ref mut meter)?;
  discard w::pair(ref mut out,b"specialization",instances::identity(ref observed,row),ref mut meter)?;
  discard w::raw(ref mut out,b",\"cleanup\":false,\"sites\":[",ref mut meter)?;
  discard declarations::span(ref mut out,ref source,row.declaration.module_index,site.start,site.end,ref mut meter)?;discard w::raw(ref mut out,b"]}",ref mut meter)?;
 }
 discard w::raw(ref mut out,b"]",ref mut meter)?;return`)
	main = strings.Replace(main, "@effects(filesystem,console,environment) fn i32 main(){", "@effects(filesystem,console,environment) fn Result<i32,u32> execute(){", 1)
	main = strings.ReplaceAll(main, "return 7;", "return .ok(7);")
	main = strings.ReplaceAll(main, "return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};", "return .ok(if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4});")
	main = strings.ReplaceAll(main, "return switch(drops::collect", "return .ok(switch(drops::collect")
	main = strings.ReplaceAll(main, "io::write_usize(code as usize);7}};", "io::write_usize(code as usize);7}});")
	return main + `
fn Result<v::Vec<drops::Site>,u32> batch_sites(ref drops::Collection all,drops::Range range){var copy=v::empty<drops::Site>();var usize k=range.start;while(k<range.end){if(!v::push<drops::Site>(ref mut copy,drops::at(all,k)).is_ok){return .err(2);}k+=1;}return .ok(copy);}
@effects(filesystem,console,environment) fn i32 main(){return switch(execute()){case .ok(p):{p}case .err(_):{7}};}`
}

func TestCandidateAtlasUnsafeSites(t *testing.T) {
	type fixture struct {
		name, root string
		batch      bool
		want       map[string][]semanticsnapshot.Span
	}
	var fixtures []fixture
	const ordinary = "module main;fn i32 id(i32 n){return n;}\n"
	const foreign = "module main;unsafe extern C fn i32 external_value(i32 n);\n"
	for _, item := range []struct {
		name, source string
		sdk          bool
	}{
		{"none", "module main;fn i32 main(){return 0;}", false},
		{"nested-calls", ordinary + "fn i32 main(){unsafe{discard id(id(1));}return id(2);}", false},
		{"empty-scope", ordinary + "fn i32 main(){unsafe{let i32 n=1;discard n+1;}return id(2);}", false},
		{"disjoint-scopes", ordinary + "fn i32 main(){unsafe{discard id(1);}discard id(2);unsafe{discard id(3);}return 0;}", false},
		{"nested-scopes", ordinary + "fn i32 main(){unsafe{discard id(unsafe{id(1)});unsafe{discard id(2);}}return 0;}", false},
		{"unsafe-function", ordinary + "unsafe fn i32 work(i32 n){return id(n);}fn i32 main(){return unsafe{work(1)};}", false},
		{"foreign-call", foreign + "fn i32 main(){return unsafe{external_value(1)};}", false},
		{"foreign-branch", foreign + "fn i32 f(bool b){if(b){return unsafe{external_value(1)};}return unsafe{external_value(2)};}fn i32 main(){return f(true);}", false},
		{"assert-terminal", foreign + "fn i32 main(){unsafe{discard external_value(1);}assert(false);return unsafe{external_value(2)};}", false},
		{"grouped-assert", foreign + "fn i32 main(){assert((false));return unsafe{external_value(2)};}", false},
		{"borrow-address", "module main;fn i32 main(){let u8 n=1;discard unsafe{ref n as ptr u8};return 0;}", false},
		{"mutable-address", "module main;fn i32 main(){var u8 n=1;discard unsafe{ref mut n as ptr mut u8};return 0;}", false},
		{"opaque-address", "module main;fn i32 main(){let i32 n=1;discard unsafe{ref n as ptr void};return 0;}", false},
		{"borrowed-parameter", "module main;unsafe fn ptr u8 raw(ref u8 n){return n as ptr u8;}fn i32 main(){let u8 n=1;discard unsafe{raw(ref n)};return 0;}", false},
		{"record-address", "module main;@repr(C) struct Item{i32 value;}fn i32 main(){let n=Item{.value=1};discard unsafe{ref n as ptr Item};return 0;}", false},
		{"string-address", "module main;fn i32 main(){let value=\"hello\";discard unsafe{value as ptr u8};return 0;}", false},
		{"byte-address", "module main;fn i32 main(){let value=b\"hello\";discard unsafe{value as ptr u8};return 0;}", false},
		{"raw-bytes", "module main;fn i32 main(){let u8 n=0;let ptr u8 p=unsafe{ref n as ptr u8};let ref u8[] data=unsafe{byte_slice::from_raw_parts(ref n,p,1)};discard data;return 0;}", false},
		{"raw-string", "module main;fn i32 main(){let u8 n=0;let ptr u8 p=unsafe{ref n as ptr u8};let ref str data=unsafe{str::from_raw_parts(ref n,p,1)};discard data;return 0;}", false},
		{"raw-cstr", "module main;fn i32 main(){let u8 n=0;let ptr u8 p=unsafe{ref n as ptr u8};let ref CStr data=unsafe{CStr::from_raw_parts(ref n,p,1)};discard data;return 0;}", false},
		{"raw-mut-bytes", "module main;fn i32 main(){var u8 n=0;let ptr mut u8 p=unsafe{ref mut n as ptr mut u8};let ref mut u8[] data=unsafe{byte_slice::from_raw_parts_mut(ref mut n,p,1)};discard data;return 0;}", false},
		{"safe-borrow-and-cast", "module main;fn i32 main(){let u8 n=1;unsafe{let ref u8 p=ref n;let u32 wide=n as u32;discard *p;discard wide;}let ptr void p=null;discard p;return 0;}", false},
		{"defer", ordinary + "fn void mark(i32 n){discard n;}fn i32 main(){unsafe{defer mark(id(1));}return 0;}", false},
		{"generic", "module main;fn T id<T>(T n){return n;}unsafe fn T work<T>(T n){return id<T>(n);}fn i32 main(){unsafe{discard work<u8>(1);discard work<i32>(2);}return 0;}", false},
		{"cleanup", ordinary + "resource Token{i32 value;cleanup{unsafe{discard id(1);}}}fn i32 main(){let t=Token{.value=1};return 0;}", false},
		{"generic-cleanup", ordinary + "resource Token<T>{T value;cleanup{unsafe{discard id(1);}}}fn i32 main(){let t=Token<i32>{.value=1};return 0;}", false},
		{"sdk-vector-unused", candidateAtlasSDKVectorUnusedSource, true},
		{"sdk-vector-used", candidateAtlasSDKVectorUsedSource, true},
		{"sdk-vector-pop", candidateAtlasSDKVectorPopSource, true},
	} {
		root := writeWorkspaceSessionFixture(t, item.source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("unsafe oracle %s: %d %s", item.name, code, &diagnostic)
		}
		want := map[string][]semanticsnapshot.Span{}
		for _, p := range prepared.partitions {
			for _, f := range p.Functions() {
				want[atlasDropKey(f.Function, "", false)] = f.UnsafeSites
			}
			for _, s := range p.Specializations() {
				if s.Aggregate == nil {
					want[atlasDropKey(s.Declaration, s.ID, false)] = s.Function.UnsafeSites
				}
				if s.Cleanup != nil {
					want[atlasDropKey(s.Declaration, s.ID, true)] = s.Cleanup.UnsafeSites
				}
			}
		}
		fixtures = append(fixtures, fixture{item.name, root, item.sdk, want})
	}
	controlRoot := ""
	for _, f := range fixtures {
		for _, sites := range f.want {
			if controlRoot == "" && len(sites) > 0 {
				controlRoot = f.root
			}
		}
	}
	if controlRoot == "" {
		t.Fatal("missing independent nonempty unsafe boundary fixture")
	}
	main := candidateAtlasUnsafeSitesMain(t)
	candidateAtlasProbeSize(t, "project_atlas_source", main, 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				var code int
				var out, stderr string
				if f.batch {
					code, out, stderr = candidateAtlasSDKCLIProcess(t, image, f.root, []string{"batch"})
				} else {
					code, out, stderr = fetchCLIProcess(t, image, f.root, nil, "")
				}
				if code != 0 || stderr != "" {
					t.Fatalf("unsafe collector: %d %.3000s %.500s", code, out, stderr)
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
						t.Fatal("unknown/duplicate unsafe owner", key)
					}
					seen[key] = true
					var actual []semanticsnapshot.Span
					for _, s := range row.Sites {
						actual = append(actual, semanticsnapshot.Span{PackageInstance: s.Package, Source: s.Source, StartByte: s.Start, EndByte: s.End})
					}
					if !slices.Equal(actual, want) {
						t.Fatalf("unsafe sites %s\n got %#v\nwant %#v", key, actual, want)
					}
				}
				if len(seen) != len(f.want) {
					t.Fatalf("unsafe owner coverage: %d/%d", len(seen), len(f.want))
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("unsafe collector changed workspace")
				}
			})
			if t.Failed() {
				return
			}
		}
		for _, mode := range []string{"zero", "past", "budget", "short-work", "exact-work", "more-work", "short-count", "exact-count", "more-count", "over-count", "bulk-budget", "bulk-short-work", "bulk-exact-work", "bulk-more-work", "bulk-short-count", "bulk-exact-count", "bulk-more-count", "bulk-over-count"} {
			code, out, stderr := fetchCLIProcess(t, image, controlRoot, []string{mode}, "")
			want, failure := 0, ""
			if mode == "zero" || mode == "past" || mode == "over-count" || mode == "bulk-over-count" {
				want, failure = 7, "9"
			} else if mode == "budget" || mode == "short-work" || mode == "short-count" || mode == "bulk-budget" || mode == "bulk-short-work" || mode == "bulk-short-count" {
				want, failure = 7, "3"
			}
			if code != want || out != failure || stderr != "" {
				t.Fatalf("unsafe control %s: %d/%d %q/%q %q", mode, code, want, out, failure, stderr)
			}
		}
		for i, root := range []string{fixtures[0].root, controlRoot} {
			expected := i * 7
			code, out, stderr := fetchCLIProcess(t, image, root, []string{"bulk-zero-count"}, "")
			if code != expected || stderr != "" || expected == 0 && out != "" || expected == 7 && out != "3" {
				t.Fatalf("unsafe zero count %d: %d %q %q", i, code, out, stderr)
			}
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph", "project_atlas_unsafe_sites", "project_compiler_source_mode")
}
