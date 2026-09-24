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
	inputs := scannerCReferenceInputs()
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

func scannerCReferenceInputs() []struct {
	name string
	data []byte
} {
	return []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"identifier and keyword", []byte("module fairy; true false null defer")},
		{"all keyword spellings", []byte("as assert break case cleanup const continue discard else enum extern false fn for if import in let module mut null pub ptr ref region resource return struct switch true unsafe var while defer")},
		{"BOM CRLF and nested comment", append([]byte{0xef, 0xbb, 0xbf}, []byte("module main;\r\n/// doc\r\n/* outer /* inner */ done */\r\n")...)},
		{"symbols and whitespace", []byte("a <<= b::c -> d..e += f\t\n")},
		{"all symbol spellings", []byte("<<= >>= += -= *= /= %= &= ^= |= :: -> => == != <= >= << >> && || .. @ ( ) { } [ ] , ; : . ? = + - * / % ! ~ & | ^ < >")},
		{"unclosed comment", []byte("let x; /* still open")},
		{"invalid bytes", []byte{'m', ' ', 0xff, ' ', 0xe3, 0x95, 0x9c, ' ', '#', 0}},
		{"overlong two-byte source", []byte{'a', ' ', 0xc0, 0x80, ' ', 'b'}},
		{"overlong three-byte source", []byte{'a', ' ', 0xe0, 0x80, 0x80, ' ', 'b'}},
		{"surrogate source", []byte{'a', ' ', 0xed, 0xa0, 0x80, ' ', 'b'}},
		{"above Unicode maximum source", []byte{'a', ' ', 0xf4, 0x90, 0x80, 0x80, ' ', 'b'}},
		{"invalid leading byte source", []byte{'a', ' ', 0xf5, 0x80, 0x80, 0x80, ' ', 'b'}},
		{"truncated two-byte source", []byte{'a', ' ', 0xc2}},
		{"truncated three-byte source", []byte{'a', ' ', 0xe2, 0x82}},
		{"truncated four-byte source", []byte{'a', ' ', 0xf0, 0x9f, 0x92}},
		{"valid upper-bound scalar source", []byte{'a', ' ', 0xf4, 0x8f, 0xbf, 0xbf, ' ', 'b'}},
		{"overlong string bytes", []byte{'"', 'a', 0xc0, 0x80, 'b', '"'}},
		{"surrogate string bytes", []byte{'"', 'a', 0xed, 0xa0, 0x80, 'b', '"'}},
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
}

