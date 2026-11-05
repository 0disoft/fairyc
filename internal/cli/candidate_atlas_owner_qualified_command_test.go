package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestCandidateAtlasOwnerQualifiedCommand(t *testing.T) {
	unsupported := candidateAtlasUnsupportedFixtures(t)
	const main = `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment)fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyCandidateAtlasOwnerQualifiedCommand(t, image)
		verifyCandidateAtlasUnsupported(t, image, unsupported)
	})
}

func verifyCandidateAtlasOwnerQualifiedCommand(t *testing.T, image string) {
	t.Helper()
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	t.Setenv("PATH", "")
	for _, fixture := range atlasOwnerQualifiedFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			root := atlasOwnerQualifiedWorkspace(t, fixture)
			prepare := func() (atlasmap.Partition, string, string, *atlasmap.WorkspaceCallIndex) {
				t.Helper()
				var diagnostic bytes.Buffer
				prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
				if code != ExitSuccess {
					t.Fatalf("owner command oracle: %d %s", code, &diagnostic)
				}
				index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
				if err != nil {
					t.Fatal(err)
				}
				for _, partition := range prepared.partitions {
					if partition.Header().PackageUID == fixture.uid {
						for _, symbol := range partition.Symbols() {
							if symbol.Declaration.Name == fixture.declaration {
								return partition, symbol.Visible, symbol.Full, index
							}
						}
					}
				}
				t.Fatal("missing exact qualified command subject")
				return atlasmap.Partition{}, "", "", nil
			}
			partition, visible, full, index := prepare()
			calls := 1
			if fixture.name == "unused-associated" {
				calls = 0
			}
			if len(index.Calls()) != calls {
				t.Fatalf("owner source call inventory: %d want %d", len(index.Calls()), calls)
			}
			arguments := func(selector string) []string {
				args := append([]string{"experimental", "atlas-impact"}, impactArgs(fixture.uid, selector)...)
				return replaceWorkspaceSessionArgument(args, "--max-calls", strconv.Itoa(max(1, calls)))
			}
			query := func(current atlasmap.Partition, selector string, index *atlasmap.WorkspaceCallIndex) {
				t.Helper()
				args := arguments(selector)
				if selector == full {
					args = append(args, "--locked")
				}
				code, output, diagnostic := fetchCLIProcess(t, image, root, args, "")
				var got atlasmap.ImpactResult
				if code != ExitSuccess || diagnostic != "" || json.Unmarshal([]byte(output), &got) != nil {
					t.Fatalf("owner impact: %d %s %s", code, output, diagnostic)
				}
				want, err := index.Impact(current.Header().PackageInstance, selector, 8, 64)
				if err != nil {
					t.Fatal(err)
				}
				compareCandidateAtlasAggregateImpact(t, got, want, compiler)
			}
			before := remoteTree(t, root)
			query(partition, visible, index)
			query(partition, full, index)
			for _, bound := range [][2]string{{"--max-packages", "1"}, {"--max-output-bytes", "1"}} {
				code, output, diagnostic := fetchCLIProcess(t, image, root, replaceWorkspaceSessionArgument(arguments(visible), bound[0], bound[1]), "")
				if code != ExitInvalidInvocation || output != "" || diagnostic == "" {
					t.Fatalf("owner %s cutoff: %d %q %q", bound[0], code, output, diagnostic)
				}
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("owner impact query changed workspace")
			}
			if fixture.name != "shared" {
				return
			}
			writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(fixture.source, "return this.value;", "return this.value+1;", 1))
			stale := remoteTree(t, root)
			code, output, diagnostic := fetchCLIProcess(t, image, root, arguments(visible), "")
			if code != 7 || output != "" || diagnostic == "" || !reflect.DeepEqual(stale, remoteTree(t, root)) {
				t.Fatalf("owner stale lock: %d %q %q", code, output, diagnostic)
			}
			current, currentVisible, currentFull, currentIndex := prepare()
			if currentVisible != visible || currentFull != full || current.Header().PackageInstance == partition.Header().PackageInstance || current.Header().PackageSource == partition.Header().PackageSource || current.Header().InterfaceHash != partition.Header().InterfaceHash {
				t.Fatal("private method body edit lost stable identity or current source provenance")
			}
			after := remoteTree(t, root)
			query(current, visible, currentIndex)
			if !reflect.DeepEqual(after, remoteTree(t, root)) {
				t.Fatal("fresh owner query changed workspace")
			}
		})
	}
}
