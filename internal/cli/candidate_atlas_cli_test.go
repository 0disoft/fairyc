package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Keep the authored main and dispatcher in the real CLI image. No companion is
// built or installed: Atlas has to resolve, check and traverse entirely in Fairy.
func TestCandidateAtlasCLI(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows full CLI; Linux remains batched")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for full CLI replay")
	}
	enumRoots := candidateAtlasEnumOwnerFixtures(t)
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	build, support := candidateParityBuildPartsWithLimit(t, root, map[string]bool{"CloseHandle": true, "CreateFileW": true, "GetFileInformationByHandleEx": true, "GetFinalPathNameByHandleW": true, "GetLastError": true}, 2<<20)
	largest := 0
	for _, source := range build.Sources {
		if len(source) > largest {
			largest = len(source)
		}
	}
	t.Logf("C input bytes: header=%d runtime=%d units=%d largest=%d", len(build.Header), len(support), len(build.Sources), largest)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			imageDir := root
			if retained := os.Getenv("FAIRYC_ATLAS_CLI_ARTIFACTS"); retained != "" {
				imageDir = filepath.Join(retained, profile.Name)
				if err := os.MkdirAll(imageDir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			image := filepath.Join(imageDir, "atlas-cli-"+profile.Name+".exe")
			digest := sha256.New()
			policy, _ := json.Marshal(profile)
			for _, part := range append([]string{string(policy), tc.Name, tc.Path, tc.Version, "stack-reserve=16777216", build.Header, support}, build.Sources...) {
				digest.Write([]byte(part))
				digest.Write([]byte{0})
			}
			sourceID := hex.EncodeToString(digest.Sum(nil))
			type receipt struct{ Source, Image string }
			var saved receipt
			record := filepath.Join(imageDir, "atlas-image.json")
			data, _ := os.ReadFile(record)
			binary, _ := os.ReadFile(image)
			hash := sha256.Sum256(binary)
			if json.Unmarshal(data, &saved) != nil || len(binary) == 0 || saved.Source != sourceID || saved.Image != hex.EncodeToString(hash[:]) {
				artifact, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(image, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				hash = sha256.Sum256(artifact.Bytes)
				data, err = json.Marshal(receipt{sourceID, hex.EncodeToString(hash[:])})
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(record, data, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				t.Log("reuse verified generated-C/profile/toolchain/image digest")
			}
			t.Logf("authored CLI image: source=%s image=%x", sourceID, hash)

			t.Setenv("PATH", "")
			// Keep every original verifier and its ordering. Named groups allow
			// bounded replays against the same freshly authenticated full image;
			// selecting one group does not attest any unselected group.
			for _, check := range []struct {
				name   string
				verify func(*testing.T, string)
			}{
				{"impact-ordinary", verifyCandidateAtlasCommand},
				{"impact-families", verifyCandidateAtlasFamilies},
				{"mcp-ordinary", verifyCandidateMCPCommand},
				{"mcp-aggregates", verifyCandidateMCPAggregateCommand},
				{"mcp-static-operations", verifyCandidateMCPStaticOperationCommand},
				{"mcp-copied-closures", verifyCandidateMCPCopiedClosureCommand},
				{"edit-ordinary", verifyCandidateEditCommand},
				{"edit-publication", verifyCandidateEditCLIApply},
				{"edit-aggregates", verifyCandidateEditAggregateCommand},
				{"edit-static-operations", verifyCandidateEditStaticOperationCommand},
				{"edit-copied-closures", verifyCandidateEditCopiedClosureCommand},
				{"enum-owner", func(t *testing.T, image string) { verifyCandidateAtlasEnumOwners(t, image, enumRoots) }},
				{"impact-generics", verifyCandidateAtlasGenericCommand},
				{"impact-static-operations", verifyCandidateAtlasStaticOperationCommand},
				{"impact-copied-closures", verifyCandidateAtlasCopiedClosureCommand},
				{"impact-aggregates", verifyCandidateAtlasAggregateCommand},
				{"impact-sdk", verifyCandidateAtlasSDKCommand},
				{"impact-sdk-vector", verifyCandidateAtlasSDKVectorCommand},
				{"impact-sdk-aggregates", verifyCandidateAtlasSDKUserAggregateCommand},
				{"mcp-sdk", verifyCandidateMCPSDKCommand},
				{"mcp-sdk-vector", verifyCandidateMCPSDKVectorCommand},
				{"mcp-sdk-aggregates", verifyCandidateMCPSDKUserAggregateCommand},
				{"mcp-sdk-aggregate-cleanup", verifyCandidateMCPSDKAggregateCleanupCommand},
				{"edit-sdk", verifyCandidateEditSDKCommand},
				{"edit-sdk-vector", verifyCandidateEditSDKVectorCommand},
				{"edit-sdk-aggregates", verifyCandidateEditSDKUserAggregateCommand},
				{"edit-sdk-aggregate-cleanup", verifyCandidateEditSDKAggregateCleanupCommand},
				{"help", verifyCandidateAtlasCLIHelp},
				{"catalog-ordinary", verifyCandidateCatalogCommand},
				{"catalog-sources", verifyCandidateCatalogSourceCommand},
				{"catalog-sdk", verifyCandidateCatalogSDKCommand},
				{"impact-owner-qualified", verifyCandidateAtlasOwnerQualifiedCommand},
				{"mcp-owner-qualified", verifyCandidateMCPOwnerQualifiedCommand},
				{"edit-owner-qualified", verifyCandidateEditOwnerQualifiedCommand},
				{"impact-owner-namespaces", verifyCandidateAtlasOwnerNamespaceCommand},
				{"mcp-owner-namespaces", verifyCandidateMCPOwnerNamespaceCommand},
				{"edit-owner-namespaces", verifyCandidateEditOwnerNamespaceCommand},
				{"calls-ordinary", verifyCandidateCallsCommand},
				{"calls-sources", verifyCandidateCallsSourceCommand},
				{"calls-sdk", verifyCandidateCallsSDKCommand},
				{"pack-ordinary", verifyCandidatePackCommand},
				{"pack-sources", verifyCandidatePackSourceCommand},
				{"pack-sdk", verifyCandidatePackSDKCommand},
			} {
				t.Run(check.name, func(t *testing.T) { check.verify(t, image) })
			}
		})
	}
}

