package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

const closureUnitCacheSource = `module root;struct Holder{fn i32(i32) f;}fn fn i32(i32) make(i32 base){let fn i32(i32) f=fn [i32 base] i32(i32 x){return base+x;};return f;}fn i32 main(){let original=make(40);let copy=original;let holder=Holder{.f=copy};let stored=holder;let extracted=stored.f;return extracted(2);}`

func candidateClosureUnitCacheWorkspace(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_unit_cache.fy"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_unit_cache.fy", raw), source.DefaultLimits())
	if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
		t.Fatalf("closure unit cache fixture syntax: %v %+v", err, p.Tree.Diagnostics)
	}
	main := strings.Replace(string(raw), "module tests::closure_unit_cache;", "module main;", 1) + "\n@effects(filesystem,environment,clock,console) fn i32 main(){return run();}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_cache_unit_store", "project_cache_unit_key", "project_compiler_input", "project_module_visibility", "project_coordinate_graph", "project_source_plan", "project_cache_environment", "module_unit_c_emit")
	writeTestSource(t, root, "src/main.fy", main)
	return root
}

// This checks the real imported Fairy source, including all authored acceptance
// controls. It neither builds a native image nor runs the fixture assertions.
func TestCandidateClosureUnitCacheSource(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root := candidateClosureUnitCacheWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure unit cache source: %d %s %s", code, out, diagnostic)
	}
	t.Log("explicit FYCACHE2/FYPUNIT2/FYPROG02 APIs and authored key/version/source/work/disk controls source checked; no native execution")
}

func candidateClosureUnitCacheInput(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"example.closure-unit-cache\"\nname = \"closure-unit-cache\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n")
	writeWorkspaceFile(t, root, "src/root.fy", text)
	return root
}

// Parent runs this after freezing sources and extending the raw disk transport
// to accept FYCACHE2. The current disk owner is deliberately outside this patch.
// Both producer profiles reopen one real retained directory in fresh processes;
// each returned scalar-closure C packet executes under both consumer profiles.
func TestCandidateClosureUnitCacheNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows closure unit cache replay; Linux execution remains batched")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	producer := candidateClosureUnitCacheWorkspace(t)
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
			path := filepath.Join(producer, "closure-unit-cache-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			root := candidateClosureUnitCacheInput(t, closureUnitCacheSource)
			cache := t.TempDir()
			replay := func(mode string, want int, aggregate bool) {
				t.Helper()
				feature := "ordinary"
				if aggregate {
					feature = "aggregate"
				}
				code, out := candidateAtlasProcess(t, path, nil, filepath.ToSlash(root), filepath.ToSlash(cache), mode, feature)
				if code != 0 {
					t.Fatalf("%s/%s: exit=%d output=%q", feature, mode, code, out)
				}
				if mode == "invalid" {
					if out != "" {
						t.Fatalf("failed source published output: %q", out)
					}
					return
				}
				if aggregate {
					if out != "closure aggregate unit cache checked\n" {
						t.Fatalf("missing combined-mode receipt: %q", out)
					}
					return // Aggregate UNIT emission is a separate owner gate.
				}
				packet, err := candidatebuild.Decode([]byte(out))
				if err != nil {
					t.Fatalf("%s C packet: %v", mode, err)
				}
				for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					built, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, consumer, packet.Header, packet.Literals, packet.Units, consumerSupport)
					if err != nil {
						t.Fatalf("%s/%s: %v", mode, consumer.Name, err)
					}
					candidateAggregateExecuteImage(t, built.Bytes, want, "")
				}
			}
			replay("cold", 42, false)
			replay("warm", 42, false)
			original := candidateAggregateCacheFiles(t, cache)
			if len(original) != 1 {
				t.Fatalf("cold closure unit records=%d want=1", len(original))
			}
			for name, raw := range original {
				if !bytes.HasPrefix(raw, []byte("FYCACHE2")) {
					t.Fatalf("wrong unit envelope: %s", name)
				}
				bad := bytes.Clone(raw)
				bad[len(bad)-1] ^= 1
				if err := os.WriteFile(filepath.Join(cache, name), bad, 0600); err != nil {
					t.Fatal(err)
				}
				replay("repair", 42, false)
				repaired := candidateAggregateCacheFiles(t, cache)
				if len(repaired) != 1 || !bytes.Equal(repaired[name], raw) {
					t.Fatal("corrupt content-addressed generation was not repaired exactly")
				}
			}
			replay("warm", 42, false)
			sourcePath := filepath.Join(root, "src", "root.fy")
			changed := strings.Replace(closureUnitCacheSource, "base+x", "base+x+1", 1)
			if err := os.WriteFile(sourcePath, []byte(changed), 0600); err != nil {
				t.Fatal(err)
			}
			replay("changed", 43, false)
			replay("warm", 43, false)
			before := candidateAggregateCacheFiles(t, cache)
			if len(before) != 2 {
				t.Fatalf("source change should retain both generations, got %d", len(before))
			}
			if err := os.WriteFile(sourcePath, []byte(strings.Replace(changed, "extracted(2)", "extracted(true)", 1)), 0600); err != nil {
				t.Fatal(err)
			}
			replay("invalid", 0, false)
			after := candidateAggregateCacheFiles(t, cache)
			if len(before) != len(after) {
				t.Fatal("failed source changed the retained set")
			}
			for name, raw := range before {
				if !bytes.Equal(after[name], raw) {
					t.Fatalf("failed source changed %s", name)
				}
			}
			if err := os.WriteFile(sourcePath, []byte(closureUnitCacheSource), 0600); err != nil {
				t.Fatal(err)
			}
			replay("warm", 42, false) // Skip incompatible newer generation, keep old.
			replay("exhausted", 42, false)
			replay("io", 42, false)
			root = candidateClosureUnitCacheInput(t, `module root;struct Cell<T>{T value;}fn i32 apply(fn i32(i32) action,Cell<i32> cell){return action(cell.value);}fn i32 main(){let base=40;let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return apply(action,Cell<i32>{.value=2});}`)
			cache = t.TempDir()
			replay("cold", 42, true)
			replay("warm", 42, true)
		}) {
			return
		}
	}
}
