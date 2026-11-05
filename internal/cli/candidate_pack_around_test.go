package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func candidatePackAroundFixtures(t *testing.T) []candidateCatalogFixture {
	t.Helper()
	fixtures := candidatePackRelationFixtures(t)
	var source strings.Builder
	source.WriteString("module main;import answer as a;fn i32 leaf(){return 1;}\n")
	for i := 0; i < 6; i++ {
		fmt.Fprintf(&source, "fn i32 hub_%d(){return leaf();}\n", i)
	}
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&source, "fn i32 arm_%d(){return hub_%d()+hub_%d();}\n", i, i%6, (i+1)%6)
	}
	source.WriteString("fn i32 fanout(){return ")
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&source, "arm_%d()+", i)
	}
	source.WriteString("a::value();}fn i32 ring_a(){return ring_b();}fn i32 ring_b(){return ring_a();}fn void self(){self();}fn i32 main(){return fanout();}")
	for v, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV2, atlasmap.LogicalSchemaV3} {
		root := writeWorkspaceSessionFixture(t, source.String())
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, schema)
		if code != 0 {
			t.Fatalf("frontier source v%d: %d %s", v+1, code, &diagnostic)
		}
		for _, p := range prepared.partitions {
			if p.Header().PackageUID != "example.session.app" {
				continue
			}
			data, err := []func(atlasmap.Partition, uint64) ([]byte, error){atlasmap.Encode, atlasmap.EncodeV2, atlasmap.EncodeV3}[v](p, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			e := prepared.expectations[p.Header().PackageInstance]
			index, err := []func(atlasmap.Partition, atlasmap.LoadExpectation) (*atlasmap.QueryIndex, error){atlasmap.NewQueryIndex, atlasmap.NewQueryIndexV2, atlasmap.NewQueryIndexV3}[v](p, e)
			if err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, candidateCatalogFixture{fmt.Sprintf("frontier-v%d", v+1), p.Header().PackageUID, v + 1, data, candidateAtlasCurrent(t, e), index})
		}
	}
	return fixtures
}

