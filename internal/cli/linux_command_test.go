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
	runLinuxCandidateFixture(t, []string{"linux_command"}, "linux_command", `
fn i32 main() {
    preserves_exact_bytes_empty_arguments_and_owned_storage();
    rejects_invalid_paths_and_arguments_without_changing_inputs();
    accounts_for_pointer_table_at_exact_byte_and_empty_argument_limits();
    return 0;
}
`, "")
}

func runLinuxCandidateFixture(t *testing.T, modules []string, fixtureName, entry, extraSupport string) {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.linux-command"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	for _, name := range modules {
		module, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, "src/"+name+".fy", string(module))
	}
	fixture, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", fixtureName+".fy"))
	if err != nil {
		t.Fatal(err)
	}
	client := strings.NewReplacer("module tests::"+fixtureName+";", "module main;", "@test ", "").Replace(string(fixture)) + entry
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
	var runtimeSymbols []string
	for _, symbol := range unit.ForeignSymbols {
		if strings.HasPrefix(symbol, "fy_std_") || strings.HasPrefix(symbol, "fy_private_vector_") {
			runtimeSymbols = append(runtimeSymbols, symbol)
		}
	}
	var support string
	if id == target.X8664LinuxGNU {
		support, err = linuxVectorSupportSource(runtimeSymbols)
	} else {
		support, err = windowsVectorSupportSource(runtimeSymbols, false)
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support + "\n" + extraSupport})
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{Timeout: 20 * time.Second})
			if err != nil || outcome.ExitCode != 0 || outcome.TrapCategory != "" || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("native argv transport: %+v %v", outcome, err)
			}
		})
	}
}
