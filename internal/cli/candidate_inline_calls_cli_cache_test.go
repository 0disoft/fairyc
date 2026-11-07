package cli

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

// Forward declaration, positional noncommutative inputs, initializer + return.
// Keep the fixed marker vectors independent of the Fairy artifact oracle.
const inlineCallsCLIProvider = `module answer;
@inline pub fn i32 value(){let i32 first=leaf(48,4);return leaf(first,2);}
@inline pub fn i32 leaf(i32 a,i32 b){return a-b;}
@inline pub fn i32 spare(i32 a,i32 b){return a-b-1;}
@inline pub fn i32 unused(){return 7;}
@inline pub fn i32 legacy(i32 x){let i32 a=x+2;let b=a*2;return b/2;}
fn i32 private_helper(){return 1;}
`

type inlineCallsCLIEdit struct {
	name, source string
	value        int
	mixed        bool
}

func inlineCallsCLIEdits() []inlineCallsCLIEdit {
	leaf := strings.Replace(inlineCallsCLIProvider, "return a-b;", "return a-b+1;", 1)
	unused := strings.Replace(leaf, "return 7;", "return 8;", 1)
	private := strings.Replace(unused, "return 1;", "return 2;", 1)
	retarget := strings.Replace(private, "first=leaf(48,4);return leaf(first,2)", "first=spare(48,4);return spare(first,2)", 1)
	return []inlineCallsCLIEdit{
		{"leaf-only", leaf, 44, false},
		{"unused-inline-export", unused, 44, false},
		{"private-ordinary", private, 44, true},
		{"retarget-inline", retarget, 40, false},
	}
}

func inlineCallsCLIRefusals(valid string) []inlineCallsCLIEdit {
	return []inlineCallsCLIEdit{
		{name: "annotation-removed", source: strings.Replace(valid, "@inline pub fn i32 spare", "pub fn i32 spare", 1)},
		{name: "retarget-ordinary", source: strings.Replace(valid, "return spare(first,2)", "return ordinary(first,2)", 1) + "pub fn i32 ordinary(i32 a,i32 b){return a-b;}\n"},
		{name: "public-label", source: strings.Replace(valid, "spare(48,4)", "spare(.a=48,.b=4)", 1)},
	}
}

func inlineCallsCLIWorkspace(t *testing.T, generic bool) string {
	t.Helper()
	root := inlineCLIWorkspace(t)
	writeTestSource(t, root, "libs/answer/src/answer.fy", inlineCallsCLIProvider)
	declaration, call := "", "a::value()"
	checks := "assert(a::value()==42);"
	if generic {
		declaration = "fn i32 evaluate<T>(T marker){discard marker;return a::value();}"
		call = "evaluate<i32>(1)"
		checks = "assert(evaluate<i32>(1)==42);assert(evaluate<bool>(true)==42);"
	}
	main := "module main;import answer as a;" + declaration + "fn i32 main(){"
	if generic {
		main += "assert(evaluate<i32>(1)==evaluate<bool>(true));"
	}
	writeTestSource(t, root, "src/main.fy", main+"return "+call+";}")
	writeTestSource(t, root, "tests/inline.fy", "module tests::inline;import answer as a;"+declaration+inlineCLIPick+`@test fn void inline_calls_contract(){`+checks+`
var i32 n=0;assert(a::leaf(pick(ref mut n),pick(ref mut n))==-1);assert(n==2);
assert(a::leaf(9,2)==7);assert(a::leaf(2,9)==-7);assert(a::legacy(40)==42);}`)
	return root
}

// Native workspaces survive failures beside the digest-authenticated hosts.
// Fixture-only validation does not set this opt-in directory.
func inlineCallsCLIRetain(t *testing.T, root string) string {
	t.Helper()
	parent := os.Getenv("FAIRYC_INLINE_CALLS_CLI_ATTEMPT")
	if parent == "" {
		return root
	}
	if err := os.MkdirAll(parent, 0700); err != nil {
		t.Fatal(err)
	}
	destination, err := os.MkdirTemp(parent, "workspace-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(destination, os.DirFS(root)); err != nil {
		t.Fatal(err)
	}
	t.Logf("retained workspace: %s", destination)
	return destination
}

