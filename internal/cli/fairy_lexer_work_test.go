package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyLexerWorkBudget(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	for _, group := range []struct {
		name   string
		inputs []string
		extra  int
	}{
		{"common", []string{"", "\xef\xbb\xbfabc", "abc", "true false", "a1 + foo", "\r\n x", " \r\n\t", " \r x", "//abc\r\nx", "///a\n", "/*x*/a", "/*a/*b*/c*/", "/*open", "+ <<=", "\xff \x00", "😀a", "한", "\xc2\xa0"}, 0},
		{"integers", []string{"0", "12", "0xFF", "0b10", "1_2", "1abc", "0x"}, 0},
		{"decimals", []string{"1.25", "1e3", "1e-3", "0e-0", "1.200", "1e+", "1e1 1e1"}, 8},
		{"extreme", []string{"1e100", "1e-100", "0e100", "1e9223372036854775807", "1e-9223372036854775808", "1e9223372036854775808", "1e-9223372036854775809"}, 0},
		{"strings", []string{`""`, `"a"`, `b"abc"`, `c"abc"`, `"a 한 😀"`, "\"line\nend", "\"line\rend", `"open`, `"`, `b"`, `c"`}, 0},
		{"escapes", []string{`"\n\r\t\\"`, `"\x41"`, `c"\x00"`, `"\u{1f600}"`, `b"\u{41}"`, `"\q"`}, 0},
		{"malformed-escapes", []string{`"\x"`, `"\x1Z"`, `"\u{}"`, `"\u{1234567}"`, `"\u{bad missing"`, `"\u{d800}"`, "\"unfinished\\"}, 0},
		{"string-bytes", []string{"\"\x00\xff\"", "b\"\xff\"", "c\"\x00\"", "\"\xf0\x90\"", `"\u{110000}"`, `"\xc2\xa0"`}, 0},
		{"mixed", []string{`"a" 1.25 x`, "1e3 \"\\q\" \r", `/*x*/ c"\0"`}, 8},
	} {
		t.Run(group.name, func(t *testing.T) { testWindowsLexerWorkInputs(t, group.inputs, group.extra, 256) })
	}
	for _, limit := range []int{0, 1} {
		t.Run(fmt.Sprintf("diagnostics-%d", limit), func(t *testing.T) {
			testWindowsLexerWorkInputs(t, []string{`"\q\q" tail`, "\"a\x00\xff\"", `c"\0" 1.2`}, 0, limit)
		})
	}
}

func testWindowsLexerWorkInputs(t *testing.T, inputs []string, extra, diagnosticLimit int) {
	root := t.TempDir()
	for _, name := range []string{"fairy.toml", "src/scanner.fy", "src/numbers.fy", "src/strings.fy", "src/diagnostics.fy", "src/utf8.fy", "src/budget.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, name, string(data))
	}
	var program, expected strings.Builder
	program.WriteString("module main; import scanner as scanner; @effects(console) fn i32 main() {\n")
	var names []string
	var starts []int
	for _, input := range inputs {
		var workLimits []int
		for limit := 0; limit <= len(input)+1+extra; limit++ {
			workLimits = append(workLimits, limit)
		}
		if extra > 0 {
			workLimits = append(workLimits, 28, 29, 31, 64)
		}
		for _, limit := range workLimits {
			for _, tokenLimit := range []int{0, 1, 1048576} {
				names = append(names, fmt.Sprintf("input=%q work=%d tokens=%d diagnostics=%d", input, limit, tokenLimit, diagnosticLimit))
				starts = append(starts, expected.Len())
				program.WriteString("assert(scanner::report_work(b\"")
				for _, b := range []byte(input) {
					fmt.Fprintf(&program, "\\x%02X", b)
				}
				fmt.Fprintf(&program, "\", false, 4096, %d, %d, %d) == 0);\n", tokenLimit, diagnosticLimit, limit)
				limits := source.DefaultLimits()
				limits.MaxWorkUnits = limit
				limits.MaxTokens = tokenLimit
				limits.MaxDiagnostics = diagnosticLimit
				result := lexer.Lex(source.NewFile("input.fy", []byte(input)), source.NewBudget(limits, len(input)))
				for _, token := range result.Tokens {
					fmt.Fprintf(&expected, "%d %d %d %d %d ", token.Kind, token.Trivia, token.Span.StartByte, token.Span.EndByte, len(token.Cooked))
					for _, b := range []byte(token.Cooked) {
						fmt.Fprintf(&expected, "%d ", b)
					}
				}
				fmt.Fprintf(&expected, "%d ", len(result.Diagnostics))
				for _, d := range result.Diagnostics {
					for _, value := range []string{d.Code, string(d.Repair), d.Token} {
						fmt.Fprintf(&expected, "%d ", len(value))
						for _, b := range []byte(value) {
							fmt.Fprintf(&expected, "%d ", b)
						}
					}
					fmt.Fprintf(&expected, "%d %d ", d.Span.StartByte, d.Span.EndByte)
				}
			}
		}
	}
	program.WriteString("return 0; }\n")
	if expected.Len() >= 60000 {
		t.Fatal("work-budget observations exceed native output bound")
	}
	writeTestSource(t, root, "src/main.fy", program.String())
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("work budget release=%v code=%d stderr=%s", release, code, stderr)
		}
		if stdout != expected.String() {
			want := expected.String()
			i := 0
			for i < len(stdout) && i < len(want) && stdout[i] == want[i] {
				i++
			}
			caseIndex := 0
			for caseIndex+1 < len(starts) && starts[caseIndex+1] <= i {
				caseIndex++
			}
			t.Fatalf("work budget release=%v %s byte=%d got=%q want=%q", release, names[caseIndex], i, stdout[i:min(i+180, len(stdout))], want[i:min(i+180, len(want))])
		}
	}
	t.Logf("compared %d work/token budget combinations at both profiles", len(names))
}
