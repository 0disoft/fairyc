package cli

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

var measureLexerCost = flag.Bool("fairyc-lexer-cost", false, "run bounded native lexer cost observations without a performance gate")

func TestWindowsFairyLexerCostBaseline(t *testing.T) {
	if !*measureLexerCost {
		t.Skip("opt-in measurement, not a correctness or performance gate")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	sourceHash := sha256.New()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	for _, name := range []string{"fairy.toml", "src/scanner.fy", "src/numbers.fy", "src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy"} {
		data, err := os.ReadFile(filepath.Join(example, name))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(sourceHash, "%s\x00%d\x00", name, len(data))
		sourceHash.Write(data)
	}
	tokenHeavy := "module main;\n" + strings.Repeat("fn i32 f() { return 42; }\n", 600)
	tokenHeavy += strings.Repeat(" ", 16384-len(tokenHeavy))
	scenarios := []struct {
		name, input string
		cycles      int
	}{
		{"empty-driver", "", 0},
		{"small-source", "module main; fn i32 main() { return 42; }", 10},
		{"token-heavy-16k", tokenHeavy, 10},
		{"literal-10k", `"` + strings.Repeat("a", 10000) + `"`, 10},
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		for _, scenario := range scenarios {
			result := lexer.Lex(source.NewFile("input.fy", []byte(scenario.input)), source.NewBudget(source.DefaultLimits(), len(scenario.input)))
			cooked, diagnostic := 0, 0
			for _, token := range result.Tokens {
				if token.Kind != lexer.Keyword {
					cooked += len(token.Cooked)
				}
			}
			for _, d := range result.Diagnostics {
				diagnostic += len(d.Token)
			}
			var program strings.Builder
			program.WriteString("module main; import scanner as scanner; import std::bytes as bytes; fn i32 main() { var usize index = 0;\n")
			fmt.Fprintf(&program, "while (index < %d) { var bool allocated = true; let bytes::Buffer input = switch (bytes::copy(b\"", scenario.cycles)
			for _, b := range []byte(scenario.input) {
				fmt.Fprintf(&program, "\\x%02X", b)
			}
			program.WriteString(`")) { case .ok(value): { value } case .err(_): { allocated = false; bytes::empty() } };
if (!allocated) { return 3; }
let usize count = switch (scanner::collect(input, false, 32768, 1048576, 256, 16777216)) {
case .ok(stream): { let scanner::Storage observation = scanner::storage(ref stream);
`)
			fmt.Fprintf(&program, "assert(observation.source_bytes == %d && observation.cooked_bytes == %d && observation.diagnostic_bytes == %d && observation.tokens == %d && observation.diagnostics == %d); observation.tokens }\n", len(scenario.input), cooked, diagnostic, len(result.Tokens), len(result.Diagnostics))
			fmt.Fprintf(&program, "case .err(_): { 0 } }; if (count != %d) { return 4; } index += 1; } return 0; }", len(result.Tokens))
			root := t.TempDir()
			copyLexerStreamExample(t, root)
			writeTestSource(t, root, "src/main.fy", program.String())
			var stderr strings.Builder
			compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
			if code != ExitSuccess {
				t.Fatalf("compile %s/%s: %d %s", scenario.name, profile.Name, code, stderr.String())
			}
			samples := make([]float64, 0, 5)
			for sample := -1; sample < 5; sample++ {
				start := time.Now()
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, strings.NewReader(""))
				elapsed := time.Since(start).Seconds() * 1000
				if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
					t.Fatalf("sample %s/%s %d %.3fms: %+v %v", scenario.name, profile.Name, sample, elapsed, outcome, err)
				}
				if sample >= 0 {
					samples = append(samples, elapsed)
				}
			}
			ordered := slices.Clone(samples)
			slices.Sort(ordered)
			record := map[string]any{
				"schema": "fairyc.lexer-cost-observation/v1", "scenario": scenario.name, "profile": profile.Name,
				"os": runtime.GOOS, "arch": runtime.GOARCH, "logical_cpus": runtime.NumCPU(), "go": runtime.Version(), "msvc": compiled.artifact.Toolchain.Version,
				"source_sha256": fmt.Sprintf("%x", sourceHash.Sum(nil)), "program_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(program.String()))), "pe_sha256": fmt.Sprintf("%x", sha256.Sum256(compiled.artifact.Bytes)),
				"cycles_per_process": scenario.cycles, "warmup_processes": 1, "successful_samples": 5, "concurrency": 1,
				"source_bytes": len(scenario.input), "cooked_bytes": cooked, "diagnostic_token_bytes": diagnostic, "tokens": len(result.Tokens),
				"driver_roundtrip_ms": samples, "median_ms": ordered[2],
			}
			encoded, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			t.Log(string(encoded))
		}
	}
}
