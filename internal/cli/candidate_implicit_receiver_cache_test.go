package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestCandidateImplicitReceiverPersistence(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows implicit receiver persistence; Linux execution is batched")
	}
	for _, port := range []string{"unit", "disk"} {
		t.Run(port, func(t *testing.T) {
			name := "generic_aggregate_" + port + "_cache"
			raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", name+".fy"))
			if err != nil {
				t.Fatal(err)
			}
			main := strings.Replace(string(raw), "module tests::"+name+";", "module main;import syntax as syntax;", 1) + "\n@effects(filesystem,environment,clock,console) fn i32 main(){return run();}\n"
			// The original cache fixture has one ordinary root function. Method
			// fixtures have several; select the actual written main, not the first.
			const oldEntry = "if(f.kind==3 && input::origin(source,f.module_index).owner==0)"
			if strings.Count(main, oldEntry) != 1 {
				t.Fatal("missing cache fixture entry selector")
			}
			main = strings.Replace(main, oldEntry, `if((f.kind==1 || f.kind==3) && input::origin(source,f.module_index).owner==0 && bytes::equal(syntax::declaration_name(input::source(source,f.module_index),f.declaration),b"main"))`, 1)
			candidateAtlasProbeSize(t, "project_compiler_core", main, 512<<10, func(t *testing.T, image string) {
				toolchain, err := cdriver.ResolveWindowsHostMSVC()
				if err != nil {
					t.Fatal(err)
				}
				support, err := cruntime.ScalarSource(80)
				if err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", "")
				for _, f := range implicitReceiverExecutionSources() {
					t.Run(f.name, func(t *testing.T) {
						root := candidateAggregateCacheWorkspace(t, candidateAggregateLayoutFixture{
							root:  strings.Replace(f.source, "module main;", "module root;", 1),
							maker: "module maker;pub fn i32 unused(){return 1;}",
						})
						if code, _, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess {
							t.Fatalf("Go cache source: %d %s", code, diagnostic)
						}
						before := remoteTree(t, root)
						cache := t.TempDir()
						var cold []byte
						var saved map[string][]byte
						for _, mode := range []string{"ordinarycold", "ordinarywarm"} {
							code, out := candidateAtlasProcess(t, image, nil, filepath.ToSlash(root), filepath.ToSlash(cache), mode)
							if code != 0 {
								t.Fatalf("%s: %d %.3000s", mode, code, out)
							}
							packet, err := candidatebuild.Decode([]byte(out))
							if err != nil {
								t.Fatal(err)
							}
							if mode == "ordinarycold" {
								cold = []byte(out)
								saved = candidateAggregateCacheFiles(t, cache)
								want := 1
								if port == "unit" {
									want = 3
								}
								if len(saved) != want {
									t.Fatalf("stored records: %d/%d", len(saved), want)
								}
								continue
							}
							if string(cold) != out || !reflect.DeepEqual(saved, candidateAggregateCacheFiles(t, cache)) {
								t.Fatal("restored C or immutable cache bytes changed")
							}
							built, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, cdriver.LowProfile, packet.Header, packet.Literals, packet.Units, support)
							if err != nil {
								t.Fatal(err)
							}
							candidateAggregateExecuteImage(t, built.Bytes, 0, "")
						}
						if !reflect.DeepEqual(before, remoteTree(t, root)) {
							t.Fatal("cache replay changed workspace bytes")
						}
					})
				}
			}, "project_cache_artifact", "project_cache_keys", "project_cache_artifact_wire", "project_cache_lookup", "module_program_artifact_wire", "project_compiler_input", "project_coordinate_graph", "project_module_visibility", "project_cache_environment", "project_cache_disk", "project_cache_unit_store", "project_cache_unit_key", "module_project_complete_c", "file_source")
		})
	}
}
