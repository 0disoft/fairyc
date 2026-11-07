package cli

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

const closureMixedCacheRoot = `module main;import maker as m;fn i32 main(){let original=m::forward(40);let copy=original;let extracted=copy.action;let value=extracted(2);assert(m::probe()==0);return value;}`
const closureMixedCacheRelay = `module maker;import helper as h;pub fn h::Holder forward(i32 base){let action=h::four(base);let holder=h::Holder{.action=action};let copy=holder;return h::keep(copy);}pub fn i32 probe(){let zero=h::zero();let one=h::one(40);let two=h::two(40);let three=h::three(40);assert(zero(2)==42 && one(2)==42 && two(2)==42 && three(2)==42);return 0;}`
const closureMixedCacheProvider = `module helper;
pub struct Holder{fn i32(i32) action;}
pub fn Holder keep(Holder value){let copy=value;return copy;}
pub fn fn i32(i32) zero(){let fn i32(i32) action=fn [] i32(i32 x){return 40+x;};return action;}
pub fn fn i32(i32) one(i32 base){let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}
pub fn fn i32(i32) two(i32 base){let a=0;let fn i32(i32) action=fn [i32 base,i32 a] i32(i32 x){return base+a+x;};return action;}
pub fn fn i32(i32) three(i32 base){let a=0;let b=0;let fn i32(i32) action=fn [i32 base,i32 a,i32 b] i32(i32 x){return base+a+b+x;};return action;}
pub fn fn i32(i32) four(i32 base){let a=0;let b=0;let c=0;let fn i32(i32) action=fn [i32 base,i32 a,i32 b,i32 c] i32(i32 x){return base+a+b+c+x;};return action;}
pub fn i32 checked_later(){return 0;}`

func candidateClosureMixedCacheWorkspace(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_mixed_cache.fy"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_mixed_cache.fy", raw), source.DefaultLimits())
	if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
		t.Fatalf("closure mixed cache syntax: %v %+v", err, p.Tree.Diagnostics)
	}
	main := strings.Replace(string(raw), "module tests::closure_mixed_cache;", "module main;", 1) + "\n@effects(filesystem,environment,clock,console) fn i32 main(){return run();}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_compiler_core", "project_compiler_entry", "project_compiler_input", "project_coordinate_graph", "project_module_visibility", "project_cache_environment", "project_cache_disk", "project_cache_unit_store", "module_project_complete_c", "module_unit_body_ir_closures")
	writeTestSource(t, root, "src/main.fy", main)
	return root
}

func TestCandidateClosureMixedCacheSource(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root := candidateClosureMixedCacheWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure mixed cache source: %d %s %s", code, out, diagnostic)
	}
	for _, local := range []bool{false, true} {
		input := candidateClosureMixedCacheInput(t, local)
		if code, out, diagnostic := runForTest([]string{"check"}, input); code != ExitSuccess || diagnostic != "" {
			t.Fatalf("original closure workspace local=%t: %d %s %s", local, code, out, diagnostic)
		}
	}
	// Generic aggregate admission is a private prototype; stable Go check
	// deliberately rejects it. Use its existing explicit package oracle.
	workspace, _, err := loadWorkspaceWithLock(candidateClosureMixedCacheAggregateInput(t), false)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	checked, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !checked.Verified || len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		t.Fatalf("private aggregate closure source: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	t.Log("real mixed Complete factories, source entry selection and sealed Complete publisher source checked; no native execution")
}

func candidateClosureMixedCacheAggregateInput(t *testing.T) string {
	t.Helper()
	root := candidateClosureMixedCacheInput(t, false)
	// Aggregate UNIT admission follows actual generic aggregate types in the
	// canonical graph. Calling the aggregate factory alone adds no such types.
	main := strings.Replace(closureMixedCacheRoot, "fn i32 main()", "struct Cell<T>{T value;}fn i32 main()", 1)
	main = strings.Replace(main, "let value=extracted(2);", "let cell=Cell<i32>{.value=extracted(2)};let value=cell.value;", 1)
	writeWorkspaceFile(t, root, "src/main.fy", main)
	return root
}

