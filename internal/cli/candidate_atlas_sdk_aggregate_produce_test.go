package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func candidateAtlasSDKUserAggregateProduceMain(t *testing.T) string {
	t.Helper()
	main := candidateAtlasSDKVectorProduceMain(t)
	// Retain compiler phase, source range and original stage for composition
	// failures; the product diagnostic contract is unchanged.
	if strings.Count(main, `discard io::write_usize(p.stage as usize);`) != 1 {
		t.Fatal("SDK aggregate probe requires one compiler diagnostic port")
	}
	main = strings.Replace(main, `discard io::write_usize(p.stage as usize);`, `discard io::write_usize(p.phase as usize);discard io::write("/");discard io::write_usize(p.module_index);discard io::write("/");discard io::write_usize(p.start);discard io::write("-");discard io::write_usize(p.end);discard io::write("/");discard io::write_usize(p.stage as usize);`, 1)
	for _, replacement := range []struct {
		from, to string
		count    int
	}{
		{"compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)", "compiler::build_complete_aggregate_sources(input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut source_work,ref mut link_work)", 1},
		{"schema::prepare(", "schema::prepare_aggregates(", 1},
		{"instances::collect(", "instances::collect_aggregates(", 2},
	} {
		if strings.Count(main, replacement.from) != replacement.count {
			t.Fatalf("SDK aggregate probe requires %d exact ports: %q", replacement.count, replacement.from)
		}
		main = strings.ReplaceAll(main, replacement.from, replacement.to)
	}
	return main
}

func TestCandidateAtlasSDKUserAggregateProduction(t *testing.T) {
	type fixture struct {
		name, root string
		partitions map[string]atlasmap.Partition
	}
	var fixtures []fixture
	for _, source := range candidateAtlasSDKUserAggregateFixtures() {
		root := writeWorkspaceSessionFixture(t, source.source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != ExitSuccess {
			candidateAtlasSDKUserAggregateOracleFailure(t, root)
			t.Fatalf("SDK aggregate oracle: %d %s", code, &diagnostic)
		}
		f := fixture{name: source.name, root: root, partitions: map[string]atlasmap.Partition{}}
		for _, partition := range prepared.partitions {
			encoded, err := atlasmap.EncodeV3(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			f.partitions[decoded.Header().PackageInstance] = decoded
		}
		fixtures = append(fixtures, f)
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateAtlasSDKUserAggregateProduceMain(t), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				code, out, stderr := candidateAtlasSDKCLIProcess(t, image, f.root, nil)
				if code != 0 || stderr != "" {
					t.Fatalf("SDK/user aggregate production: %d %s %s", code, out, stderr)
				}
				var documents []json.RawMessage
				if err := json.Unmarshal([]byte(out), &documents); err != nil || len(documents) != len(f.partitions) {
					t.Fatalf("SDK aggregate partitions: %v %d, want %d", err, len(documents), len(f.partitions))
				}
				global := candidateAtlasSDKVectorGlobalOracle(t, f.partitions)
				seen := map[string]bool{}
				for _, raw := range documents {
					got, err := atlasmap.DecodeV3(raw, 4<<20)
					if err != nil {
						t.Fatalf("Go rejected mixed SDK aggregate map: %v", err)
					}
					instance := got.Header().PackageInstance
					want, ok := f.partitions[instance]
					if !ok || seen[instance] {
						t.Fatalf("unexpected or duplicate mixed package %s", instance)
					}
					seen[instance] = true
					verifyCandidateAtlasSDKVectorPartition(t, got, want)
					verifyCandidateAtlasSDKVectorCompleteness(t, got, want, global)
				}
				if f.name == "SDK Vec and struct" {
					for _, bad := range []struct{ mode, output string }{
						{"source-budget", "instances/3"},
						{"export-budget", "encode/3"},
						{"validation-budget", "ready/3"},
						{"join-budget", "join/3"},
						{"missing-coordinate", "instances/7"},
					} {
						code, out, stderr := candidateAtlasSDKCLIProcess(t, image, f.root, []string{bad.mode})
						if code != 7 || out != bad.output || stderr != "" {
							t.Fatalf("mixed SDK aggregate %s: %d %q %q, want 7 %q", bad.mode, code, out, stderr, bad.output)
						}
					}
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("mixed SDK aggregate producer changed workspace")
				}
			})
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}
