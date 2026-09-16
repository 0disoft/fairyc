package cli

import (
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

// Copy the real scanner and its imports, excluding parser callers. Change only
// token_info's argument boundary; do not invent a second scanner implementation.
func tokenIndexWorkspace(t *testing.T, wrapped bool) (string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	sources := map[string]string{
		"fairy.toml":  "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.token-index\"\nname = \"token-index\"\nversion = \"1.0.0\"\nlanguage = \"" + sdkstdlib.LanguageRevision + "\"\nsource = \"src\"\n",
		"src/main.fy": tokenIndexClient,
	}
	for _, name := range []string{"scanner", "numbers", "diagnostics", "utf8", "strings", "budget"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		sources["src/"+name+".fy"] = string(data)
	}
	if wrapped {
		old := "pub fn TokenInfo token_info(ref Stream stream, usize index) {\n    let ref Token token = v::at<Token>(ref stream.tokens, index);"
		changed := "pub struct TokenIndex { usize raw; }\npub struct ByteOffset { usize raw; }\n\npub fn TokenInfo token_info(ref Stream stream, TokenIndex index) {\n    let ref Token token = v::at<Token>(ref stream.tokens, index.raw);"
		// Repository worktrees may use CRLF; normalize both variants equally below.
		scanner := strings.ReplaceAll(sources["src/scanner.fy"], "\r\n", "\n")
		if strings.Count(scanner, old) != 1 {
			t.Fatal("token_info source boundary changed; review experiment")
		}
		sources["src/scanner.fy"] = strings.Replace(scanner, old, changed, 1)
		sources["src/main.fy"] = strings.NewReplacer(
			"let usize token_index = 2;", "let scanner::TokenIndex token_index = scanner::TokenIndex { .raw = 2 };",
			"let usize byte_offset = 7;", "let scanner::ByteOffset byte_offset = scanner::ByteOffset { .raw = 7 };",
			"info.start == byte_offset", "info.start == byte_offset.raw",
		).Replace(tokenIndexClient)
	}
	hashes := make(map[string]string, len(sources))
	for name, contents := range sources {
		contents = strings.ReplaceAll(contents, "\r\n", "\n")
		writeTestSource(t, root, name, contents)
		hashes[name] = tokenIndexHash([]byte(contents))
	}
	return root, hashes
}

func TestTokenIndexStructBoundary(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		root, _ := tokenIndexWorkspace(t, wrapped)
		code, _, stderr := runForTest([]string{"check"}, root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("valid wrapped=%v: %d %s", wrapped, code, stderr)
		}
		file := filepath.Join(root, "src/main.fy")
		original, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, wrong := range []string{"byte_offset", "2"} {
			mutated := strings.Replace(string(original), "token_info(ref stream, token_index)", "token_info(ref stream, "+wrong+")", 1)
			if mutated == string(original) {
				t.Fatal("mutation did not reach the call")
			}
			writeTestSource(t, root, "src/main.fy", mutated)
			code, _, stderr = runForTest([]string{"check"}, root)
			if !wrapped {
				if code != ExitSuccess || stderr != "" {
					t.Fatalf("raw boundary should admit %s: %d %s", wrong, code, stderr)
				}
			} else if code != 3 || !strings.Contains(stderr, "E1006") {
				t.Fatalf("wrapper must reject %s as type mismatch: %d %s", wrong, code, stderr)
			} else {
				t.Logf("rejected %s: %s", wrong, strings.TrimSpace(stderr))
			}
		}
	}
}

func TestTokenIndexStructCostMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	type sample struct {
		Repetition      int               `json:"repetition"`
		Wrapped         bool              `json:"wrapped"`
		Profile         string            `json:"profile"`
		Sources         map[string]string `json:"source_sha256"`
		CheckNS         int64             `json:"check_ns"`
		CompileNS       int64             `json:"compile_ns"`
		CBytes          int               `json:"c_bytes"`
		CSHA256         string            `json:"c_sha256"`
		ExecutableBytes int               `json:"executable_bytes"`
		Sections        map[string]uint32 `json:"pe_virtual_section_bytes"`
		Objects         map[string]int    `json:"object_bytes"`
		PDBBytes        int               `json:"pdb_bytes"`
		ExitCode        int               `json:"exit_code"`
	}
	var samples []sample
	var msvc string
	for repetition := 0; repetition < 2; repetition++ {
		order := []bool{false, true}
		if repetition == 1 {
			order = []bool{true, false}
		}
		for _, wrapped := range order {
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				root, hashes := tokenIndexWorkspace(t, wrapped)
				start := time.Now()
				code, _, stderr := runForTest([]string{"check"}, root)
				checkNS := time.Since(start).Nanoseconds()
				if code != ExitSuccess || stderr != "" {
					t.Fatalf("check: %d %s", code, stderr)
				}
				var diagnostics bytes.Buffer
				lowered, code := lowerWorkspaceExecutable(root, true, workspaceCompilePolicy{stableDiagnostics: true}, "token index measurement", &diagnostics)
				if code != ExitSuccess || diagnostics.Len() != 0 || !lowered.sdkStorage {
					t.Fatalf("lower: %d %s", code, diagnostics.String())
				}
				unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
				if err != nil {
					t.Fatal(err)
				}
				generatedC := unit.Source
				start = time.Now()
				compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: true}, workspaceCompilePolicy{stableDiagnostics: true}, &diagnostics)
				compileNS := time.Since(start).Nanoseconds()
				if code != ExitSuccess || diagnostics.Len() != 0 {
					t.Fatalf("compile: %d %s", code, diagnostics.String())
				}
				artifact := compiled.artifact
				msvc = artifact.Toolchain.Version
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("execute: %+v %v", outcome, err)
				}
				image, err := pe.NewFile(bytes.NewReader(artifact.Bytes))
				if err != nil {
					t.Fatal(err)
				}
				sections := make(map[string]uint32)
				for _, section := range image.Sections {
					sections[section.Name] = section.VirtualSize
				}
				if err := image.Close(); err != nil {
					t.Fatal(err)
				}
				objects := make(map[string]int)
				for _, object := range artifact.Objects {
					objects[object.Name] = len(object.Bytes)
				}
				samples = append(samples, sample{repetition, wrapped, profile.Name, hashes, checkNS, compileNS, len(generatedC), tokenIndexHash([]byte(generatedC)), len(artifact.Bytes), sections, objects, len(artifact.ProgramDatabase), int(outcome.ExitCode)})
			}
		}
	}
	testSource, err := os.ReadFile("token_index_experiment_test.go")
	if err != nil {
		t.Fatal(err)
	}
	report := struct {
		Schema           string   `json:"schema"`
		CompilerSource   string   `json:"compiler_source"`
		TestSourceSHA256 string   `json:"test_source_sha256"`
		TestExecutable   string   `json:"test_executable_identity"`
		Go               string   `json:"go_version"`
		MSVC             string   `json:"msvc_version"`
		Method           string   `json:"method"`
		Samples          []sample `json:"samples"`
	}{"fairyc.token-index-struct-experiment/v1", "5e171171a5ab0319e4376eff2973596be0f72929", tokenIndexHash(testSource), identity.String(), runtime.Version(), msvc,
		"fresh workspace; check includes lock creation; private storage-aware C emission precedes measured shared stable build compiler; compile excludes writing final EXE; ambient caches; execute without rebuild; no runtime benchmark; no PDB requested", samples}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("TOKEN_INDEX_REPORT " + string(encoded))
}

func tokenIndexHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

const tokenIndexClient = `module main;
import scanner as scanner;
import std::bytes as bytes;
fn i32 inspect(scanner::Stream stream) {
    let usize token_index = 2;
    let usize byte_offset = 7;
    let scanner::TokenInfo info = scanner::token_info(ref stream, token_index);
    assert(info.kind == 3 && info.start == byte_offset && info.end == 12);
    assert(scanner::token_count(ref stream) == 5);
    return 0;
}
fn i32 main() {
    let bytes::Buffer input = switch (bytes::copy(b"module alpha;")) {
        case .ok(value): { value }
        case .err(_): { bytes::empty() }
    };
    if (bytes::len(ref input) != 13) { return 1; }
    return switch (scanner::collect(input, false, 16384, 256, 32, 1048576)) {
        case .ok(stream): { inspect(stream) }
        case .err(_): { 2 }
    };
}
`