func inlineCallsCLIMarkers(t *testing.T, text string) map[string]string {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("answer.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil {
		t.Fatalf("Go provider parse: %v", err)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "answer.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified {
		t.Fatalf("Go provider check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	markers := map[string]string{}
	for _, fn := range checked.Semantic.HIR.Modules[0].Functions {
		markers[fn.Name] = fn.InlineBody
	}
	t.Logf("provider-sha256=%x markers=%v", sha256.Sum256([]byte(text)), markers)
	return markers
}

func inlineCallsCLIFingerprints(t *testing.T, root string) {
	t.Helper()
	for _, path := range []string{"src/main.fy", "tests/inline.fy", "libs/answer/src/answer.fy"} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s sha256=%x", path, sha256.Sum256(raw))
	}
}

// The existing fixture supports exact provider trap locations and report parity.
// Both unused initializers execute; reverse changes only their source order.
var inlineCallsCLITraps = inlineTrapFixture{
	cases: []inlineTrapCase{
		{"unused_division_first", "outer(2147483647,0)", "division-by-zero", 2},
		{"unused_overflow_second", "outer(2147483647,1)", "integer-overflow", 3},
		{"unused_overflow_first", "reverse(2147483647,0)", "integer-overflow", 3},
	},
	provider: `module answer;
@inline pub fn i32 divide(i32 a,i32 b){return a/b;}
@inline pub fn i32 overflow(i32 a){return a+1;}
@inline pub fn i32 outer(i32 a,i32 b){let unused=divide(a,b);let next=overflow(a);return 42;}
@inline pub fn i32 reverse(i32 a,i32 b){let unused=overflow(a);let next=divide(a,b);return 42;}
`,
	columns:    []int{47, 43, 43},
	assertions: "assert(a::outer(40,2)==42);assert(a::reverse(40,2)==42);",
}

// First step: Go control only, with no large authored Fairy compiler-host build.
// Stable Go run controls may compile small native targets through their usual path.
func TestCandidateInlineCallsCLIFixtures(t *testing.T) {
	baseline := inlineCallsCLIMarkers(t, inlineCallsCLIProvider)
	for name, want := range map[string]string{
		"value":  "fyinlinefn1-sha256-9b8978c9825c3bcc925ddeb7fe1c1b7f023aad5cc2756916b82a7e1061c887af",
		"leaf":   "fyinlinefn1-sha256-e1c7867e72719b32aaf40f42cbe5afec989b512e5f88e14cd312525177e2ebe2",
		"legacy": "fyinlinefn1-sha256-6aac2a75fa26963e634daa3970fe133abdd3cd7e440ae96e9a12cefd94231aaf",
	} {
		if baseline[name] != want {
			t.Fatalf("fixed %s marker: %s want %s", name, baseline[name], want)
		}
	}
	for _, generic := range []bool{false, true} {
		t.Run(fmt.Sprintf("generic-%t", generic), func(t *testing.T) {
			root := inlineCallsCLIWorkspace(t, generic)
			inlineCallsCLIFingerprints(t, root)
			if code, out, diagnostic := runForTest([]string{"test", "--json"}, root); code != 0 || diagnostic != "" || !strings.Contains(out, `"passed":1`) {
				t.Fatalf("Go test control: %d %s %s", code, out, diagnostic)
			}
			edits := append([]inlineCallsCLIEdit{{name: "baseline", source: inlineCallsCLIProvider, value: 42}}, inlineCallsCLIEdits()...)
			for i, edit := range edits {
				writeTestSource(t, root, "libs/answer/src/answer.fy", edit.source)
				markers := inlineCallsCLIMarkers(t, edit.source)
				if i > 0 && edit.name != "retarget-inline" && markers["value"] != baseline["value"] {
					t.Fatal("outer marker changed without changing outer text")
				}
				if edit.name == "leaf-only" && markers["leaf"] == baseline["leaf"] || edit.name == "retarget-inline" && markers["value"] == baseline["value"] {
					t.Fatal("leaf/target identity failed to change")
				}
				if code, out, diagnostic := runForTest([]string{"run"}, root); code != edit.value || out != "" || diagnostic != "" {
					t.Fatalf("Go %s: %d want %d %q %s", edit.name, code, edit.value, out, diagnostic)
				}
			}
			for _, edit := range inlineCallsCLIRefusals(edits[len(edits)-1].source) {
				writeTestSource(t, root, "libs/answer/src/answer.fy", edit.source)
				if code, _, diagnostic := runForTest([]string{"check"}, root); code != ExitSourceDiagnostics || diagnostic == "" {
					t.Fatalf("Go refusal %s: %d %s", edit.name, code, diagnostic)
				}
			}
			for i, item := range inlineCallsCLITraps.cases {
				root := inlineCallsCLITraps.workspace(t, i, generic)
				if code, out, diagnostic := runForTest([]string{"run"}, root); code != ExitProgramTrap || out != "" {
					t.Fatalf("Go trap: %d %q %s", code, out, diagnostic)
				} else {
					want := fmt.Sprintf("trap: %s\n  at libs/answer/src/answer.fy:%d:%d\n", item.category, item.line, inlineCallsCLITraps.columns[i])
					if diagnostic != want {
						t.Fatalf("Go trap provenance: %q want %q", diagnostic, want)
					}
				}
				if code, out, diagnostic := runForTest([]string{"test", "--json", "--locked"}, root); code != ExitProgramTrap || diagnostic != "" {
					t.Fatalf("Go trap test: %d %s %s", code, out, diagnostic)
				} else {
					inlineCallsCLITraps.checkReport(t, out, false, false)
				}
			}
		})
	}
}

func inlineCallsCLIPacket(t *testing.T, path string, definitions int) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := candidatebuild.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(packet.Header, "static inline int32_t fy_fn_"); n != 2*definitions {
		t.Fatalf("reachable shared-header inline prototypes/definitions: %d want %d", n, 2*definitions)
	}
	for _, unit := range packet.Units {
		if strings.Contains(unit, "static inline ") {
			t.Fatal("inline body escaped shared header")
		}
	}
	t.Logf("actual packet sha256=%x inline definitions=%d body units=%d", sha256.Sum256(raw), definitions, len(packet.Units))
}