func TestPrivateVectorSDKCandidateCStringCookedLimit(t *testing.T) {
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
	const probe = `
int main(void) {
    struct sample {
        const char *source;
        unsigned kind;
        size_t opener, initial;
        int status;
        size_t total, hex_length;
        const char *hex;
    };
    static const struct sample cases[] = {
        {"\"A\"", FC_STRING, 1, 65534, 0, 65536, 2, "41"},
        {"\"A\"", FC_STRING, 1, 65535, FC_CANDIDATE_COOKED_SIZE, 65535, 0, ""},
        {"b\"A\"", FC_BYTE_STRING, 2, 65534, 0, 65536, 2, "41"},
        {"c\"A\"", FC_C_STRING, 2, 65532, 0, 65536, 4, "4100"},
        {"c\"A\"", FC_C_STRING, 2, 65533, FC_CANDIDATE_COOKED_SIZE, 65533, 0, ""},
        {"\"\"", FC_STRING, 1, 65536, 0, 65536, 0, ""},
        {"\"\\u{1f600}\"", FC_STRING, 1, 65528, 0, 65536, 8, "f09f9880"},
        {"\"\\u{1f600}\"", FC_STRING, 1, 65529, FC_CANDIDATE_COOKED_SIZE, 65529, 0, ""},
    };
    for (size_t i = 0; i < sizeof(cases) / sizeof(cases[0]); i++) {
        const struct sample *item = &cases[i];
        struct fc_token tokens[1] = {0};
        struct fc_diagnostic diagnostics[2] = {0};
        struct fc_scanner scanner = {(const unsigned char *)item->source,
                                     strlen(item->source), 0, 0, 0, tokens, diagnostics};
        scanner.cooked_total = item->initial;
        int status = fc_scan_string(&scanner, item->kind, item->opener);
        int failed = status != item->status || scanner.cooked_total != item->total ||
                     scanner.token_count != (item->status == 0 ? 1u : 0u) ||
                     scanner.diagnostic_count != 0;
        if (!failed && item->status == 0)
            failed = tokens[0].kind != item->kind ||
                     tokens[0].cooked_length != item->hex_length ||
                     memcmp(tokens[0].cooked, item->hex, item->hex_length) != 0;
        fc_release_cooked(&scanner);
        if (failed) return (int)(i + 1u);
    }
    return 0;
}
`
	code := "#define FC_CANDIDATE_COOKED_LIMIT\n#define FC_SCANNER_LIBRARY\n" + string(core) + probe
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("string cooked limit: %+v %v", outcome, err)
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

func TestPrivateVectorSDKCandidateIndependentCScannerGeometricTokens(t *testing.T) {
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
	const probe = `
int main(void) {
#ifdef _WIN32
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 ||
        _setmode(_fileno(stdout), _O_BINARY) == -1) return FC_BAD_INPUT;
#endif
    int count = getchar();
    if (count < 0 || count > 16 || getchar() != EOF) return FC_BAD_INPUT;
    unsigned char source[16];
    memset(source, '@', (size_t)count);
    struct fc_diagnostic diagnostics[17] = {0};
    struct fc_scanner scanner = {source, (size_t)count, 0, 0, 0, NULL, diagnostics};
    int status = fc_scan(&scanner);
    size_t expected = count < 8 ? 8u : count < 16 ? 16u : 32u;
    if (status == 0 && (scanner.token_count != (size_t)count + 1u ||
                        scanner.token_capacity != expected ||
                        scanner.diagnostic_count != 0)) status = 9;
    if (status == 0) fc_print(&scanner);
    fc_release_cooked(&scanner);
    free(scanner.tokens);
    return status;
}`
	code := "#define FC_CANDIDATE_GEOMETRIC_TOKENS\n#define FC_SCANNER_LIBRARY\n" + string(core) + probe
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			for _, count := range []byte{0, 7, 8, 15, 16} {
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, bytes.NewReader([]byte{count}))
				want := scannerGoObservation([]byte(strings.Repeat("@", int(count))))
				if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" ||
					outcome.TrapControl != "" || outcome.Stdout != want {
					t.Fatalf("tokens=%d: %+v %v\nwant %q", count, outcome, err, want)
				}
			}
			t.Run("token limit", func(t *testing.T) {
				if !strings.Contains(scannerCBenchmarkSource(t), "#define FC_CANDIDATE_TOKEN_LIMIT\n") ||
					!strings.Contains(scannerCGeometricBenchmarkSource(t), "#define FC_CANDIDATE_TOKEN_LIMIT\n") {
					t.Fatal("C budget images omitted the token limit")
				}
				const limitProbe = `
int main(void) {
#ifdef _WIN32
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 ||
        _setmode(_fileno(stdout), _O_BINARY) == -1) return FC_BAD_INPUT;
#endif
    int limit = getchar(), length = getchar();
    if (limit < 0 || limit > 3 || length < 0 || length > 16) return FC_BAD_INPUT;
    unsigned char source[16] = {0};
    if (fread(source, 1, (size_t)length, stdin) != (size_t)length || getchar() != EOF)
        return FC_BAD_INPUT;
    struct fc_token *tokens = NULL;
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
    tokens = (struct fc_token *)calloc((size_t)length + 2u, sizeof(*tokens));
#endif
    struct fc_diagnostic *diagnostics = NULL;
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    diagnostics = (struct fc_diagnostic *)calloc((size_t)length + 2u, sizeof(*diagnostics));
#endif
    if (
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
        tokens == NULL ||
#endif
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
        diagnostics == NULL ||
#endif
        0) { free(tokens); free(diagnostics); return 3; }
    struct fc_scanner scanner = {source, (size_t)length, 0, 0, 0, tokens, diagnostics};
    scanner.token_limit = (size_t)limit;
    int status = fc_scan(&scanner);
    if (status == 0) {
        qsort(scanner.diagnostics, scanner.diagnostic_count,
              sizeof(*scanner.diagnostics), fc_compare_diagnostic);
        fc_print(&scanner);
    }
    fc_release_cooked(&scanner);
    free(scanner.tokens);
    free(scanner.diagnostics);
    return status;
}`
				cases := []struct {
					name  string
					data  []byte
					limit byte
				}{
					{"empty", nil, 0},
					{"first token", []byte("a"), 0},
					{"exactly one token", []byte("a"), 1},
					{"middle token", []byte("a b"), 1},
					{"last token", []byte("a b"), 2},
					{"exactly three tokens", []byte("a b"), 3},
					{"invalid diagnostic suppressed", []byte("#x"), 0},
					{"BOM", []byte{0xef, 0xbb, 0xbf, 'a'}, 0},
					{"BOM exactly at limit", []byte{0xef, 0xbb, 0xbf}, 1},
					{"cooked value discarded", []byte("1e3 x"), 0},
				}
				for _, storage := range []struct {
					name    string
					defines string
				}{
					{"preallocated", ""},
					{"geometric", "#define FC_CANDIDATE_GEOMETRIC_TOKENS\n#define FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS\n"},
				} {
					t.Run(storage.name, func(t *testing.T) {
						code := "#define FC_CANDIDATE_TOKEN_LIMIT\n#define FC_CANDIDATE_DIAGNOSTIC_LIMIT\n#define FC_SCANNER_LIBRARY\n" + storage.defines + string(core) + limitProbe
						image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
						if err != nil {
							t.Fatal(err)
						}
						for _, item := range cases {
							input := append([]byte{item.limit, byte(len(item.data))}, item.data...)
							outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(input))
							limits := source.DefaultLimits()
							limits.MaxTokens = int(item.limit)
							want := scannerGoObservationWithLimits(item.data, limits)
							if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" ||
								outcome.TrapControl != "" || outcome.Stdout != want {
								t.Fatalf("%s: %+v %v\nwant %q", item.name, outcome, err, want)
							}
						}
					})
				}
				const fairyMain = `module main;
import scanner as scanner;
import diagnostics as diagnostics;
import std::bytes as bytes;
fn i32 check(ref u8[] input, usize limit, usize wanted_tokens, usize position, usize rejected_index, bool over) {
    let bytes::Buffer owned = switch (bytes::copy(input)) {
        case .ok(value): { value }
        case .err(_): { return 3; }
    };
    let scanner::Stream stream = switch (scanner::collect(owned, false, 32768, limit, 256, 16777216)) {
        case .ok(value): { value }
        case .err(code): { return code; }
    };
    if (scanner::token_count(ref stream) != wanted_tokens) { return 10; }
    let scanner::TokenInfo eof = scanner::token_info(ref stream, wanted_tokens - 1);
    if (eof.kind != 1 || eof.start != input.len || eof.end != input.len) { return 11; }
    if (!over) {
        if (scanner::diagnostic_count(ref stream) != 0) { return 12; }
    } else {
        if (scanner::diagnostic_count(ref stream) != 1) { return 13; }
        let diagnostics::Info exhausted = scanner::diagnostic_info(ref stream, 0);
        if (exhausted.code != 10 || exhausted.start != position || exhausted.end != position) { return 14; }
        let scanner::TokenInfo rejected = scanner::token_info(ref stream, rejected_index);
        if (rejected.kind != 0 || rejected.end != position) { return 15; }
        if (position < input.len) {
            let scanner::TokenInfo tail = scanner::token_info(ref stream, wanted_tokens - 2);
            if (tail.kind != 0 || tail.start != position || tail.end != input.len) { return 16; }
        }
    }
    return 0;
}
fn i32 main() {
    if (check(b""[..], 0, 1, 0, 0, false) != 0) { return 21; }
    if (check(b"a"[..], 0, 2, 1, 0, true) != 0) { return 22; }
    if (check(b"a"[..], 1, 2, 0, 0, false) != 0) { return 23; }
    if (check(b"a b"[..], 1, 4, 2, 1, true) != 0) { return 24; }
    if (check(b"a b"[..], 2, 4, 3, 2, true) != 0) { return 25; }
    if (check(b"a b"[..], 3, 4, 0, 0, false) != 0) { return 26; }
    if (check(b"#x"[..], 0, 3, 1, 0, true) != 0) { return 27; }
    if (check(b"\xEF\xBB\xBFa"[..], 0, 3, 3, 0, true) != 0) { return 28; }
    if (check(b"\xEF\xBB\xBF"[..], 1, 2, 0, 0, false) != 0) { return 29; }
    if (check(b"1e3 x"[..], 0, 3, 3, 0, true) != 0) { return 30; }
    return 0;
}`
				variants := realScannerVectorVariants(t)
				if len(variants) != 5 || variants[0].name != "explicit index" {
					t.Fatal("real scanner source changed")
				}
				program, entry, _ := scannerBudgetFairyProgramWithMain(t, variants[0], fairyMain)
				fairyImage, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
				fairyOutcome, err := cdriver.ExecuteWindowsHost(fairyImage)
				if err != nil || fairyOutcome.ExitCode != 0 || fairyOutcome.Stdout != "" || fairyOutcome.Stderr != "" {
					t.Fatalf("Fairy token limit: %+v %v", fairyOutcome, err)
				}
			})
		})
	}
}

