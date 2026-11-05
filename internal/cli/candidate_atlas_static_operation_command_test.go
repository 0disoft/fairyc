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

func TestCandidateAtlasStaticOperationCommand(t *testing.T) {
	main := `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment)fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, verifyCandidateAtlasStaticOperationCommand)
}

func verifyCandidateAtlasStaticOperationCommand(t *testing.T, image string) {
	t.Helper()
	root := atlasStaticOperationWorkspace(t, atlasStaticOperationUsed, atlasStaticOperationProvider)
	prepare := func() (atlasmap.Partition, *atlasmap.WorkspaceCallIndex) {
		t.Helper()
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != ExitSuccess {
			t.Fatalf("static operation command oracle: %d %s", code, &diagnostic)
		}
		index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
		if err != nil || len(index.Calls()) != 6 {
			t.Fatalf("static operation call inventory: %v", err)
		}
		for _, partition := range prepared.partitions {
			if partition.Header().PackageUID == "example.session.answer" {
				return partition, index
			}
		}
		t.Fatal("missing provider package")
		return atlasmap.Partition{}, nil
	}
	member, index := prepare()
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	t.Setenv("PATH", "")
	query := func(partition atlasmap.Partition, selector string, current *atlasmap.WorkspaceCallIndex) {
		t.Helper()
		args := append([]string{"experimental", "atlas-impact"}, impactArgs(partition.Header().PackageUID, selector)...)
		args = replaceWorkspaceSessionArgument(args, "--max-calls", "6")
		code, output, diagnostic := fetchCLIProcess(t, image, root, args, "")
		if code != ExitSuccess || diagnostic != "" {
			t.Fatalf("static operation impact: %d %s %s", code, output, diagnostic)
		}
		var got atlasmap.ImpactResult
		if err := json.Unmarshal([]byte(output), &got); err != nil {
			t.Fatal(err)
		}
		want, err := current.Impact(partition.Header().PackageInstance, selector, 8, 64)
		if err != nil || got.Schema != want.Schema || got.Scope != want.Scope || got.State != "partial" || !reflect.DeepEqual(got.Subject, want.Subject) || !reflect.DeepEqual(got.Nodes, want.Nodes) || got.TraversalComplete != want.TraversalComplete || !reflect.DeepEqual(got.Omissions, want.Omissions) || len(got.Contexts) != len(want.Contexts) {
			t.Fatalf("static operation impact differs: %v; got %s want %+v", err, output, want)
		}
		for i, context := range got.Contexts {
			verifyCandidateAtlasFamilyContext(t, context, want.Contexts[i], compiler)
		}
	}
	before := remoteTree(t, root)
	var symbol string
	selected := 0
	for _, s := range member.Symbols() {
		if s.Declaration.Name != "transform" && s.Declaration.Name != "algorithm" && s.Declaration.Name != "twice" && s.Declaration.Name != "logical" {
			continue
		}
		selected++
		query(member, s.Visible, index)
		query(member, s.Full, index)
		if s.Declaration.Name == "transform" {
			symbol = s.Visible
		}
	}
	if selected != 4 || symbol == "" {
		t.Fatal("missing operation/provider selections", selected)
	}
	base := append([]string{"experimental", "atlas-impact"}, impactArgs(member.Header().PackageUID, symbol)...)
	base = replaceWorkspaceSessionArgument(base, "--max-calls", "6")
	for _, bound := range [][2]string{{"--max-calls", "5"}, {"--max-packages", "1"}, {"--max-output-bytes", "1"}} {
		code, output, diagnostic := fetchCLIProcess(t, image, root, replaceWorkspaceSessionArgument(base, bound[0], bound[1]), "")
		if code != ExitInvalidInvocation || output != "" || diagnostic == "" {
			t.Fatalf("static operation %s: %d %q %q", bound[0], code, output, diagnostic)
		}
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("static operation queries changed workspace")
	}
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", strings.Replace(atlasStaticOperationProvider, "value*2", "value*3", 1))
	stale := remoteTree(t, root)
	code, output, diagnostic := fetchCLIProcess(t, image, root, base, "")
	if code != 7 || output != "" || diagnostic == "" || !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatalf("static operation stale lock: %d %q %q", code, output, diagnostic)
	}
	current, currentIndex := prepare()
	if member.Header().GenericInterfaceHash == current.Header().GenericInterfaceHash || member.Header().PackageInstance == current.Header().PackageInstance {
		t.Fatal("provider edit did not change current identity")
	}
	after := remoteTree(t, root)
	query(current, symbol, currentIndex)
	if !reflect.DeepEqual(after, remoteTree(t, root)) {
		t.Fatal("refreshed operation query changed workspace")
	}
}
