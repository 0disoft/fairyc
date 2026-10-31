//go:build windows && amd64

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"golang.org/x/sys/windows"
)

func dependencyImagePath(t *testing.T, image []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "program.exe")
	if err := os.WriteFile(path, image, 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func checkDependencyCRT(t *testing.T, artifact cdriver.WindowsHostArtifact, receipt *windowsDependencyReceipt) {
	t.Helper()
	objects := make([][]byte, len(artifact.Objects))
	for i, obj := range artifact.Objects {
		objects[i] = obj.Bytes
	}
	crt, err := pecoff.InspectMSVCDefaultLibraries(objects, artifact.Bytes, artifact.LinkMap, artifact.LinkerVerbose)
	if err != nil {
		t.Fatal(err)
	}
	receipt.StaticCRT = crt.Symbols
	for _, library := range crt.Libraries {
		if library.Name != "libcmt.lib" && library.Name != "libucrt.lib" && library.Name != "libvcruntime.lib" {
			continue
		}
		file, err := os.Open(library.Path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.New()
		_, err = io.Copy(digest, file)
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("CRT hash %v %v", err, closeErr)
		}
		receipt.CRTArchives = append(receipt.CRTArchives, dependencyArchive{Name: library.Name, Path: library.Path, SHA256: fmt.Sprintf("%x", digest.Sum(nil))})
	}
	if len(receipt.CRTArchives) != 3 {
		t.Fatal("missing static CRT archives")
	}
	for _, library := range receipt.Imports.Libraries {
		if library.Name == "ucrtbase.dll" || strings.HasPrefix(library.Name, "api-ms-win-crt-") || strings.HasPrefix(library.Name, "vcruntime") || strings.HasPrefix(library.Name, "msvcp") {
			t.Fatalf("unexpected /MT direct CRT DLL import %s", library.Name)
		}
	}
}

func declaredDependency(t *testing.T, receipt *windowsDependencyReceipt, name string, functions ...string) {
	t.Helper()
	library := pecoff.ImportedLibrary{Name: name}
	for _, function := range functions {
		library.Functions = append(library.Functions, pecoff.ImportedFunction{Name: function})
	}
	resolution, err := resolveDependencyLibrary(library)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, module := range receipt.Modules {
		if strings.EqualFold(module.Path, resolution.Path) && module.SHA256 == resolution.SHA256 {
			found = true
		}
	}
	if !found {
		t.Fatalf("declared runtime DLL %s not loaded in child", name)
	}
	receipt.Dynamic = append(receipt.Dynamic, resolution)
}

func TestWindowsDependencyCorpus(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	tracer := buildDependencyTracer(t, tc)
	var rows []windowsDependencyReceipt
	for _, sample := range []struct {
		name   string
		exit   int32
		stdout string
		args   []string
	}{
		{"std_bytes_quickstart", 0, "", nil}, {"std_alloc_quickstart", 42, "", nil},
		{"std_text_from_buffer_quickstart", 42, "", nil}, {"std_vec_quickstart", 0, "42", nil},
		{"process_args_quickstart", 42, "", []string{"", "Fairy", "--locked", `a b"c\d\`, "한글λ"}},
	} {
		workspace, err := workspaceloader.Load(filepath.Join("..", "..", "..", "examples", sample.name))
		if err != nil {
			t.Fatal(err)
		}
		checked, err := workspacecompiler.Check(workspace)
		if err != nil || !checked.Verified {
			t.Fatalf("check %v", err)
		}
		id, err := workspace.RootPackageID()
		if err != nil {
			t.Fatal(err)
		}
		unit, err := cbackend.EmitStorageTransfers(checked.MIR, hir.QualifiedModulePath(id, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		support, err := nativesupport.Source(target.X8664WindowsMSVC, unit.ForeignSymbols, nativesupport.Options{SDKStorage: true})
		if err != nil {
			t.Fatal(err)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			artifact, err := cdriver.BuildWindowsHostWithLibraryLog(tc, profile, unit.Source, support)
			if err != nil {
				t.Fatal(err)
			}
			receipt, out, stderr, err := traceDependencyProcess(tracer, dependencyImagePath(t, artifact.Bytes), sample.args)
			if err != nil || receipt.Exit != sample.exit || string(out) != sample.stdout || len(stderr) != 0 {
				t.Fatalf("%s/%s: exit=%d out=%q err=%q %v", sample.name, profile.Name, receipt.Exit, out, stderr, err)
			}
			receipt.Case, receipt.Profile = sample.name, profile.Name
			receipt.UnitSHA256, receipt.RuntimeSHA256 = debugDigest([]byte(unit.Source)), debugDigest([]byte(support))
			checkDependencyCRT(t, artifact, &receipt)
			if err := validateDependencyResolutions(&receipt); err != nil {
				t.Fatal(err)
			}
			rows = append(rows, receipt)
			t.Logf("%s/%s: static CRT=3 archives imports=%d load events=%d exit=%d", sample.name, profile.Name, len(receipt.Imports.Libraries), len(receipt.Modules), receipt.Exit)
		}
	}
	// Exercise the original private native resolver, including its NTSTATUS error path.
	rooted, err := candidatehost.WindowsSource("fairyc_private_windows_directory_open_v1( fairyc_private_windows_root_file_open_v1(")
	if err != nil {
		t.Fatal(err)
	}
	getProc := regexp.MustCompile(`GetProcAddress\(module, "([A-Za-z0-9_]+)"\)`)
	want := map[string]bool{"NtCreateFile": false, "NtSetInformationFile": false, "RtlNtStatusToDosError": false}
	for _, match := range getProc.FindAllStringSubmatch(rooted, -1) {
		if _, ok := want[match[1]]; !ok {
			t.Fatalf("undeclared runtime lookup %s", match[1])
		}
		want[match[1]] = true
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("declared runtime lookup missing from source: %s", name)
		}
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		for _, scenario := range []struct{ name, missing, argument, output string }{
			{"rooted-native", "", "ok", "dynamic: create+rename+NTSTATUS mapping\r\n"},
			{"missing-NtCreateFile", "NtCreateFile", "missing-open", "dynamic: missing-open error=127\r\n"},
			{"missing-RtlNtStatusToDosError", "RtlNtStatusToDosError", "missing-open", "dynamic: missing-open error=127\r\n"},
			{"missing-NtSetInformationFile", "NtSetInformationFile", "missing-set", "dynamic: missing-set error=127\r\n"},
		} {
			runtime := rooted
			if scenario.missing != "" {
				runtime = strings.ReplaceAll(runtime, `GetProcAddress(module, "`+scenario.missing+`")`, `GetProcAddress(module, "FairycMissingNativeExport")`)
			}
			artifact, err := cdriver.BuildWindowsHostWithLibraryLog(tc, profile, dependencyRootedUnitC, runtime)
			if err != nil {
				t.Fatal(err)
			}
			receipt, out, stderr, err := traceDependencyProcess(tracer, dependencyImagePath(t, artifact.Bytes), []string{t.TempDir(), scenario.argument})
			if err != nil || receipt.Exit != 42 || string(out) != scenario.output || len(stderr) != 0 {
				t.Fatalf("%s/%s: exit=%d out=%q err=%q %v", scenario.name, profile.Name, receipt.Exit, out, stderr, err)
			}
			receipt.Case, receipt.Profile = scenario.name, profile.Name
			receipt.UnitSHA256, receipt.RuntimeSHA256 = debugDigest([]byte(dependencyRootedUnitC)), debugDigest([]byte(runtime))
			// This C-only path uses wmain for lossless Windows paths. The existing
			// CRT selector's main/exe_main contract belongs to the generated corpus above.
			if err := validateDependencyResolutions(&receipt); err != nil {
				t.Fatal(err)
			}
			declaredDependency(t, &receipt, "ntdll.dll", "NtCreateFile", "NtSetInformationFile", "RtlNtStatusToDosError")
			rows = append(rows, receipt)
			t.Logf("%s/%s: %s", scenario.name, profile.Name, strings.TrimSpace(string(out)))
		}
	}
	goTool := filepath.Join(t.TempDir(), "debug-tool.exe")
	if output, err := exec.Command("go", "build", "-mod=readonly", "-o", goTool, ".").CombinedOutput(); err != nil {
		t.Fatalf("build debug tool: %v %s", err, output)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		_, manifest, files := buildRetainedDebugFixture(t, tc, profile)
		store := t.TempDir()
		key, err := retainWindowsDebugBundle(store, manifest, files)
		if err != nil {
			t.Fatal(err)
		}
		rva := retainedDebugMapRVA(t, files, "main", "unit.obj")
		receipt, out, stderr, err := traceDependencyProcess(tracer, goTool, []string{"--debug-lookup", "--store", store, "--image-sha", key, "--rva", fmt.Sprint(rva)})
		if err != nil || receipt.Exit != 0 || len(stderr) != 0 {
			t.Fatalf("debug lookup: exit=%d %v %s %s", receipt.Exit, err, out, stderr)
		}
		var lookup retainedDebugLookup
		if err := json.Unmarshal(out, &lookup); err != nil || lookup.Symbol != "main" || lookup.ImageSHA256 != key {
			t.Fatalf("debug lookup evidence: %v %s", err, out)
		}
		if err := validateDependencyResolutions(&receipt); err != nil {
			t.Fatal(err)
		}
		declaredDependency(t, &receipt, "dbghelp.dll", "SymInitializeW", "SymCleanup", "SymSetOptions", "SymGetOptions", "SymLoadModuleExW", "SymFromAddr", "SymGetLineFromAddrW64")
		receipt.Case, receipt.Profile = "developer-symbolication", profile.Name
		rows = append(rows, receipt)
		t.Logf("debug-tool/%s: imports=%d load events=%d DbgHelp=7 exact main:%d", profile.Name, len(receipt.Imports.Libraries), len(receipt.Modules), lookup.Line)
	}
	bootstrap := filepath.Join(t.TempDir(), "fairyc.exe")
	build := exec.Command("go", "build", "-mod=readonly", "-o", bootstrap, "./cmd/fairyc")
	build.Dir = filepath.Join("..", "..", "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Go CLI: %v %s", err, output)
	}
	receipt, out, stderr, err := traceDependencyProcess(tracer, bootstrap, []string{"--help"})
	if err != nil || receipt.Exit != 0 || !strings.Contains(string(out), "fairyc") || len(stderr) != 0 {
		t.Fatalf("Go CLI: exit=%d %v %s %s", receipt.Exit, err, out, stderr)
	}
	if err := validateDependencyResolutions(&receipt); err != nil {
		t.Fatal(err)
	}
	receipt.Case, receipt.Profile = "go-bootstrap-help", "go-default"
	rows = append(rows, receipt)
	probe, err := openProcessMemoryProbe(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	working, pagefile, err := probe.snapshot()
	closeErr := probe.close()
	if err != nil || closeErr != nil || working == 0 || pagefile == 0 {
		t.Fatalf("PSAPI memory query: %v %v", err, closeErr)
	}
	psapi, err := resolveDependencyLibrary(pecoff.ImportedLibrary{Name: "psapi.dll", Functions: []pecoff.ImportedFunction{{Name: "GetProcessMemoryInfo"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 21 {
		t.Fatalf("incomplete dependency matrix: %d", len(rows))
	}
	revision, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	report := struct {
		Schema         string                     `json:"schema"`
		SourceRevision string                     `json:"sourceRevision"`
		Toolchain      string                     `json:"toolchain"`
		TracerSHA256   string                     `json:"tracerSourceSHA256"`
		Passed         bool                       `json:"passed"`
		Rows           []windowsDependencyReceipt `json:"rows"`
		PSAPI          dependencyResolution       `json:"psapiInTestProcess"`
		WindowsBuild   uint32                     `json:"windowsBuild"`
		GoVersion      string                     `json:"goVersion"`
	}{"fairyc.windows-dependencies/v1", strings.TrimSpace(string(revision)), tc.Version, debugDigest([]byte(windowsDependencyTracerC)), true, rows, psapi, windows.RtlGetVersion().BuildNumber, runtime.Version()}
	if path := os.Getenv("FAIRYC_DEPENDENCY_EVIDENCE"); path != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("dependency matrix: %d traces; PSAPI memory query passed", len(rows))
}

func TestWindowsDependencyMissingInputs(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	checkDependencyMissingInputs(t, tc, buildDependencyTracer(t, tc))
}

func checkDependencyMissingInputs(t *testing.T, tc cdriver.WindowsHostToolchain, tracer string) {
	t.Helper()
	artifact, err := cdriver.BuildWindowsMSVCProbe(tc, cdriver.LowProfile, "#include <windows.h>\nint main(void){return (int)(GetCurrentProcessId()==0);}\n")
	if err != nil {
		t.Fatal(err)
	}
	imports, err := pecoff.InspectAMD64Dependencies(artifact.Bytes)
	if err != nil || len(imports.Libraries) == 0 {
		t.Fatalf("control imports: %v", err)
	}
	for _, scenario := range []struct {
		name string
		edit func([]byte)
		want uint32
	}{
		{"missing-DLL", func(image []byte) {
			// Binary PE bytes need byte-preserving ASCII folding, not Unicode case conversion.
			folded := append([]byte(nil), image...)
			for i, c := range folded {
				if c >= 'A' && c <= 'Z' {
					folded[i] = c + ('a' - 'A')
				}
			}
			offset := bytes.Index(folded, []byte("kernel32.dll\x00"))
			if offset < 0 {
				t.Fatal("no kernel32")
			}
			copy(image[offset:], "fairy404.dll\x00")
		}, 0xc0000135},
		{"missing-import", func(image []byte) {
			name := imports.Libraries[0].Functions[0].Name
			offset := bytes.Index(image, append([]byte(name), 0))
			if name == "" || offset < 0 {
				t.Fatal("no named import")
			}
			copy(image[offset:], strings.Repeat("Z", len(name)))
		}, 0xc0000139},
	} {
		image := append([]byte(nil), artifact.Bytes...)
		scenario.edit(image)
		receipt, _, _, err := traceDependencyProcess(tracer, dependencyImagePath(t, image), nil)
		if err != nil || uint32(receipt.Exit) != scenario.want {
			t.Fatalf("%s: exit=%#x want=%#x %v", scenario.name, uint32(receipt.Exit), scenario.want, err)
		}
		if err := validateDependencyResolutions(&receipt); err == nil {
			t.Fatal("missing dependency resolution accepted")
		}
		t.Logf("loader negative %s: exit=%#x", scenario.name, scenario.want)
	}
	if _, err := resolveDependencyLibrary(pecoff.ImportedLibrary{Name: "ntdll.dll", Functions: []pecoff.ImportedFunction{{Name: "FairycMissingNativeExport"}}}); err == nil {
		t.Fatal("absent native export accepted")
	}
	system, err := queryWindowsSystemDirectory()
	if err != nil {
		t.Fatal(err)
	}
	module, err := dependencySystemFile(filepath.Join(system, "kernel32.dll"))
	if err != nil {
		t.Fatal(err)
	}
	module.SHA256 = strings.Repeat("0", 64)
	if recheckDependencyModule(module) == nil {
		t.Fatal("DLL hash mismatch accepted")
	}
	if _, err := dependencySystemFile(filepath.Join(t.TempDir(), "kernel32.dll")); err == nil {
		t.Fatal("non-system DLL owner accepted")
	}
	if _, err := dependencySystemFile(filepath.Join(system, "fairy404.dll")); err == nil {
		t.Fatal("missing system DLL accepted")
	}
}
