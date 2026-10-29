package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
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
	selectLinuxWorkspaceReaders(t, root)
	selectLinuxCEmission(t, root)
	entryPath := filepath.Join(root, "src", "main.fy")
	entry, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(entry), "import file_cli as file_cli;") != 1 {
		t.Fatal("candidate dispatcher import changed")
	}
	writeTestSource(t, root, "src/main.fy", strings.Replace(string(entry), "import file_cli as file_cli;", "import linux_file_cli as file_cli;\nimport linux_c_file as linux_wire;\nimport linux_build_cli as linux_build;", 1)+`
fn i32 cli_probe() {
    assert(file_cli::command(b"--help",2)==1 && file_cli::command(b"help",2)==1 && file_cli::command(b"-h",2)==1);
    assert(file_cli::command(b"--version",2)==2 && file_cli::command(b"targets",2)==3);
    assert(file_cli::command(b"--version",3)==0 && file_cli::command(b"targets",3)==0);
    assert(file_cli::command(b"--tokens-file",3)==4 && file_cli::command(b"--mir-file",3)==4);
    assert(file_cli::command(b"--tokens-file",2)==0 && file_cli::command(b"--mir-file",4)==0);
    assert(file_cli::command(b"check",2)==6 && file_cli::command(b"build",3)==7 && file_cli::command(b"run",4)==5);
    assert(file_cli::command(b"test",2)==5 && file_cli::command(b"new",3)==5);
    assert(file_cli::command(b"--c-file",3)==0 && file_cli::command(b"unknown",2)==0);
    assert(file_cli::source_mode(b"--declarations-file") && file_cli::source_mode(b"--check-names-file"));
    assert(file_cli::source_mode(b"--bindings-file") && file_cli::source_mode(b"--check-types-file") && file_cli::source_mode(b"--hir-file"));
    assert(bytes::equal(linux_wire::wire_name(6),b"fy_std_file_result_v0") && linux_wire::field_type(6,1)==1);
    assert(bytes::equal(linux_wire::field_name(6,1),b"error_number") && bytes::equal(linux_wire::field_name(7,1),b"os_code"));
    let output=switch(linux_build::output(b"client")){case .ok(p):{p}case .err(_):{assert(false);return 1;}};
    assert(bytes::equal(bytes::view(ref output),b"build/client"));
    let driver=switch(linux_build::companion(b"/opt/bin/fairyc")){case .ok(p):{p}case .err(_):{assert(false);return 1;}};
    assert(bytes::equal(bytes::view(ref driver),b"/opt/bin/fairyc-candidate-linux-build"));
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
	backend := buildLinuxCompanion(t)
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
				{[]string{"check", "--help"}, 0, "check [--locked]"},
				{[]string{"check", "--cache-root", "ignored"}, 2, "accepts only optional --locked"},
				{[]string{"check", "--unknown"}, 2, "accepts only optional --locked"},
				{[]string{"build", "--help"}, 0, "build [--release] [--locked]"},
				{[]string{"build", "--release", "--release"}, 2, ""},
				{[]string{"build", "--cache-root", "ignored"}, 2, "accepts only optional --release and --locked"},
				{[]string{"run"}, 3, "Linux workspace CLI is not connected"},
			}
			for _, tc := range cases {
				result, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: tc.args, WorkingDirectory: root, Timeout: 30 * time.Second})
				if err != nil || result.ExitCode != tc.code || !strings.Contains(result.Stdout+result.Stderr, tc.text) {
					t.Fatalf("%v: %+v %v", tc.args, result, err)
				}
			}
			checkLinuxWorkspace(t, toolchain, artifact)
			checkLinuxBuild(t, artifact, backend)
		})
	}
}

func selectLinuxCEmission(t *testing.T, root string) {
	t.Helper()
	for name, change := range map[string][2]string{
		"module_unit_c_sdk":  {"import module_unit_c_file as file_storage;", "import linux_c_file as file_storage;"},
		"module_unit_c_emit": {`#define FY_CANDIDATE_PROJECT_HEADER_V1\n`, `#define FY_CANDIDATE_PROJECT_HEADER_V1\n/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\n`},
	} {
		path := filepath.Join(root, "src", name+".fy")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(data), change[0]) != 1 {
			t.Fatalf("Linux emitter boundary changed: %s", name)
		}
		writeTestSource(t, root, "src/"+name+".fy", strings.Replace(string(data), change[0], change[1], 1))
	}
}

