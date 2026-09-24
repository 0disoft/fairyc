package compiler_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// This C source is independently authored and still rejects decimal
// expansions beyond the default work budget until it can match the full scanner.
func TestPrivateVectorSDKCandidateIndependentCScannerCore(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "scanner_reference.c"))
	if err != nil {
		t.Fatal(err)
	}
	inputs := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"identifier and keyword", []byte("module fairy; true false null defer")},
		{"BOM CRLF and nested comment", append([]byte{0xef, 0xbb, 0xbf}, []byte("module main;\r\n/// doc\r\n/* outer /* inner */ done */\r\n")...)},
		{"symbols and whitespace", []byte("a <<= b::c -> d..e += f\t\n")},
		{"unclosed comment", []byte("let x; /* still open")},
		{"invalid bytes", []byte{'m', ' ', 0xff, ' ', 0xe3, 0x95, 0x9c, ' ', '#', 0}},
		{"lone CR", []byte("abc \rnext")},
		{"real scanner sample", []byte("let n=12345; /*open")},
		{"integer spellings", []byte("0 000_42 0xDEAD_BEEF 0b1010_0011 0xffffffffffffffffffffffffffffffff")},
		{"invalid integers", []byte("0x 0b 1__2 12abc 0xG 0b102")},
		{"decimal floats", []byte("1.25 1_024.5e-2 1e2 1e-2 1.0e2 0.000 00.0e2 1e_2 -1.5")},
		{"invalid floats", []byte("1e 1e+ 1.2abc 1e+foo 1e999999999999999999999999999999")},
		{"plain strings", []byte(`"text" b"bytes" c"cstr" "" b"" c""`)},
		{"simple escapes", []byte(`"a\n\\\"" b"\x00A" c"\0" c"\x00"`)},
		{"raw Unicode", []byte(`"한" b"한" c"한"`)},
		{"Unicode escapes", []byte(`"\u{41}\u{03bb}\u{1F600}\u{10ffff}\u{0}"`)},
		{"invalid Unicode scalars", []byte(`"\u{d800}\u{110000}\u{}\u{1234567}"`)},
		{"invalid Unicode recovery", []byte(`"\u{12g}x" "\u{12" b"\u{41}" c"\u{41}"`)},
		{"invalid string bytes", []byte{'"', 'a', 0xff, 'b', '"'}},
		{"invalid escaped UTF-8", []byte(`"\xff"`)},
		{"invalid escapes", []byte(`"\q" "\xG" "\xF"`)},
		{"unclosed string", []byte(`"open`)},
		{"newline in string", []byte("\"line\nnext")},
		{"trailing escape", []byte{'"', 'x', '\\'}},
		{"raw C NUL", []byte{'c', '"', 'a', 0, 'b', '"'}},
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, string(code))
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range inputs {
				t.Run(item.name, func(t *testing.T) {
					outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, scannerCInput(item.data))
					if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("C scanner execution: %+v %v", outcome, err)
					}
					want := scannerGoObservation(item.data)
					if outcome.Stdout != want {
						t.Fatalf("C scanner differs from Go lexer\n got: %q\nwant: %q", outcome.Stdout, want)
					}
				})
			}
			for _, unsupported := range []string{"1e17000000", "1e-17000000"} {
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, scannerCInput([]byte(unsupported)))
				if err != nil || outcome.ExitCode != 7 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("unsupported %q: %+v %v", unsupported, outcome, err)
				}
			}
		})
	}
}

// A compact digest keeps large cooked tokens below the native driver's output
// cap while checking their full-payload digest against the independent Go lexer.
func TestPrivateVectorSDKCandidateIndependentCScannerLargeDecimal(t *testing.T) {
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
	const cMain = `
int main(void) {
#ifdef _WIN32
    if (_setmode(_fileno(stdout), _O_BINARY) == -1) return FC_BAD_INPUT;
#endif
    static const char *const inputs[] = {"1e70000", "1e-70000"};
    for (size_t i = 0; i < sizeof(inputs) / sizeof(inputs[0]); i++) {
        const char *raw = inputs[i];
        struct fc_token tokens[4] = {0};
        struct fc_diagnostic diagnostics[4] = {0};
        struct fc_scanner scanner = {(const unsigned char *)raw, strlen(raw),
                                     0, 0, 0, tokens, diagnostics};
        int status = fc_scan(&scanner);
        if (status != 0 || scanner.token_count != 2 ||
            scanner.diagnostic_count != 0 || tokens[0].kind != FC_FLOAT) {
            fc_release_cooked(&scanner);
            return status != 0 ? status : 9;
        }
        uint64_t digest = UINT64_C(14695981039346656037);
        for (size_t j = 0; j < tokens[0].cooked_length; j++) {
            digest ^= (uint64_t)tokens[0].cooked[j];
            digest *= UINT64_C(1099511628211);
        }
        printf("%zu %llu\n", tokens[0].cooked_length,
               (unsigned long long)digest);
        fc_release_cooked(&scanner);
    }
    return 0;
}
`
	code := "#define FC_SCANNER_LIBRARY\n" + string(core) + "\n" + cMain
	var want strings.Builder
	for _, raw := range []string{"1e70000", "1e-70000"} {
		lexed := lexer.Lex(source.NewFile("input.fy", []byte(raw)),
			source.NewBudget(source.DefaultLimits(), len(raw)))
		if len(lexed.Tokens) != 2 || len(lexed.Diagnostics) != 0 || lexed.Tokens[0].Kind != lexer.Float {
			t.Fatalf("Go lexer did not accept %q: %+v", raw, lexed.Diagnostics)
		}
		digest := fnv.New64a()
		if _, err := digest.Write([]byte(lexed.Tokens[0].Cooked)); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&want, "%d %d\n", len(lexed.Tokens[0].Cooked), digest.Sum64())
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" ||
				outcome.Stdout != want.String() {
				t.Fatalf("large decimal parity: %+v %v\nwant %q", outcome, err, want.String())
			}
		})
	}
}

