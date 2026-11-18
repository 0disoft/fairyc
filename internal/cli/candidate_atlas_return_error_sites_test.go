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

func TestCandidateAtlasReturnErrorSites(t *testing.T) {
	type fixture struct {
		name, root string
		want       map[string][]semanticsnapshot.Span
	}
	var fixtures []fixture
	for _, item := range atlasReturnErrorCollectorCases() {
		root := writeWorkspaceSessionFixture(t, item.source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("return-error oracle %s: %d %s", item.name, code, &diagnostic)
		}
		want := map[string][]semanticsnapshot.Span{}
		for _, part := range prepared.partitions {
			for _, f := range part.Functions() {
				want[atlasDropKey(f.Function, "", false)] = f.ReturnErrorSites
			}
			for _, s := range part.Specializations() {
				if s.Aggregate == nil {
					want[atlasDropKey(s.Declaration, s.ID, false)] = s.Function.ReturnErrorSites
				}
				if s.Cleanup != nil {
					want[atlasDropKey(s.Declaration, s.ID, true)] = s.Cleanup.ReturnErrorSites
				}
			}
		}
		fixtures = append(fixtures, fixture{item.name, root, want})
	}
	// The development main uses ? only for test-owned JSON assembly. Keep its
	// typed failure handler outside the compiler source under test.
	main := candidateAtlasObservedSitesMain(t, "project_atlas_return_error_sites")
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
				code, out, stderr := candidateAtlasSDKCLIProcess(t, image, f.root, nil)
				if code != 0 || stderr != "" {
					t.Fatalf("return-error collector: %d %.3000s %.500s", code, out, stderr)
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
						t.Fatalf("unknown/duplicate return-error owner %s", key)
					}
					seen[key] = true
					var actual []semanticsnapshot.Span
					for _, s := range row.Sites {
						actual = append(actual, semanticsnapshot.Span{PackageInstance: s.Package, Source: s.Source, StartByte: s.Start, EndByte: s.End})
					}
					if !slices.Equal(actual, want) {
						t.Fatalf("return-error sites %s\n got %#v\nwant %#v", key, actual, want)
					}
				}
				if len(seen) != len(f.want) {
					t.Fatalf("return-error owner coverage: %d/%d", len(seen), len(f.want))
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
			for _, control := range []struct {
				name          string
				fixture, code int
				output        string
			}{{"bulk-zero-count-empty", 0, 0, ""}, {"bulk-zero-count-nonempty", 2, 7, "3"}} {
				t.Run(control.name, func(t *testing.T) {
					code, out, stderr := candidateAtlasSDKCLIProcess(t, image, fixtures[control.fixture].root, []string{"bulk-zero-count"})
					if code != control.code || out != control.output || stderr != "" {
						t.Fatalf("return-error empty/nonempty zero cap: %d %q %q", code, out, stderr)
					}
				})
			}
			for _, mode := range []string{"zero", "past", "budget", "short-work", "exact-work", "more-work", "short-count", "exact-count", "more-count", "over-count", "bulk-budget", "bulk-short-work", "bulk-exact-work", "bulk-more-work", "bulk-short-count", "bulk-exact-count", "bulk-more-count", "bulk-over-count"} {
				t.Run(mode, func(t *testing.T) {
					code, out, stderr := candidateAtlasSDKCLIProcess(t, image, fixtures[2].root, []string{mode})
					want := 0
					failure := ""
					if mode == "zero" || mode == "past" || mode == "over-count" || mode == "bulk-over-count" {
						want, failure = 7, "9"
					} else if mode == "budget" || mode == "short-work" || mode == "short-count" || mode == "bulk-budget" || mode == "bulk-short-work" || mode == "bulk-short-count" {
						want, failure = 7, "3"
					}
					if code != want || out != failure || stderr != "" {
						t.Fatalf("return-error control %s: %d/%d %q/%q %q", mode, code, want, out, failure, stderr)
					}
				})
			}
		})
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph", "project_atlas_return_error_sites", "project_compiler_source_mode")
}

func atlasReturnErrorCollectorCases() []atlasReturnErrorOracleFixture {
	const prefix = "module main;fn Result<i32,i32> read(){return .ok(7);}fn i32 main(){return 0;}"
	return append(atlasReturnErrorSiteOracleCases(), []atlasReturnErrorOracleFixture{
		{name: "parenthesized-operand", source: prefix + "fn Result<i32,i32> exercise(){let n=(read())?;return .ok(n);}"},
		{name: "grouped-propagation", source: prefix + "fn Result<i32,i32> exercise(){let n=(read()?);return .ok(n);}"},
		{name: "logical-condition", source: prefix + "fn Result<i32,i32> exercise(bool flag){if(flag && read()?==7){return .ok(1);}return .ok(0);}"},
		{name: "generic", source: prefix + "pub fn Result<T,i32> wrapped<T>(T value){return .ok(value);}pub fn Result<T,i32> forward<T>(T value){let n=wrapped<T>(value)?;return .ok(n);}fn void exercise(){discard forward<i32>(7);discard forward<u8>(8);}"},
		{name: "cleanup-owner", source: prefix + "resource Token<T>{T value;cleanup{discard read();}}fn Result<i32,i32> exercise(){let token=Token<i32>{.value=1};let n=read()?;return .ok(n);}"},
	}...)
}

func TestAtlasReturnErrorCollectionSources(t *testing.T) {
	for _, f := range atlasReturnErrorCollectorCases() {
		t.Run(f.name, func(t *testing.T) {
			var diagnostic bytes.Buffer
			if _, code := prepareAtlasWorkspaceSchema(writeWorkspaceSessionFixture(t, f.source), false, &diagnostic, atlasmap.LogicalSchemaV3); code != ExitSuccess {
				t.Fatalf("independent return-error source admission: %d %s", code, &diagnostic)
			}
		})
	}
}
