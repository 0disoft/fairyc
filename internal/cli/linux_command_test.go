package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestLinuxCommandTransport(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.linux-command"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	module, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "linux_command.fy"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "linux_command.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/linux_command.fy", string(module))
	client := strings.NewReplacer("module tests::linux_command;", "module main;", "@test ", "").Replace(string(fixture))
	client += `
fn i32 main() {
    preserves_exact_bytes_empty_arguments_and_owned_storage();
    rejects_invalid_paths_and_arguments_without_changing_inputs();
    accounts_for_pointer_table_at_exact_byte_and_empty_argument_limits();
    return 0;
}
`
	writeTestSource(t, root, "src/main.fy", client)
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "Linux argv transport", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower: code=%d %s", code, &diagnostics)
	}
	units := map[target.ID]cbackend.Unit{}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, profile)
		if err != nil {
			t.Fatal(err)
		}
		units[target.ID(unit.Target)] = unit
	}
	id, err := cdriver.HostNativeTarget()
	if err != nil {
		t.Skipf("both-target emission passed; native execution unavailable: %v", err)
	}
	compiler := ""
	if id == target.X8664LinuxGNU {
		compiler = "gcc"
	}
	tc, err := cdriver.ResolveNativeToolchain(id, compiler)
	if err != nil {
		t.Fatal(err)
	}
	unit := units[id]
	var support string
	if id == target.X8664LinuxGNU {
		support, err = linuxVectorSupportSource(unit.ForeignSymbols)
	} else {
		support, err = windowsVectorSupportSource(unit.ForeignSymbols, false)
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support})
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{Timeout: 20 * time.Second, RejectOutput: true})
			if err != nil || outcome.ExitCode != 0 || outcome.TrapCategory != "" {
				t.Fatalf("native argv transport: %+v %v", outcome, err)
			}
		})
	}
}
