package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

type atlasAllocationOracleFixture struct {
	name, source string
	sites        []string
}

func atlasAllocationSiteOracleCases() []atlasAllocationOracleFixture {
	const prefix = "module main;import std::alloc as alloc;"
	const entry = "fn i32 main(){return 0;}"
	return []atlasAllocationOracleFixture{
		{"none", prefix + entry, nil},
		{"place", prefix + "fn void exercise(Region r){discard r.place(7);}" + entry, []string{"r.place(7)"}},
		{"two-places", prefix + "fn void exercise(Region r){discard r.place(7);discard r.place(8);}" + entry, []string{"r.place(7)", "r.place(8)"}},
		{"branches", prefix + "fn void exercise(Region r,bool flag){if(flag){discard r.place(7);}else{discard r.place(8);}}" + entry, []string{"r.place(7)", "r.place(8)"}},
		{"loop", prefix + "fn void exercise(Region r){var i32 i=0;while(i<2){discard r.place(i);i+=1;}}" + entry, []string{"r.place(i)"}},
		{"lexical", prefix + "fn void exercise(alloc::Arena owned){var alloc::Arena arena=owned;region r in ref mut arena{discard r.place(7);}}" + entry, []string{"r.place(7)"}},
		{"zeroed-bytes", prefix + "fn void exercise(Region r){discard r.alloc_zeroed_bytes(0);}" + entry, []string{"r.alloc_zeroed_bytes(0)"}},
		{"copy-bytes", prefix + "fn void exercise(Region r){discard r.copy_bytes(b\"ab\");}" + entry, []string{"r.copy_bytes(b\"ab\")"}},
		{"before-later-trap", prefix + "fn void take(Result<ref i32,alloc::AllocError> value,i32 n){discard value;}fn void exercise(Region r){take(r.place(7),{assert(false);0});}" + entry, []string{"r.place(7)"}},
		{"after-earlier-trap", prefix + "fn void take(i32 n,Result<ref i32,alloc::AllocError> value){discard value;}fn void exercise(Region r){take({assert(false);0},r.place(7));}" + entry, nil},
		{"trapping-operand", prefix + "fn void exercise(Region r){discard r.place({assert(false);7});}" + entry, nil},
		{"trapping-byte-count", prefix + "fn void exercise(Region r){discard r.alloc_zeroed_bytes({assert(false);0});}" + entry, nil},
		{"trapping-byte-view", prefix + "fn void exercise(Region r){discard r.copy_bytes({assert(false);b\"ab\"});}" + entry, nil},
		{"heap-is-not-region", prefix + "fn void exercise(){discard alloc::create();}" + entry, nil},
	}
}

func TestAtlasAllocationSitesOracle(t *testing.T) {
	for _, f := range atlasAllocationSiteOracleCases() {
		t.Run(f.name, func(t *testing.T) {
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(writeWorkspaceSessionFixture(t, f.source), false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("allocation source: %d %s", code, &diagnostic)
			}
			seen, sites := false, 0
			for _, partition := range prepared.partitions {
				encoded, err := atlasmap.EncodeV3(partition, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				for _, summary := range decoded.Functions() {
					if summary.Function.Module != "main" {
						continue
					}
					seen = seen || summary.Function.Name == "main"
					var want []semanticsnapshot.Span
					if summary.Function.Name == "exercise" {
						for _, text := range f.sites {
							if strings.Count(f.source, text) != 1 {
								t.Fatalf("ambiguous allocation source selector %q", text)
							}
							start := strings.Index(f.source, text)
							want = append(want, semanticsnapshot.Span{PackageInstance: summary.Function.PackageInstance, Source: "main.fy", StartByte: start, EndByte: start + len(text)})
						}
					}
					if !slices.Equal(summary.AllocationSites, want) {
						t.Fatalf("%s allocation sites: got %+v want original calls %+v", summary.Function.Name, summary.AllocationSites, want)
					}
					sites += len(summary.AllocationSites)
				}
			}
			if !seen || sites != len(f.sites) {
				t.Fatalf("incomplete allocation oracle: main=%v sites=%d want %d", seen, sites, len(f.sites))
			}
		})
	}
}
