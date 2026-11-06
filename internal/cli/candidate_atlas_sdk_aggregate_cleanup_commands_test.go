package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

// Compile the three actual command closures once per profile. The full authored
// main registers the same verifiers separately; this small harness is not that
// full-image gate and supplies no source, map, Complete or aggregate-mode facts.
const atlasSDKAggregateCleanupCommandsMain = `module main;
import project_atlas_cli as atlas;import project_mcp_cli as mcp;import project_edit_cli as edit;
import std::process as process;import std::bytes as bytes;
@effects(filesystem,console,environment)fn i32 main(){
 var u8[64] raw=[0;64];let n=switch(process::arg(2,ref mut raw)){case .ok(p):{p}case .err(_):{return 2;}};let ref u8[] command=ref raw;
 if(bytes::equal(command[..n],b"atlas-mcp")){return mcp::arguments();}
 if(bytes::equal(command[..n],b"atlas-edit")){return edit::arguments();}
 return atlas::arguments();
}`

func TestCandidateAtlasSDKAggregateCleanupCommands(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for bounded SDK/cleanup MCP and edit replay")
	}
	candidateAtlasProbeSize(t, "project_atlas_cli", atlasSDKAggregateCleanupCommandsMain, 2<<20, func(t *testing.T, image string) {
		t.Run("mcp", func(t *testing.T) { verifyCandidateMCPSDKAggregateCleanupCommand(t, image) })
		t.Run("edit", func(t *testing.T) { verifyCandidateEditSDKAggregateCleanupCommandSelection(t, image, "main") })
		t.Run("current-sdk", func(t *testing.T) {
			data, err := os.ReadFile(image)
			if err != nil {
				t.Fatal(err)
			}
			imageHash := fmt.Sprintf("%x", sha256.Sum256(data))
			if expected := os.Getenv("FAIRYC_ATLAS_RESUME_IMAGE_SHA256"); expected != "" && expected != imageHash {
				t.Fatalf("resume image changed: got %s want %s", imageHash, expected)
			}
			root, before := atlasSDKAggregateCleanupOracle(t)
			writeWorkspaceFile(t, root, "src/main.fy", atlasSDKAggregateCleanupEditedSource())
			var diagnostic bytes.Buffer
			current, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != 0 {
				t.Fatalf("current SDK resume oracle: %d %s", code, &diagnostic)
			}
			verifyAtlasSDKAggregateCleanupEditedOracle(t, before, current)
			verifyCandidateAtlasSDKAggregateCleanupCurrentSelection(t, image, root, current, "std::vec")
			t.Logf("current SDK source-sha256=%x image-sha256=%s", sha256.Sum256([]byte(atlasSDKAggregateCleanupEditedSource())), imageHash)
		})
	}, "project_mcp_cli", "project_edit_cli")
}

func verifyCandidateMCPSDKAggregateCleanupCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateMCPSDKCommandFixture(t, image, atlasSDKAggregateCleanupSource, "std::vec", "at")
}

func verifyCandidateEditSDKAggregateCleanupCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateEditSDKAggregateCleanupCommandSelection(t, image, "")
}

func verifyCandidateEditSDKAggregateCleanupCommandSelection(t *testing.T, image, module string) {
	t.Helper()
	_, before := atlasSDKAggregateCleanupOracle(t)
	verifyCandidateEditSDKCommandFixture(t, image, candidateEditSDKFixture{
		source:  atlasSDKAggregateCleanupSource,
		body:    atlasSDKAggregateCleanupEditBody,
		invalid: `{let value=Cell<bool>{.value=42};return 1;}`,
		module:  "std::vec", name: "at",
		verifyCurrent: func(t *testing.T, image, root string, current preparedAtlasWorkspace) {
			data, err := os.ReadFile(filepath.Join(root, "src/main.fy"))
			if err != nil || string(data) != atlasSDKAggregateCleanupEditedSource() {
				t.Fatalf("current cleanup source differs from independent overlay: %v", err)
			}
			verifyAtlasSDKAggregateCleanupEditedOracle(t, before, current)
			verifyCandidateAtlasSDKAggregateCleanupCurrentSelection(t, image, root, current, module)
		},
	})
}

func verifyCandidateAtlasSDKAggregateCleanupCurrentSelection(t *testing.T, image, root string, current preparedAtlasWorkspace, module string) {
	t.Helper()
	index, err := atlasmap.NewWorkspaceCallIndexV3(current.partitions, current.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 8192})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	untouched := remoteTree(t, root)
	queries := 0
	for _, p := range current.partitions {
		for _, s := range p.Symbols() {
			if module != "" && s.Declaration.Module != module {
				continue
			}
			if !(s.Declaration.Module == "std::vec" && s.Declaration.Name == "at" || s.Declaration.Module == "main" && s.Declaration.Name == "cell") {
				continue
			}
			queries++
			args := append([]string{"experimental", "atlas-impact"}, impactArgs(p.Header().PackageUID, s.Full)...)
			args = replaceWorkspaceSessionArgument(args, "--max-calls", strconv.Itoa(len(index.Calls())))
			limit := 3 * time.Minute
			if s.Declaration.Module == "std::vec" {
				// This added SDK-wide query exceeds the legacy fixture's 3m
				// bound under /Od. Keep all existing helper defaults intact.
				limit = 6 * time.Minute
			}
			code, out, diagnostic := candidateAtlasSDKCLIProcessLimit(t, image, root, args, limit)
			var got atlasmap.ImpactResult
			if code != 0 || diagnostic != "" || json.Unmarshal([]byte(out), &got) != nil {
				t.Fatalf("edited cleanup current query: %d %s %s", code, out, diagnostic)
			}
			want, err := index.Impact(p.Header().PackageInstance, s.Full, 8, 64)
			if err != nil {
				t.Fatal(err)
			}
			compareCandidateAtlasAggregateImpact(t, got, want, compiler)
		}
	}
	wantQueries := 2
	if module != "" {
		wantQueries = 1
	}
	if queries != wantQueries || !reflect.DeepEqual(untouched, remoteTree(t, root)) {
		t.Fatalf("edited current queries=%d want %d, or workspace changed", queries, wantQueries)
	}
	t.Logf("edited current queries=%d; resolved calls=%d; compiler=%s", queries, len(index.Calls()), compiler)
}
