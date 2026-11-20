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

func TestCandidateAtlasGenericCommand(t *testing.T) {
	enumRoots := candidateAtlasEnumOwnerFixtures(t)
	main := `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment) fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyCandidateAtlasGenericCommand(t, image)
		verifyCandidateAtlasEnumOwners(t, image, enumRoots)
	})
}

func verifyCandidateAtlasGenericCommand(t *testing.T, image string) {
	t.Helper()
	root := candidateAtlasGenericProductionWorkspace(t, atlasGenericProductionUsedSource)
	prepare := func() (preparedAtlasWorkspace, *atlasmap.WorkspaceCallIndex) {
		t.Helper()
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("generic command oracle: %d %s", code, &diagnostic)
		}
		index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
		if err != nil {
			t.Fatal(err)
		}
		return prepared, index
	}
	prepared, index := prepare()
	if len(index.Calls()) != 7 {
		t.Fatalf("generic call coverage: %d", len(index.Calls()))
	}
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	t.Setenv("PATH", "")
	query := func(p atlasmap.Partition, selector string, current *atlasmap.WorkspaceCallIndex) atlasmap.ImpactResult {
		t.Helper()
		args := append([]string{"experimental", "atlas-impact"}, impactArgs(p.Header().PackageUID, selector)...)
		args = replaceWorkspaceSessionArgument(args, "--max-calls", "7")
		code, out, stderr := fetchCLIProcess(t, image, root, args, "")
		if code != 0 || stderr != "" {
			t.Fatalf("generic command: %d %s %s", code, out, stderr)
		}
		var got atlasmap.ImpactResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("generic impact JSON: %v %s", err, out)
		}
		want, err := current.Impact(p.Header().PackageInstance, selector, 8, 64)
		if err != nil {
			t.Fatal(err)
		}
		if got.Schema != want.Schema || got.Scope != want.Scope || got.State != "partial" || !reflect.DeepEqual(got.Subject, want.Subject) || got.SubjectSpecialization != want.SubjectSpecialization || !reflect.DeepEqual(got.Nodes, want.Nodes) || got.TraversalComplete != want.TraversalComplete || !reflect.DeepEqual(got.Omissions, want.Omissions) || len(got.Contexts) != len(want.Contexts) {
			t.Fatalf("generic impact differs: got %s want %+v", out, want)
		}
		for i, context := range got.Contexts {
			verifyCandidateAtlasFamilyContext(t, context, want.Contexts[i], compiler)
		}
		return got
	}
	before := remoteTree(t, root)
	var selected atlasmap.Partition
	var symbol string
	selections := 0
	for _, p := range prepared.partitions {
		for _, s := range p.Symbols() {
			if s.Declaration.Name != "id" && s.Declaration.Name != "choose" && !(p.Header().PackageUID == "example.session.app" && s.Declaration.Name == "unused") {
				continue
			}
			selections++
			for _, selector := range []string{s.Visible, s.Full} {
				query(p, selector, index)
			}
			if s.Declaration.Name == "id" {
				selected, symbol = p, s.Visible
			}
		}
	}
	if selections != 3 || symbol == "" {
		t.Fatalf("generic symbol coverage: %d", selections)
	}
	implementations := 0
	for _, implementation := range selected.Specializations() {
		if implementation.Declaration.Name == "id" {
			query(selected, symbol+"#"+implementation.ID, index)
			implementations++
		}
	}
	if implementations != 2 {
		t.Fatalf("expected two concrete id implementations, found %d", implementations)
	}
	base := append([]string{"experimental", "atlas-impact"}, impactArgs(selected.Header().PackageUID, symbol)...)
	for _, bound := range [][2]string{{"--max-calls", "6"}, {"--max-packages", "1"}} {
		code, out, stderr := fetchCLIProcess(t, image, root, replaceWorkspaceSessionArgument(base, bound[0], bound[1]), "")
		if code != 2 || out != "" || stderr == "" {
			t.Fatalf("generic %s cutoff: %d %q %q", bound[0], code, out, stderr)
		}
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("generic command mutated workspace")
	}
	old := query(selected, symbol, index)
	writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(atlasGenericProductionUsedSource, "pub fn T unused<T>(T value){return value;}", "pub fn T unused<T>(T value){let T copied=value;return copied;}", 1))
	stale := remoteTree(t, root)
	code, out, stderr := fetchCLIProcess(t, image, root, base, "")
	if code != 7 || out != "" || stderr == "" || !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatalf("generic stale lock: %d %q %q", code, out, stderr)
	}
	_, current := prepare()
	after := remoteTree(t, root)
	fresh := query(selected, symbol, current)
	changed := false
	for _, context := range fresh.Contexts {
		if context.PackageUID != "example.session.app" {
			continue
		}
		for _, previous := range old.Contexts {
			if previous.PackageUID == context.PackageUID {
				changed = context.InterfaceHash == previous.InterfaceHash && context.GenericInterfaceHash != previous.GenericInterfaceHash && context.PackageSource != previous.PackageSource && context.PackageInstance != previous.PackageInstance
				if !changed {
					t.Logf("root provenance: before %s %s %s; after %s %s %s", previous.InterfaceHash, previous.GenericInterfaceHash, previous.PackageSource, context.InterfaceHash, context.GenericInterfaceHash, context.PackageSource)
				}
			}
		}
	}
	if !changed || !reflect.DeepEqual(after, remoteTree(t, root)) {
		t.Fatal("unused template edit lost exact current generic provenance or changed workspace")
	}
}
