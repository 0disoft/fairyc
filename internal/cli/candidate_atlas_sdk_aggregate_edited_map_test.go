package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
)

// This fills the edited whole-map observation left separate from the existing
// editor/impact replay. Fairy receives only the actual edited workspace, never
// Go maps, a Complete, or a replacement aggregate-mode selector.
func TestCandidateAtlasSDKAggregateEditedMap(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for edited SDK/aggregate native maps")
	}
	root, original := atlasSDKAggregateCleanupOracle(t)
	var previousUser atlasmap.LoadExpectation
	for _, expectation := range original.expectations {
		if expectation.Current.UID == "example.session.app" {
			previousUser = expectation
		}
	}
	if previousUser.Current.UID == "" {
		t.Fatal("missing original user current-source expectation")
	}
	edited := atlasSDKAggregateCleanupEditedSource()
	if edited == atlasSDKAggregateCleanupSource {
		t.Fatal("edited cleanup fixture did not change the source")
	}
	writeWorkspaceFile(t, root, "src/main.fy", edited)
	var diagnostic bytes.Buffer
	current, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess || diagnostic.Len() != 0 {
		t.Fatalf("fresh edited whole-map Go oracle: %d %s", code, &diagnostic)
	}
	// Independent existing expectations require Cell<bool>/cell<bool>, Check<bool>
	// and its three cross-package SDK calls, and retain the same 14 SDK instances,
	// pinned declarations, symbols, original source and interface identities.
	verifyAtlasSDKAggregateCleanupEditedOracle(t, original, current)
	want := make(map[string]atlasmap.Partition, len(current.partitions))
	for _, partition := range current.partitions {
		encoded, err := atlasmap.EncodeV3(partition, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		instance := decoded.Header().PackageInstance
		if _, duplicate := want[instance]; duplicate {
			t.Fatalf("duplicate Go partition %s", instance)
		}
		want[instance] = decoded
	}
	global := candidateAtlasSDKVectorGlobalOracle(t, want)
	t.Logf("exact edited source sha256=%x; packages=%d implementations=%d resolved-calls=%d", sha256.Sum256([]byte(edited)), len(want), global.implementations, global.calls)
	candidateAtlasProbeSize(t, "project_atlas_source", atlasSDKAggregateCleanupMain, 2<<20, func(t *testing.T, image string) {
		imageBytes, err := os.ReadFile(image)
		if err != nil {
			t.Fatal(err)
		}
		compiler := buildidentity.Identity(sha256.Sum256(imageBytes))
		t.Logf("current native observer sha256=%x", sha256.Sum256(imageBytes))
		// Empty PATH prevents a compiler/Go subprocess from producing source facts.
		t.Setenv("PATH", "")
		before := remoteTree(t, root)
		code, out, stderr := candidateAtlasSDKCLIProcess(t, image, root, nil)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("edited production source maps: exit=%d stdout=%s stderr=%s", code, out, stderr)
		}
		var documents []json.RawMessage
		if err := json.Unmarshal([]byte(out), &documents); err != nil || len(documents) != len(want) {
			t.Fatalf("edited native partitions: %v count=%d want=%d", err, len(documents), len(want))
		}
		seen := make(map[string]bool, len(want))
		for _, raw := range documents {
			got, err := atlasmap.DecodeV3(raw, 4<<20)
			if err != nil {
				t.Fatalf("edited native whole map rejected by Go: %v", err)
			}
			h := got.Header()
			expected, ok := want[h.PackageInstance]
			if !ok || seen[h.PackageInstance] {
				t.Fatalf("unexpected/duplicate edited native partition: %s", h.PackageInstance)
			}
			if h.CompilerBuild != compiler.String() || !reflect.DeepEqual(got.Files(), expected.Files()) || !reflect.DeepEqual(got.Modules(), expected.Modules()) {
				t.Fatalf("edited map compiler/file/module records differ: %s", h.PackageUID)
			}
			seen[h.PackageInstance] = true
			expectation, ok := current.expectations[h.PackageInstance]
			if !ok {
				t.Fatalf("missing independent current-source expectation: %s", h.PackageInstance)
			}
			if err := atlasmap.ValidateCurrent(got, expectation.Current); err != nil {
				t.Fatalf("edited native map is not bound to fresh source: %s: %v", h.PackageUID, err)
			}
			verifyCandidateAtlasSDKVectorPartition(t, got, expected)
			verifyCandidateAtlasSDKVectorCompleteness(t, got, expected, global)
			if h.PackageUID == "example.session.app" {
				if err := atlasmap.ValidateCurrent(got, previousUser.Current); !errors.Is(err, atlasmap.ErrStaleMap) {
					t.Fatalf("edited user map vs original source: %v source=%s -> %s instance=%s -> %s", err, previousUser.Current.Source, h.PackageSource, previousUser.Current.Instance, h.PackageInstance)
				}
			}
			t.Logf("whole edited map matched %s: declarations=%d functions=%d implementations=%d relations=%d completeness=%d", h.PackageUID, len(got.Declarations()), len(got.Functions()), len(got.Specializations()), h.Counts.Relations, len(got.Completeness()))
		}
		if len(seen) != len(want) || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatal("missing edited map or native source loader changed the workspace")
		}
		published, err := os.ReadFile(filepath.Join(root, "src", "main.fy"))
		if err != nil || !bytes.Equal(published, []byte(edited)) {
			t.Fatalf("edited source changed during map production: %v", err)
		}
	}, "project_atlas_cli")
}