func TestCandidatePackAround(t *testing.T) {
	fixtures := candidatePackAroundFixtures(t)
	main := strings.Replace(candidatePackShowMain, "pack::encode_show(", "pack::encode_around(", 1)
	candidateAtlasProbe(t, "project_pack", main, func(t *testing.T, image string) {
		processes := 0
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				run := func(q candidatePackQueryControls) (int, string) {
					t.Helper()
					processes++
					return candidatePackQueryProcess(t, image, f, q)
				}
				check := func(name string, q candidatePackQueryControls) {
					t.Helper()
					t.Run(name, func(t *testing.T) {
						want, err := candidatePackQueryOracle(t, f, q, true)
						code, got := run(q)
						if err != nil && (code != 7 || got != "") || err == nil && (code != 0 || got != string(want)) {
							t.Fatalf("around %+v: %d %v\n got=%s\nwant=%s", q, code, err, got, want)
						}
					})
				}
				base := candidatePackQueryControls{Read: f.version, Version: f.version, Mask: 1023, Records: 65536, Relations: 65536, Depth: 64, Selection: 4 << 20, Output: 4 << 20}
				catalog, err := f.index.CatalogSymbols(atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 65536, MaxSelectionBytes: 4 << 20}})
				if err != nil || !catalog.Complete {
					t.Fatalf("source catalog: %v", err)
				}
				largest := -1
				for _, entry := range catalog.Symbols {
					q := base
					q.Symbol = entry.Symbol.Full
					for direction := 0; direction < 3; direction++ {
						q.Direction = direction
						check(entry.Name+"-"+strconv.Itoa(direction), q)
					}
					q.Direction = 2
					r, err := f.index.Around(q.Symbol, q.request())
					if err != nil {
						t.Fatal(err)
					}
					if r.Usage.Records > largest {
						largest = r.Usage.Records
						base = q
					}
					q.Symbol = entry.Symbol.Visible
					check(entry.Name+"-visible", q)
				}
				for i := 0; i < 10; i++ {
					q := base
					q.Mask = 1 << i
					check("kind-"+strconv.Itoa(i), q)
				}
				for _, mask := range []int{6, 24, 341, 682} {
					q := base
					q.Mask = mask
					check("mask-"+strconv.Itoa(mask), q)
				}
				full, err := f.index.Around(base.Symbol, base.request())
				if err != nil {
					t.Fatal(err)
				}
				if strings.HasPrefix(f.name, "frontier-") && (len(full.Nodes) < 20 || full.Usage.Depth < 2 || len(full.Relations) < 40) {
					t.Fatalf("wide/convergent fixture lost independent coverage: %+v", full.Usage)
				}
				rootBudget := base
				rootBudget.Records = 1
				rootBudget.Relations = 0
				rootBudget.Depth = 0
				rootResult, err := f.index.Around(base.Symbol, rootBudget.request())
				if err != nil {
					t.Fatal(err)
				}
				mandatory, err := json.Marshal(struct {
					Context atlasmap.QueryContext `json:"context"`
					Root    atlasmap.AroundNode   `json:"root"`
				}{rootResult.Context, rootResult.Nodes[0]})
				if err != nil {
					t.Fatal(err)
				}
				if uint64(len(mandatory)+1) != rootResult.Usage.SelectionBytes {
					t.Fatal("independent mandatory root cost differs")
				}
				bounds := []uint64{0, 1, rootResult.Usage.SelectionBytes - 1, rootResult.Usage.SelectionBytes, rootResult.Usage.SelectionBytes + 1}
				for n := 0; n <= full.Usage.Records+1; n++ {
					q := base
					q.Records = n
					check("records-"+strconv.Itoa(n), q)
					if n > 0 {
						prefix, err := f.index.Around(base.Symbol, q.request())
						if err != nil {
							t.Fatal(err)
						}
						cost := prefix.Usage.SelectionBytes
						bounds = append(bounds, cost-1, cost, cost+1)
					}
				}
				slices.Sort(bounds)
				for _, n := range slices.Compact(bounds) {
					q := base
					q.Selection = n
					check("selection-"+strconv.FormatUint(n, 10), q)
				}
				for n := 0; n <= full.Usage.Relations+1; n++ {
					q := base
					q.Relations = n
					check("relations-"+strconv.Itoa(n), q)
				}
				for n := 0; n <= full.Usage.Depth+1; n++ {
					for direction := 0; direction < 3; direction++ {
						q := base
						q.Depth = n
						q.Direction = direction
						check(fmt.Sprintf("depth-%d-direction-%d", n, direction), q)
					}
				}
				for _, n := range []int{0, 1, full.Usage.Relations} {
					q := base
					q.Records = 1
					q.Relations = n
					q.Selection = rootResult.Usage.SelectionBytes
					check("precedence-"+strconv.Itoa(n), q)
				}
				want, err := candidatePackQueryOracle(t, f, base, true)
				if err != nil {
					t.Fatal(err)
				}
				for _, n := range []int{0, len(want) - 1, len(want), len(want) + 1} {
					q := base
					q.Output = uint64(n)
					check("output-"+strconv.Itoa(n), q)
				}
				for _, selector := range []string{"main", strings.ToUpper(base.Symbol), "fysym1-sha256-128-" + strings.Repeat("0", 32), "fysym1-sha256-256-" + strings.Repeat("0", 64)} {
					q := base
					q.Symbol = selector
					check("bad-selector-"+selector, q)
				}
				for _, mask := range []int{0, 1024} {
					q := base
					q.Mask = mask
					check("invalid-mask-"+strconv.Itoa(mask), q)
				}
				q := base
				q.Direction = 3
				check("invalid-direction", q)
				q = base
				q.Version = f.version%3 + 1
				check("wrong-family", q)
				for _, mode := range []string{"corrupt", "stale"} {
					q = base
					q.Mode = mode
					if code, got := run(q); code != 7 || got != "" {
						t.Fatalf("%s published: %d %q", mode, code, got)
					}
				}
				q = base
				q.Mode = "measure"
				code, text := run(q)
				cost, err := strconv.Atoi(text)
				if code != 0 || err != nil || cost <= 0 || cost >= 16<<20 {
					t.Fatalf("invalid work: %d %q %v", code, text, err)
				}
				for _, extra := range []int{-1, 0, 1} {
					q.Mode = "meter"
					q.Meter = (16 << 20) - cost - extra
					code, got := run(q)
					if extra < 0 && (code != 7 || got != "") || extra >= 0 && (code != 0 || got != string(want)) {
						t.Fatalf("work boundary %d: %d %q", extra, code, got)
					}
				}
			})
		}
		t.Logf("%d native around observations", processes)
	})
}
