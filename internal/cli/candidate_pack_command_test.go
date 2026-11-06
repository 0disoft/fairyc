package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

// The source producer currently collects calls, drops, trap categories and unsafe sites. Catalog/call output never
// exposed the Go oracle's other body facts, but packs expose their counts and
// node summaries too. Project that explicitly declared scope before querying;
// retain every original Go declaration, call, drop, trap category, unsafe span and specialization identity.
func candidatePackCommandFixture(t *testing.T, image, root string, version int) candidateCatalogFixture {
	t.Helper()
	prepared := candidateCallsCommandFixture(t, image, root, version-1).prepared
	p, found := selectAtlasPartition(prepared.partitions, "example.session.app")
	if !found {
		t.Fatal("missing independent pack root")
	}
	data, err := []func(atlasmap.Partition, uint64) ([]byte, error){atlasmap.Encode, atlasmap.EncodeV2, atlasmap.EncodeV3}[version-1](p, 4<<20)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Map struct {
			Functions       []json.RawMessage
			Specializations []struct{ Function, Cleanup json.RawMessage }
		}
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	bodies := slices.Clone(envelope.Map.Functions)
	for _, s := range envelope.Map.Specializations {
		for _, body := range []json.RawMessage{s.Function, s.Cleanup} {
			if len(body) > 0 && string(body) != "null" {
				bodies = append(bodies, body)
			}
		}
	}
	count := 0
	for _, body := range bodies {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(body, &fields); err != nil {
			t.Fatal(err)
		}
		var calls []json.RawMessage
		if err := json.Unmarshal(fields["calls"], &calls); err != nil {
			t.Fatal(err)
		}
		var drops []json.RawMessage
		if err := json.Unmarshal(fields["drop_sites"], &drops); err != nil {
			t.Fatal(err)
		}
		var traps []json.RawMessage
		if err := json.Unmarshal(fields["trap_categories"], &traps); err != nil {
			t.Fatal(err)
		}
		var unsafeSites []json.RawMessage
		if err := json.Unmarshal(fields["unsafe_sites"], &unsafeSites); err != nil {
			t.Fatal(err)
		}
		count += len(calls) + len(drops) + len(traps) + len(unsafeSites)
		for _, field := range []string{"move_sites", "allocation_sites", "return_error_sites", "borrows"} {
			if _, ok := fields[field]; !ok {
				t.Fatalf("missing independent body field %s", field)
			}
			prefix := []byte(fmt.Sprintf("%q:", field))
			data = bytes.ReplaceAll(data, append(slices.Clone(prefix), fields[field]...), append(prefix, []byte("[]")...))
		}
	}
	data = bytes.Replace(data, []byte(fmt.Sprintf(`"relations":%d`, p.Header().Counts.Relations)), []byte(fmt.Sprintf(`"relations":%d`, count)), 1)
	p, err = []func([]byte, uint64) (atlasmap.Partition, error){atlasmap.Decode, atlasmap.DecodeV2, atlasmap.DecodeV3}[version-1](atlasRechecksum(t, data), 4<<20)
	if err != nil {
		t.Fatal("independent call/drop pack partition", err)
	}
	index, err := []func(atlasmap.Partition, atlasmap.LoadExpectation) (*atlasmap.QueryIndex, error){atlasmap.NewQueryIndex, atlasmap.NewQueryIndexV2, atlasmap.NewQueryIndexV3}[version-1](p, prepared.expectations[p.Header().PackageInstance])
	if err != nil {
		t.Fatal(err)
	}
	return candidateCatalogFixture{uid: p.Header().PackageUID, version: version, index: index}
}

func candidatePackCommandSubject(t *testing.T, f candidateCatalogFixture, name string, full bool) string {
	t.Helper()
	r, err := f.index.CatalogSymbols(atlasmap.SymbolCatalogRequest{Name: name, Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 64, MaxSelectionBytes: 1 << 20}})
	if err != nil || len(r.Symbols) != 1 {
		t.Fatalf("independent pack subject %s: %v %#v", name, err, r.Symbols)
	}
	if full {
		return r.Symbols[0].Symbol.Full
	}
	return r.Symbols[0].Symbol.Visible
}

