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

const closureProjectCacheSource = `module root;struct Holder{fn i32(i32) f;}fn fn i32(i32) make(i32 base){let fn i32(i32) f=fn [i32 base] i32(i32 x){return base+x;};return f;}fn i32 main(){let original=make(40);let copy=original;let holder=Holder{.f=copy};let stored=holder;let extracted=stored.f;return extracted(2);}`

func candidateClosureProjectCacheWorkspace(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_project_cache.fy"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_project_cache.fy", raw), source.DefaultLimits())
	if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
		t.Fatalf("closure project cache fixture syntax: %v %+v", err, p.Tree.Diagnostics)
	}
	main := strings.Replace(string(raw), "module tests::closure_project_cache;", "module main;", 1) + "\n@effects(filesystem,environment,clock,console) fn i32 main(){return run();}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_cache_artifact", "project_cache_lookup", "project_cache_keys", "project_compiler_cached", "module_project_complete_c", "project_compiler_input", "project_module_visibility", "project_coordinate_graph", "project_source_plan", "project_cache_environment", "module_unit_c_emit")
	writeTestSource(t, root, "src/main.fy", main)
	return root
}

// This checks the real imported Fairy source, including all authored acceptance
// controls. It neither builds a native image nor runs the fixture assertions.
func TestCandidateClosureProjectCacheSource(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root := candidateClosureProjectCacheWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure project cache source: %d %s %s", code, out, diagnostic)
	}
	t.Log("explicit FYCACHE2/FYCKEYS2/FYPRJ002 APIs and authored key/version/source/work/disk controls source checked; no native execution")
}

func candidateClosureProjectCacheInput(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"example.closure-project-cache\"\nname = \"closure-project-cache\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n")
	writeWorkspaceFile(t, root, "src/root.fy", text)
	writeWorkspaceFile(t, root, "src/extra.fy", `module extra;fn fn i32(i32) make(i32 seed){let fn i32(i32) other=fn [i32 seed] i32(i32 x){return seed-x;};return other;}`)
	return root
}

// Parent runs Native only after freezing all source writes and caller integration.
// Both producer profiles reopen one real retained directory in fresh processes;
// each returned whole-project C packet executes under both consumer profiles.
func TestCandidateClosureProjectCacheNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows closure project cache replay; Linux execution remains batched")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	producer := candidateClosureProjectCacheWorkspace(t)
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
			path := filepath.Join(producer, "closure-project-cache-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			root := candidateClosureProjectCacheInput(t, closureProjectCacheSource)
			mismatch := candidateClosureProjectCacheInput(t, strings.Replace(closureProjectCacheSource, "base+x", "base-x", 1))
			cache := t.TempDir()
			replay := func(mode string, want int, aggregate bool) {
				t.Helper()
				feature := "ordinary"
				if aggregate {
					feature = "aggregate"
				}
				code, out := candidateAtlasProcess(t, path, nil, filepath.ToSlash(root), filepath.ToSlash(cache), mode, feature, filepath.ToSlash(mismatch))
				if code != 0 {
					t.Fatalf("%s/%s: exit=%d output=%q", feature, mode, code, out)
				}
				if mode == "invalid" {
					if out != "" {
						t.Fatalf("failed source published output: %q", out)
					}
					return
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
			replay("pipeline-warm", 42, false)
			original := candidateAggregateCacheFiles(t, cache)
			if len(original) != 1 {
				t.Fatalf("cold closure project records=%d want=1", len(original))
			}
			for name, raw := range original {
				if !bytes.HasPrefix(raw, []byte("FYCACHE2")) {
					t.Fatalf("wrong project envelope: %s", name)
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
			changed := strings.Replace(closureProjectCacheSource, "base+x", "base+x+1", 1)
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
			if err := os.WriteFile(sourcePath, []byte(closureProjectCacheSource), 0600); err != nil {
				t.Fatal(err)
			}
			replay("warm", 42, false) // Original source route remains retained after a newer source generation.
			replay("exhausted", 42, false)
			replay("io", 42, false)
			cache = t.TempDir()
			replay("pipeline-cold", 42, false)
			replay("pipeline-warm", 42, false)
			aggregateSource := `module root;struct Cell<T>{T value;}fn i32 apply(fn i32(i32) action,Cell<i32> cell){return action(cell.value);}fn i32 main(){let base=40;let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return apply(action,Cell<i32>{.value=2});}`
			root = candidateClosureProjectCacheInput(t, aggregateSource)
			mismatch = candidateClosureProjectCacheInput(t, strings.Replace(aggregateSource, "base+x", "base-x", 1))
			cache = t.TempDir()
			replay("cold", 42, true)
			replay("warm", 42, true)
			replay("pipeline-warm", 42, true)
		}) {
			return
		}
	}
}
