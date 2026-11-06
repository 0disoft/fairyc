package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

type atlasBorrowOracleSite struct {
	function, selector, expression, kind string
	escapes                              bool
}

type atlasBorrowOracleFixture struct {
	name, source string
	sites        []atlasBorrowOracleSite
}

func atlasBorrowSiteOracleCases() []atlasBorrowOracleFixture {
	type site = atlasBorrowOracleSite
	const prefix = "module main;fn i32 main(){return 0;}"
	return []atlasBorrowOracleFixture{
		{"none", prefix, nil},
		{"shared-local", prefix + "fn void exercise(){let i32 value=7;let ref i32 view=ref value;discard view;}", []site{{"exercise", "ref value", "ref value", "shared", false}}},
		{"mutable-local", prefix + "fn void exercise(){var i32 value=7;let ref mut i32 view=ref mut value;*view=8;}", []site{{"exercise", "ref mut value", "ref mut value", "mutable", false}}},
		{"copied-reference", prefix + "fn void exercise(){let i32 value=7;let ref i32 view=ref value;let ref i32 other=view;discard other;}", []site{{"exercise", "ref value", "ref value", "shared", false}}},
		{"static-bytes", prefix + `fn void exercise(){let ref u8[] data=b"A\0\xff";discard data;}`, []site{{"exercise", `b"A\0\xff"`, `b"A\0\xff"`, "shared", false}}},
		{"static-c-string", prefix + `fn void exercise(){let ref CStr name=c"fairy";discard name;}`, []site{{"exercise", `c"fairy"`, `c"fairy"`, "shared", false}}},
		{"returned-static", prefix + `fn ref u8[] exercise(){return b"bytes";}`, []site{{"exercise", `b"bytes"`, `b"bytes"`, "shared", true}}},
		{"returned-parameter", prefix + "fn ref i32 exercise(ref i32 input){return input;}", []site{{"exercise", "return input;", "input", "shared", true}}},
		{"returned-mutable-parameter", prefix + "fn ref mut i32 exercise(ref mut i32 input){return input;}", []site{{"exercise", "return input;", "input", "mutable", true}}},
		{"returned-forwarded-parameter", prefix + "fn ref i32 exercise(ref i32 input){let ref i32 view=input;return view;}", []site{{"exercise", "return view;", "view", "shared", true}}},
		{"call-bounded", prefix + "fn i32 read(ref i32 input){return *input;}fn i32 exercise(){let i32 value=7;return read(ref value);}", nil},
		{"discarded", prefix + "fn void exercise(){let i32 value=7;discard ref value;}", nil},
		{"after-trap", prefix + "fn void exercise(){let i32 value=7;assert(false);let ref i32 view=ref value;discard view;}", nil},
		{"branches", prefix + "fn void exercise(bool flag){let i32 value=7;if(flag){let ref i32 left=ref value;discard left;}else{let ref i32 right=ref value;discard right;}}", []site{{"exercise", "left=ref value", "ref value", "shared", false}, {"exercise", "right=ref value", "ref value", "shared", false}}},
		{"field", prefix + "struct Pair{i32 first;i32 second;}fn void exercise(){let Pair pair=Pair{.first=1,.second=2};let ref i32 view=ref pair.first;discard view;}", []site{{"exercise", "ref pair.first", "ref pair.first", "shared", false}}},
		{"array-slice", prefix + "fn void exercise(){let i32[2] data=[1,2];let ref i32[] view=ref data[..];discard view;}", []site{{"exercise", "ref data[..]", "ref data[..]", "shared", false}}},
		{"retained-call-result", prefix + "fn ref i32 identity(ref i32 input){return input;}fn void exercise(){let i32 value=7;let ref i32 view=identity(ref value);discard view;}", []site{{"identity", "return input;", "input", "shared", true}, {"exercise", "ref value", "ref value", "shared", false}}},
	}
}

// Fix the original semantic inventory before a Fairy collector exists. Direct
// call/discard borrows are not retained facts; a forwarded parameter may instead
// create an escaping fact at the returned expression's span.
func TestAtlasBorrowSitesOracle(t *testing.T) {
	for _, f := range atlasBorrowSiteOracleCases() {
		t.Run(f.name, func(t *testing.T) {
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(writeWorkspaceSessionFixture(t, f.source), false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("borrow oracle source: %d %s", code, &diagnostic)
			}
			seenMain, count := false, 0
			for _, p := range prepared.partitions {
				data, err := atlasmap.EncodeV3(p, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				p, err = atlasmap.DecodeV3(data, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				for _, function := range p.Functions() {
					if function.Function.Module != "main" {
						continue
					}
					seenMain = seenMain || function.Function.Name == "main"
					var want []semanticsnapshot.Borrow
					for _, site := range f.sites {
						if site.function != function.Function.Name {
							continue
						}
						if strings.Count(f.source, site.selector) != 1 || strings.Count(site.selector, site.expression) != 1 {
							t.Fatalf("ambiguous original borrow selector: %+v", site)
						}
						start := strings.Index(f.source, site.selector) + strings.Index(site.selector, site.expression)
						want = append(want, semanticsnapshot.Borrow{Span: semanticsnapshot.Span{PackageInstance: function.Function.PackageInstance, Source: "main.fy", StartByte: start, EndByte: start + len(site.expression)}, Kind: site.kind, Escapes: site.escapes})
					}
					if !slices.Equal(function.Borrows, want) {
						t.Fatalf("%s semantic borrows: got %+v want original %+v", function.Function.Name, function.Borrows, want)
					}
					count += len(function.Borrows)
				}
			}
			if !seenMain || count != len(f.sites) {
				t.Fatalf("incomplete borrow oracle: main=%v sites=%d want %d", seenMain, count, len(f.sites))
			}
		})
	}
}

func TestAtlasBorrowSitesUnsupportedSource(t *testing.T) {
	for _, f := range []struct{ name, source, diagnostic string }{
		{"dereferenced-reborrow", "fn ref i32 exercise(ref i32 input){let ref i32 view=ref *input;return view;}", "borrow of a non-place expression"},
		{"unborrowed-array-slice", "fn void exercise(){let i32[2] data=[1,2];let ref i32[] view=data[..];discard view;}", "slicing outside executable array views"},
	} {
		t.Run(f.name, func(t *testing.T) {
			var diagnostic bytes.Buffer
			_, code := prepareAtlasWorkspaceSchema(writeWorkspaceSessionFixture(t, "module main;fn i32 main(){return 0;}"+f.source), false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSourceDiagnostics || !strings.Contains(diagnostic.String(), f.diagnostic) {
				t.Fatalf("unsupported borrow source: %d %s", code, &diagnostic)
			}
		})
	}
}
