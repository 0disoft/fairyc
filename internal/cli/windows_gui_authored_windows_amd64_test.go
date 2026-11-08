package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/pecoff"
)

func windowsGUIExample(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"fairy.toml", "fairy.windows.json", "src/main.fy", "native/window.c", "native/app.rc"} {
		data, err := os.ReadFile(filepath.Join("../../examples/win32_minimal", name))
		if err != nil {
			t.Fatal(err)
		}
		writeWorkspaceFile(t, root, name, string(data))
	}
	return root
}

func windowsGUIExecute(t *testing.T, image string, subsystem uint16) []byte {
	t.Helper()
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pecoff.InspectExecutableSubsystem(data, subsystem); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, image, "--smoke").CombinedOutput(); err != nil || len(out) != 0 {
		t.Fatalf("window/message loop: %v %s", err, out)
	}
	t.Logf("subsystem %d executable: %d bytes", subsystem, len(data))
	return data
}

func TestWindowsGUIExample(t *testing.T) {
	root := windowsGUIExample(t)
	for _, flags := range [][]string{nil, {"--release"}} {
		if code, out, diag := runForTest(append([]string{"build"}, flags...), root); code != 0 || diag != "" {
			t.Fatalf("build %v: %d %s %s", flags, code, out, diag)
		}
		image := filepath.Join(root, "build/win32-minimal.exe")
		windowsGUIExecute(t, image, 2)
		name := "development-control"
		if len(flags) != 0 {
			name = "release-control"
		}
		t.Run(name, func(t *testing.T) {
			python, err := exec.LookPath("python")
			if err != nil {
				t.Skip("optional Win32 JSONL controller requires Python 3")
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, python, "-B", "../../examples/win32_minimal/control.py", "--exe", image, "--verify").CombinedOutput()
			if err != nil {
				t.Fatalf("CLI control events: %v %s", err, out)
			}
			var report struct {
				OK            bool
				Verification  string
				ProcessExited bool
			}
			if err := json.Unmarshal(out, &report); err != nil || !report.OK || report.Verification != "real-win32-controls" || !report.ProcessExited {
				t.Fatalf("invalid CLI control receipt: %v %s", err, out)
			}
		})
	}
}

func TestWindowsGUIEventFailure(t *testing.T) {
	root := windowsGUIExample(t)
	path := filepath.Join(root, "src/main.fy")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(source), `b"Result: "`, `b"Broken: "`, 1)
	if broken == string(source) {
		t.Fatal("missing Fairy result mutation target")
	}
	writeWorkspaceFile(t, root, "src/main.fy", broken)
	if code, out, diag := runForTest([]string{"build", "--release"}, root); code != 0 || diag != "" {
		t.Fatalf("build negative control: %d %s %s", code, out, diag)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, filepath.Join(root, "build/win32-minimal.exe"), "--smoke").CombinedOutput()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 5 || len(out) != 0 {
		t.Fatalf("broken button event must fail smoke with 5: %v %s", err, out)
	}
}

// Build only the companion's transitive authored modules with the Go bootstrap.
// Application compilation/linking below executes the real Fairy companion.
func windowsGUIAuthoredDriver(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.gui-driver\"\nname = \"gui-driver\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n")
	writeWorkspaceFile(t, root, "src/main.fy", "module main;import native_build_cli as cli;@effects(filesystem,console,environment,clock) fn i32 main(){return cli::run();}")
	moduleRE := regexp.MustCompile(`(?m)^module\s+([\w:]+);`)
	importRE := regexp.MustCompile(`(?m)^import\s+([\w:]+)`)
	modules := map[string]string{}
	sources, err := filepath.Glob("../../examples/lexer_candidate/src/*.fy")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		match := moduleRE.FindSubmatch(data)
		if len(match) == 2 {
			modules[string(match[1])] = string(data)
		}
	}
	seen := map[string]bool{}
	pending := []string{"native_build_cli"}
	for len(pending) > 0 {
		name := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[name] || strings.HasPrefix(name, "std::") {
			continue
		}
		seen[name] = true
		source, ok := modules[name]
		if !ok {
			t.Fatalf("missing companion module %s", name)
		}
		writeWorkspaceFile(t, root, "src/"+name+".fy", source)
		for _, imported := range importRE.FindAllStringSubmatch(source, -1) {
			pending = append(pending, imported[1])
		}
	}
	build, support := candidateParityBuildPartsWithSystemImports(t, root, map[string]bool{"CloseHandle": true, "CreateFileW": true, "GetFileInformationByHandleEx": true, "GetFinalPathNameByHandleW": true, "GetLastError": true})
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := cdriver.BuildWindowsMSVCCompilerHostPartsWithStackReserve(tc, cdriver.LowProfile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if parent := os.Getenv("FAIRYC_WINDOWS_GUI_DRIVER_ARTIFACTS"); parent != "" {
		home = parent
		if err := os.MkdirAll(home, 0700); err != nil {
			t.Fatal(err)
		}
	}
	image := filepath.Join(home, "fairyc-candidate-build.exe")
	if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), "python", "-B", "tools/package-candidate-runtime-assets.py", "--output", filepath.Join(home, "fairyc-runtime-assets.pack"))
	command.Dir = "../.."
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("runtime asset pack: %v %s", err, out)
	}
	return image
}