func candidateClosureMixedCacheInput(t *testing.T, local bool) string {
	t.Helper()
	fixture := candidateAggregateLayoutFixture{root: closureMixedCacheRoot, maker: closureMixedCacheRelay}
	provider := closureMixedCacheProvider
	if local {
		// Only a body-local safe-fn type creates the leaf. Neither public nor
		// private signatures contain a closure; the dependencies stay scalar.
		fixture.root = `module main;import maker as m;fn i32 main(){let base=m::answer();let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};let copy=action;return copy(2);}`
		fixture.maker = `module maker;import helper as h;pub fn i32 answer(){return h::answer();}`
		provider = `module helper;pub fn i32 answer(){return 40;}`
	}
	root := candidateAggregateCacheWorkspace(t, fixture)
	if err := os.Rename(filepath.Join(root, "src", "root.fy"), filepath.Join(root, "src", "main.fy")); err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, "libs/helper/src/helper.fy", provider)
	return root
}

// Names and actual content hashes establish retained-set identity, not mtimes
// or directory counts alone. Every snapshot also authenticates the disk leaf.
func candidateClosureMixedCacheFiles(t *testing.T, root string) map[string][sha256.Size]byte {
	t.Helper()
	result := make(map[string][sha256.Size]byte)
	for name, raw := range candidateAggregateCacheFiles(t, root) {
		hash := sha256.Sum256(raw)
		if len(name) != 144 || name[:11] != "fyproject0-" || name[75] != '-' || name[140:] != ".fyc" || name[76:140] != fmt.Sprintf("%x", hash) {
			t.Fatalf("invalid content-addressed unit filename %q", name)
		}
		result[name] = hash
	}
	return result
}

func candidateClosureMixedCacheRetained(t *testing.T, before, after map[string][sha256.Size]byte, added int) {
	t.Helper()
	if len(after) != len(before)+added {
		t.Fatalf("unit generations before=%d after=%d want added=%d", len(before), len(after), added)
	}
	for name, hash := range before {
		if actual, ok := after[name]; !ok || actual != hash {
			t.Fatalf("retained unit changed or disappeared: %s", name)
		}
	}
}

func candidateClosureMixedCacheFamilies(t *testing.T, root string, scalar, closure int) {
	t.Helper()
	counts := map[string]int{}
	files := candidateAggregateCacheFiles(t, root)
	for _, raw := range files {
		if len(raw) < 8 {
			t.Fatal("truncated unit envelope")
		}
		counts[string(raw[:8])]++
	}
	if counts["FYCACHE0"] != scalar || counts["FYCACHE2"] != closure || len(files) != scalar+closure {
		t.Fatalf("unit families=%v want scalar=%d closure=%d", counts, scalar, closure)
	}
}

