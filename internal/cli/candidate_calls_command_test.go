package cli

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
)

func candidateCallsCommandFixture(t *testing.T, image, root string, version int) candidateCallsFixture {
	t.Helper()
	var diagnostic bytes.Buffer
	p, code := prepareAtlasWorkspaceSchema(root, true, &diagnostic, []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV2, atlasmap.LogicalSchemaV3}[version])
	if code != 0 {
		t.Fatalf("locked calls source oracle: %d %s", code, &diagnostic)
	}
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := buildidentity.Identity(sha256.Sum256(data))
	// Narrow only the independently known non-call collection classes and image
	// identity. Call edges, original declarations, sources and IDs stay Go-owned.
	for i, part := range p.partitions {
		next, expected := candidateCatalogCommandPartition(t, part, p.expectations[part.Header().PackageInstance], compiler, version+1)
		p.partitions[i], p.expectations[part.Header().PackageInstance] = next, expected
	}
	return candidateCallsFixture{version: version, prepared: p}
}

func candidateCallsCommandSubject(t *testing.T, f candidateCallsFixture, uid, name string, full bool) (string, string) {
	t.Helper()
	for _, p := range f.prepared.partitions {
		if p.Header().PackageUID != uid {
			continue
		}
		for _, s := range p.Symbols() {
			if s.Declaration.Name == name {
				if full {
					return p.Header().PackageInstance, s.Full
				}
				return p.Header().PackageInstance, s.Visible
			}
		}
	}
	t.Fatalf("missing independent calls subject %s/%s", uid, name)
	return "", ""
}

func candidateCallsCommandArgs(version int, uid, symbol string, direction atlasmap.QueryDirection, budget atlasmap.WorkspaceCallBudget, output uint64) []string {
	return []string{"experimental", "atlas-calls", "list", "--catalog-version", strconv.Itoa(version), "--package-uid", uid, "--symbol", symbol, "--direction", string(direction), "--max-packages", strconv.Itoa(budget.MaxPackages), "--max-calls", strconv.Itoa(budget.MaxCalls), "--max-output-bytes", strconv.FormatUint(output, 10)}
}

func candidateCallsCommandSetup(t *testing.T, root string) {
	t.Helper()
	var diagnostic bytes.Buffer
	if _, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3); code != 0 {
		t.Fatalf("calls source setup: %d %s", code, &diagnostic)
	}
}