func TestWindowsGUIAuthoredDriver(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 to bootstrap and execute the Fairy companion")
	}
	driver := windowsGUIAuthoredDriver(t)
	root := windowsGUIExample(t)
	// Raw packet construction is independent of the example application ABI.
	// The public CLI section below exercises the real Fairy event handler.
	header := "#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n#define FY_CANDIDATE_PROJECT_HEADER_V1\n#include <stdint.h>\n#endif\n"
	literals := "/* fairyc.candidate-project-literals-c/v1 */\n#include \"fy_project.h\"\n"
	unit := "/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\nextern int32_t fy_fn_1(void){return 0;}\nint32_t fy_candidate_entry(void){return fy_fn_1();}\nint main(void){return (int)fy_candidate_entry();}\n"
	var packet bytes.Buffer
	encode := func() {
		packet.Reset()
		fmt.Fprintf(&packet, "FYCPARTS1\n%d\n%s%d\n%s1\n%d\n%s", len(header), header, len(literals), literals, len(unit), unit)
		if _, err := candidatebuild.Decode(packet.Bytes()); err != nil {
			t.Fatal(err)
		}
		writeWorkspaceFile(t, root, "packet.bin", packet.String())
	}
	encode()
	command := copiedClosureCommand{t: t, image: driver, root: root}
	args := []string{"--packet-input", filepath.Join(root, "packet.bin"), "--output", filepath.Join(root, "program.exe"), "--windows-root", root}
	for _, flags := range [][]string{nil, {"--release"}} {
		command.run(0, append(args, flags...)...)
		windowsGUIExecute(t, filepath.Join(root, "program.exe"), 2)
	}
	before, err := os.ReadFile(filepath.Join(root, "program.exe"))
	if err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, "native/app.rc", "invalid resource text\n")
	_, diagnostics := command.run(1, args...)
	if !strings.Contains(diagnostics, "compile Windows resource") {
		t.Fatal(diagnostics)
	}
	after, err := os.ReadFile(filepath.Join(root, "program.exe"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("resource failure replaced prior executable", err)
	}
	// The same authored driver still consumes ordinary console packets when
	// no Windows root is selected, even if its cwd contains GUI settings.
	encode()
	command.run(0, args[:len(args)-2]...)
	windowsGUIExecute(t, filepath.Join(root, "program.exe"), 3)
	if err := os.Remove(filepath.Join(root, "fairy.windows.json")); err != nil {
		t.Fatal(err)
	}
	command.run(0, args...)
	windowsGUIExecute(t, filepath.Join(root, "program.exe"), 3)
	// Retain all authored public CLI commands and replace only the bootstrap
	// C builder with the freshly built Fairy companion and its pinned pack.
	cli := copiedClosureCLIHost(t, cdriver.LowProfile)
	for _, name := range []string{"fairyc-candidate-build.exe", "fairyc-runtime-assets.pack"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(driver), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(cli), name), data, 0700); err != nil {
			t.Fatal(err)
		}
	}
	public := copiedClosureCommand{t: t, image: cli, root: windowsGUIExample(t)}
	for _, flags := range [][]string{nil, {"--release"}} {
		public.run(0, append([]string{"build"}, flags...)...)
		windowsGUIExecute(t, filepath.Join(public.root, "build/win32-minimal.exe"), 2)
		public.run(0, append(append([]string{"run"}, flags...), "--", "--smoke")...)
	}
}
