package compiler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// Check the same checked-in lexical corpus through both C storage choices and
// both Fairy loop forms. The Go lexer supplies an independent output oracle.
func TestPrivateVectorSDKCandidateCScannerLexicalParity(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	core, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "scanner_reference.c"))
	if err != nil {
		t.Fatal(err)
	}
	inputs := scannerCReferenceInputs()
	if len(inputs) != 25 {
		t.Fatalf("lexical corpus changed: %d cases", len(inputs))
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	fairyMain := scannerLexicalParityMain(inputs)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			for _, storage := range []struct {
				name      string
				geometric bool
			}{{"preallocated", false}, {"geometric", true}} {
				t.Run("C/"+storage.name, func(t *testing.T) {
					image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile,
						scannerCBudgetDefines(storage.geometric)+string(core))
					if err != nil {
						t.Fatal(err)
					}
					for _, item := range inputs {
						t.Run(item.name, func(t *testing.T) {
							outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, scannerCInput(item.data))
							want := scannerGoObservation(item.data)
							if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" ||
								outcome.TrapControl != "" || outcome.Stdout != want {
								t.Fatalf("candidate C/Go lexical mismatch: %+v %v\nwant %q", outcome, err, want)
							}
						})
					}
				})
			}
			for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
				t.Run("Fairy/"+variant.name, func(t *testing.T) {
					program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, fairyMain)
					image, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("Fairy lexical execution: %+v %v", outcome, err)
					}
					lines := strings.Split(strings.TrimSuffix(outcome.Stdout, "\n"), "\n")
					if len(lines) != len(inputs) {
						t.Fatalf("Fairy lexical reports=%d, want %d: %q", len(lines), len(inputs), outcome.Stdout)
					}
					for i, item := range inputs {
						want := scannerFairyNumericObservation(item.data)
						if lines[i] != want {
							t.Fatalf("%s Fairy/Go lexical mismatch\n got: %q\nwant: %q", item.name, lines[i], want)
						}
					}
				})
			}
		})
	}
}

func scannerLexicalParityMain(inputs []struct {
	name string
	data []byte
}) string {
	var main strings.Builder
	main.WriteString("module main;\nimport scanner as scanner;\nimport std::io as io;\n")
	main.WriteString("@link_name(\"fairyc_fixture_vector_clean\") unsafe extern C fn i32 clean();\n")
	main.WriteString("@effects(console) fn i32 main() {\n")
	for _, item := range inputs {
		fmt.Fprintf(&main, "    assert(scanner::report_work(b\"%s\"[..], false, 32768, 1048576, 256, 16777216) == 0);\n",
			scannerFairyByteLiteral(item.data))
		main.WriteString("    assert(switch (io::write(\"\\n\")) { case .ok(_): { true } case .err(_): { false } });\n")
	}
	main.WriteString("    assert(unsafe { clean() } == 1);\n    return 0;\n}\n")
	return main.String()
}

func scannerFairyByteLiteral(data []byte) string {
	var escaped strings.Builder
	for _, value := range data {
		if value == '\\' || value == '"' {
			escaped.WriteByte('\\')
			escaped.WriteByte(value)
		} else if value >= 0x20 && value <= 0x7e {
			escaped.WriteByte(value)
		} else {
			fmt.Fprintf(&escaped, "\\x%02X", value)
		}
	}
	return escaped.String()
}

func scannerFairyNumericObservation(data []byte) string {
	result := lexer.Lex(source.NewFile("input.fy", data), source.NewBudget(source.DefaultLimits(), len(data)))
	var output strings.Builder
	writeBytes := func(value []byte) {
		fmt.Fprintf(&output, "%d ", len(value))
		for _, item := range value {
			fmt.Fprintf(&output, "%d ", item)
		}
	}
	for _, token := range result.Tokens {
		fmt.Fprintf(&output, "%d %d %d %d ", token.Kind, token.Trivia,
			token.Span.StartByte, token.Span.EndByte)
		writeBytes([]byte(token.Cooked))
	}
	fmt.Fprintf(&output, "%d ", len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		writeBytes([]byte(diagnostic.Code))
		writeBytes([]byte(diagnostic.Repair))
		writeBytes([]byte(diagnostic.Token))
		fmt.Fprintf(&output, "%d %d ", diagnostic.Span.StartByte, diagnostic.Span.EndByte)
	}
	return output.String()
}
