package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestCandidateGenericAggregateUnitCache(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows aggregate unit cache replay; Linux execution is batched")
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_unit_cache.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::generic_aggregate_unit_cache;", "module main;", 1) + "\n@effects(filesystem,environment,clock,console) fn i32 main(){return run();}\n"
	candidateAtlasProbeSize(t, "project_compiler_core", main, 512<<10, func(t *testing.T, image string) {
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
		base := candidateAggregateLayoutFixture{name: "consumed generic body", root: "module root;import maker as m;fn i32 main(){return m::score(m::Cell<i32>{.value=42});}", maker: aggregateLayoutMaker + "pub fn i32 score(Cell<i32> value){return h::finish<i32>(value.value);}fn i32 hidden(){return 1;}"}
		root := candidateAggregateCacheWorkspace(t, base)
		cache := t.TempDir()
		replay := func(mode string, code int, output string) {
			t.Helper()
			status, out := candidateAtlasProcess(t, image, nil, filepath.ToSlash(root), filepath.ToSlash(cache), mode)
			if status != 0 {
				t.Fatalf("%s: %d %q", mode, status, out)
			}
			if mode == "invalid" {
				if out != "" {
					t.Fatalf("failed Complete published C: %q", out)
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
		if len(original) != 3 {
			t.Fatalf("unit records = %d", len(original))
		}
		for name, raw := range original {
			corrupt := bytes.Clone(raw)
			corrupt[len(corrupt)-1] ^= 1
			if err := os.WriteFile(filepath.Join(cache, name), corrupt, 0600); err != nil {
				t.Fatal(err)
			}
			replay("repair", 42, "")
			repaired, err := os.ReadFile(filepath.Join(cache, name))
			if err != nil || !bytes.Equal(raw, repaired) {
				t.Fatalf("unit repair %s: %v", name, err)
			}
			break
		}
		// Each process reopens the same retained directory and authenticates the
		// actual compiler image. No in-memory reuse can explain these hits.
		maker := filepath.Join(root, "libs", "maker", "src", "maker.fy")
		if err := os.WriteFile(maker, []byte(strings.Replace(base.maker, "return 1;", "return 2;", 1)), 0600); err != nil {
			t.Fatal(err)
		}
		replay("private", 42, "")
		replay("warm", 42, "")
		helper := filepath.Join(root, "libs", "helper", "src", "helper.fy")
		if err := os.WriteFile(helper, []byte(strings.Replace(aggregatePlanHelper, "return value;", "return value+1;", 1)), 0600); err != nil {
			t.Fatal(err)
		}
		replay("generic", 43, "")
		replay("warm", 43, "")
		before := candidateAggregateCacheFiles(t, cache)
		source := filepath.Join(root, "src", "root.fy")
		if err := os.WriteFile(source, []byte(strings.Replace(base.root, ".value=42", ".missing=42", 1)), 0600); err != nil {
			t.Fatal(err)
		}
		replay("invalid", 0, "")
		after := candidateAggregateCacheFiles(t, cache)
		if len(before) != len(after) {
			t.Fatal("failed Complete published a cache prefix")
		}
		for name, data := range before {
			if !bytes.Equal(data, after[name]) {
				t.Fatalf("failed Complete altered %s", name)
			}
		}
		if err := os.WriteFile(source, []byte(base.root), 0600); err != nil {
			t.Fatal(err)
		}
		replay("warm", 43, "")
		replay("exhausted", 43, "")
		for _, native := range candidateAggregateExecutions()[3:] {
			root = candidateAggregateCacheWorkspace(t, native.fixture)
			cache = t.TempDir()
			replay("cold", native.code, native.output)
			replay("warm", native.code, native.output)
		}
		root = candidateAggregateCacheWorkspace(t, candidateAggregateLayoutFixture{
			root:  "module root;import maker as m;fn i32 main(){return m::answer();}",
			maker: "module maker;import helper as h;pub fn i32 answer(){return h::finish<i32>(42);}",
		})
		cache = t.TempDir()
		replay("ordinarycold", 42, "")
		replay("ordinarywarm", 42, "")
	}, "project_compiler_input", "project_coordinate_graph", "project_module_visibility", "project_cache_environment", "project_cache_disk", "project_cache_unit_store", "project_cache_unit_key", "module_project_complete_c")
}

func candidateAggregateCacheWorkspace(t *testing.T, fixture candidateAggregateLayoutFixture) string {
	t.Helper()
	root := t.TempDir()
	manifest := func(name, extra string) string {
		return fmt.Sprintf("schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"example.aggregate.%s\"\nname = %q\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n%s", name, name, extra)
	}
	writeWorkspaceFile(t, root, "fairy.toml", manifest("root", "[workspace]\nmembers = [\"libs/helper\", \"libs/maker\"]\n[dependencies.maker]\npath = \"libs/maker\"\nversion = \"1.0.0\"\nfeatures = []\n"))
	writeWorkspaceFile(t, root, "src/root.fy", fixture.root)
	maker := fixture.maker
	if maker == "" {
		maker = aggregateLayoutMaker
	}
	writeWorkspaceFile(t, root, "libs/maker/fairy.toml", manifest("maker", "[dependencies.helper]\npath = \"libs/helper\"\nversion = \"1.0.0\"\nfeatures = []\n"))
	writeWorkspaceFile(t, root, "libs/maker/src/maker.fy", maker)
	writeWorkspaceFile(t, root, "libs/helper/fairy.toml", manifest("helper", ""))
	writeWorkspaceFile(t, root, "libs/helper/src/helper.fy", aggregatePlanHelper)
	return root
}

func candidateAggregateCacheFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	result := make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf("unexpected cache directory: %s", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		result[entry.Name()] = data
	}
	return result
}
