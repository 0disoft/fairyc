package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestCandidateCacheDiskWindows(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows cache regression requires Windows amd64")
	}
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	fixture, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "project_cache_disk.fy"))
	if err != nil {
		t.Fatal(err)
	}
	client := strings.NewReplacer("module tests::project_cache_disk;", "module main;", "@test ", "").Replace(string(fixture)) + `
@effects(filesystem,clock,console) fn i32 main(){
    content_hash_matches_independent_padding_vectors_and_exact_work();
    immutable_binary_generations_round_trip_and_preserve_collision_metadata();
    corruption_is_skipped_and_exact_observed_collision_is_repaired();
    retention_enforces_exact_entry_byte_limits_and_preserves_data_on_early_failure();
    orphan_recovery_skips_live_stage_and_publication_failure_preserves_previous_bytes();
    return 0;
}
`
	writeTestSource(t, root, "src/main.fy", client)
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "Windows cache regression", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower: %d %s", code, &diagnostics)
	}
	unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	var standard []string
	for _, symbol := range unit.ForeignSymbols {
		if !strings.HasPrefix(symbol, "fairyc_private_windows_") {
			standard = append(standard, symbol)
		}
	}
	support, err := nativesupport.Source(target.X8664WindowsMSVC, standard, nativesupport.Options{SDKStorage: true})
	if err != nil {
		t.Fatal(err)
	}
	host, err := candidatehost.WindowsSource(strings.Join(unit.ForeignSymbols, "(\n") + "(")
	if err != nil {
		t.Fatal(err)
	}
	tc, err := cdriver.ResolveNativeToolchain(target.X8664WindowsMSVC, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support + "\n" + host})
			if err != nil {
				t.Fatal(err)
			}
			data := t.TempDir()
			manifest := filepath.Join(data, "cache-root.txt")
			if err := os.WriteFile(manifest, []byte(filepath.ToSlash(data)), 0600); err != nil {
				t.Fatal(err)
			}
			result, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: []string{manifest}, Timeout: 30 * time.Second, RejectOutput: true})
			if err != nil || result.ExitCode != 0 || result.TrapCategory != "" {
				t.Fatalf("cache: %+v %v", result, err)
			}
		})
	}
}
