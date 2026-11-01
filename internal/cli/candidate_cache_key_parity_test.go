package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Exercise the authored implementation against complete bytes independently
// produced by the active Go oracle, through real native Windows executables.
func TestCandidateCacheKeyV5Parity(t *testing.T) {
	if (runtime.GOOS != "windows" && runtime.GOOS != "linux") || runtime.GOARCH != "amd64" {
		t.Skip("requires native Windows/Linux AMD64; Linux execution remains deferred on Windows")
	}
	for _, family := range []struct{ name, generator, file, calls string }{
		{"ordinary", "verify-candidate-cache-key.go", "project_cache_key.fy", `matches_actual_go_v5_bytes_and_all_invalidation_members(); rejects_missing_environment_and_body_or_package_facts(); rejects_unsorted_duplicate_empty_and_malformed_sets(); retains_detached_key_and_obeys_exact_phase_budget(); rejects_old_v4_key_namespace();`},
		{"generic", "verify-candidate-generic-cache-key.go", "project_generic_cache_key.fy", `matches_actual_go_generic_keys_and_complete_encoding(); rejects_incomplete_environment_dependency_coverage_and_noncanonical_ids(); retains_detached_keys_and_respects_shared_phase_budget();`},
	} {
		t.Run(family.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			fixture := filepath.Join(root, "oracle.bin")
			command := exec.CommandContext(t.Context(), "go", "run", "-mod=readonly", filepath.Join("tools", family.generator), "--output", fixture)
			command.Dir = filepath.Join("..", "..")
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("Go oracle: %v: %s", err, output)
			}
			for _, name := range []string{"project_cache_key.fy", "project_generic_cache_key.fy", "project_cache_encoding.fy", "sha256_stream.fy", "sha256.fy", "file_source.fy"} {
				data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
				if err != nil {
					t.Fatal(err)
				}
				writeTestSource(t, root, "src/"+name, string(data))
			}
			data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", family.file))
			if err != nil {
				t.Fatal(err)
			}
			program := strings.Replace(string(data), "module tests::"+strings.TrimSuffix(family.file, ".fy")+";", "module main;", 1)
			program = strings.ReplaceAll(program, "@test\n", "")
			program += "\n@effects(filesystem,environment) fn i32 main(){" + family.calls + "return 0;}\n"
			writeTestSource(t, root, "src/main.fy", program)
			for _, args := range [][]string{{"run", "--", fixture}, {"run", "--release", "--locked", "--", fixture}} {
				code, out, diagnostics := runForTest(args, root)
				if code != ExitSuccess || out != "" || diagnostics != "" {
					t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
				}
			}
		})
	}
}
