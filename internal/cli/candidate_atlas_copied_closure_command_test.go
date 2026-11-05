package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestCandidateAtlasCopiedClosureCommand(t *testing.T) {
	unsupported := candidateAtlasUnsupportedFixtures(t)
	main := `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment)fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyCandidateAtlasCopiedClosureCommand(t, image)
		verifyCandidateAtlasUnsupported(t, image, unsupported)
	})
}

func verifyCandidateAtlasCopiedClosureCommand(t *testing.T, image string) {
	t.Helper()
	source := atlasCopiedClosureSource(t, "mixed-owners")
	root := writeWorkspaceSessionFixture(t, source)
	prepare := func() (atlasmap.Partition, *atlasmap.WorkspaceCallIndex) {
		t.Helper()
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != ExitSuccess {
			t.Fatalf("closure command oracle: %d %s", code, &diagnostic)
		}
		index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
		if err != nil || len(index.Calls()) != 6 {
			t.Fatalf("closure call inventory: %v", err)
		}
		for _, partition := range prepared.partitions {
			if partition.Header().PackageUID == "example.session.app" {
				return partition, index
			}
		}
		t.Fatal("missing closure root")
		return atlasmap.Partition{}, nil
	}
	partition, index := prepare()
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	t.Setenv("PATH", "")
	arguments := func(selector string) []string {
		args := append([]string{"experimental", "atlas-impact"}, impactArgs("example.session.app", selector)...)
		return replaceWorkspaceSessionArgument(args, "--max-calls", "6")
	}
	query := func(current atlasmap.Partition, selector string, calls *atlasmap.WorkspaceCallIndex) {
		t.Helper()
		code, output, diagnostic := fetchCLIProcess(t, image, root, arguments(selector), "")
		if code != ExitSuccess || diagnostic != "" {
			t.Fatalf("closure impact: %d %s %s", code, output, diagnostic)
		}
		var got atlasmap.ImpactResult
		if err := json.Unmarshal([]byte(output), &got); err != nil {
			t.Fatal(err)
		}
		want, err := calls.Impact(current.Header().PackageInstance, selector, 8, 64)
		if err != nil {
			t.Fatal(err)
		}
		compareCandidateAtlasAggregateImpact(t, got, want, compiler)
	}
	before := remoteTree(t, root)
	var owner, oldHelper string
	selected := 0
	for _, symbol := range partition.Symbols() {
		if symbol.Declaration.Name != "choose" && !strings.HasPrefix(symbol.Declaration.Name, "_fy_closure_") {
			continue
		}
		selected++
		query(partition, symbol.Visible, index)
		query(partition, symbol.Full, index)
		if symbol.Declaration.Name == "choose" {
			owner = symbol.Visible
		} else if strings.HasPrefix(symbol.Declaration.Name, "_fy_closure__fy_generic_") {
			oldHelper = symbol.Visible
		}
	}
	if selected != 4 || owner == "" || oldHelper == "" {
		t.Fatal("missing distinct ordinary/generic closure selectors", selected)
	}
	for _, bound := range [][2]string{{"--max-calls", "5"}, {"--max-packages", "1"}, {"--max-output-bytes", "1"}} {
		code, output, diagnostic := fetchCLIProcess(t, image, root, replaceWorkspaceSessionArgument(arguments(owner), bound[0], bound[1]), "")
		if code != ExitInvalidInvocation || output != "" || diagnostic == "" {
			t.Fatalf("closure %s: %d %q %q", bound[0], code, output, diagnostic)
		}
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("closure queries changed workspace")
	}
	writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(source, "offset=40", "offset=41", 1))
	stale := remoteTree(t, root)
	code, output, diagnostic := fetchCLIProcess(t, image, root, arguments(owner), "")
	if code != 7 || output != "" || diagnostic == "" || !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatalf("closure stale lock: %d %q %q", code, output, diagnostic)
	}
	current, currentIndex := prepare()
	if current.Header().PackageInstance == partition.Header().PackageInstance || reflect.DeepEqual(current.Specializations(), partition.Specializations()) {
		t.Fatal("closure body edit did not refresh concrete owners")
	}
	after := remoteTree(t, root)
	query(current, owner, currentIndex)
	code, output, diagnostic = fetchCLIProcess(t, image, root, arguments(oldHelper), "")
	if code != ExitInvalidInvocation || output != "" || diagnostic == "" || !reflect.DeepEqual(after, remoteTree(t, root)) {
		t.Fatalf("old concrete helper selector: %d %q %q", code, output, diagnostic)
	}
}
