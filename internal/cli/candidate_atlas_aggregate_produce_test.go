package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func candidateAtlasAggregateProduceMain() string {
	main := candidateAtlasGenericProduceMain()
	main = strings.ReplaceAll(main, "compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)", "compiler::build_complete_aggregate_sources(input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut source_work,ref mut link_work)")
	main = strings.ReplaceAll(main, "schema::prepare(", "schema::prepare_aggregates(")
	main = strings.ReplaceAll(main, "instances::collect(", "instances::collect_aggregates(")
	lines := strings.Split(main, "\n")
	for i, line := range lines {
		if strings.Contains(line, "let observed=switch(") {
			lines[i] = strings.Replace(line, "discard io::write_usize(code as usize)", `if(mode.len==0){discard io::write("instances/");}discard io::write_usize(code as usize)`, 1)
		}
		if strings.Contains(line, "let part=switch(produce::package_generics") {
			lines[i] = strings.Replace(line, "discard io::write_usize(code as usize)", `discard io::write("package/");discard io::write_usize(code as usize)`, 1)
		}
		if strings.Contains(line, "let f=switch(facts::collect") {
			lines[i] = strings.Replace(line, "discard io::write_usize(code as usize)", `discard io::write("facts/");discard io::write_usize(code as usize)`, 1)
		}
	}
	return strings.Join(lines, "\n")
}

func TestCandidateAtlasAggregateProduction(t *testing.T) {
	type fixture struct {
		name, root string
		partitions map[string]atlasmap.Partition
	}
	cases := candidateAtlasAggregateFixtures()
	cases = append(cases,
		candidateAggregateLayoutFixture{name: "concrete repr C", maker: aggregateLayoutMaker + `@repr(C)pub struct Marker<T>{c::int value;}`, root: `module root;import maker as m;fn i32 main(){let marker=m::Marker<i32>{.value=42 as c::int};return marker.value as i32;}`},
		candidateAggregateLayoutFixture{name: "empty generic cleanup", maker: aggregateLayoutMaker + `pub resource Empty<T>{cleanup{}}pub fn Empty<i32> empty(){return Empty<i32>{};}`, root: `module root;import maker as m;fn i32 main(){let empty=m::empty();return 42;}`},
	)
	var fixtures []fixture
	for _, source := range cases {
		root := candidateAggregateCacheWorkspace(t, source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			candidateAtlasAggregateOracleFailure(t, root)
			t.Fatalf("aggregate production oracle %s: %d %s", source.name, code, &diagnostic)
		}
		f := fixture{name: source.name, root: root, partitions: map[string]atlasmap.Partition{}}
		for _, p := range prepared.partitions {
			encoded, err := atlasmap.EncodeV3(p, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			p, err = atlasmap.DecodeV3(encoded, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			f.partitions[p.Header().PackageInstance] = p
		}
		fixtures = append(fixtures, f)
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateAtlasAggregateProduceMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				code, out, stderr := fetchCLIProcess(t, image, f.root, nil, "")
				if code != 0 || stderr != "" {
					t.Fatalf("aggregate production: %d %s %s", code, out, stderr)
				}
				var documents []json.RawMessage
				if err := json.Unmarshal([]byte(out), &documents); err != nil || len(documents) != len(f.partitions) {
					t.Fatalf("aggregate partitions: %v %d, want %d", err, len(documents), len(f.partitions))
				}
				seen := map[string]bool{}
				for _, raw := range documents {
					got, err := atlasmap.DecodeV3(raw, 4<<20)
					if err != nil {
						t.Fatalf("Go rejected aggregate map: %v", err)
					}
					h := got.Header()
					want, ok := f.partitions[h.PackageInstance]
					if !ok || seen[h.PackageInstance] {
						t.Fatal("unexpected or duplicate package", h.PackageInstance)
					}
					seen[h.PackageInstance] = true
					if !reflect.DeepEqual(got.Declarations(), want.Declarations()) || !reflect.DeepEqual(got.Symbols(), want.Symbols()) || h.InterfaceHash != want.Header().InterfaceHash || h.GenericInterfaceHash != want.Header().GenericInterfaceHash {
						t.Fatalf("aggregate source declarations/identities differ %s: got %+v want %+v", h.PackageUID, got.Declarations(), want.Declarations())
					}
					compareAtlasProducedFunctions(t, got.Functions(), want.Functions())
					actual, expected := got.Specializations(), want.Specializations()
					if len(actual) != len(expected) || h.Counts.Specializations != len(expected) {
						t.Fatalf("aggregate implementation count differs %s: %d != %d", h.PackageUID, len(actual), len(expected))
					}
					for i, a := range actual {
						w := expected[i]
						if a.ID != w.ID || a.Declaration != w.Declaration || !reflect.DeepEqual(a.TypeArguments, w.TypeArguments) || !reflect.DeepEqual(a.Aggregate, w.Aggregate) || (a.Cleanup == nil) != (w.Cleanup == nil) {
							t.Fatalf("aggregate specialization differs %s: got %+v want %+v", h.PackageUID, a, w)
						}
						if w.Aggregate == nil {
							compareAtlasProducedFunctions(t, []semanticsnapshot.FunctionSummary{a.Function}, []semanticsnapshot.FunctionSummary{w.Function})
						} else if !reflect.DeepEqual(a.Function, semanticsnapshot.FunctionSummary{}) {
							t.Fatal("aggregate published as a function")
						}
						if a.Cleanup != nil {
							compareAtlasProducedFunctions(t, []semanticsnapshot.FunctionSummary{*a.Cleanup}, []semanticsnapshot.FunctionSummary{*w.Cleanup})
						}
					}
				}
				for _, bad := range []struct{ mode, text string }{{"budget", "3"}, {"missing-coordinate", "7"}} {
					code, out, stderr := fetchCLIProcess(t, image, f.root, []string{bad.mode}, "")
					if code != 7 || out != bad.text || stderr != "" {
						t.Fatalf("%s: %d %q %q", bad.mode, code, out, stderr)
					}
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("aggregate production changed workspace")
				}
			})
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}