func candidatePackCommandControls(t *testing.T, f candidateCatalogFixture, name string) candidatePackQueryControls {
	t.Helper()
	return candidatePackQueryControls{Version: f.version, Symbol: candidatePackCommandSubject(t, f, name, false), Direction: 2, Mask: 1023, Records: 128, Relations: 128, Depth: 3, Selection: 1 << 20, Output: 1 << 20}
}

func candidatePackCommandArgs(f candidateCatalogFixture, q candidatePackQueryControls, action string) []string {
	args := []string{"experimental", "atlas-pack", action, "--pack-version", strconv.Itoa(q.Version), "--package-uid", f.uid, "--max-records", strconv.Itoa(q.Records), "--max-selection-bytes", strconv.FormatUint(q.Selection, 10), "--max-output-bytes", strconv.FormatUint(q.Output, 10)}
	if action != "overview" {
		args = append(args, "--symbol", q.Symbol, "--direction", string(q.request().Direction), "--max-relations", strconv.Itoa(q.Relations))
		for _, kind := range q.request().Relations {
			args = append(args, "--relation", string(kind))
		}
		if action == "around" {
			args = append(args, "--max-depth", strconv.Itoa(q.Depth))
		}
	}
	return append(args, "--locked")
}

func candidatePackCommandOracle(t *testing.T, f candidateCatalogFixture, q candidatePackQueryControls, action string) ([]byte, error) {
	t.Helper()
	if action == "overview" {
		return candidatePackOverviewOracle(t, f, q.Records, q.Selection, q.Output)
	}
	if action == "show" {
		q.Depth = 0
	}
	return candidatePackQueryOracle(t, f, q, action == "around")
}

func checkCandidatePackCommand(t *testing.T, image, root string, f candidateCatalogFixture, q candidatePackQueryControls, action string, sdk bool) {
	t.Helper()
	golden, err := candidatePackCommandOracle(t, f, q, action)
	want := 0
	if err != nil {
		want = 2
	}
	args := candidatePackCommandArgs(f, q, action)
	run := fetchCLIProcess
	if sdk {
		run = func(t *testing.T, image, root string, args []string, _ string) (int, string, string) {
			return candidateAtlasSDKCLIProcess(t, image, root, args)
		}
	}
	code, out, stderr := run(t, image, root, args, "")
	if code != want || want == 0 && (out != string(golden) || stderr != "") || want != 0 && (out != "" || stderr == "") {
		var actual, expected struct{ Pack json.RawMessage }
		_ = json.Unmarshal([]byte(out), &actual)
		_ = json.Unmarshal(golden, &expected)
		t.Fatalf("pack %s v%d: %v\ncode=%d/%d err=%.500s\n%s\nbody %s", action, q.Version, err, code, want, stderr, fetchCLITextDiff(out, string(golden)), fetchCLITextDiff(string(actual.Pack), string(expected.Pack)))
	}
}

