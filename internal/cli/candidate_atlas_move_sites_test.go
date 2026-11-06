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

func candidateAtlasMoveSitesMain(t *testing.T) string {
	t.Helper()
	main := candidateAtlasDropSitesMain()
	main = strings.Replace(main, "import project_atlas_drop_sites as drops;", "import project_atlas_move_sites as drops;", 1)
	start := strings.Index(main, " var usize combined_work=")
	if start < 0 {
		t.Fatal("missing combined probe start")
	}
	stop := strings.Index(main[start:], " if(bytes::equal(mode,b\"bulk-budget\")")
	if stop < 0 {
		t.Fatal("missing combined probe end")
	}
	main = main[:start] + main[start+stop:]
	start = strings.Index(main, "  let r=drops::range(ref all,frame-1);")
	if start < 0 {
		t.Fatal("missing batch comparison start")
	}
	stop = strings.Index(main[start:], "\n  if(bytes::equal(mode,")
	if stop < 0 {
		t.Fatal("missing batch comparison end")
	}
	main = main[:start] + `  let r=drops::range(ref all,frame-1);if(r.end-r.start!=v::len<drops::Site>(ref sites)){return 7;}var usize n=0;while(n<v::len<drops::Site>(ref sites)){let a=drops::at(ref all,r.start+n);let b=*v::at<drops::Site>(ref sites,n);if(a.start!=b.start || a.end!=b.end){return 7;}n+=1;}` + main[start+stop:]
	main = strings.Replace(main, `|| bytes::equal(mode,b"bulk-over-count")){`, `|| bytes::equal(mode,b"bulk-over-count") || bytes::equal(mode,b"bulk-zero-count")){`, 1)
	main = strings.Replace(main, `else if(bytes::equal(mode,b"bulk-over-count")){maximum=65537;}`, `else if(bytes::equal(mode,b"bulk-over-count")){maximum=65537;}else if(bytes::equal(mode,b"bulk-zero-count")){maximum=0;}`, 1)
	main = strings.Replace(main, `  if(bytes::equal(mode,b"zero")`, `  if((bytes::equal(mode,b"short-count") || bytes::equal(mode,b"exact-count") || bytes::equal(mode,b"more-count")) && v::len<drops::Site>(ref sites)==0){frame+=1;continue;}
  if(bytes::equal(mode,b"zero")`, 1)
	return main
}

func TestCandidateAtlasMoveSites(t *testing.T) {
	type fixture struct {
		name, root string
		want       map[string][]semanticsnapshot.Span
	}
	var fixtures []fixture
	for _, item := range atlasMoveCollectorCases() {
		root := writeWorkspaceSessionFixture(t, item.source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("move oracle %s: %d %s", item.name, code, &diagnostic)
		}
		want := map[string][]semanticsnapshot.Span{}
		for _, part := range prepared.partitions {
			for _, f := range part.Functions() {
				want[atlasDropKey(f.Function, "", false)] = f.MoveSites
			}
			for _, s := range part.Specializations() {
				if s.Aggregate == nil {
					want[atlasDropKey(s.Declaration, s.ID, false)] = s.Function.MoveSites
				}
				if s.Cleanup != nil {
					want[atlasDropKey(s.Declaration, s.ID, true)] = s.Cleanup.MoveSites
				}
			}
		}
		fixtures = append(fixtures, fixture{item.name, root, want})
	}
	// The development main uses ? only for test-owned JSON assembly. Keep its
	// typed failure handler outside the compiler source under test.
	main := candidateAtlasMoveSitesMain(t)
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
					t.Fatalf("move collector: %d %.3000s %.500s", code, out, stderr)
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
						t.Fatalf("unknown/duplicate move owner %s", key)
					}
					seen[key] = true
					var actual []semanticsnapshot.Span
					for _, s := range row.Sites {
						actual = append(actual, semanticsnapshot.Span{PackageInstance: s.Package, Source: s.Source, StartByte: s.Start, EndByte: s.End})
					}
					if !slices.Equal(actual, want) {
						t.Fatalf("move sites %s\n got %#v\nwant %#v", key, actual, want)
					}
				}
				if len(seen) != len(f.want) {
					t.Fatalf("move owner coverage: %d/%d", len(seen), len(f.want))
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("collector changed workspace")
				}
			})
			if t.Failed() {
				return
			}
		}
		for _, control := range []struct {
			fixture, code int
			output        string
		}{{0, 0, ""}, {2, 7, "3"}} {
			code, out, stderr := fetchCLIProcess(t, image, fixtures[control.fixture].root, []string{"bulk-zero-count"}, "")
			if code != control.code || out != control.output || stderr != "" {
				t.Fatalf("move empty/nonempty zero cap: %d %q %q", code, out, stderr)
			}
		}
		for _, mode := range []string{"zero", "past", "budget", "short-work", "exact-work", "more-work", "short-count", "exact-count", "more-count", "over-count", "bulk-budget", "bulk-short-work", "bulk-exact-work", "bulk-more-work", "bulk-short-count", "bulk-exact-count", "bulk-more-count", "bulk-over-count"} {
			code, out, stderr := fetchCLIProcess(t, image, fixtures[2].root, []string{mode}, "")
			want := 0
			failure := ""
			if mode == "zero" || mode == "past" || mode == "over-count" || mode == "bulk-over-count" {
				want, failure = 7, "9"
			} else if mode == "budget" || mode == "short-work" || mode == "short-count" || mode == "bulk-budget" || mode == "bulk-short-work" || mode == "bulk-short-count" {
				want, failure = 7, "3"
			}
			if code != want || out != failure || stderr != "" {
				t.Fatalf("move control %s: %d/%d %q/%q %q", mode, code, want, out, failure, stderr)
			}
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph", "project_atlas_move_sites", "project_compiler_source_mode")
}