func inlineCallsCLIExactCache(t *testing.T, command *copiedClosureCommand, value int, status string, args ...string) string {
	t.Helper()
	out, diagnostic := command.run(value, append(args, "--cache-root", "cache", "--cache-status")...)
	if diagnostic != status+"\n" {
		t.Fatalf("%v: diagnostics=%q want %q", args, diagnostic, status+"\n")
	}
	return out
}

func inlineCallsCLIStaleLock(t *testing.T, command *copiedClosureCommand) {
	t.Helper()
	out, diagnostic := command.run(7, "check", "--locked", "--cache-root", "cache", "--cache-status")
	if out != "" || diagnostic != "cache: skipped\nfairyc: package workspace failure: lockfile is stale for the current workspace\n" {
		t.Fatalf("stale locked source: %q %q", out, diagnostic)
	}
}

func inlineCallsCLIRebuilt(t *testing.T, command *copiedClosureCommand, value int, mixed bool, args ...string) {
	t.Helper()
	out, diagnostic := command.run(value, append(args, "--cache-root", "cache", "--cache-status")...)
	if out != "" || diagnostic != "cache: mixed\n" && (mixed || diagnostic != "cache: cold stored\n") {
		t.Fatalf("current sources were not rebuilt (mixed=%t): %q %q", mixed, out, diagnostic)
	}
	t.Logf("rebuilt: %s", strings.TrimSpace(diagnostic))
}