func verifyCandidatePackCommand(t *testing.T, image string) {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, `module main;import answer as a;
fn i32 leaf(){return a::value();}pub fn i32 isolated(){return 0;}
fn i32 self(i32 n){if(n==0){return leaf();}return self(n-1)+leaf();}
fn i32 main(){return self(2)+leaf()+leaf();}`)
	candidateCallsCommandSetup(t, root)
	before := remoteTree(t, root)
	observations := 0
	var f candidateCatalogFixture
	var q candidatePackQueryControls
	for version := 1; version <= 3; version++ {
		f = candidatePackCommandFixture(t, image, root, version)
		q = candidatePackCommandControls(t, f, "main")
		checkCandidatePackCommand(t, image, root, f, q, "overview", false)
		observations++
		for direction := 0; direction < 3; direction++ {
			q.Direction = direction
			q.Symbol = candidatePackCommandSubject(t, f, "main", direction == 2)
			for _, action := range []string{"show", "around"} {
				checkCandidatePackCommand(t, image, root, f, q, action, false)
				observations++
			}
		}
	}
	for _, action := range []string{"overview", "show", "around"} {
		golden, err := candidatePackCommandOracle(t, f, q, action)
		if err != nil {
			t.Fatal(err)
		}
		for _, output := range []int{len(golden) - 1, len(golden), len(golden) + 1} {
			next := q
			next.Output = uint64(output)
			checkCandidatePackCommand(t, image, root, f, next, action, false)
			observations++
		}
		next := q
		next.Selection = 1
		checkCandidatePackCommand(t, image, root, f, next, action, false)
		observations++
	}
	q.Depth = 0
	checkCandidatePackCommand(t, image, root, f, q, "around", false)
	observations++
	q.Symbol = candidatePackCommandSubject(t, f, "self", false)
	q.Depth = 3
	checkCandidatePackCommand(t, image, root, f, q, "around", false)
	observations++
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("locked packs changed the workspace")
	}

	base := candidatePackCommandArgs(f, q, "around")
	runFailure := func(dir string, args []string, want int) {
		t.Helper()
		observations++
		code, out, stderr := fetchCLIProcess(t, image, dir, args, "")
		if code != want || out != "" || stderr == "" {
			t.Fatalf("pack refusal %v: %d/%d %.500s %.500s", args, code, want, out, stderr)
		}
	}
	for _, change := range [][2]string{{"--package-uid", "example.missing"}, {"--symbol", "fysym1-sha256-128-00000000000000000000000000000000"}} {
		runFailure(root, replaceWorkspaceSessionArgument(base, change[0], change[1]), 2)
	}
	for _, args := range [][]string{{"experimental", "atlas-pack"}, {"experimental", "atlas-pack", "walk"}, append(slices.Clone(base), "--locked"), replaceWorkspaceSessionArgument(base, "--pack-version", "4"), append(candidatePackCommandArgs(f, q, "show"), "--max-depth", "0")} {
		empty := t.TempDir()
		runFailure(empty, args, 2)
		if len(remoteTree(t, empty)) != 0 {
			t.Fatal("pack argument refusal performed workspace I/O")
		}
	}
	empty := t.TempDir()
	runFailure(empty, base, 7)
	if len(remoteTree(t, empty)) != 0 {
		t.Fatal("missing locked workspace created files")
	}
	observations++
	if code, _, stderr := fetchCLIProcess(t, image, root, base, "stdout"); code != 4 || stderr == "" {
		t.Fatalf("pack stdout failure: %d %q", code, stderr)
	}
	lockBefore := mustReadAtlasFile(t, filepath.Join(root, "fairy.lock"))
	writeWorkspaceFile(t, root, "src/main.fy", string(mustReadAtlasFile(t, filepath.Join(root, "src/main.fy")))+"\n")
	stale := remoteTree(t, root)
	for _, action := range []string{"overview", "show", "around"} {
		runFailure(root, candidatePackCommandArgs(f, q, action), 7)
	}
	if !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatal("stale locked packs changed the workspace")
	}
	// Prepare fresh independent expectations, then restore the stale lock so the
	// authored unlocked command must perform the actual refresh itself.
	candidateCallsCommandSetup(t, root)
	f = candidatePackCommandFixture(t, image, root, 3)
	q = candidatePackCommandControls(t, f, "main")
	freshLock := mustReadAtlasFile(t, filepath.Join(root, "fairy.lock"))
	writeWorkspaceFile(t, root, "fairy.lock", string(lockBefore))
	if !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatal("independent oracle failed to restore stale workspace")
	}
	unlocked := candidatePackCommandArgs(f, q, "around")
	unlocked = unlocked[:len(unlocked)-1]
	golden, err := candidatePackCommandOracle(t, f, q, "around")
	if err != nil {
		t.Fatal(err)
	}
	observations++
	code, out, stderr := fetchCLIProcess(t, image, root, unlocked, "")
	if actual := mustReadAtlasFile(t, filepath.Join(root, "fairy.lock")); bytes.Equal(lockBefore, actual) || !bytes.Equal(freshLock, actual) {
		t.Fatal("unlocked pack did not refresh source lock")
	}
	if code != 0 || out != string(golden) || stderr != "" {
		t.Fatalf("unlocked fresh pack: %d %.5000s %.500s; want %.5000s", code, out, stderr, golden)
	}
	t.Logf("pack ordinary command native processes: %d", observations)
}