func TestPrivateVectorSDKCandidateIndependentCScannerWorkload(t *testing.T) {
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
	workload, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "scanner_workload_reference.c"))
	if err != nil {
		t.Fatal(err)
	}
	code := "#define FC_SCANNER_LIBRARY\n" + string(core) + "\n" + string(workload)
	data := []byte("let n=12345; /*open")
	lexed := lexer.Lex(source.NewFile("input.fy", data), source.NewBudget(source.DefaultLimits(), len(data)))
	perRound := len(data) + len(lexed.Tokens) + len(lexed.Diagnostics)
	for _, token := range lexed.Tokens {
		perRound += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		perRound += len(diagnostic.Token)
	}
	if len(lexed.Tokens) == 0 || len(lexed.Diagnostics) == 0 || perRound <= len(data) {
		t.Fatal("workload lost token, cooked or diagnostic coverage")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			for _, rounds := range []uint32{1, 2} {
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, scannerCWorkloadInput(rounds, data))
				if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
					t.Fatalf("C workload rounds=%d: %+v %v", rounds, outcome, err)
				}
				want := fmt.Sprintf("%sS %d\n", scannerGoObservation(data), perRound*int(rounds))
				if outcome.Stdout != want {
					t.Fatalf("C workload rounds=%d differs\n got: %q\nwant: %q", rounds, outcome.Stdout, want)
				}
			}
		})
	}
}

func TestPrivateVectorSDKCandidateIndependentCScannerBenchmarkSmoke(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	code := scannerCBenchmarkSource(t)
	const sample = "let n=12345; /*open"
	lexed := lexer.Lex(source.NewFile("input.fy", []byte(sample)), source.NewBudget(source.DefaultLimits(), len(sample)))
	perRound := len(sample) + len(lexed.Tokens) + len(lexed.Diagnostics)
	for _, token := range lexed.Tokens {
		perRound += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		perRound += len(diagnostic.Token)
	}
	if perRound <= len(sample) || len(lexed.Diagnostics) == 0 {
		t.Fatal("benchmark sample lost cooked or diagnostic coverage")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, strings.NewReader("12345 1000\n"))
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("C benchmark smoke: %+v %v", outcome, err)
			}
			prefix := scannerGoObservation([]byte(sample))
			if !strings.HasPrefix(outcome.Stdout, prefix) {
				t.Fatalf("C benchmark reporter differs: %q", outcome.Stdout)
			}
			fields := strings.Fields(strings.TrimPrefix(outcome.Stdout, prefix))
			if len(fields) != 4 {
				t.Fatalf("C benchmark fields=%d: %q", len(fields), outcome.Stdout)
			}
			completed, completedErr := strconv.ParseUint(fields[0], 10, 32)
			ticks, ticksErr := strconv.ParseUint(fields[1], 10, 64)
			frequency, frequencyErr := strconv.ParseUint(fields[2], 10, 64)
			total, totalErr := strconv.ParseUint(fields[3], 10, 64)
			if completedErr != nil || ticksErr != nil || frequencyErr != nil || totalErr != nil ||
				completed != 1000 || ticks == 0 || frequency == 0 || total != uint64(perRound*1000) {
				t.Fatalf("C benchmark values=%v errors=%v,%v,%v,%v", fields, completedErr, ticksErr, frequencyErr, totalErr)
			}
		})
	}
}

func scannerCBenchmarkSource(t *testing.T) string {
	t.Helper()
	parts := []string{"#define FC_CANDIDATE_DECIMAL_LIMIT\n#define FC_CANDIDATE_COOKED_LIMIT\n#define FC_SCANNER_LIBRARY\n#define FC_SCANNER_WORKLOAD_LIBRARY\n"}
	for _, name := range []string{"scanner_reference.c", "scanner_workload_reference.c", "support.c", "scanner_workload_benchmark.c"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", name))
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, string(data), "\n")
	}
	return strings.Join(parts, "")
}

func scannerCWorkloadInput(rounds uint32, data []byte) *bytes.Reader {
	encoded := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint32(encoded, rounds)
	binary.LittleEndian.PutUint32(encoded[4:], uint32(len(data)))
	copy(encoded[8:], data)
	return bytes.NewReader(encoded)
}

func scannerCInput(data []byte) *bytes.Reader {
	encoded := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(encoded, uint32(len(data)))
	copy(encoded[4:], data)
	return bytes.NewReader(encoded)
}

func scannerGoObservation(data []byte) string {
	result := lexer.Lex(source.NewFile("input.fy", data), source.NewBudget(source.DefaultLimits(), len(data)))
	var output strings.Builder
	for _, token := range result.Tokens {
		cooked := hex.EncodeToString([]byte(token.Cooked))
		if cooked == "" {
			cooked = "-"
		}
		fmt.Fprintf(&output, "T %d %d %d %d %s\n", token.Kind, token.Trivia,
			token.Span.StartByte, token.Span.EndByte, cooked)
	}
	for _, diagnostic := range result.Diagnostics {
		text := hex.EncodeToString([]byte(diagnostic.Token))
		if text == "" {
			text = "-"
		}
		fmt.Fprintf(&output, "D %s %d %d %s %s\n", diagnostic.Code,
			diagnostic.Span.StartByte, diagnostic.Span.EndByte, diagnostic.Repair, text)
	}
	return output.String()
}
