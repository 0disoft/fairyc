package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

// Compares three real lexer-candidate loops with their previous explicit-index
// spelling. Timed native calls include process startup and console output.
func TestByteViewRealLexerWorkloadComparison(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	const scannerInput = "let n=12345; /*open"
	lexed := lexer.Lex(source.NewFile("input.fy", []byte(scannerInput)), source.NewBudget(source.DefaultLimits(), len(scannerInput)))
	hasCookedNumber := false
	for _, token := range lexed.Tokens {
		if token.Kind == lexer.Integer && len(token.Cooked) > 0 {
			hasCookedNumber = true
		}
	}
	if !hasCookedNumber || len(lexed.Diagnostics) == 0 || len(lexed.Diagnostics[0].Token) == 0 {
		t.Fatal("scanner workload no longer exercises cooked bytes and diagnostic text")
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	pages, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "pages.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/pages.fy", string(pages))
	const driver = `module main;
import pages as pages; import scanner as scanner;
@effects(console) fn i32 main() {
    let i32 status = pages::run();
    if (status != 0) { return status; }
    return scanner::report_work(b"%s"[..], false, 32768, 1048576, 256, 16777216);
}`
	changes := []struct{ path, before, after string }{
		{"src/diagnostics.fy", "var usize i = 0; while (i < value.len) { number(value[i] as usize); i += 1; }", "for (let u8 byte in value) { number(byte as usize); }"},
		{"src/pages.fy", "var usize index = 0; while (index < value.len) { number(window, output, value[index] as usize); index += 1; }", "for (let u8 byte in value) { number(window, output, byte as usize); }"},
		{"src/scanner.fy", "var usize n = 0; while (n < numeric.len) { number(numeric[n] as usize); n += 1; }", "for (let u8 byte in numeric) { number(byte as usize); }"},
	}
	current := make(map[string]string, len(changes))
	previous := make(map[string]string, len(changes))
	for _, change := range changes {
		data, err := os.ReadFile(filepath.Join(root, change.path))
		if err != nil {
			t.Fatal(err)
		}
		current[change.path] = string(data)
		if strings.Count(current[change.path], change.after) != 1 {
			t.Fatalf("expected one adopted loop in %s", change.path)
		}
		previous[change.path] = strings.Replace(current[change.path], change.after, change.before, 1)
	}
	variants := []struct {
		name    string
		sources map[string]string
	}{{"explicit-index", previous}, {"byte-view", current}}
	roots := make([]string, len(variants))
	for index, variant := range variants {
		roots[index] = t.TempDir()
		copyLexerStreamExample(t, roots[index])
		writeTestSource(t, roots[index], "src/pages.fy", string(pages))
		writeTestSource(t, roots[index], "src/main.fy", fmt.Sprintf(driver, scannerInput))
		for _, change := range changes {
			writeTestSource(t, roots[index], change.path, variant.sources[change.path])
		}
	}
	inputs := [][]byte{
		[]byte("module main; fn i32 f(){let i32 n=12345; /*open"),
		[]byte(`"` + strings.Repeat("a", 10000) + `"`),
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifacts := make([]cdriver.WindowsHostArtifact, len(variants))
		buildNS := make([]int64, len(variants))
		sourceSHA := make([]string, len(variants))
		exeBytes := make([]int, len(variants))
		unitBytes := make([]int, len(variants))
		outputs := make([][]string, len(variants))
		for variantIndex, variant := range variants {
			digest := sha256.New()
			for _, change := range changes {
				text := variant.sources[change.path]
				fmt.Fprintf(digest, "%s\x00%d\x00", change.path, len(text))
				digest.Write([]byte(text))
			}
			sourceSHA[variantIndex] = fmt.Sprintf("%x", digest.Sum(nil))
			var diagnostics strings.Builder
			started := time.Now()
			compiled, code := compileWindowsWorkspace(roots[variantIndex], workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
			buildNS[variantIndex] = time.Since(started).Nanoseconds()
			if code != ExitSuccess {
				t.Fatalf("%s/%s compile: %d %s", profile.Name, variant.name, code, diagnostics.String())
			}
			artifacts[variantIndex] = compiled.artifact
			exeBytes[variantIndex] = len(compiled.artifact.Bytes)
			for _, object := range compiled.artifact.Objects {
				if object.Name == "unit.obj" {
					unitBytes[variantIndex] = len(object.Bytes)
				}
			}
			if unitBytes[variantIndex] == 0 {
				t.Fatal("missing generated unit.obj")
			}
			for _, input := range inputs {
				var page strings.Builder
				values := streamObservation(t, input, false)
				for _, value := range values[:min(4096, len(values))] {
					fmt.Fprintf(&page, "%d ", value)
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewReader(streamPageFrame(input, false, 0)))
				if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" || !strings.HasPrefix(outcome.Stdout, page.String()) || len(outcome.Stdout) <= page.Len() {
					t.Fatalf("%s/%s output: %+v %v", profile.Name, variant.name, outcome, err)
				}
				outputs[variantIndex] = append(outputs[variantIndex], outcome.Stdout)
			}
		}
		for input := range inputs {
			if outputs[0][input] != outputs[1][input] {
				t.Fatalf("%s input %d changed lexer output", profile.Name, input)
			}
		}
		elapsedNS := [2][]int64{{}, {}}
		frame := streamPageFrame(inputs[1], false, 0)
		for pair := -1; pair < 5; pair++ {
			order := [2]int{0, 1}
			if pair%2 == 0 {
				order = [2]int{1, 0}
			}
			for _, lane := range order {
				started := time.Now()
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifacts[lane], bytes.NewReader(frame))
				elapsed := time.Since(started).Nanoseconds()
				if err != nil || outcome.ExitCode != 0 || outcome.Stdout != outputs[lane][1] || outcome.Stderr != "" || outcome.TrapControl != "" {
					t.Fatalf("%s pair %d %s: %+v %v", profile.Name, pair, variants[lane].name, outcome, err)
				}
				if pair >= 0 {
					elapsedNS[lane] = append(elapsedNS[lane], elapsed)
				}
			}
		}
		report, err := json.Marshal(map[string]any{
			"profile": profile.Name, "msvc": artifacts[0].Toolchain.Version,
			"source_sha256": sourceSHA, "exe_bytes": exeBytes, "unit_object_bytes": unitBytes,
			"single_build_ns": buildNS, "process_roundtrip_ns": elapsedNS,
			"warmup_pairs": 1, "measured_pairs": 5, "input_bytes": len(inputs[1]),
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("BYTE_VIEW_REAL_WORKLOAD %s", report)
	}
}
