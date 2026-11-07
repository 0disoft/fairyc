package cli

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// main and tests select only add/i32. The multiply sibling and the entire i64
// bundle are supplied but unrequested; changing either must reach the consumer.
const groupedStaticCLIProvider = `module answer;
@interface(Arithmetic) pub fn T add<T>(T left,T right){}
@interface(Arithmetic) pub fn T multiply<T>(T left,T right){}
@implements(add,i32) fn i32 add32(i32 left,i32 right){return left+right+0;}
@implements(multiply,i32) fn i32 mul32(i32 left,i32 right){return left*right+0;}
@implements(add,i64) fn i64 add64(i64 left,i64 right){return left+right+0;}
@implements(multiply,i64) fn i64 mul64(i64 left,i64 right){return left*right+0;}
pub fn T algorithm<T>(T left,T right){return add<T>(left,right);}
pub fn i32 ordinary(i32 left,i32 right){return add32(left,right);}
fn i32 private_helper(){return 1;}
`

const groupedStaticCLIMain = `module main;import answer as a;
fn i32 main(){assert(a::ordinary(20,22)==a::algorithm<i32>(20,22));return a::algorithm<i32>(20,22);}`

const groupedStaticCLITests = `module tests::grouped;import answer as a;
fn i32 pick(ref mut i32 count){*count+=1;return *count;}
@test fn void grouped_contract(){
assert(a::ordinary(20,22)==42);assert(a::algorithm<i32>(20,22)==42);
assert(a::add<i32>(9,2)==11);var i32 count=0;
assert(a::algorithm<i32>(pick(ref mut count),pick(ref mut count))==3);assert(count==2);}`

type groupedStaticCLIEdit struct {
	name, source string
	value        int
	stableKeys   int
}

func groupedStaticCLIEdits() []groupedStaticCLIEdit {
	used := strings.Replace(groupedStaticCLIProvider, "add32(i32 left,i32 right){return left+right+0;}", "add32(i32 left,i32 right){return left+right+2;}", 1)
	sibling := strings.Replace(used, "mul32(i32 left,i32 right){return left*right+0;}", "mul32(i32 left,i32 right){return left*right+1;}", 1)
	tuple := strings.Replace(sibling, "mul64(i64 left,i64 right){return left*right+0;}", "mul64(i64 left,i64 right){return left*right+1;}", 1)
	private := strings.Replace(tuple, "private_helper(){return 1;}", "private_helper(){return 2;}", 1)
	return []groupedStaticCLIEdit{
		{"used-provider", used, 44, 0},
		{"unused-sibling-provider", sibling, 44, 0},
		{"unused-i64-provider-tuple", tuple, 44, 0},
		{"private-body", private, 44, 1},
	}
}

func groupedStaticCLIRefusals(valid string) []groupedStaticCLIEdit {
	return []groupedStaticCLIEdit{
		{name: "partial-unused-i64", source: strings.Replace(valid, "@implements(multiply,i64)", "", 1)},
		{name: "duplicate-unused-i64", source: valid + "@implements(add,i64) fn i64 duplicate(i64 left,i64 right){return left;}\n"},
		{name: "wrong-unused-i64-signature", source: strings.Replace(valid, "mul64(i64 left,i64 right)", "mul64(i32 left,i64 right)", 1)},
	}
}

func groupedStaticCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.grouped-static-cli"
name = "grouped-static-cli"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
[workspace]
members = ["libs/answer"]
[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`)
	writeTestSource(t, root, "libs/answer/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.grouped-static-provider"
name = "answer"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "libs/answer/src/answer.fy", groupedStaticCLIProvider)
	writeTestSource(t, root, "src/main.fy", groupedStaticCLIMain)
	writeTestSource(t, root, "tests/grouped.fy", groupedStaticCLITests)
	if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func groupedStaticCLIFingerprints(t *testing.T, root string) {
	t.Helper()
	for _, name := range []string{"fairy.toml", "libs/answer/fairy.toml", "libs/answer/src/answer.fy", "src/main.fy", "tests/grouped.fy"} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s sha256=%x", name, sha256.Sum256(raw))
	}
}

// These controls use the Go dispatcher/interpreter, not an authored Fairy host.
func TestCandidateGroupedStaticCLIFixtures(t *testing.T) {
	root := groupedStaticCLIWorkspace(t)
	groupedStaticCLIFingerprints(t, root)
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Go control executable=%s", identity.String())
	if code, out, diagnostic := runForTest([]string{"test", "--json"}, root); code != 0 || diagnostic != "" || !strings.Contains(out, `"passed":1`) || !strings.Contains(out, `"failed":0`) {
		t.Fatalf("Go interpreter test: %d %s %s", code, out, diagnostic)
	}
	edits := append([]groupedStaticCLIEdit{{name: "baseline", source: groupedStaticCLIProvider, value: 42}}, groupedStaticCLIEdits()...)
	previous := groupedStaticCLIProvider
	for _, edit := range edits {
		if len(edit.source) != len(groupedStaticCLIProvider) || edit.name != "baseline" && edit.source == previous {
			t.Fatal("edit must be real and equal-width", edit.name)
		}
		writeTestSource(t, root, "libs/answer/src/answer.fy", edit.source)
		if code, out, diagnostic := runForTest([]string{"check"}, root); code != 0 || diagnostic != "" {
			t.Fatalf("Go %s check: %d %s %s", edit.name, code, out, diagnostic)
		}
		workspace, err := workspaceloader.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		result, err := workspacecompiler.Check(workspace)
		if err != nil || !result.Verified {
			t.Fatalf("Go %s: %v %+v", edit.name, err, result.Semantic)
		}
		id, err := workspace.RootPackageID()
		if err != nil {
			t.Fatal(err)
		}
		out, err := interpreter.Execute(result.MIR, hir.QualifiedModulePath(id, "main")+"::main")
		if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != int32(edit.value) {
			t.Fatalf("Go %s value: %+v %v", edit.name, out, err)
		}
		names := map[string]string{}
		for _, d := range result.GenericProvenance.Declarations() {
			names[string(d.ID)] = d.Name
		}
		selected := map[string]int{}
		for _, origin := range result.GenericProvenance.Specializations() {
			selected[names[string(origin.Declaration)]]++
		}
		if !reflect.DeepEqual(selected, map[string]int{"algorithm": 1, "add": 1}) {
			t.Fatal("fixture unexpectedly selected multiply or i64 tuple", selected)
		}
		t.Logf("Go %s provider-sha256=%x result=%d", edit.name, sha256.Sum256([]byte(edit.source)), out.Value)
		previous = edit.source
	}
	for _, refusal := range groupedStaticCLIRefusals(previous) {
		writeTestSource(t, root, "libs/answer/src/answer.fy", refusal.source)
		if code, _, diagnostic := runForTest([]string{"check"}, root); code != ExitSourceDiagnostics || !strings.Contains(diagnostic, "E4501") {
			t.Fatalf("Go %s refusal: %d %s", refusal.name, code, diagnostic)
		}
		t.Logf("Go refusal %s provider-sha256=%x", refusal.name, sha256.Sum256([]byte(refusal.source)))
	}
}

func groupedStaticCLIExact(t *testing.T, command *copiedClosureCommand, code int, status string, args ...string) string {
	t.Helper()
	out, diagnostic := command.run(code, append(args, "--cache-root", "cache", "--cache-status")...)
	if diagnostic != status+"\n" {
		t.Fatalf("%v: diagnostic=%q want %q", args, diagnostic, status+"\n")
	}
	return out
}

func groupedStaticCLIStaleLock(t *testing.T, command *copiedClosureCommand) {
	t.Helper()
	out, diagnostic := command.run(7, "check", "--locked", "--cache-root", "cache", "--cache-status")
	if out != "" || diagnostic != "cache: skipped\nfairyc: package workspace failure: lockfile is stale for the current workspace\n" {
		t.Fatalf("stale lock: %q %q", out, diagnostic)
	}
}

func groupedStaticCLIWarm(t *testing.T, command *copiedClosureCommand, value int) {
	t.Helper()
	before := copiedClosureCacheSnapshot(t, command.root)
	lock, err := os.ReadFile(filepath.Join(command.root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	groupedStaticCLIExact(t, command, 0, "cache: hit", "check", "--locked")
	if out := groupedStaticCLIExact(t, command, value, "cache: hit", "run", "--release", "--locked"); out != "" {
		t.Fatal(out)
	}
	afterLock, err := os.ReadFile(filepath.Join(command.root, "fairy.lock"))
	if err != nil || string(lock) != string(afterLock) || !reflect.DeepEqual(before, copiedClosureCacheSnapshot(t, command.root)) {
		t.Fatal("fresh locked processes changed lock or cache names/bytes", err)
	}
}

// Project envelopes and per-unit records share FYCACHE0 and the filename family.
// Decode only the actual first member; FYCKEYS0 identifies the project ledger.
func groupedStaticCLIProject(t *testing.T, root string, before, after map[string][32]byte) (string, []byte) {
	t.Helper()
	var selected string
	var ledger []byte
	for name := range after {
		if _, exists := before[name]; exists {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, "cache", name))
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) < 24 || string(raw[16:24]) != "FYCKEYS0" {
			continue // Includes FYPUNIT0 records, which must never be corrupted here.
		}
		length := binary.BigEndian.Uint64(raw[8:16])
		if length < 8 || length > uint64(len(raw)-16) || selected != "" {
			t.Fatal("missing, malformed or ambiguous selected project ledger", name)
		}
		selected, ledger = name, raw[16:16+int(length)]
	}
	if selected == "" {
		t.Fatal("no newly selected FYCKEYS0 project generation")
	}
	t.Logf("selected project=%s sha256=%x ledger-sha256=%x", selected, after[selected], sha256.Sum256(ledger))
	return selected, ledger
}

// Read existing ledger facts, without recomputing the production cache key.
// Each row contains instance, coordinate, ordinary canonical key, generic key.
func groupedStaticCLILedger(t *testing.T, raw []byte) map[string][32]byte {
	t.Helper()
	if len(raw) < 16 || string(raw[:8]) != "FYCKEYS0" {
		t.Fatal("not a project ledger")
	}
	count, cursor := binary.BigEndian.Uint64(raw[8:16]), 16
	if count != 2 {
		t.Fatalf("fixture must have two package key rows, got %d", count)
	}
	rows := map[string][32]byte{}
	for range count {
		var fields [4][]byte
		for j := range fields {
			if cursor > len(raw)-8 {
				t.Fatal("short ledger length")
			}
			length := binary.BigEndian.Uint64(raw[cursor : cursor+8])
			cursor += 8
			if length == 0 || length > uint64(len(raw)-cursor) {
				t.Fatal("short ledger member")
			}
			fields[j] = raw[cursor : cursor+int(length)]
			cursor += int(length)
		}
		coordinate := hex.EncodeToString(fields[1])
		if _, duplicate := rows[coordinate]; duplicate {
			t.Fatal("duplicate package coordinate")
		}
		rows[coordinate] = sha256.Sum256(fields[3])
	}
	if cursor != len(raw) {
		t.Fatal("trailing project ledger bytes")
	}
	return rows
}

func groupedStaticCLIAcceptance(t *testing.T, image, artifacts string, full bool) {
	t.Helper()
	root, err := os.MkdirTemp(artifacts, "workspace-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(root, os.DirFS(groupedStaticCLIWorkspace(t))); err != nil {
		t.Fatal(err)
	}
	t.Logf("retained workspace=%s", root)
	groupedStaticCLIFingerprints(t, root)
	command := copiedClosureCommand{t: t, image: image, root: root}
	groupedStaticCLIExact(t, &command, 0, "cache: cold stored", "check")
	current := copiedClosureCacheSnapshot(t, root)
	_, ledger := groupedStaticCLIProject(t, root, nil, current)
	keys := groupedStaticCLILedger(t, ledger)
	groupedStaticCLIWarm(t, &command, 42)
	for _, flags := range [][]string{nil, {"--release"}} {
		if out := groupedStaticCLIExact(t, &command, 0, "cache: hit", append([]string{"build", "--locked"}, flags...)...); !strings.Contains(out, "built:") {
			t.Fatal(out)
		}
		path := filepath.Join(root, "build", "grouped-static-cli.exe")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		out, err := exec.CommandContext(ctx, path).CombinedOutput()
		cancel()
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 42 || len(out) != 0 {
			t.Fatalf("standalone grouped product: %v %q", err, out)
		}
		t.Logf("product flags=%v sha256=%x exit42 empty streams", flags, sha256.Sum256(raw))
		if out := groupedStaticCLIExact(t, &command, 42, "cache: hit", append([]string{"run", "--locked"}, flags...)...); out != "" {
			t.Fatal(out)
		}
	}
	out, diagnostic := command.run(0, "test", "--list", "--json", "--locked")
	if diagnostic != "" || !strings.Contains(out, "grouped_contract") || !strings.Contains(out, `"count":1`) {
		t.Fatalf("test list: %s %s", out, diagnostic)
	}
	for _, flags := range [][]string{nil, {"--native"}, {"--native", "--release"}} {
		out, diagnostic := command.run(0, append([]string{"test", "--json", "--locked"}, flags...)...)
		if diagnostic != "" || !strings.Contains(out, `"passed":1`) || !strings.Contains(out, `"failed":0`) {
			t.Fatalf("grouped test flags=%v: %s %s", flags, out, diagnostic)
		}
	}
	// Tests use their own source-set generations. Compare main package keys only
	// with the new whole-project ledger selected by each subsequent main run.
	var selected, valid string
	edits := groupedStaticCLIEdits()
	if !full {
		edits = edits[:1] // Low keeps one representative invalidation.
	}
	for _, edit := range edits {
		before := copiedClosureCacheSnapshot(t, root)
		writeTestSource(t, root, "libs/answer/src/answer.fy", edit.source)
		t.Logf("edit=%s provider-sha256=%x", edit.name, sha256.Sum256([]byte(edit.source)))
		groupedStaticCLIStaleLock(t, &command)
		out, diagnostic := command.run(edit.value, "run", "--cache-root", "cache", "--cache-status")
		if out != "" || diagnostic != "cache: mixed\n" && diagnostic != "cache: cold stored\n" || edit.stableKeys == 1 && diagnostic != "cache: mixed\n" {
			t.Fatalf("edit %s rebuild: %q %q", edit.name, out, diagnostic)
		}
		after := copiedClosureCacheSnapshot(t, root)
		selected, ledger = groupedStaticCLIProject(t, root, before, after)
		next := groupedStaticCLILedger(t, ledger)
		stable := 0
		for coordinate, old := range keys {
			if value, exists := next[coordinate]; !exists {
				t.Fatal("edit changed package coordinate")
			} else if value == old {
				stable++
			}
		}
		if stable != edit.stableKeys {
			t.Fatalf("%s: stable dependent/package generic keys=%d want %d", edit.name, stable, edit.stableKeys)
		}
		t.Logf("edit=%s cache=%s stable-package-generic-keys=%d/2", edit.name, strings.TrimSpace(diagnostic), stable)
		keys, valid = next, edit.source
		groupedStaticCLIWarm(t, &command, edit.value)
	}
	groupedStaticCLIRepairAndRefusals(t, &command, selected, valid, full)
	t.Logf("completed %d fresh authored CLI processes + 2 standalone products; full-vectors=%t retained=%s", command.calls, full, root)
}

func groupedStaticCLIRepairAndRefusals(t *testing.T, command *copiedClosureCommand, selected, valid string, full bool) {
	t.Helper()
	root := command.root
	good := copiedClosureCacheSnapshot(t, root)
	if err := os.WriteFile(filepath.Join(root, "cache", selected), []byte("corrupt selected grouped project generation"), 0600); err != nil {
		t.Fatal(err)
	}
	out, diagnostic := command.run(44, "run", "--locked", "--cache-root", "cache", "--cache-status")
	if out != "" || diagnostic != "cache: units hit\n" && diagnostic != "cache: mixed\n" && diagnostic != "cache: cold stored\n" {
		t.Fatalf("selected project was not repaired: %q %q", out, diagnostic)
	}
	if !reflect.DeepEqual(good, copiedClosureCacheSnapshot(t, root)) {
		t.Fatal("repair did not restore exact project bytes or changed other generations")
	}
	t.Logf("selected-generation repair=%s selected=%s good-sha256=%x", strings.TrimSpace(diagnostic), selected, good[selected])
	if full {
		groupedStaticCLIWarm(t, command, 44)
	}
	refusals := groupedStaticCLIRefusals(valid)
	if !full {
		refusals = refusals[:1]
	}
	for _, refusal := range refusals {
		writeTestSource(t, root, "libs/answer/src/answer.fy", refusal.source)
		groupedStaticCLIStaleLock(t, command)
		out, diagnostic := command.run(ExitSourceDiagnostics, "check", "--cache-root", "cache", "--cache-status")
		if out != "" || !strings.HasPrefix(diagnostic, "cache: cold uncached\n") || !strings.Contains(diagnostic, "answer.fy") {
			t.Fatalf("%s source not refused: %q %q", refusal.name, out, diagnostic)
		}
		if !reflect.DeepEqual(good, copiedClosureCacheSnapshot(t, root)) {
			t.Fatal("refused current source mutated good persistent cache")
		}
		t.Logf("refusal=%s provider-sha256=%x diagnostics=%q", refusal.name, sha256.Sum256([]byte(refusal.source)), diagnostic)
		writeTestSource(t, root, "libs/answer/src/answer.fy", valid)
		// Source diagnostics may follow lock publication: refresh unlocked first,
		// requiring the preserved good cache hit, then assert locked stability.
		groupedStaticCLIExact(t, command, 0, "cache: hit", "check")
		if full {
			groupedStaticCLIWarm(t, command, 44)
		} else {
			before := copiedClosureCacheSnapshot(t, root)
			groupedStaticCLIExact(t, command, 0, "cache: hit", "check", "--locked")
			if !reflect.DeepEqual(before, copiedClosureCacheSnapshot(t, root)) {
				t.Fatal("low restored-source locked hit changed cache bytes")
			}
		}
	}
}

// Resume only the interrupted low-profile tail. The already completed baseline
// and mutation assertions remain in attempt1's retained log; do not replay them.
func groupedStaticCLIResumeLow(t *testing.T, image, artifacts string) {
	t.Helper()
	root := os.Getenv("FAIRYC_GROUPED_STATIC_CLI_RESUME_LOW")
	relative, err := filepath.Rel(artifacts, root)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		t.Fatal("resume workspace must be inside the retained attempt", err)
	}
	valid := groupedStaticCLIEdits()[3].source
	raw, err := os.ReadFile(filepath.Join(root, "libs", "answer", "src", "answer.fy"))
	if err != nil || string(raw) != valid {
		t.Fatal("low resume source does not match completed private-body vector", err)
	}
	selected := os.Getenv("FAIRYC_GROUPED_STATIC_CLI_RESUME_PROJECT")
	if selected == "" || filepath.Base(selected) != selected {
		t.Fatal("resume requires the exact selected project basename")
	}
	raw, err = os.ReadFile(filepath.Join(root, "cache", selected))
	if err != nil || len(raw) < 24 || string(raw[16:24]) != "FYCKEYS0" {
		t.Fatal("resume selection is not a whole-project generation", err)
	}
	command := copiedClosureCommand{t: t, image: image, root: root}
	groupedStaticCLIExact(t, &command, 0, "cache: hit", "check", "--locked")
	groupedStaticCLIRepairAndRefusals(t, &command, selected, valid, false)
	t.Logf("low resumed tail complete: %d processes, repair + partial unused tuple refusal; baseline/mutations owned by interrupted attempt1", command.calls)
}

// The gate is set only after the coordinator reports grouped source/artifact
// readiness. Existing host timeout, 256KiB parts and product bounds are retained.
func TestCandidateGroupedStaticCLICache(t *testing.T) {
	if os.Getenv("FAIRYC_GROUPED_STATIC_CLI_GATE") != "passed" {
		t.Skip("wait for coordinator grouped pipeline pass before large host build")
	}
	profiles := copiedClosureCLIProfiles(t)
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	cache, err := filepath.Abs(filepath.Join("..", "..", ".cache"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	attempt := os.Getenv("FAIRYC_GROUPED_STATIC_CLI_REUSE_ARTIFACTS")
	if attempt == "" {
		attempt, err = os.MkdirTemp(cache, "grouped-static-cli-")
		if err != nil {
			t.Fatal(err)
		}
	} else {
		relative, err := filepath.Rel(cache, attempt)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			t.Fatal("reused artifact directory must be inside isolated cache", err)
		}
	}
	t.Setenv("FAIRYC_CLOSURE_CLI_ARTIFACTS", attempt)
	t.Logf("unique retained host/workspace artifacts=%s", attempt)
	for _, profile := range profiles {
		t.Run(profile.Name, func(t *testing.T) {
			started := time.Now()
			image := copiedClosureCLIHost(t, profile)
			receipt, err := os.ReadFile(filepath.Join(filepath.Dir(image), "host.json"))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("host profile=%s resolve-elapsed=%s identity=%s", profile.Name, time.Since(started), receipt)
			if profile.Name == "low" && os.Getenv("FAIRYC_GROUPED_STATIC_CLI_RESUME_LOW") != "" {
				groupedStaticCLIResumeLow(t, image, attempt)
			} else {
				groupedStaticCLIAcceptance(t, image, attempt, profile.Name == "optimized")
			}
			t.Logf("profile=%s actual elapsed=%s", profile.Name, time.Since(started))
		})
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("test executable=%s", identity.String())
}
