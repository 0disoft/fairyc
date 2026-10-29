package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestLinuxFileCLI(t *testing.T) {
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	entryPath := filepath.Join(root, "src", "main.fy")
	entry, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(entry), "import file_cli as file_cli;") != 1 {
		t.Fatal("candidate dispatcher import changed")
	}
	writeTestSource(t, root, "src/main.fy", strings.Replace(string(entry), "import file_cli as file_cli;", "import linux_file_cli as file_cli;", 1)+`
fn i32 cli_probe() {
    assert(file_cli::command(b"--help",2)==1 && file_cli::command(b"help",2)==1 && file_cli::command(b"-h",2)==1);
    assert(file_cli::command(b"--version",2)==2 && file_cli::command(b"targets",2)==3);
    assert(file_cli::command(b"--version",3)==0 && file_cli::command(b"targets",3)==0);
    assert(file_cli::command(b"--tokens-file",3)==4 && file_cli::command(b"--mir-file",3)==4);
    assert(file_cli::command(b"--tokens-file",2)==0 && file_cli::command(b"--mir-file",4)==0);
    assert(file_cli::command(b"check",2)==5 && file_cli::command(b"build",3)==5 && file_cli::command(b"run",4)==5);
    assert(file_cli::command(b"test",2)==5 && file_cli::command(b"new",3)==5);
    assert(file_cli::command(b"--c-file",3)==0 && file_cli::command(b"unknown",2)==0);
    assert(file_cli::source_mode(b"--declarations-file") && file_cli::source_mode(b"--check-names-file"));
    assert(file_cli::source_mode(b"--bindings-file") && file_cli::source_mode(b"--check-types-file") && file_cli::source_mode(b"--hir-file"));
    return 0;
}
`)
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "Linux CLI", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower Linux CLI: %d %s", code, &diagnostics)
	}
	var linux cbackend.Unit
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, symbol := range unit.ForeignSymbols {
			if strings.Contains(symbol, "windows_") {
				t.Fatalf("Windows dependency leaked into Linux entry: %s", symbol)
			}
		}
		if profile.Target == target.X8664LinuxGNU {
			linux = unit
		}
	}
	support, err := nativesupport.Source(target.X8664LinuxGNU, linux.ForeignSymbols, nativesupport.Options{SDKStorage: true, CandidateHost: true})
	if err != nil || !strings.Contains(support, "fairyc_private_linux_root_file_read_v1") {
		t.Fatalf("Linux CLI runtime selection: %v", err)
	}
	t.Log("both-target full Linux entry emission and closed Linux support selection passed")
	t.Run("portable-dispatch", func(t *testing.T) {
		id, err := cdriver.HostNativeTarget()
		if err != nil {
			t.Skip(err)
		}
		unit, err := cbackend.EmitStorageTransfers(lowered.program, strings.TrimSuffix(lowered.entryName, "::main")+"::cli_probe", cbackend.Profile{Target: id, Dialect: cbackend.DialectC11})
		if err != nil {
			t.Fatal(err)
		}
		support, err := nativesupport.Source(id, unit.ForeignSymbols, nativesupport.Options{SDKStorage: true})
		if err != nil {
			t.Fatal(err)
		}
		toolchain, err := cdriver.ResolveNativeToolchain(id, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			artifact, err := toolchain.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support})
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{RejectOutput: true, Timeout: 30 * time.Second})
			if err != nil || outcome.ExitCode != 0 || outcome.TrapCategory != "" {
				t.Fatalf("%s dispatch: %+v %v", profile.Name, outcome, err)
			}
		}
	})
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native Linux CLI execution requires Linux amd64; emission and linkage selection passed")
	}
	toolchain, err := cdriver.ResolveNativeToolchain(target.X8664LinuxGNU, "gcc")
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "input.fy")
	if err := os.WriteFile(source, []byte("module main; fn i32 main(){return 0;}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "large.fy"), bytes.Repeat([]byte{' '}, 65537), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("input.fy", filepath.Join(root, "link.fy")); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := toolchain.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: linux.Source, RuntimeSource: support})
			if err != nil {
				t.Fatal(err)
			}
			cases := []struct {
				args []string
				code int32
				text string
			}{
				{[]string{"--help"}, 0, "lexer-candidate-linux"},
				{[]string{"--version"}, 0, "fairyc"},
				{[]string{"targets"}, 0, "x86_64-linux-gnu"},
				{[]string{"--check-types-file", "input.fy"}, 0, ""},
				{[]string{"--tokens-file", source}, 0, ""},
				{[]string{"--tokens-file", "large.fy"}, 2, "source limit"},
				{[]string{"--tokens-file", "link.fy"}, 2, "input file read failed"},
				{[]string{"--tokens-file", "absent.fy"}, 2, "input file read failed"},
				{[]string{"--tokens-file", "src"}, 2, "input file read failed"},
				{[]string{"--version", "extra"}, 2, "usage:"},
				{[]string{"--tokens-file"}, 2, "usage:"},
				{[]string{"unknown"}, 2, "usage:"},
				{[]string{"check"}, 3, "Linux workspace CLI is not connected"},
				{[]string{"build"}, 3, "Linux workspace CLI is not connected"},
				{[]string{"run"}, 3, "Linux workspace CLI is not connected"},
			}
			for _, tc := range cases {
				result, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: tc.args, WorkingDirectory: root, Timeout: 30 * time.Second})
				if err != nil || result.ExitCode != tc.code || !strings.Contains(result.Stdout+result.Stderr, tc.text) {
					t.Fatalf("%v: %+v %v", tc.args, result, err)
				}
			}
		})
	}
}