func atlasMoveCollectorCases() []atlasMoveOracleFixture {
	const resource = "module main;resource Token{i32 value;cleanup{}}\n"
	return append(append(atlasMoveSiteOracleCases(), atlasMoveImplicitBorrowCases()...), []atlasMoveOracleFixture{
		{name: "reassign", source: resource + "fn i32 main(){var first=Token{.value=1};let second=Token{.value=2};first=second;return first.value;}"},
		{name: "consuming-method", source: resource + "fn void Token::finish(Token this){}fn i32 main(){let token=Token{.value=1};token.finish();return 0;}"},
		{name: "borrowing-method", source: resource + "fn i32 Token::read(ref Token this){return this.value;}fn i32 main(){let token=Token{.value=1};let view=ref token;return view.read();}"},
		{name: "mutable-method", source: resource + "fn void Token::set(ref mut Token this,i32 n){this.value=n;}fn i32 main(){var token=Token{.value=1};let view=ref mut token;view.set(3);return token.value;}"},
		{name: "block-result", source: resource + "fn i32 main(){let other={let token=Token{.value=1};token};return other.value;}"},
		{name: "block-external", source: resource + "fn i32 main(){let token=Token{.value=1};let other={token};return other.value;}"},
		{name: "ignored-block", source: resource + "fn i32 main(){discard {let token=Token{.value=1};token};return 0;}"},
		{name: "ignored-external-block", source: resource + "fn i32 main(){let token=Token{.value=1};discard {token};return 0;}"},
		{name: "propagate-block-live", source: resource + "fn Result<i32,i32> work(){let Result<Token,i32> result=.ok(Token{.value=1});let token={let inner=Token{.value=2};result}?;return .ok(token.value);}fn i32 main(){discard work();return 0;}"},
		{name: "propagate-block-moved", source: resource + "fn void finish(Token inner){}fn Result<i32,i32> work(){let Result<Token,i32> result=.ok(Token{.value=1});let token={let inner=Token{.value=2};finish(inner);result}?;return .ok(token.value);}fn i32 main(){discard work();return 0;}"},
		{name: "switch-result", source: resource + "fn Token take(Option<Token> maybe){return switch(maybe){case .some(token):{token}case .none:{Token{.value=0}}};}fn i32 main(){let Option<Token> maybe=.some(Token{.value=1});let token=take(maybe);return token.value;}"},
		{name: "if-result", source: resource + "fn Token choose(bool flag){return if(flag){let left=Token{.value=1};left}else{let right=Token{.value=2};right};}fn i32 main(){let token=choose(false);return token.value;}"},
		{name: "else-if-result", source: resource + "fn Token choose(bool flag,bool second){return if(flag){let left=Token{.value=1};left}else if(second){let middle=Token{.value=2};middle}else{let right=Token{.value=3};right};}fn i32 main(){let token=choose(false,true);return token.value;}"},
		{name: "if-external", source: resource + "fn Token choose(bool flag,Token token){return if(flag){token}else{token};}fn i32 main(){let token=choose(false,Token{.value=1});return token.value;}"},
		{name: "if-external-scoped", source: resource + "fn Token choose(bool flag,Token token){return if(flag){let other=Token{.value=1};token}else{let other=Token{.value=2};token};}fn i32 main(){let token=choose(false,Token{.value=1});return token.value;}"},
		{name: "loop", source: resource + "fn i32 main(){var i32 total=0;while(total<2){let token=Token{.value=1};let other=token;total+=other.value;if(total>1){break;}}return total;}"},
		{name: "generic", source: "module main;resource Token<T>{T value;cleanup{}}pub fn i32 move<T>(T value){let token=Token<T>{.value=value};let other=token;return 0;}fn i32 main(){discard move<i32>(1);return move<u8>(2);}"},
		{name: "cleanup", source: resource + "resource Wrap<T>{T marker;cleanup{let token=Token{.value=1};let other=token;}}fn i32 main(){let value=Wrap<i32>{.marker=1};return 0;}"},
		{name: "nested-tag", source: resource + "fn i32 main(){let token=Token{.value=1};let Option<Token> maybe=.some(token);let Result<Option<Token>,Token> result=.ok(maybe);discard result;return 0;}"},
		{name: "zero-field", source: "module main;resource Ticket{cleanup{}}fn i32 main(){let first=Ticket{};let second=first;return 0;}"},
	}...)
}

func TestAtlasMoveCollectionSources(t *testing.T) {
	for _, f := range atlasMoveCollectorCases() {
		t.Run(f.name, func(t *testing.T) {
			var diagnostic bytes.Buffer
			if _, code := prepareAtlasWorkspaceSchema(writeWorkspaceSessionFixture(t, f.source), false, &diagnostic, atlasmap.LogicalSchemaV3); code != ExitSuccess {
				t.Fatalf("move collection source: %d %s", code, &diagnostic)
			}
		})
	}
}

// Implicit receiver borrows must retain independent Go move observations;
// successful checking alone must not fabricate an empty observation.
func atlasMoveImplicitBorrowCases() []atlasMoveOracleFixture {
	const resource = "module main;resource Token{i32 value;cleanup{}}\n"
	return []atlasMoveOracleFixture{
		{name: "shared-owner", source: resource + "fn i32 Token::read(ref Token this){return this.value;}fn i32 main(){let token=Token{.value=1};return token.read();}"},
		{name: "exclusive-owner", source: resource + "fn void Token::set(ref mut Token this,i32 n){this.value=n;}fn i32 main(){var token=Token{.value=1};token.set(3);return token.value;}"},
	}
}
