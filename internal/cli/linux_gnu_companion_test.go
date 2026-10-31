package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

var linuxGNUCompanionModules = []string{"linux_gnu_companion", "linux_gnu_build", "linux_gnu_host", "linux_gnu_policy", "linux_command", "linux_elf", "linux_runtime", "linux_runtime_contract", "native_packet_decode", "sha256", "linux_runtime_assets", "linux_runtime_asset_pin", "linux_companion_input", "linux_workspace_fs", "linux_directory", "linux_rooted_file", "linux_host_paths", "native_cli_options"}

func TestLinuxGNUCompanionOptions(t *testing.T) {
	runLinuxCandidateFixture(t, linuxGNUCompanionModules, "linux_gnu_companion", `fn i32 main(){accepts_build_run_and_explicit_compiler_options();rejects_ambiguous_flags_and_nonprivate_trap_outputs();return 0;}`, "")
}

func linuxGNUCompanionUnit(t *testing.T) cbackend.BuildUnit {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.gnu-companion"
name = "companion"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	for _, name := range linuxGNUCompanionModules {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, "src/"+name+".fy", string(data))
	}
	writeTestSource(t, root, "src/main.fy", "module main;\nimport linux_gnu_companion as companion;\n@effects(filesystem,console,environment,clock) fn i32 main(){return companion::run();}\n")
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "GNU companion", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower GNU companion: %d %s", code, &diagnostics)
	}
	var linux cbackend.BuildUnit
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := emitNativeBuildUnit(lowered.program, lowered.entryName, profile, true)
		if err != nil {
			t.Fatal(err)
		}
		for _, symbol := range unit.Unit.ForeignSymbols {
			if strings.Contains(symbol, "windows_") {
				t.Fatalf("Windows dependency in GNU companion: %s", symbol)
			}
		}
		if profile.Target == target.X8664LinuxGNU {
			linux = unit
		}
	}
	return linux
}

func TestLinuxGNUCompanionEntryEmission(t *testing.T) {
	unit := linuxGNUCompanionUnit(t)
	support, err := nativesupport.Source(target.X8664LinuxGNU, unit.Unit.ForeignSymbols, nativesupport.Options{SDKStorage: true, CandidateHost: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(support, "fairyc_private_linux_gnu_publish_v1") || !strings.Contains(support, "fairyc_private_linux_gnu_run_v1") {
		t.Fatal("missing GNU native adapter selection")
	}
	t.Log("complete companion emits for both targets with closed Linux support and no Windows foreign symbols")
}
