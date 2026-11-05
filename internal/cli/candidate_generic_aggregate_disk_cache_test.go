package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestCandidateGenericAggregateDiskCache(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows aggregate whole-project cache replay; Linux execution is batched")
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_disk_cache.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::generic_aggregate_disk_cache;", "module main;", 1) + "\n@effects(filesystem,environment,clock,console) fn i32 main(){return run();}\n"
	candidateAtlasProbeSize(t, "project_cache_artifact", main, 512<<10, func(t *testing.T, image string) {
		toolchain, err := cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
		support, err := cruntime.ScalarSource(80)
		if err != nil {
			t.Fatal(err)
		}
		support += "\n#include <stdio.h>\nvoid observe_cleanup(int32_t value){printf(\"%d,\",(int)value);}\n"
		t.Setenv("PATH", "")
		base := candidateAggregateLayoutFixture{root: "module root;import maker as m;fn i32 main(){return m::score(m::Cell<i32>{.value=42});}", maker: aggregateLayoutMaker + "pub fn i32 score(Cell<i32> value){return h::finish<i32>(value.value);}fn i32 hidden(){return 1;}"}
		root := candidateAggregateCacheWorkspace(t, base)
		cache := t.TempDir()
		replay := func(mode string, code int, output string) {
			t.Helper()
			t.Logf("whole-project cache %s", mode)
			status, out := candidateAtlasProcess(t, image, nil, filepath.ToSlash(root), filepath.ToSlash(cache), mode)
			if status != 0 {
				t.Fatalf("%s: %d %q", mode, status, out)
			}
			if mode == "invalid" {
				if out != "" {
					t.Fatal("failed Complete emitted C")
				}
				return
			}
			packet, err := candidatebuild.Decode([]byte(out))
			if err != nil {
				t.Fatalf("%s C packet: %v", mode, err)
			}
			for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				built, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, consumer, packet.Header, packet.Literals, packet.Units, support)
				if err != nil {
					t.Fatalf("%s/%s: %v", mode, consumer.Name, err)
				}
				candidateAggregateExecuteImage(t, built.Bytes, code, output)
			}
		}
		replay("cold", 42, "")
		replay("warm", 42, "")
		original := candidateAggregateCacheFiles(t, cache)
		if len(original) != 1 {
			t.Fatalf("whole-project records = %d", len(original))
		}
		for name, data := range original {
			if !bytes.HasPrefix(data, []byte("FYCACHE1")) {
				t.Fatal("ordinary envelope used for aggregate project")
			}
			writeWorkspaceFile(t, root, "old.fyc", string(data))
			corrupt := bytes.Clone(data)
			corrupt[len(corrupt)-1] ^= 1
			if err := os.WriteFile(filepath.Join(cache, name), corrupt, 0600); err != nil {
				t.Fatal(err)
			}
			replay("repair", 42, "")
			repaired, err := os.ReadFile(filepath.Join(cache, name))
			if err != nil || !bytes.Equal(data, repaired) {
				t.Fatalf("whole-project repair: %v", err)
			}
		}
		writeWorkspaceFile(t, root, "libs/maker/src/maker.fy", strings.Replace(base.maker, "return 1;", "return 2;", 1))
		replay("private", 42, "")
		replay("warm", 42, "")
		writeWorkspaceFile(t, root, "libs/helper/src/helper.fy", strings.Replace(aggregatePlanHelper, "return value;", "return value+1;", 1))
		replay("generic", 43, "")
		replay("warm", 43, "")
		before := candidateAggregateCacheFiles(t, cache)
		writeWorkspaceFile(t, root, "src/root.fy", strings.Replace(base.root, ".value=42", ".missing=42", 1))
		replay("invalid", 0, "")
		after := candidateAggregateCacheFiles(t, cache)
		if len(before) != len(after) {
			t.Fatal("failed Complete changed cache inventory")
		}
		for name, data := range before {
			if !bytes.Equal(data, after[name]) {
				t.Fatalf("failed Complete altered %s", name)
			}
		}
		writeWorkspaceFile(t, root, "src/root.fy", base.root)
		replay("exhausted", 43, "")
		for _, native := range candidateAggregateExecutions()[3:] {
			root = candidateAggregateCacheWorkspace(t, native.fixture)
			cache = t.TempDir()
			replay("cold", native.code, native.output)
			replay("warm", native.code, native.output)
		}
		root = candidateAggregateCacheWorkspace(t, candidateAggregateLayoutFixture{root: "module root;import maker as m;fn i32 main(){return m::answer();}", maker: "module maker;import helper as h;pub fn i32 answer(){return h::finish<i32>(42);}"})
		cache = t.TempDir()
		replay("ordinarycold", 42, "")
		replay("ordinarywarm", 42, "")
	}, "project_compiler_input", "project_coordinate_graph", "project_module_visibility", "project_cache_environment", "project_cache_disk", "project_cache_unit_store", "project_cache_unit_key", "module_project_complete_c", "file_source")
}