// Parent schedules this after the shared source freeze. Each producer profile
// gets its own retained directory; each replay launches a fresh OS process.
// Consumer C comes only from Complete::partition_entry, never a standalone unit.
func TestCandidateClosureMixedCacheNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows mixed closure cache; Linux execution remains batched")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	producer := candidateClosureMixedCacheWorkspace(t)
	build, support := candidateParityBuildParts(t, producer)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	consumerSupport, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		if !t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCCompilerHostPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(producer, "closure-mixed-cache-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			root := candidateClosureMixedCacheAggregateInput(t)
			cache := t.TempDir()
			feature := "aggregate"
			replay := func(mode string, want int) {
				t.Helper()
				code, out := candidateAtlasProcess(t, path, nil, filepath.ToSlash(root), filepath.ToSlash(cache), mode, feature)
				if code != 0 {
					t.Fatalf("%s/%s: exit=%d output=%q", feature, mode, code, out)
				}
				if mode == "invalid" {
					if out != "" {
						t.Fatalf("failed Complete emitted C: %q", out)
					}
					return
				}
				packet, err := candidatebuild.Decode([]byte(out))
				if err != nil {
					t.Fatalf("%s/%s Complete packet: %v", feature, mode, err)
				}
				// Matching low/optimized producer and consumer profiles avoid a
				// redundant cross product while executing both actual C profiles.
				built, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, profile, packet.Header, packet.Literals, packet.Units, consumerSupport)
				if err != nil {
					t.Fatalf("%s/%s/%s: %v", feature, mode, profile.Name, err)
				}
				candidateAggregateExecuteImage(t, built.Bytes, want, "")
			}
			// Check the aggregate/closure combination first, then all ordinary
			// and local-only controls. All profiles keep the same assertions.
			replay("cold", 42)
			aggregateFiles := candidateClosureMixedCacheFiles(t, cache)
			candidateClosureMixedCacheFamilies(t, cache, 0, 3)
			replay("warm", 42)
			candidateClosureMixedCacheRetained(t, aggregateFiles, candidateClosureMixedCacheFiles(t, cache), 0)
			root = candidateClosureMixedCacheInput(t, false)
			cache = t.TempDir()
			feature = "ordinary"
			replay("cold", 42)
			original := candidateClosureMixedCacheFiles(t, cache)
			if len(original) != 3 {
				t.Fatalf("cold unit generations=%d want=3", len(original))
			}
			candidateClosureMixedCacheFamilies(t, cache, 0, 3)
			replay("warm", 42)
			candidateClosureMixedCacheRetained(t, original, candidateClosureMixedCacheFiles(t, cache), 0)

			provider := filepath.Join(root, "libs", "helper", "src", "helper.fy")
			changed := strings.Replace(closureMixedCacheProvider, "base+a+b+c+x", "base+a+b+c+x+1", 1)
			write := func(path, text string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			write(provider, changed)
			replay("changed", 43) // Two restored callers, one current provider.
			afterProvider := candidateClosureMixedCacheFiles(t, cache)
			candidateClosureMixedCacheRetained(t, original, afterProvider, 1)
			replay("warm", 43)
			candidateClosureMixedCacheRetained(t, afterProvider, candidateClosureMixedCacheFiles(t, cache), 0)

			write(provider, closureMixedCacheProvider)
			caller := filepath.Join(root, "src", "main.fy")
			write(caller, strings.Replace(closureMixedCacheRoot, "extracted(2)", "extracted(3)", 1))
			replay("changed", 43) // Old provider generation plus a cold caller.
			afterCaller := candidateClosureMixedCacheFiles(t, cache)
			candidateClosureMixedCacheRetained(t, afterProvider, afterCaller, 1)
			replay("warm", 43)
			candidateClosureMixedCacheRetained(t, afterCaller, candidateClosureMixedCacheFiles(t, cache), 0)

			// The last package has a body-only type error. An uncached root
			// succeeds first; misses>0 proves this is not an early rejection.
			write(caller, strings.Replace(closureMixedCacheRoot, "extracted(2)", "extracted(4)", 1))
			write(provider, strings.Replace(closureMixedCacheProvider, "checked_later(){return 0;}", "checked_later(){return true;}", 1))
			replay("invalid", 0)
			candidateClosureMixedCacheRetained(t, afterCaller, candidateClosureMixedCacheFiles(t, cache), 0)
			retained := cache
			cache = t.TempDir()
			replay("invalid", 0) // Entirely cold prefix must also stay unpublished.
			if files := candidateClosureMixedCacheFiles(t, cache); len(files) != 0 {
				t.Fatalf("failed later unit published %d cold prefix records", len(files))
			}
			cache = retained
			write(provider, closureMixedCacheProvider)
			write(caller, closureMixedCacheRoot)
			replay("warm", 42)
			candidateClosureMixedCacheRetained(t, afterCaller, candidateClosureMixedCacheFiles(t, cache), 0)

			// A real package identity change invalidates canonical unit keys;
			// it must not be explained merely by a changed source body.
			manifestPath := filepath.Join(root, "fairy.toml")
			manifest, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			write(manifestPath, strings.Replace(string(manifest), "example.aggregate.root", "example.closure-mixed.changed", 1))
			replay("cold", 42)
			afterKey := candidateClosureMixedCacheFiles(t, cache)
			candidateClosureMixedCacheRetained(t, afterCaller, afterKey, 3)
			write(manifestPath, string(manifest))
			replay("warm", 42)
			candidateClosureMixedCacheRetained(t, afterKey, candidateClosureMixedCacheFiles(t, cache), 0)

			// One local-only closure UNIT and two legacy scalar UNITs must
			// coexist and all hit after reopening this exact directory.
			root = candidateClosureMixedCacheInput(t, true)
			cache = t.TempDir()
			feature = "local"
			replay("cold", 42)
			localFiles := candidateClosureMixedCacheFiles(t, cache)
			candidateClosureMixedCacheFamilies(t, cache, 2, 1)
			replay("warm", 42)
			candidateClosureMixedCacheRetained(t, localFiles, candidateClosureMixedCacheFiles(t, cache), 0)

		}) {
			return // Never schedule optimized work after a low-profile failure.
		}
	}
}
