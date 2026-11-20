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
	fullSymbols := map[string]string{}
	implementations := map[string]string{}
	var lifted string
	for _, partition := range prepared.partitions {
		if partition.Header().PackageUID != uid {
			continue
		}
		for _, symbol := range partition.Symbols() {
			symbols[symbol.Declaration.Name] = symbol.Visible
			fullSymbols[symbol.Declaration.Name] = symbol.Full
		}
		for _, implementation := range partition.Specializations() {
			if implementation.Declaration.Name != "leaf" || len(implementation.TypeArguments) != 1 {
				continue
			}
			typ := implementation.TypeArguments[0].Name
			implementations[typ] = implementation.ID
			if typ == "bool" && len(implementation.Function.Calls) == 1 {
				lifted = implementation.Function.Calls[0].Target.Name
			}
		}
	}
	if lifted == "" || symbols[lifted] == "" || implementations["bool"] == "" || implementations["i32"] == "" {
		t.Fatal("missing concrete leaf targets")
	}
	for _, tc := range []struct {
		name, selector, implementation string
		want                           map[string]int
	}{
		{"lifted", symbols[lifted], "", map[string]int{lifted: 0, "leaf": 1, "relay": 2, "boolean": 3, "main": 4}},
		{"declaration", symbols["leaf"], "", map[string]int{"leaf": 0, "relay": 1, "boolean": 2, "signed": 2, "main": 3}},
		{"bool", symbols["leaf"] + "#" + implementations["bool"], implementations["bool"], map[string]int{"leaf": 0, "relay": 1, "boolean": 2, "main": 3}},
		{"i32", fullSymbols["leaf"] + "#" + implementations["i32"], implementations["i32"], map[string]int{"leaf": 0, "relay": 1, "signed": 2, "main": 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"experimental", "atlas-impact"}, impactArgs(uid, tc.selector)...)
			code, out, stderr := runForTest(args, root)
			var impact atlasmap.ImpactResult
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &impact) != nil ||
				!impact.TraversalComplete || len(impact.Nodes) != len(tc.want) || impact.SubjectSpecialization != tc.implementation {
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
	t.Run("mcp concrete selector", func(t *testing.T) {
		selector := symbols["leaf"] + "#" + implementations["bool"]
		input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"cli-test","version":"1"}}}` + "\n" +
			`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"atlas_impact","arguments":{"packageUid":"` + uid + `","symbol":"` + selector + `"}}}` + "\n"
		args := []string{"experimental", "atlas-mcp", "--allow-impact", "--max-packages", "8", "--max-calls", "128",
			"--max-depth", "8", "--max-symbols", "64", "--max-output-bytes", "100000", "--max-requests", "3"}
		var out, stderr bytes.Buffer
		if code := RunWithInput(args, root, strings.NewReader(input), &out, &stderr); code != ExitSuccess || stderr.Len() != 0 {
			t.Fatalf("MCP %d %s %s", code, &out, &stderr)
		}
		replies := strings.Split(strings.TrimSpace(out.String()), "\n")
		var reply struct {
			Result struct {
				Error   bool                  `json:"isError"`
				Content atlasmap.ImpactResult `json:"structuredContent"`
			}
		}
		if len(replies) != 2 || json.Unmarshal([]byte(replies[1]), &reply) != nil || reply.Result.Error ||
			reply.Result.Content.SubjectSpecialization != implementations["bool"] || len(reply.Result.Content.Nodes) != 4 {
			t.Fatalf("unexpected MCP result: %s", &out)
		}
	})
	for _, selector := range []string{
		symbols["relay"] + "#" + implementations["bool"],
		symbols["leaf"] + "#g0_" + strings.Repeat("0", 64),
		symbols["leaf"] + "#",
	} {
		args := append([]string{"experimental", "atlas-impact"}, impactArgs(uid, selector)...)
		if code, out, stderr := runForTest(args, root); code != ExitInvalidInvocation || out != "" || stderr == "" {
			t.Fatalf("invalid implementation %q: %d %q %q", selector, code, out, stderr)
		}
	}
}