func inlineCallsCLIWarm(t *testing.T, command *copiedClosureCommand, value int) {
	t.Helper()
	before := copiedClosureCacheSnapshot(t, command.root)
	lock, err := os.ReadFile(filepath.Join(command.root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	inlineCallsCLIExactCache(t, command, 0, "cache: hit", "check", "--locked")
	if out := inlineCallsCLIExactCache(t, command, value, "cache: hit", "run", "--locked", "--release"); out != "" {
		t.Fatal(out)
	}
	locked, err := os.ReadFile(filepath.Join(command.root, "fairy.lock"))
	if err != nil || string(lock) != string(locked) || !reflect.DeepEqual(before, copiedClosureCacheSnapshot(t, command.root)) {
		t.Fatal("fresh locked processes changed lock or project generation names/bytes", err)
	}
}

func inlineCallsCLIAcceptance(t *testing.T, image string, generic bool) {
	t.Helper()
	root := inlineCallsCLIRetain(t, inlineCallsCLIWorkspace(t, generic))
	inlineCallsCLIFingerprints(t, root)
	packet := filepath.Join(root, "observed.bin")
	t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
	command := copiedClosureCommand{t: t, image: image, root: root}
	inlineCallsCLIExactCache(t, &command, 0, "cache: cold stored", "check")
	inlineCallsCLIWarm(t, &command, 42)
	inlineCallsCLIPacket(t, packet, 2)
	for _, flags := range [][]string{nil, {"--release"}} {
		if out := inlineCallsCLIExactCache(t, &command, 0, "cache: hit", append([]string{"build", "--locked"}, flags...)...); !strings.Contains(out, "built:") {
			t.Fatal(out)
		}
		inlineCallsCLIPacket(t, packet, 2)
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		out, err := exec.CommandContext(ctx, filepath.Join(root, "build", "inline-cli.exe")).CombinedOutput()
		cancel()
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 42 || len(out) != 0 {
			t.Fatalf("standalone inline-call executable: %v %q", err, out)
		}
		if out := inlineCallsCLIExactCache(t, &command, 42, "cache: hit", append([]string{"run", "--locked"}, flags...)...); out != "" {
			t.Fatal(out)
		}
	}
	out, diagnostic := command.run(0, "test", "--list", "--json", "--locked")
	if diagnostic != "" || !strings.Contains(out, "inline_calls_contract") || !strings.Contains(out, `"count":1`) {
		t.Fatalf("test list: %s %s", out, diagnostic)
	}
	for _, flags := range inlineTrapTestFlags {
		out, diagnostic := command.run(0, append([]string{"test", "--json", "--locked"}, flags...)...)
		if diagnostic != "" || !strings.Contains(out, `"passed":1`) || !strings.Contains(out, `"failed":0`) {
			t.Fatalf("inline calls test: %s %s", out, diagnostic)
		}
	}
	set := func(text string) { t.Helper(); writeTestSource(t, root, "libs/answer/src/answer.fy", text) }
	var selected string
	for _, edit := range inlineCallsCLIEdits() {
		t.Logf("edit %s source-sha256=%x", edit.name, sha256.Sum256([]byte(edit.source)))
		before := copiedClosureCacheSnapshot(t, root)
		set(edit.source)
		inlineCallsCLIStaleLock(t, &command)
		inlineCallsCLIRebuilt(t, &command, edit.value, edit.mixed, "run")
		after := copiedClosureCacheSnapshot(t, root)
		if reflect.DeepEqual(before, after) {
			t.Fatal("source edit did not publish a current generation")
		}
		inlineCallsCLIWarm(t, &command, edit.value)
		inlineCallsCLIPacket(t, packet, 2)
		if edit.name == "retarget-inline" {
			for name := range after {
				if _, exists := before[name]; !exists {
					// The store shares fyproject0 filenames/FYCACHE0 between
					// project ledgers and individual unit records. Select only
					// the whole-project FYCKEYS0 member, not a new FYPUNIT0.
					raw, err := os.ReadFile(filepath.Join(root, "cache", name))
					if err != nil {
						t.Fatal(err)
					}
					if len(raw) < 24 || string(raw[16:24]) != "FYCKEYS0" {
						continue
					}
					if selected != "" {
						t.Fatal("expected exactly one newly selected project generation")
					}
					selected = name
				}
			}
		}
	}
	if selected == "" {
		t.Fatal("selected generation not identified")
	}
	good := copiedClosureCacheSnapshot(t, root)
	if err := os.WriteFile(filepath.Join(root, "cache", selected), []byte("corrupt selected generation"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("corrupted only selected generation: %s sha256=%x", selected, good[selected])
	out, diagnostic = command.run(40, "run", "--locked", "--cache-root", "cache", "--cache-status")
	// Unit records remain valid: their reuse may heal the missing project
	// envelope. Whole-project hit would incorrectly accept the corrupt record.
	if out != "" || diagnostic != "cache: units hit\n" && diagnostic != "cache: mixed\n" && diagnostic != "cache: cold stored\n" {
		t.Fatalf("selected project generation was not repaired: %q %q", out, diagnostic)
	}
	t.Logf("selected-generation repair: %s", strings.TrimSpace(diagnostic))
	if !reflect.DeepEqual(good, copiedClosureCacheSnapshot(t, root)) {
		t.Fatal("selected generation repair did not restore exact good bytes or changed other generations")
	}
	inlineCallsCLIWarm(t, &command, 40)
	valid := inlineCallsCLIEdits()[3].source
	for _, refusal := range inlineCallsCLIRefusals(valid) {
		t.Logf("refusal %s source-sha256=%x", refusal.name, sha256.Sum256([]byte(refusal.source)))
		set(refusal.source)
		inlineCallsCLIStaleLock(t, &command)
		out, diagnostic := command.run(ExitSourceDiagnostics, "check", "--cache-root", "cache", "--cache-status")
		if out != "" || diagnostic != "cache: cold uncached\nlibs/answer/answer.fy:2:1: source is outside the supported compiler subset [bytes 15..22]\n" {
			t.Fatalf("invalid target/marker was not refused: %q %q", out, diagnostic)
		}
		if !reflect.DeepEqual(good, copiedClosureCacheSnapshot(t, root)) {
			t.Fatal("refused source changed good cache bytes")
		}
		set(valid)
		// Source admission follows normal lock projection. Refresh that lock
		// from restored good source while requiring the preserved project hit.
		inlineCallsCLIExactCache(t, &command, 0, "cache: hit", "check")
		inlineCallsCLIWarm(t, &command, 40)
	}
	t.Logf("generic=%t: %d fresh CLI processes + 2 standalone builds", generic, command.calls)
}

func inlineCallsCLITrapAcceptance(t *testing.T, image string, oracle inlineTrapOracle) {
	t.Helper()
	for family := 0; family < 2; family++ {
		var last string
		calls := 0
		for i, item := range inlineCallsCLITraps.cases {
			root := inlineCallsCLIRetain(t, inlineCallsCLITraps.workspace(t, i, family == 1))
			inlineCallsCLIFingerprints(t, root)
			packet := filepath.Join(root, "observed.bin")
			t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
			command := copiedClosureCommand{t: t, image: image, root: root}
			for warm := 0; warm < 2; warm++ {
				args := []string{"run", "--cache-root", "cache", "--cache-status"}
				status := "cache: cold stored\n"
				var before map[string][32]byte
				if warm == 1 {
					args = append(args, "--locked", "--release")
					status = "cache: hit\n"
					before = copiedClosureCacheSnapshot(t, root)
				}
				out, diagnostic := command.run(ExitProgramTrap, args...)
				if out != "" || diagnostic != status+oracle.diagnostics[family][i] {
					t.Fatalf("family=%d %s warm=%d: %q %q", family, item.name, warm, out, diagnostic)
				}
				if warm == 1 && !reflect.DeepEqual(before, copiedClosureCacheSnapshot(t, root)) {
					t.Fatal("trapping warm process changed cache bytes")
				}
			}
			inlineCallsCLIPacket(t, packet, 3)
			calls += command.calls
			last = root
		}
		command := copiedClosureCommand{t: t, image: image, root: last}
		for i, flags := range inlineTrapTestFlags {
			out, diagnostic := command.run(ExitProgramTrap, append([]string{"test", "--json", "--locked"}, flags...)...)
			inlineCallsCLITraps.checkReport(t, out, i != 0, i == 2)
			if diagnostic != "" || out != oracle.reports[family][i] {
				t.Fatalf("family=%d engine=%d: %q %q want %q", family, i, out, diagnostic, oracle.reports[family][i])
			}
		}
		t.Logf("traps generic=%t: %d fresh CLI processes", family == 1, calls+command.calls)
	}
}

// Start only after the coordinator's Fairy artifact pipeline passes. This test
// retains both profiles, ordinary limits, strict C warnings and fresh CLI execs.
func TestCandidateInlineCallsCLICache(t *testing.T) {
	t.Setenv("FAIRYC_CLOSURE_CLI_PROFILE", "")
	profiles := copiedClosureCLIProfiles(t)
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	// Independent Go references do not consume Fairy output as their oracle.
	oracle := inlineCallsCLITraps.reference(t)
	for _, profile := range profiles {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			for _, generic := range []bool{false, true} {
				t.Run(fmt.Sprintf("generic-%t", generic), func(t *testing.T) {
					inlineCallsCLIAcceptance(t, image, generic)
				})
			}
			t.Run("traps", func(t *testing.T) { inlineCallsCLITrapAcceptance(t, image, oracle) })
		})
	}
}