func selectLinuxWorkspaceReaders(t *testing.T, root string) {
	t.Helper()
	types := regexp.MustCompile(`\bfs::(FileError|FileKind|FileId|File|ReadDir|DirEntry|PathKind)\b`)
	for _, name := range []string{"project_directory", "project_source_catalog", "project_source_file", "project_source_set", "project_lock_write"} {
		path := filepath.Join(root, "src", name+".fy")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source := string(data)
		if name != "project_lock_write" {
			source = strings.Replace(source, "import std::fs as fs;", "import linux_workspace_fs as fs;", 1)
			source = types.ReplaceAllString(source, "fs::Workspace${1}")
		} else {
			source = strings.Replace(source, "import std::fs as fs;", "import std::fs as fs;\nimport linux_workspace_fs as directory;", 1)
			for _, operation := range []string{"read_dir_utf8", "close_dir", "path_kind"} {
				source = strings.ReplaceAll(source, "fs::"+operation+"(", "directory::"+operation+"(")
			}
			source = strings.ReplaceAll(source, "return .err(io(4,error));", "return .err(directory_io(4,error));")
			source += "\nfn Problem directory_io(u32 stage,directory::WorkspaceFileError error){var Problem p=fail(stage,error.status);p.os_code=error.os_code;p.completed=error.completed;return p;}\n"
		}
		writeTestSource(t, root, "src/"+name+".fy", source)
	}
}

func checkLinuxWorkspace(t *testing.T, toolchain cdriver.NativeToolchain, artifact cdriver.NativeArtifact) {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.linux-workspace-check"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return 0;}\n")
	run := func(args []string, want int32) string {
		t.Helper()
		result, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: args, WorkingDirectory: root, Timeout: time.Minute})
		if err != nil || result.ExitCode != want {
			t.Fatalf("workspace %v: %+v %v", args, result, err)
		}
		if len(args) == 2 && args[1] == "--locked" && want == 0 {
			code, stdout, stderr := runForTest(args, root)
			if code != 0 || result.Stdout != stdout || result.Stderr != stderr {
				t.Fatalf("locked check differs from Go: %+v vs (%d, %q, %q)", result, code, stdout, stderr)
			}
		}
		return result.Stdout + result.Stderr
	}
	run([]string{"check", "--locked"}, 7)
	if text := run([]string{"check"}, 0); !strings.Contains(text, "check ok:") {
		t.Fatal(text)
	}
	lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	run([]string{"check", "--locked"}, 0)
	after, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil || !bytes.Equal(lock, after) {
		t.Fatal("locked check changed lock", err)
	}
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return true;}\n")
	run([]string{"check", "--locked"}, 7)
	if text := run([]string{"check"}, 3); !strings.Contains(text, "E1006") {
		t.Fatal(text)
	}
	writeTestSource(t, root, "src/main.fy", "module main; import std::bytes as bytes; fn i32 main(){return 0;}\n")
	run([]string{"check"}, 0)
	run([]string{"check", "--locked"}, 0)
	for _, name := range []string{"fairy.toml", "src/main.fy", "tests/answer.fy", "libs/answer/fairy.toml", "libs/answer/src/answer.fy", "libs/answer/tests/answer.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "workspace_quickstart", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, name, string(data))
	}
	run([]string{"check"}, 0)
	run([]string{"check", "--locked"}, 0)
	if err := os.Symlink("main.fy", filepath.Join(root, "src", "link.fy")); err != nil {
		t.Fatal(err)
	}
	run([]string{"check"}, 7)
	if err := os.Remove(filepath.Join(root, "src", "link.fy")); err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/MAIN.fy", "module extra;\n")
	run([]string{"check"}, 7)
}
