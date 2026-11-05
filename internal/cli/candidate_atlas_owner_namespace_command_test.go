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

func atlasOwnerNamespaceCommandSource(t *testing.T, ordinary bool) string {
	t.Helper()
	source := atlasOwnerQualifiedNamespaceSource(ordinary)
	if ordinary {
		return source
	}
	const cell = "fn i32 Cell::get(ref Cell this){return this.value;}"
	const other = "fn i32 Other::get(ref Other this){return this.value;}"
	reordered := strings.Replace(source, cell+"\n"+other, other+"\n"+cell, 1)
	if reordered == source {
		t.Fatal("reverse-declaration namespace fixture no longer matches source")
	}
	return reordered
}

func verifyCandidateAtlasOwnerNamespaceCommand(t *testing.T, image string) {
	t.Helper()
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	t.Setenv("PATH", "")
	for _, ordinary := range []bool{false, true} {
		t.Run(fmt.Sprint("ordinary=", ordinary), func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, atlasOwnerNamespaceCommandSource(t, ordinary))
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("namespace command oracle: %d %s", code, &diagnostic)
			}
			index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
			if err != nil {
				t.Fatal(err)
			}
			calls := 2
			if ordinary {
				calls = 3
			}
			if len(index.Calls()) != calls {
				t.Fatalf("namespace call inventory: %d want %d", len(index.Calls()), calls)
			}
			before := remoteTree(t, root)
			selected := 0
			for _, partition := range prepared.partitions {
				if partition.Header().PackageUID != "example.session.app" {
					continue
				}
				for _, symbol := range partition.Symbols() {
					if symbol.Declaration.Name != "Cell::get" && symbol.Declaration.Name != "Other::get" && symbol.Declaration.Name != "get" {
						continue
					}
					selected++
					for _, selector := range []string{symbol.Visible, symbol.Full} {
						args := append([]string{"experimental", "atlas-impact"}, impactArgs("example.session.app", selector)...)
						args = replaceWorkspaceSessionArgument(args, "--max-calls", strconv.Itoa(calls))
						if selector == symbol.Full {
							args = append(args, "--locked")
						}
						code, output, diagnostic := fetchCLIProcess(t, image, root, args, "")
						var got atlasmap.ImpactResult
						if code != ExitSuccess || diagnostic != "" || json.Unmarshal([]byte(output), &got) != nil {
							t.Fatalf("namespace impact %s: %d %s %s", symbol.Declaration.Name, code, output, diagnostic)
						}
						want, err := index.Impact(partition.Header().PackageInstance, selector, 8, 64)
						if err != nil {
							t.Fatal(err)
						}
						compareCandidateAtlasAggregateImpact(t, got, want, compiler)
					}
					if symbol.Declaration.Name == "Cell::get" {
						args := append([]string{"experimental", "atlas-impact"}, impactArgs("example.session.app", symbol.Visible)...)
						args = replaceWorkspaceSessionArgument(args, "--max-calls", strconv.Itoa(calls-1))
						code, output, diagnostic := fetchCLIProcess(t, image, root, args, "")
						if code != ExitInvalidInvocation || output != "" || diagnostic == "" {
							t.Fatalf("namespace one-short call bound: %d %s %s", code, output, diagnostic)
						}
					}
				}
			}
			if selected != calls || !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatalf("namespace subject coverage or immutability: selected=%d want=%d", selected, calls)
			}
		})
	}
}

func verifyCandidateMCPOwnerNamespaceCommand(t *testing.T, image string) {
	t.Helper()
	for _, ordinary := range []bool{false, true} {
		t.Run(fmt.Sprint("ordinary=", ordinary), func(t *testing.T) {
			source := atlasOwnerNamespaceCommandSource(t, ordinary)
			root := writeWorkspaceSessionFixture(t, source)
			name := "Other::get"
			if ordinary {
				name = "get"
			}
			verifyCandidateMCPWorkspaceFixture(t, image, root, source, workspaceSessionMemberSource, "main", name)
		})
	}
}

func verifyCandidateEditOwnerNamespaceCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateEditSDKCommandFixture(t, image, candidateEditSDKFixture{
		source:    atlasOwnerQualifiedNamespaceSource(true),
		body:      "{return 2;}",
		invalid:   "{return false;}",
		module:    "main",
		name:      "Other::get",
		deniedUID: "example.session.app",
		subject:   "get",
	})
}
