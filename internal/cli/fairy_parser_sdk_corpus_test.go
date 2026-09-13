package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

type parserSDKFile struct{ disk, name string }

// Keep the admission baseline independent of the observation inventory so that
// removing an inventory entry cannot silently remove its regression requirement.
func requiredParserSDKFiles() map[string]string {
	return map[string]string{
		"optional/std/vec.fy": "std/vec.fy",
		"std/bytes.fy":        "std/bytes.fy",
		"std/text.fy":         "std/text.fy",
		"std/alloc.fy":        "std/alloc.fy",
		"std/process.fy":      "std/process.fy",
		"std/io.fy":           "std/io.fy",
	}
}

func validateParserSDKFiles(files []parserSDKFile) error {
	missing := requiredParserSDKFiles()
	disks, names := map[string]bool{}, map[string]bool{}
	for _, file := range files {
		if file.disk == "" || file.name == "" || disks[file.disk] || names[file.name] {
			return fmt.Errorf("empty or duplicate SDK identity: %+v", file)
		}
		disks[file.disk], names[file.name] = true, true
		if name, required := missing[file.disk]; required {
			if name != file.name {
				return fmt.Errorf("required SDK identity changed: %s: %s != %s", file.disk, file.name, name)
			}
			delete(missing, file.disk)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("missing required SDK files: %v", missing)
	}
	return nil
}

func requireParserSDKComplete(file parserSDKFile, full []int) error {
	if _, required := requiredParserSDKFiles()[file.disk]; required {
		if len(full) < 6 || full[0] != 0 || full[2] != 0 {
			return fmt.Errorf("complete-SDK regression %s: %v", file.name, full)
		}
	}
	return nil
}

// Read unchanged SDK sources as data; never compile them as candidate modules.
func TestWindowsFairyParserSDKCorpus(t *testing.T) {
	files := []parserSDKFile{
		{"optional/std/vec.fy", "std/vec.fy"}, {"std/bytes.fy", "std/bytes.fy"}, {"std/text.fy", "std/text.fy"},
		{"std/alloc.fy", "std/alloc.fy"}, {"std/process.fy", "std/process.fy"},
		{"std/io.fy", "std/io.fy"},
		{"std/fs.fy", "std/fs.fy"}, // Observation-only; preserve failures under existing limits.
	}
	if err := validateParserSDKFiles(files); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	for _, name := range []string{"syntax.fy", "integrity.fy", "file_syntax.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, "src/"+name, string(data))
	}
	writeTestSource(t, root, "src/main.fy", parserFileCorpusProgram(t))
	baseline := map[string]string{}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		missing := requiredParserSDKFiles()
		var stderr strings.Builder
		compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
		if code != ExitSuccess {
			t.Fatalf("compile: %d %s", code, stderr.String())
		}
		for _, file := range files {
			data, err := os.ReadFile(filepath.Join("..", "..", "internal", "sdkstdlib", "sources", filepath.FromSlash(file.disk)))
			if err != nil || len(data) > 32768 {
				t.Fatalf("SDK %s size=%d: %v", file.disk, len(data), err)
			}
			oracle, err := frontend.AnalyzeSyntax(source.NewFile(file.name, data), source.DefaultLimits())
			if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
				t.Fatalf("SDK oracle %s: %v", file.name, err)
			}
			top, _ := oracle.Tree.Node(oracle.Tree.Root)
			if len(top.Children) > 128 {
				t.Fatal("SDK prefix probe budget exceeded")
			}
			observe := func(input []byte) []int {
				t.Helper()
				out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewReader(append([]byte(file.name+"\n"), input...)))
				if err != nil || out.ExitCode != 0 || out.Stderr != "" || out.TrapControl != "" {
					t.Fatalf("SDK native %s: %+v %v", file.name, out, err)
				}
				fields := strings.Fields(out.Stdout)
				if len(fields) < 6 {
					t.Fatal("truncated SDK observation")
				}
				row := make([]int, len(fields))
				for i, s := range fields {
					n, err := strconv.Atoi(s)
					if err != nil || n < 0 {
						t.Fatal("invalid SDK observation")
					}
					row[i] = n
				}
				if len(row) != 6+5*row[1] || (row[0] != 0 && row[0] != 4) {
					t.Fatalf("unexpected SDK outcome %s: %v", file.name, row)
				}
				return row
			}
			full := observe(data)
			if err := requireParserSDKComplete(file, full); err != nil {
				t.Fatal(err)
			}
			if full[0] == 0 {
				want := parserDeclarationRecords(t, file.name, data)
				position := 0
				for i := 6; i < len(full); i += 5 {
					for position < len(want) && fmt.Sprint(want[position:position+5]) != fmt.Sprint(full[i:i+5]) {
						position += 5
					}
					if position == len(want) {
						t.Fatalf("SDK retained declaration differs from Go: %s %v", file.name, full[i:i+5])
					}
					position += 5
				}
			}
			first := full
			if full[0] == 4 {
				for _, id := range top.Children {
					node, _ := oracle.Tree.Node(id)
					first = observe(data[:node.Span.EndByte])
					if first[0] == 0 && first[2] > 0 {
						break
					}
				}
				if first[0] != 0 || first[2] == 0 {
					t.Fatalf("unlocated SDK exhaustion: %s", file.name)
				}
			}
			actual := ""
			class := "supported"
			if first[2] > 0 {
				class = "unsupported"
				if first[4] >= first[5] || first[5] > len(data) {
					t.Fatal("invalid SDK first-fault span")
				}
				actual = string(data[first[4]:first[5]])
			} else {
				compareParserDeclarations(t, file.name, data, full[6:])
			}
			_, required := requiredParserSDKFiles()[file.disk]
			row := fmt.Sprintf("%s required=%t bytes=%d status=%d declarations=%d problems=%d first=%d..%d expected=%d actual=%q prefix=%t", class, required, len(data), full[0], full[1], full[2], first[4], first[5], first[3], actual, full[0] == 4)
			evidence := fmt.Sprint(full, first)
			if profile == cdriver.LowProfile {
				baseline[file.name] = evidence
				t.Logf("%s %s", file.name, row)
			} else if baseline[file.name] != evidence {
				t.Fatalf("SDK profile mismatch: %s", file.name)
			}
			delete(missing, file.disk)
		}
		if len(missing) != 0 {
			t.Fatalf("required SDK files not verified in profile %v: %v", profile, missing)
		}
		t.Logf("profile=%v SDK files=%d required=%d; required files need complete syntax and exact Go declarations; other files are observations", profile, len(files), len(requiredParserSDKFiles()))
	}
}

