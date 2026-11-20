package cli

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func impactArgs(uid, symbol string) []string {
	return []string{"--package-uid", uid, "--symbol", symbol, "--max-packages", "8", "--max-calls", "128", "--max-depth", "8", "--max-symbols", "64", "--max-output-bytes", "100000"}
}

func TestAtlasImpactLockedCurrentAndReadOnly(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	s := atlasFunctionSymbolForPackage(t, root, "example.session.answer", "value")
	args := append([]string{"experimental", "atlas-impact"}, impactArgs("example.session.answer", s.Visible)...)
	code, out, err := runForTest(args, root)
	if code != 0 || err != "" || strings.Contains(out, root) {
		t.Fatalf("impact %d %q %q", code, out, err)
	}
	var report atlasmap.ImpactResult
	if json.Unmarshal([]byte(out), &report) != nil || len(report.Nodes) != 2 || report.State == "exact" {
		t.Fatal(out)
	}
	writeWorkspaceFile(t, root, "src/main.fy", workspaceSessionRootSource+"\n")
	if code, out, _ := runForTest(args, root); code != ExitPackageFailure || out != "" {
		t.Fatal("stale lock admitted")
	}
}

const atlasConcreteImpactSource = `module main;
fn i32 leaf<T>(T marker) {
    discard marker;
    let copied = fn [] i32(i32 value) { return 40 + value; };
    return copied(2);
}
fn i32 relay<T>(T marker) { return leaf<T>(marker); }
fn i32 signed() { return relay<i32>(42); }
fn i32 boolean() { return relay<bool>(true); }
fn i32 main() { return signed() + boolean(); }
`

func TestAtlasImpactPreservesConcreteCallPaths(t *testing.T) {
	const uid = "example.session.app"
	root := writeWorkspaceSessionFixture(t, atlasConcreteImpactSource)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("prepare impact symbols: %d %s", code, &diagnostic)
	}

	symbols := map[string]string{}
	var lifted string
	for _, partition := range prepared.partitions {
		if partition.Header().PackageUID != uid {
			continue
		}
		for _, symbol := range partition.Symbols() {
			symbols[symbol.Declaration.Name] = symbol.Visible
		}
		for _, implementation := range partition.Specializations() {
			if implementation.Declaration.Name == "leaf" && len(implementation.TypeArguments) == 1 &&
				implementation.TypeArguments[0].Name == "bool" &&
				len(implementation.Function.Calls) == 1 {
				lifted = implementation.Function.Calls[0].Target.Name
			}
		}
	}
	if lifted == "" || symbols[lifted] == "" {
		t.Fatal("missing bool closure target")
	}
	for _, tc := range []struct {
		subject string
		want    map[string]int
	}{
		{lifted, map[string]int{lifted: 0, "leaf": 1, "relay": 2, "boolean": 3, "main": 4}},
		{"leaf", map[string]int{"leaf": 0, "relay": 1, "boolean": 2, "signed": 2, "main": 3}},
	} {
		t.Run(tc.subject, func(t *testing.T) {
			args := append([]string{"experimental", "atlas-impact"}, impactArgs(uid, symbols[tc.subject])...)
			code, out, stderr := runForTest(args, root)
			var impact atlasmap.ImpactResult
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &impact) != nil ||
				!impact.TraversalComplete || len(impact.Nodes) != len(tc.want) {
				t.Fatalf("impact %d %q %q", code, out, stderr)
			}
			for name, depth := range tc.want {
				if !slices.ContainsFunc(impact.Nodes, func(node atlasmap.ImpactNode) bool {
					return node.Subject.Symbol == symbols[name] && node.Depth == depth
				}) {
					t.Fatalf("missing %s at depth %d: %s", name, depth, out)
				}
			}
		})
	}
}
