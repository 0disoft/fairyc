package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestCandidateAtlasSDKVectorCommand(t *testing.T) {
	main := `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment)fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, verifyCandidateAtlasSDKVectorCommand)
}

func verifyCandidateAtlasSDKVectorCommand(t *testing.T, image string) {
	t.Helper()
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	t.Setenv("PATH", "")
	for _, fixture := range []struct {
		name, source, target string
		full                 bool
	}{
		{"unused", candidateAtlasSDKVectorUnusedSource, "take_slot", false},
		{"used", candidateAtlasSDKVectorUsedSource, "at", true},
		{"pop", candidateAtlasSDKVectorPopSource, "take_slot", false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, fixture.source)
			prepare := func() (preparedAtlasWorkspace, *atlasmap.WorkspaceCallIndex) {
				t.Helper()
				var diagnostic bytes.Buffer
				p, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
				if code != 0 {
					t.Fatalf("vector impact oracle: %d %s", code, &diagnostic)
				}
				index, err := atlasmap.NewWorkspaceCallIndexV3(p.partitions, p.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 8192})
				if err != nil {
					t.Fatal(err)
				}
				return p, index
			}
			prepared, index := prepare()
			partitions := map[string]atlasmap.Partition{}
			var sdk atlasmap.Partition
			var selector string
			for _, p := range prepared.partitions {
				partitions[p.Header().PackageInstance] = p
				for _, s := range p.Symbols() {
					if s.Declaration.Module == "std::vec" && s.Declaration.Name == fixture.target {
						sdk, selector = p, s.Visible
						if fixture.full {
							selector = s.Full
						}
					}
				}
			}
			verifyCandidateAtlasSDKVectorOracle(t, fixture.name, partitions)
			if selector == "" {
				t.Fatal("missing vector impact subject")
			}
			resolved := len(index.Calls())
			args := append([]string{"experimental", "atlas-impact"}, impactArgs(sdk.Header().PackageUID, selector)...)
			args = replaceWorkspaceSessionArgument(args, "--max-calls", strconv.Itoa(resolved))
			query := func(current *atlasmap.WorkspaceCallIndex) {
				t.Helper()
				code, out, diagnostic := candidateAtlasSDKCLIProcess(t, image, root, args)
				if code != 0 || diagnostic != "" {
					t.Fatalf("vector impact command: %d %s %s", code, out, diagnostic)
				}
				var got atlasmap.ImpactResult
				if err := json.Unmarshal([]byte(out), &got); err != nil {
					t.Fatalf("vector impact JSON: %v %s", err, out)
				}
				want, err := current.Impact(sdk.Header().PackageInstance, selector, 8, 64)
				if err != nil {
					t.Fatal(err)
				}
				compareCandidateAtlasAggregateImpact(t, got, want, compiler)
			}
			before := remoteTree(t, root)
			query(index)
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("vector impact changed workspace")
			}
			// Common limits/freshness are checked once for each native profile.
			if fixture.name != "used" {
				return
			}
			for _, limit := range [][2]string{{"--max-packages", "2"}, {"--max-calls", strconv.Itoa(resolved - 1)}} {
				code, out, diagnostic := candidateAtlasSDKCLIProcess(t, image, root, replaceWorkspaceSessionArgument(args, limit[0], limit[1]))
				if code != 2 || out != "" || diagnostic == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
					t.Fatalf("vector %s cutoff: %d %q %q", limit[0], code, out, diagnostic)
				}
			}
			writeWorkspaceFile(t, root, "src/main.fy", fixture.source+"\n")
			stale := remoteTree(t, root)
			code, out, diagnostic := candidateAtlasSDKCLIProcess(t, image, root, args)
			if code != 7 || out != "" || diagnostic == "" || !reflect.DeepEqual(stale, remoteTree(t, root)) {
				t.Fatalf("vector stale source: %d %q %q", code, out, diagnostic)
			}
			_, current := prepare()
			before = remoteTree(t, root)
			query(current)
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("vector relocked query changed workspace")
			}
		})
	}
}