func verifyCandidateAtlasCLIHelp(t *testing.T, image string) {
	t.Helper()
	for _, action := range []string{"", "overview", "show", "around"} {
		for _, flag := range []string{"--help", "-h", "help"} {
			args := []string{"experimental", "atlas-pack"}
			if action != "" {
				args = append(args, action)
			}
			args = append(args, flag)
			code, out, diagnostic := fetchCLIProcess(t, image, t.TempDir(), args, "")
			if code != 0 || out != usage || diagnostic != "" {
				t.Fatalf("pack help %v: %d %q %q", args, code, out, diagnostic)
			}
		}
	}
	for _, args := range [][]string{{"--help"}, {"experimental", "--help"}, {"experimental", "fetch-packages", "--help"}, {"experimental", "atlas-impact", "--help"}, {"experimental", "atlas-mcp", "--help"}, {"experimental", "atlas-edit", "--help"}, {"experimental", "atlas-symbols", "--help"}, {"experimental", "atlas-symbols", "list", "--help"}, {"experimental", "atlas-symbols", "list", "-h"}, {"experimental", "atlas-symbols", "list", "help"}, {"experimental", "atlas-calls", "--help"}, {"experimental", "atlas-calls", "list", "--help"}, {"experimental", "atlas-calls", "list", "-h"}, {"experimental", "atlas-calls", "list", "help"}} {
		code, out, diagnostic := fetchCLIProcess(t, image, t.TempDir(), args, "")
		if code != 0 || out != usage || diagnostic != "" {
			t.Fatalf("help %v: %d %q %q", args, code, out, diagnostic)
		}
	}
}

func TestCandidateAtlasEnumOwnerFixturesHaveValidSource(t *testing.T) {
	if len(candidateAtlasEnumOwnerFixtures(t)) == 0 {
		t.Fatal("missing enum-owned Atlas source controls")
	}
}

func TestCandidateAtlasEnumOwnerCommand(t *testing.T) {
	roots := candidateAtlasEnumOwnerFixtures(t)
	const main = `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment)fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyCandidateAtlasEnumOwners(t, image, roots)
	})
}

func candidateAtlasEnumOwnerFixtures(t *testing.T) []string {
	t.Helper()
	var roots []string
	for _, source := range []string{
		`module main;enum State{ready;}fn i32 State::answer(){return 42;}fn i32 main(){return 42;}`,
	} {
		root := writeWorkspaceSessionFixture(t, source)
		if code, _, diagnostic := runForTest([]string{"check"}, root); code != 0 {
			t.Fatalf("enum-owned source must be valid: %d %s", code, diagnostic)
		}
		roots = append(roots, root)
	}
	return roots
}
func verifyCandidateAtlasEnumOwners(t *testing.T, image string, roots []string) {
	t.Helper()
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	for _, root := range roots {
		var diagnostics bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostics, atlasmap.LogicalSchemaV3)
		if code != ExitSuccess {
			t.Fatalf("enum-owned Atlas oracle: %d %s", code, &diagnostics)
		}
		index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
		if err != nil {
			t.Fatal(err)
		}
		before := remoteTree(t, root)
		selected := false
		for _, partition := range prepared.partitions {
			if partition.Header().PackageUID != "example.session.app" {
				continue
			}
			for _, symbol := range partition.Symbols() {
				if symbol.Declaration.Name != "State::answer" {
					continue
				}
				selected = true
				for _, selector := range []string{symbol.Visible, symbol.Full} {
					args := append([]string{"experimental", "atlas-impact"}, impactArgs("example.session.app", selector)...)
					args = append(args, "--locked")
					code, out, diagnostic := fetchCLIProcess(t, image, root, args, "")
					var got atlasmap.ImpactResult
					if code != ExitSuccess || diagnostic != "" || json.Unmarshal([]byte(out), &got) != nil {
						t.Fatalf("enum-owned query: %d %q %q", code, out, diagnostic)
					}
					want, err := index.Impact(partition.Header().PackageInstance, selector, 8, 64)
					if err != nil {
						t.Fatal(err)
					}
					compareCandidateAtlasAggregateImpact(t, got, want, compiler)
				}
			}
		}
		if !selected {
			t.Fatal("missing independently prepared enum-owned subject")
		}
		args := append([]string{"experimental", "atlas-impact"}, impactArgs("example.session.app", "fysym1-sha256-128-"+strings.Repeat("a", 32))...)
		code, out, diagnostic := fetchCLIProcess(t, image, root, args, "")
		if code != 2 || out != "" || diagnostic == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("enum-owned unknown selector or workspace mutation: %d %q %q", code, out, diagnostic)
		}
	}
}