func verifyCandidateCallsCommand(t *testing.T, image string) {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, `module main;import answer as a;
pub fn i32 isolated(){return 0;}fn i32 leaf(){return a::value();}
fn i32 self(i32 n){if(n==0){return leaf();}return self(n-1)+leaf();}
fn i32 main(){return self(2)+leaf()+leaf();}`)
	candidateCallsCommandSetup(t, root)
	before := remoteTree(t, root)
	budget := atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128}
	const uid = "example.session.app"
	processes := 0
	run := func(root string, args []string, expected int, golden []byte) {
		t.Helper()
		processes++
		code, out, stderr := fetchCLIProcess(t, image, root, args, "")
		if code != expected || expected == 0 && (stderr != "" || out != string(golden)) || expected != 0 && (out != "" || stderr == "") {
			t.Fatalf("calls %v: code=%d\n got=%.4000s\n err=%.500s\nwant code=%d %.4000s", args, code, out, stderr, expected, golden)
		}
	}
	var f candidateCallsFixture
	var instance, symbol string
	var base, unlocked []string
	var golden []byte
	for version := 0; version <= 2; version++ {
		f = candidateCallsCommandFixture(t, image, root, version)
		for _, direction := range []atlasmap.QueryDirection{atlasmap.DirectionOutgoing, atlasmap.DirectionIncoming, atlasmap.DirectionBoth} {
			instance, symbol = candidateCallsCommandSubject(t, f, uid, "main", direction == atlasmap.DirectionBoth)
			var err error
			golden, err = candidateCallsOracle(t, f, budget, instance, symbol, direction, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			unlocked = candidateCallsCommandArgs(version, uid, symbol, direction, budget, 1<<20)
			base = append(slices.Clone(unlocked), "--locked")
			run(root, base, 0, golden)
		}
	}
	for _, output := range []int{len(golden) - 1, len(golden), len(golden) + 1} {
		expected := 0
		if output < len(golden) {
			expected = 2
		}
		run(root, replaceWorkspaceSessionArgument(base, "--max-output-bytes", strconv.Itoa(output)), expected, golden)
	}
	index, err := candidateCallsIndex(f, budget)
	if err != nil {
		t.Fatal(err)
	}
	for _, limits := range []atlasmap.WorkspaceCallBudget{{MaxPackages: len(f.prepared.partitions), MaxCalls: len(index.Calls())}, {MaxPackages: len(f.prepared.partitions) - 1, MaxCalls: len(index.Calls())}, {MaxPackages: len(f.prepared.partitions), MaxCalls: len(index.Calls()) - 1}} {
		golden, err := candidateCallsOracle(t, f, limits, instance, symbol, atlasmap.DirectionBoth, 1<<20)
		expected := 0
		if err != nil {
			expected = 2
		}
		run(root, append(candidateCallsCommandArgs(2, uid, symbol, atlasmap.DirectionBoth, limits, 1<<20), "--locked"), expected, golden)
	}
	_, self := candidateCallsCommandSubject(t, f, uid, "self", false)
	selfGolden, err := candidateCallsOracle(t, f, budget, instance, self, atlasmap.DirectionBoth, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	run(root, append(candidateCallsCommandArgs(2, uid, self, atlasmap.DirectionBoth, budget, 1<<20), "--locked"), 0, selfGolden)
	for _, change := range [][2]string{{"--package-uid", "example.missing"}, {"--symbol", "fysym1-sha256-128-00000000000000000000000000000000"}} {
		run(root, replaceWorkspaceSessionArgument(base, change[0], change[1]), 2, nil)
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("locked calls changed workspace")
	}
	for _, args := range [][]string{{"experimental", "atlas-calls"}, {"experimental", "atlas-calls", "show"}, append(slices.Clone(base), "--locked"), replaceWorkspaceSessionArgument(base, "--catalog-version", "3")} {
		empty := t.TempDir()
		run(empty, args, 2, nil)
		if len(remoteTree(t, empty)) != 0 {
			t.Fatal("calls argument rejection performed workspace I/O")
		}
	}
	empty := t.TempDir()
	run(empty, base, 7, nil)
	if len(remoteTree(t, empty)) != 0 {
		t.Fatal("missing locked workspace created files")
	}
	processes++
	if code, _, stderr := fetchCLIProcess(t, image, root, base, "stdout"); code != 4 || stderr == "" {
		t.Fatalf("calls stdout failure: %d %q", code, stderr)
	}
	lockBefore := mustReadAtlasFile(t, filepath.Join(root, "fairy.lock"))
	writeWorkspaceFile(t, root, "src/main.fy", string(mustReadAtlasFile(t, filepath.Join(root, "src/main.fy")))+"\n")
	stale := remoteTree(t, root)
	run(root, base, 7, nil)
	if !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatal("stale locked calls changed workspace")
	}
	// Discover the new current source selector independently, then restore the
	// exact stale lock so only the native invocation can refresh it for acceptance.
	candidateCallsCommandSetup(t, root)
	f = candidateCallsCommandFixture(t, image, root, 2)
	instance, symbol = candidateCallsCommandSubject(t, f, uid, "main", true)
	freshLock := mustReadAtlasFile(t, filepath.Join(root, "fairy.lock"))
	writeWorkspaceFile(t, root, "fairy.lock", string(lockBefore))
	if !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatal("independent source oracle failed to restore stale fixture")
	}
	unlocked = candidateCallsCommandArgs(2, uid, symbol, atlasmap.DirectionBoth, budget, 1<<20)
	processes++
	code, out, stderr := fetchCLIProcess(t, image, root, unlocked, "")
	if actual := mustReadAtlasFile(t, filepath.Join(root, "fairy.lock")); bytes.Equal(lockBefore, actual) || !bytes.Equal(freshLock, actual) {
		t.Fatal("unlocked calls did not refresh source lock")
	}
	golden, err = candidateCallsOracle(t, f, budget, instance, symbol, atlasmap.DirectionBoth, 1<<20)
	if err != nil || code != 0 || out != string(golden) || stderr != "" {
		t.Fatalf("unlocked fresh calls: %v %d %.4000s %.500s", err, code, out, stderr)
	}
	t.Logf("calls ordinary command native processes: %d", processes)
}
