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

// Read unchanged SDK sources as data; never compile them as candidate modules.
func TestWindowsFairyParserSDKCorpus(t *testing.T) {
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
		var stderr strings.Builder
		compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
		if code != ExitSuccess {
			t.Fatalf("compile: %d %s", code, stderr.String())
		}
		for _, file := range []struct{ disk, name string }{{"optional/std/vec.fy", "std/vec.fy"}, {"std/bytes.fy", "std/bytes.fy"}, {"std/text.fy", "std/text.fy"}} {
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
			row := fmt.Sprintf("%s bytes=%d status=%d declarations=%d problems=%d first=%d..%d expected=%d actual=%q prefix=%t", class, len(data), full[0], full[1], full[2], first[4], first[5], first[3], actual, full[0] == 4)
			evidence := fmt.Sprint(full, first)
			if profile == cdriver.LowProfile {
				baseline[file.name] = evidence
				t.Logf("%s %s", file.name, row)
			} else if baseline[file.name] != evidence {
				t.Fatalf("SDK profile mismatch: %s", file.name)
			}
		}
		t.Logf("profile=%v SDK files=3; incomplete/exhausted results are observations, not admission", profile)
	}
}