func TestPrivateVectorSDKCandidateIndependentCScannerGeometricDiagnostics(t *testing.T) {
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
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" {
		t.Fatal("real scanner source changed")
	}
	fairyMain := fmt.Sprintf(`module main;
import scanner as scanner;
import diagnostics as diagnostics;
import std::bytes as bytes;
fn i32 check(ref u8[] input, usize wanted_tokens, usize wanted_diagnostics, bool over) {
    let bytes::Buffer owned = switch (bytes::copy(input)) {
        case .ok(value): { value }
        case .err(_): { return 3; }
    };
    let scanner::Stream stream = switch (scanner::collect(owned, false, 32768, 1048576, 256, 16777216)) {
        case .ok(value): { value }
        case .err(code): { return code; }
    };
    if (scanner::token_count(ref stream) != wanted_tokens) { return 10; }
    if (scanner::diagnostic_count(ref stream) != wanted_diagnostics) { return 11; }
    let scanner::TokenInfo eof = scanner::token_info(ref stream, wanted_tokens - 1);
    if (eof.kind != 1 || eof.start != input.len || eof.end != input.len) { return 12; }
    if (over) {
        let diagnostics::Info last = scanner::diagnostic_info(ref stream, wanted_diagnostics - 1);
        if (last.code != 10 || last.start != 257 || last.end != 257) { return 13; }
        let scanner::TokenInfo tail = scanner::token_info(ref stream, wanted_tokens - 2);
        if (tail.kind != 0 || tail.start != wanted_tokens - 2 || tail.end != input.len) { return 14; }
    }
    return 0;
}
fn i32 main() {
    if (check(b"%s"[..], 257, 256, false) != 0) { return 21; }
    if (check(b"%s"[..], 258, 257, true) != 0) { return 22; }
    if (check(b"%s"[..], 259, 257, true) != 0) { return 23; }
    return 0;
}`, strings.Repeat("#", 256), strings.Repeat("#", 257), strings.Repeat("#", 258))
	program, entry, _ := scannerBudgetFairyProgramWithMain(t, variants[0], fairyMain)
	const probe = `
int main(void) {
#ifdef _WIN32
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 ||
        _setmode(_fileno(stdout), _O_BINARY) == -1) return FC_BAD_INPUT;
#endif
    int count = getchar();
    if (count < 0 || count > 17 || getchar() != EOF) return FC_BAD_INPUT;
    unsigned char source[17];
    memset(source, '#', (size_t)count);
    struct fc_scanner scanner = {source, (size_t)count, 0, 0, 0, NULL, NULL};
    int status = fc_scan(&scanner);
    size_t expected = count == 0 ? 0u : count <= 8 ? 8u :
                      count <= 16 ? 16u : 32u;
    if (status == 0 && (scanner.diagnostic_count != (size_t)count ||
                        scanner.diagnostic_capacity != expected ||
                        scanner.token_count != (size_t)count + 1u)) status = 9;
    if (status == 0) fc_print(&scanner);
    fc_release_cooked(&scanner);
    free(scanner.tokens);
    free(scanner.diagnostics);
    return status;
}`
	code := "#define FC_CANDIDATE_GEOMETRIC_TOKENS\n#define FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS\n#define FC_SCANNER_LIBRARY\n" + string(core) + probe
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			for _, count := range []byte{0, 1, 8, 9, 16, 17} {
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, bytes.NewReader([]byte{count}))
				want := scannerGoObservation([]byte(strings.Repeat("#", int(count))))
				if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" ||
					outcome.TrapControl != "" || outcome.Stdout != want {
					t.Fatalf("diagnostics=%d: %+v %v\nwant %q", count, outcome, err, want)
				}
			}
			t.Run("diagnostic limit", func(t *testing.T) {
				if !strings.Contains(scannerCBenchmarkSource(t), "#define FC_CANDIDATE_DIAGNOSTIC_LIMIT\n") ||
					!strings.Contains(scannerCGeometricBenchmarkSource(t), "#define FC_CANDIDATE_DIAGNOSTIC_LIMIT\n") {
					t.Fatal("C budget images omitted the diagnostic limit")
				}
				for _, storage := range []struct {
					name    string
					defines string
				}{
					{"preallocated", ""},
					{"geometric", "#define FC_CANDIDATE_GEOMETRIC_TOKENS\n#define FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS\n"},
				} {
					t.Run(storage.name, func(t *testing.T) {
						code := "#define FC_CANDIDATE_DIAGNOSTIC_LIMIT\n" + storage.defines + string(core)
						image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
						if err != nil {
							t.Fatal(err)
						}
						for _, count := range []int{256, 257, 258} {
							data := []byte(strings.Repeat("#", count))
							outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, scannerCInput(data))
							want := scannerGoObservation(data)
							if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" ||
								outcome.TrapControl != "" || outcome.Stdout != want {
								t.Fatalf("diagnostic limit %d: %+v %v\nwant %q", count, outcome, err, want)
							}
						}
					})
				}
				fairyImage, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
				fairyOutcome, err := cdriver.ExecuteWindowsHost(fairyImage)
				if err != nil || fairyOutcome.ExitCode != 0 || fairyOutcome.Stdout != "" || fairyOutcome.Stderr != "" {
					t.Fatalf("Fairy diagnostic limit: %+v %v", fairyOutcome, err)
				}
			})
			t.Run("diagnostic payload ownership", func(t *testing.T) {
				if !strings.Contains(scannerCBenchmarkSource(t), "#define FC_CANDIDATE_DIAGNOSTIC_PAYLOAD\n#define FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT\n") ||
					!strings.Contains(scannerCGeometricBenchmarkSource(t), "#define FC_CANDIDATE_DIAGNOSTIC_PAYLOAD\n#define FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT\n") {
					t.Fatal("C budget images omitted the owned diagnostic Entry layout")
				}
				const payloadProbe = `
int main(void) {
    char token[] = "*/", empty[] = "";
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    struct fc_diagnostic entries[2] = {0};
#endif
    struct fc_scanner scanner = {0};
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    scanner.diagnostics = entries;
#endif
    fc_diagnostic(&scanner, "E0001", 0, 2, "exit", token);
    int status = scanner.diagnostic_count != 1u ||
                 scanner.diagnostic_payload_failed ||
                 scanner.diagnostics[0].token_length != 2u ||
                 scanner.diagnostics[0].token == token;
    token[0] = '!';
    if (!status && memcmp(scanner.diagnostics[0].token, "*/", 2u) != 0)
        status = 2;
    fc_diagnostic(&scanner, "E0010", 2, 2, "none", empty);
    if (!status && (scanner.diagnostic_count != 2u ||
                    scanner.diagnostics[1].token_length != 0u ||
                    scanner.diagnostics[1].token != empty)) status = 3;
    fc_release_diagnostics(&scanner);
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    free(scanner.diagnostics);
#endif
    return status;
}`
				for _, storage := range []struct {
					name    string
					defines string
				}{
					{"preallocated", ""},
					{"geometric", "#define FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS\n"},
				} {
					t.Run(storage.name, func(t *testing.T) {
						code := "#define FC_CANDIDATE_DIAGNOSTIC_PAYLOAD\n#define FC_SCANNER_LIBRARY\n" + storage.defines + string(core) + payloadProbe
						image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
						if err != nil {
							t.Fatal(err)
						}
						outcome, err := cdriver.ExecuteWindowsMSVCProbe(image)
						if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
							t.Fatalf("payload ownership: %+v %v", outcome, err)
						}
					})
				}
			})
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
	geometric, err := cdriver.BuildWindowsMSVCProbe(toolchain, cdriver.OptimizedProfile,
		scannerCGeometricBenchmarkSource(t))
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(geometric, strings.NewReader("12345 1000\n"))
	if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
		t.Fatalf("geometric C benchmark smoke: %+v %v", outcome, err)
	}
	prefix := scannerGoObservation([]byte(sample))
	fields := strings.Fields(strings.TrimPrefix(outcome.Stdout, prefix))
	if !strings.HasPrefix(outcome.Stdout, prefix) || len(fields) != 4 || fields[0] != "1000" ||
		fields[3] != strconv.Itoa(perRound*1000) {
		t.Fatalf("geometric C benchmark reporter or checksum differs: %q", outcome.Stdout)
	}
}

