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

const atlasAggregateCommandRoot = `module root;import maker as m;
@effects(console)fn i32 main(){let owner=m::envelope();let empty=m::empty();
let first=m::Cell<i32>{.value=42};let second=m::Cell<bool>{.value=true};
defer m::signal(8);assert(second.value);return first.value;}`

const atlasAggregateCommandMaker = aggregateTraceMaker + `
pub resource Empty<T>{cleanup{}}
pub fn Empty<i32> empty(){return Empty<i32>{};}`

func TestCandidateAtlasAggregateCommand(t *testing.T) {
	main := `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment)fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, verifyCandidateAtlasAggregateCommand)
}

func candidateAtlasAggregateCommandOracle(t *testing.T, root string) (preparedAtlasWorkspace, *atlasmap.WorkspaceCallIndex) {
	t.Helper()
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("aggregate command oracle: %d %s", code, &diagnostic)
	}
	index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
	if err != nil {
		t.Fatal(err)
	}
	return prepared, index
}

func compareCandidateAtlasAggregateImpact(t *testing.T, got, want atlasmap.ImpactResult, compiler string) {
	t.Helper()
	if got.Schema != want.Schema || got.Scope != want.Scope || got.State != "partial" || !reflect.DeepEqual(got.Subject, want.Subject) || !reflect.DeepEqual(got.Nodes, want.Nodes) || got.TraversalComplete != want.TraversalComplete || !reflect.DeepEqual(got.Omissions, want.Omissions) || len(got.Contexts) != len(want.Contexts) {
		t.Fatalf("aggregate impact differs: got %+v want %+v", got, want)
	}
	for i, context := range got.Contexts {
		verifyCandidateAtlasFamilyContext(t, context, want.Contexts[i], compiler)
	}
}

func verifyCandidateAtlasAggregateCommand(t *testing.T, image string) {
	t.Helper()
	root := candidateAggregateCacheWorkspace(t, candidateAggregateLayoutFixture{root: atlasAggregateCommandRoot, maker: atlasAggregateCommandMaker})
	prepared, index := candidateAtlasAggregateCommandOracle(t, root)
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	resolved := len(index.Calls())
	if resolved < 2 {
		t.Fatalf("missing aggregate/cleanup call oracle: %d", resolved)
	}
	t.Logf("aggregate command resolved calls=%d", resolved)
	t.Setenv("PATH", "")
	query := func(p atlasmap.Partition, selector string, current *atlasmap.WorkspaceCallIndex) atlasmap.ImpactResult {
		t.Helper()
		args := append([]string{"experimental", "atlas-impact"}, impactArgs(p.Header().PackageUID, selector)...)
		args = replaceWorkspaceSessionArgument(args, "--max-calls", strconv.Itoa(resolved))
		code, out, stderr := fetchCLIProcess(t, image, root, args, "")
		if code != 0 || stderr != "" {
			t.Fatalf("aggregate impact command: %d %s %s", code, out, stderr)
		}
		var got atlasmap.ImpactResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err, out)
		}
		want, err := current.Impact(p.Header().PackageInstance, selector, 8, 64)
		if err != nil {
			t.Fatal(err)
		}
		compareCandidateAtlasAggregateImpact(t, got, want, compiler)
		return got
	}
	before := remoteTree(t, root)
	resources := map[atlasmap.WorkspaceSymbol]bool{}
	for _, p := range prepared.partitions {
		for _, s := range p.Symbols() {
			if s.Declaration.Kind == "resource" {
				for _, id := range []string{s.Visible, s.Full} {
					resources[atlasmap.WorkspaceSymbol{PackageInstance: p.Header().PackageInstance, Symbol: id}] = true
				}
			}
		}
	}
	var selected atlasmap.Partition
	var symbol string
	selections := 0
	for _, p := range prepared.partitions {
		for _, s := range p.Symbols() {
			if s.Declaration.Name != "finish" && s.Declaration.Name != "Token" && s.Declaration.Name != "Cell" && s.Declaration.Name != "Unused" && s.Declaration.Name != "Empty" {
				continue
			}
			selections++
			result := query(p, s.Visible, index)
			if s.Declaration.Name == "finish" {
				selected, symbol = p, s.Visible
				query(p, s.Full, index)
				resource := false
				for _, node := range result.Nodes {
					resource = resource || resources[node.Subject]
				}
				if !resource {
					t.Fatal("cross-package cleanup caller was omitted")
				}
			}
		}
	}
	if selections != 5 || symbol == "" {
		t.Fatalf("aggregate symbol coverage: %d", selections)
	}
	base := append([]string{"experimental", "atlas-impact"}, impactArgs(selected.Header().PackageUID, symbol)...)
	base = replaceWorkspaceSessionArgument(base, "--max-calls", strconv.Itoa(resolved))
	for _, bound := range [][2]string{{"--max-calls", strconv.Itoa(resolved - 1)}, {"--max-packages", "2"}} {
		code, out, stderr := fetchCLIProcess(t, image, root, replaceWorkspaceSessionArgument(base, bound[0], bound[1]), "")
		if code != 2 || out != "" || stderr == "" {
			t.Fatalf("aggregate %s cutoff: %d %q %q", bound[0], code, out, stderr)
		}
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("aggregate queries mutated workspace")
	}
	old := query(selected, symbol, index)
	writeWorkspaceFile(t, root, "libs/maker/src/maker.fy", strings.Replace(atlasAggregateCommandMaker, "pub struct Unused<T>{T value;}", "pub struct Unused<T>{T value;T other;}", 1))
	stale := remoteTree(t, root)
	code, out, stderr := fetchCLIProcess(t, image, root, base, "")
	if code != 7 || out != "" || stderr == "" || !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatalf("aggregate stale source/lock: %d %q %q", code, out, stderr)
	}
	_, current := candidateAtlasAggregateCommandOracle(t, root)
	after := remoteTree(t, root)
	fresh := query(selected, symbol, current)
	changed := false
	for _, context := range fresh.Contexts {
		for _, previous := range old.Contexts {
			if context.PackageUID == "example.aggregate.maker" && previous.PackageUID == context.PackageUID {
				changed = context.GenericInterfaceHash != previous.GenericInterfaceHash && context.PackageSource != previous.PackageSource && context.PackageInstance != previous.PackageInstance
				if !changed {
					t.Logf("maker provenance before=%+v after=%+v", previous, context)
				}
			}
		}
	}
	if !changed || !reflect.DeepEqual(after, remoteTree(t, root)) {
		t.Fatal("unused aggregate edit lost current generic provenance or mutated workspace")
	}
}