func TestParserSDKCorpusContract(t *testing.T) {
	// Declaration kind 11 retains enum identity independently of member support.
	enumSource := "module sample;\npub enum Kind { First; Second; }\n"
	enumStart, nameStart := strings.Index(enumSource, "pub"), strings.Index(enumSource, "Kind")
	want := []int{1, 0, len("module sample;"), 7, 13, 11, enumStart, len(strings.TrimSpace(enumSource)), nameStart, nameStart + len("Kind")}
	if got := parserDeclarationRecords(t, "sample.fy", []byte(enumSource)); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("enum oracle records: got %v want %v", got, want)
	}
	files := []parserSDKFile{
		{"optional/std/vec.fy", "std/vec.fy"}, {"std/bytes.fy", "std/bytes.fy"}, {"std/text.fy", "std/text.fy"},
		{"std/alloc.fy", "std/alloc.fy"}, {"std/process.fy", "std/process.fy"},
		{"std/io.fy", "std/io.fy"},
	}
	if err := validateParserSDKFiles(files); err != nil {
		t.Fatal(err)
	}
	for i, file := range files {
		t.Run("missing/"+file.name, func(t *testing.T) {
			without := append(append([]parserSDKFile{}, files[:i]...), files[i+1:]...)
			if err := validateParserSDKFiles(without); err == nil {
				t.Fatal("required inventory omission accepted")
			}
		})
		for label, row := range map[string][]int{"partial": {0, 0, 1, 3, 1, 2}, "exhausted": {4, 0, 0, 0, 0, 0}, "truncated": {0, 0}} {
			t.Run(label+"/"+file.name, func(t *testing.T) {
				if err := requireParserSDKComplete(file, row); err == nil {
					t.Fatal("incomplete required SDK accepted")
				}
			})
		}
		if err := requireParserSDKComplete(file, []int{0, 0, 0, 0, 0, 0}); err != nil {
			t.Fatal(err)
		}
	}
	for _, extra := range []parserSDKFile{files[0], {files[0].disk, "std/alias.fy"}, {"std/alias.fy", files[0].name}, {"", "std/empty.fy"}} {
		if err := validateParserSDKFiles(append(append([]parserSDKFile{}, files...), extra)); err == nil {
			t.Fatalf("invalid identity accepted: %+v", extra)
		}
	}
	changed := append([]parserSDKFile{}, files...)
	changed[0].name = "std/renamed.fy"
	if err := validateParserSDKFiles(changed); err == nil {
		t.Fatal("required disk/name remapping accepted")
	}
	// Synthetic identity only: no source file is read by this policy control.
	observation := parserSDKFile{"std/observation.fy", "std/observation.fy"}
	if err := validateParserSDKFiles(append(append([]parserSDKFile{}, files...), observation)); err != nil {
		t.Fatal(err)
	}
	if err := requireParserSDKComplete(observation, []int{0, 0, 1, 3, 1, 2}); err != nil {
		t.Fatal("observation-only syntax should not require completeness:", err)
	}
}
