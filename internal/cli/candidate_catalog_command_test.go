package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
)

// Start from independent Go source facts, then declare the candidate's existing
// narrower non-call collection scope. No expected identity, symbol, generic
// count, cursor, budget or checksum is learned from the candidate response.
func candidateCatalogCommandPartition(t *testing.T, p atlasmap.Partition, e atlasmap.LoadExpectation, compiler buildidentity.Identity, version int) (atlasmap.Partition, atlasmap.LoadExpectation) {
	t.Helper()
	encode := []func(atlasmap.Partition, uint64) ([]byte, error){atlasmap.Encode, atlasmap.EncodeV2, atlasmap.EncodeV3}[version-1]
	data, err := encode(p, 4<<20)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Map struct {
			Header struct {
				Compiler   json.RawMessage `json:"compiler_build"`
				Generation json.RawMessage `json:"generation_status"`
				Omissions  json.RawMessage `json:"omissions"`
			} `json:"header"`
			Completeness json.RawMessage `json:"completeness"`
		} `json:"map"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	type completeness struct {
		Relation string   `json:"relation"`
		Status   string   `json:"status"`
		Resolved uint64   `json:"resolved"`
		External uint64   `json:"external"`
		Unknown  uint64   `json:"unknown"`
		Reasons  []string `json:"reasons"`
	}
	var rows []completeness
	if err := json.Unmarshal(envelope.Map.Completeness, &rows); err != nil {
		t.Fatal(err)
	}
	// Ordinary Go snapshots have no generic relations. The candidate explicitly
	// retains the empty specialization class and unavailable template-body class
	// for v2/v3, as the existing independent Atlas family oracle requires.
	if version > 1 && !slices.ContainsFunc(rows, func(r completeness) bool { return r.Relation == "generic-specializations" }) {
		old := []byte(fmt.Sprintf(`"completeness":%d`, len(rows)))
		rows = append(rows,
			completeness{Relation: "generic-specializations", Status: "exact", Reasons: []string{}},
			completeness{Relation: "generic-template-bodies", Status: "unavailable", Reasons: []string{"bodies-checked-only-after-specialization"}},
		)
		slices.SortFunc(rows, func(a, b completeness) int { return strings.Compare(a.Relation, b.Relation) })
		if bytes.Count(data, old) != 1 {
			t.Fatal("ambiguous completeness count")
		}
		data = bytes.Replace(data, old, []byte(fmt.Sprintf(`"completeness":%d`, len(rows))), 1)
	}
	var omissions []string
	for i := range rows {
		r := &rows[i]
		if r.Relation != "calls" && r.Relation != "drops" && r.Relation != "may-trap" && r.Relation != "uses-unsafe" && r.Relation != "moves" && r.Relation != "allocates" && r.Relation != "returns-error" && !strings.HasPrefix(r.Relation, "generic-") {
			r.Status, r.Resolved, r.External, r.Unknown = "unavailable", 0, 0, 0
			r.Reasons = []string{"candidate-atlas-not-collected"}
		}
		if r.Status != "exact" {
			for _, reason := range r.Reasons {
				omissions = append(omissions, r.Relation+":"+r.Status+":"+reason)
			}
		}
	}
	slices.Sort(omissions)
	for _, change := range []struct {
		field string
		old   json.RawMessage
		value any
	}{
		{"compiler_build", envelope.Map.Header.Compiler, compiler.String()},
		{"generation_status", envelope.Map.Header.Generation, "partial"},
		{"omissions", envelope.Map.Header.Omissions, omissions},
		{"completeness", envelope.Map.Completeness, rows},
	} {
		value, err := json.Marshal(change.value)
		if err != nil {
			t.Fatal(err)
		}
		old := append([]byte(fmt.Sprintf("%q:", change.field)), change.old...)
		next := append([]byte(fmt.Sprintf("%q:", change.field)), value...)
		if bytes.Count(data, old) != 1 {
			t.Fatalf("ambiguous oracle field %s", change.field)
		}
		data = bytes.Replace(data, old, next, 1)
	}
	decode := []func([]byte, uint64) (atlasmap.Partition, error){atlasmap.Decode, atlasmap.DecodeV2, atlasmap.DecodeV3}[version-1]
	p, err = decode(atlasRechecksum(t, data), 4<<20)
	if err != nil {
		t.Fatal("independent narrowed map", err)
	}
	e.CompilerBuild = compiler
	return p, e
}

func candidateCatalogCommandIndex(t *testing.T, p atlasmap.Partition, e atlasmap.LoadExpectation, compiler buildidentity.Identity, version int) *atlasmap.QueryIndex {
	t.Helper()
	p, e = candidateCatalogCommandPartition(t, p, e, compiler, version)
	query := []func(atlasmap.Partition, atlasmap.LoadExpectation) (*atlasmap.QueryIndex, error){atlasmap.NewQueryIndex, atlasmap.NewQueryIndexV2, atlasmap.NewQueryIndexV3}[version-1]
	index, err := query(p, e)
	if err != nil {
		t.Fatal(err)
	}
	return index
}

func candidateCatalogCommandFixture(t *testing.T, image, root string, version int) candidateCatalogFixture {
	t.Helper()
	var diagnostic bytes.Buffer
	schema := []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV2, atlasmap.LogicalSchemaV3}[version-1]
	prepared, code := prepareAtlasWorkspaceSchema(root, true, &diagnostic, schema)
	if code != 0 {
		t.Fatalf("locked source oracle: %d %s", code, &diagnostic)
	}
	p, found := selectAtlasPartition(prepared.partitions, "example.session.app")
	if !found {
		t.Fatal("missing root partition")
	}
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	index := candidateCatalogCommandIndex(t, p, prepared.expectations[p.Header().PackageInstance], buildidentity.Identity(sha256.Sum256(data)), version)
	return candidateCatalogFixture{uid: p.Header().PackageUID, version: version, index: index}
}

func verifyCandidateCatalogCommand(t *testing.T, image string) {
	t.Helper()
	// Repeated page/byte boundaries use a small real workspace. The separate
	// source-family verifier covers the complete declaration family in each schema.
	root := writeWorkspaceSessionFixture(t, `module main;import answer as a;pub fn i32 leaf(){return a::value();}pub fn i32 other(){return leaf();}fn i32 main(){return other();}`)
	var diagnostic bytes.Buffer
	if _, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3); code != 0 {
		t.Fatalf("source setup: %d %s", code, &diagnostic)
	}
	before := remoteTree(t, root)
	calls := 0
	run := func(root string, args []string, want int, golden []byte) {
		t.Helper()
		calls++
		code, out, stderr := fetchCLIProcess(t, image, root, args, "")
		if code != want || want == 0 && (stderr != "" || !bytes.Equal([]byte(out), golden)) || want != 0 && (out != "" || stderr == "") {
			t.Fatalf("catalog %v: %d %.3500s %.500s; want %d %.3500s", args, code, out, stderr, want, golden)
		}
	}
	r := atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 64, MaxSelectionBytes: 1 << 20}}
	var base []string
	for version := 1; version <= 3; version++ {
		f := candidateCatalogCommandFixture(t, image, root, version)
		request := r
		for _, name := range []string{"", "main", "Missing"} {
			request.Name = name
			golden, err := candidateCatalogOracle(t, f, request, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			args := append(candidateCatalogArgs(f, request, 1<<20), "--locked")
			run(root, args, 0, golden)
		}
		request = r
		request.Budget.MaxSymbols = 1
		golden, err := candidateCatalogOracle(t, f, request, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		run(root, append(candidateCatalogArgs(f, request, 1<<20), "--locked"), 0, golden)
		var page struct {
			Catalog struct {
				Next string `json:"next_after"`
			} `json:"catalog"`
		}
		if json.Unmarshal(golden, &page) != nil || page.Catalog.Next == "" {
			t.Fatal("missing independent page cursor")
		}
		request.After = page.Catalog.Next
		golden, err = candidateCatalogOracle(t, f, request, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		run(root, append(candidateCatalogArgs(f, request, 1<<20), "--locked"), 0, golden)
		request = r
		request.Name = "main"
		golden, err = candidateCatalogOracle(t, f, request, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		for _, maximum := range []uint64{uint64(len(golden) - 1), uint64(len(golden)), uint64(len(golden) + 1)} {
			want := 0
			if maximum < uint64(len(golden)) {
				want = 2
			}
			run(root, append(candidateCatalogArgs(f, request, maximum), "--locked"), want, golden)
		}
		var result struct {
			Catalog struct {
				Usage struct {
					Bytes uint64 `json:"selection_bytes"`
				} `json:"usage"`
			} `json:"catalog"`
		}
		if json.Unmarshal(golden, &result) != nil || result.Catalog.Usage.Bytes < 2 {
			t.Fatal("missing independent selection size")
		}
		for _, maximum := range []uint64{result.Catalog.Usage.Bytes - 1, result.Catalog.Usage.Bytes, result.Catalog.Usage.Bytes + 1} {
			request.Budget.MaxSelectionBytes = maximum
			expected, err := candidateCatalogOracle(t, f, request, 1<<20)
			want := 0
			if err != nil {
				want = 2
			}
			run(root, append(candidateCatalogArgs(f, request, 1<<20), "--locked"), want, expected)
		}
		base = append(candidateCatalogArgs(f, r, 1<<20), "--locked")
	}
	for _, change := range [][2]string{{"--package-uid", "example.absent"}, {"--module", "absent"}, {"--after", "fysym1-sha256-256-" + strings.Repeat("0", 64)}, {"--max-output-bytes", "1"}} {
		args := append([]string{}, base...)
		if change[0] == "--module" || change[0] == "--after" {
			args = append(args, change[:]...)
		} else {
			args = replaceWorkspaceSessionArgument(args, change[0], change[1])
		}
		run(root, args, 2, nil)
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("locked catalog mutated workspace")
	}
	for _, args := range [][]string{{"experimental", "atlas-symbols"}, {"experimental", "atlas-symbols", "show"}, append(slices.Clone(base), "--locked"), replaceWorkspaceSessionArgument(base, "--catalog-version", "4")} {
		empty := t.TempDir()
		run(empty, args, 2, nil)
		if len(remoteTree(t, empty)) != 0 {
			t.Fatal("argument refusal performed I/O")
		}
	}
	code, _, stderr := fetchCLIProcess(t, image, root, base, "stdout")
	calls++
	if code != 4 || stderr == "" {
		t.Fatalf("catalog stdout failure: %d %q", code, stderr)
	}
	writeWorkspaceFile(t, root, "src/main.fy", string(mustReadAtlasFile(t, filepath.Join(root, "src/main.fy")))+"\n")
	stale := remoteTree(t, root)
	run(root, base, 7, nil)
	if !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatal("stale locked catalog mutated workspace")
	}
	unlocked := slices.Clone(base[:len(base)-1])
	code, out, stderr := fetchCLIProcess(t, image, root, unlocked, "")
	calls++
	f := candidateCatalogCommandFixture(t, image, root, 3)
	golden, err := candidateCatalogOracle(t, f, r, 1<<20)
	if err != nil || code != 0 || stderr != "" || !bytes.Equal([]byte(out), golden) {
		t.Fatalf("unlocked source/lock refresh: %v %d %.500s %.500s", err, code, out, stderr)
	}
	t.Logf("catalog command native processes: %d", calls)
}