func scannerCBenchmarkSource(t *testing.T) string {
	return scannerCBenchmarkSourceWithStorage(t, false)
}

func scannerCGeometricBenchmarkSource(t *testing.T) string {
	return scannerCBenchmarkSourceWithStorage(t, true)
}

func scannerCBenchmarkSourceWithStorage(t *testing.T, geometric bool) string {
	t.Helper()
	defines := scannerCBudgetDefines(geometric)
	parts := []string{defines + "#define FC_SCANNER_LIBRARY\n#define FC_SCANNER_WORKLOAD_LIBRARY\n"}
	for _, name := range []string{"scanner_reference.c", "scanner_workload_reference.c", "support.c", "scanner_workload_benchmark.c"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", name))
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, string(data), "\n")
	}
	return strings.Join(parts, "")
}

func scannerCBudgetDefines(geometric bool) string {
	defines := "#define FC_CANDIDATE_SOURCE_LIMIT\n#define FC_CANDIDATE_DECIMAL_LIMIT\n#define FC_CANDIDATE_COOKED_LIMIT\n#define FC_CANDIDATE_WORK_LIMIT\n#define FC_CANDIDATE_DIAGNOSTIC_LIMIT\n#define FC_CANDIDATE_TOKEN_LIMIT\n#define FC_CANDIDATE_DIAGNOSTIC_PAYLOAD\n#define FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT\n"
	if geometric {
		defines += "#define FC_CANDIDATE_GEOMETRIC_TOKENS\n#define FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS\n"
	}
	return defines
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
	return scannerGoObservationWithLimits(data, source.DefaultLimits())
}

func scannerGoObservationWithLimits(data []byte, limits source.Limits) string {
	result := lexer.Lex(source.NewFile("input.fy", data), source.NewBudget(limits, len(data)))
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
