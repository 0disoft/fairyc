package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestCandidateAtlasOwnerQualifiedProduction(t *testing.T) {
	runCandidateAtlasOwnerQualifiedProduction(t, atlasOwnerQualifiedFixtures())
}

func TestCandidateAtlasOwnerQualifiedNamespaceProduction(t *testing.T) {
	runCandidateAtlasOwnerQualifiedProduction(t, []atlasOwnerQualifiedFixture{
		{name: "two-owners", source: atlasOwnerQualifiedNamespaceSource(false)},
		{name: "ordinary-same-leaf", source: atlasOwnerQualifiedNamespaceSource(true)},
	})
}

func runCandidateAtlasOwnerQualifiedProduction(t *testing.T, sources []atlasOwnerQualifiedFixture) {
	t.Helper()
	type fixture struct {
		name, root string
		partitions map[string]atlasmap.Partition
		global     candidateAtlasSDKVectorGlobal
	}
	var fixtures []fixture
	for _, source := range sources {
		root := atlasOwnerQualifiedWorkspace(t, source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != ExitSuccess {
			t.Fatalf("owner-qualified producer oracle: %d %s", code, &diagnostic)
		}
		f := fixture{name: source.name, root: root, partitions: map[string]atlasmap.Partition{}}
		for _, partition := range prepared.partitions {
			encoded, err := atlasmap.EncodeV3(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			partition, err = atlasmap.DecodeV3(encoded, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			f.partitions[partition.Header().PackageInstance] = partition
			for _, function := range partition.Functions() {
				for _, call := range function.Calls {
					if call.TargetKind == semanticsnapshot.CallTargetResolved {
						f.global.calls++
					} else {
						f.global.external++
					}
				}
			}
		}
		fixtures = append(fixtures, f)
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateAtlasGenericProduceMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				code, output, diagnostic := fetchCLIProcess(t, image, f.root, nil, "")
				if code != ExitSuccess || diagnostic != "" {
					t.Fatalf("Fairy owner-qualified production: %d %s %s", code, output, diagnostic)
				}
				var documents []json.RawMessage
				if err := json.Unmarshal([]byte(output), &documents); err != nil || len(documents) != len(f.partitions) {
					t.Fatalf("owner-qualified partitions: %v; got %d want %d", err, len(documents), len(f.partitions))
				}
				seen := map[string]bool{}
				for _, data := range documents {
					got, err := atlasmap.DecodeV3(data, 4<<20)
					if err != nil {
						t.Fatalf("Go rejected Fairy owner-qualified map: %v", err)
					}
					instance := got.Header().PackageInstance
					want, ok := f.partitions[instance]
					if !ok || seen[instance] {
						t.Fatal("unexpected or duplicate owner-qualified partition", instance)
					}
					seen[instance] = true
					verifyCandidateAtlasSDKVectorPartition(t, got, want)
					verifyCandidateAtlasSDKVectorCompleteness(t, got, want, f.global)
				}
				if f.name == "shared" {
					for _, refusal := range []struct{ mode, output string }{{"budget", "3"}, {"missing-coordinate", "7"}} {
						code, output, diagnostic := fetchCLIProcess(t, image, f.root, []string{refusal.mode}, "")
						if code != 7 || output != refusal.output || diagnostic != "" {
							t.Fatalf("owner-qualified %s: %d %q %q", refusal.mode, code, output, diagnostic)
						}
					}
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("owner-qualified producer changed workspace")
				}
			})
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}
