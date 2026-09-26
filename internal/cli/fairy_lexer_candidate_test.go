package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyLexerCandidate(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	inputs := []string{
		"", " \t\n\r\n", "\xef\xbb\xbfmodule main;\r\n",
		"as assert break case cleanup const continue discard else enum extern false fn for if import in let module mut null pub ptr ref region resource return struct switch true unsafe var while defer",
		"if ifx xif while2 _return return_ If defer deferx state key",
		"<<= >>= += -= *= /= %= &= ^= |= :: -> => == != <= >= << >> && || .. @ ( ) { } [ ] , ; : . ? = + - * / % ! ~ & | ^ < >",
		"<<<==== >>>= ->=> .... / /= //tail\n///docs\r\n",
		"/*outer /*nested*/ tail*/fn /*\xff\x00*/ main() {}",
		strings.Repeat("a", 4096),
		"0 000 0123 0xFf 0b101010 1_000 0xA_B 0b1_0 184467440737095516160 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
		"1.25 1e3 1e-3 001.200 0e3 0e-3 1__0.0 1e_2 1_e+2 1..2 1.foo .5 1. 1e+0",
		"0x 0b 0xG 0b2 1_ 1__0 1abc 1e 1e+ 1e-foo 1.2foo 0XFF 0B10 0x_1 ; tail",
		"\r \r\n \r foo\v\f\x00\xff 한 😀 /*open",
		"\xc2\xa0 \xe0\xa0\x80 \xed\x9f\xbf \xf0\x90\x80\x80 \xf4\x8f\xbf\xbf",
		"\xc0\xaf \xed\xa0\x80 \xf4\x90\x80\x80 \xe2\x82 a\xef\xbb\xbf \x80",
		strings.Repeat("\x00", 256),
		strings.Repeat("\x00", 257),
		`"" b"" c"" "hello 한 😀" b"bytes" c"cstring"`,
		`"\\\"\n\r\t\0\x41" "\u{0}\u{7f}\u{80}\u{7ff}\u{800}\u{ffff}\u{10000}\u{10ffff}"`,
		`b"\xff\x00" c"\0\x00" "\xc2\xa0" "\xff" "\u{d800}\u{110000}"`,
		`"\q\x\x1Z\u\u{}\u{1234567}" b"\u{41}" c"\u{41}"`,
		"\"unterminated\nnext\r\n\"again\rend",
		"\"nul\x00bad\xff\" b\"\x00\xff\" c\"\x00\xff\"",
		"\"unfinished\\",
		"\"\\u{bad\nmissing\" tail",
		"\"escaped\\\nline\"",
	}
	for _, seed := range []string{`"a\x41\u{1f600}한"`, `c"\0"`, `b"\x00"`} {
		for end := 0; end <= len(seed); end++ {
			inputs = append(inputs, seed[:end])
		}
	}
	root := t.TempDir()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	for _, name := range []string{"fairy.toml", "src/scanner.fy", "src/numbers.fy", "src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy"} {
		data, err := os.ReadFile(filepath.Join(example, name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, name, string(data))
	}
	var program, expected strings.Builder
	program.WriteString("module main;\nimport scanner as scanner;\nimport diagnostics as diagnostics;\nimport budget as budget;\nimport std::vec as v;\nimport std::bytes as bytes;\nfn void budget_checks() {\n")
	for _, limit := range []int{0, 1, 3, 16} {
		for _, charges := range [][]int{{0, 1, 0}, {1, 1, 1, 1}, {3, 0, 1}, {16, 1}, {17, 0}} {
			fmt.Fprintf(&program, "{ var budget::Work work = budget::create(%d);\n", limit)
			used := 0
			for _, charge := range charges {
				accepted := charge <= limit-used
				if accepted {
					used += charge
				}
				fmt.Fprintf(&program, "assert(budget::charge(ref mut work, %d) == %t); assert(work.remaining == %d);\n", charge, accepted, limit-used)
			}
			program.WriteString("}\n")
		}
	}
	// Exact usize maximum exercises overflow avoidance without allocating input.
	program.WriteString(`{
    let usize maximum = 18446744073709551615;
    var budget::Work work = budget::create(maximum);
    assert(budget::charge(ref mut work, maximum));
    assert(work.remaining == 0);
    assert(!budget::charge(ref mut work, 1));
    assert(work.remaining == 0);
    assert(budget::charge(ref mut work, 0));
    assert(work.remaining == 0);
}
`)
	program.WriteString("}\n@effects(console)\nfn i32 main() { budget_checks();\n")
	writeInput := func(input string, index, status int) {
		name := fmt.Sprintf("input%d", index)
		fmt.Fprintf(&program, "{ let bytes::Buffer %s = switch (bytes::copy(b\"", name)
		for _, b := range []byte(input) {
			fmt.Fprintf(&program, "\\x%02X", b)
		}
		program.WriteString("\")) { case .ok(value) : { value } case .err(_) : { assert(false); } };\n")
		fmt.Fprintf(&program, "assert(scanner::report(bytes::view(ref %s)) == %d); }\n", name, status)
	}
	for i, input := range inputs {
		writeInput(input, i, 0)
		result := lexer.Lex(source.NewFile("input.fy", []byte(input)), source.NewBudget(source.DefaultLimits(), len(input)))
		for _, token := range result.Tokens {
			fmt.Fprintf(&expected, "%d %d %d %d %d ", token.Kind, token.Trivia, token.Span.StartByte, token.Span.EndByte, len(token.Cooked))
			for _, b := range []byte(token.Cooked) {
				fmt.Fprintf(&expected, "%d ", b)
			}
		}
		fmt.Fprintf(&expected, "%d ", len(result.Diagnostics))
		for _, d := range result.Diagnostics {
			if d.Code != "E0004" && d.Code != "E0001" && d.Code != "E0005" && d.Code != "E0006" && d.Code != "E0010" {
				t.Fatalf("unexpected oracle diagnostic: %+v", d)
			}
			for _, value := range []string{d.Code, string(d.Repair), d.Token} {
				fmt.Fprintf(&expected, "%d ", len(value))
				for _, b := range []byte(value) {
					fmt.Fprintf(&expected, "%d ", b)
				}
			}
			fmt.Fprintf(&expected, "%d %d ", d.Span.StartByte, d.Span.EndByte)
		}
	}
	program.WriteString(`{
    var bytes::Buffer oversized = bytes::empty();
    var usize index = 0;
    while (index < 32769) {
        assert(switch (bytes::append_byte(ref mut oversized, 97)) { case .ok(_) : { true } case .err(_) : { false } });
        index += 1;
    }
    assert(scanner::report(bytes::view(ref oversized)) == 4);
}
`)
	writeInput("1e20000", 101, 4)
	writeInput("1e-20000", 102, 4)
	writeInput(strings.Repeat("1e10000 ", 7), 103, 4)
	// Caller-selected byte/token budgets preserve the reference exhaustion stream.
	budgetNames := map[int]string{}
	for _, input := range []string{"", "abc", "a b", "\xef\xbb\xbfabc", "\r x", " \r x", "/*open", "\x00x", "1abc x", `"\q" x`, "true false"} {
		for _, limit := range []int{0, 1, 2, len(input)} {
			for _, bytesLimited := range []bool{false, true} {
				budgetNames[expected.Len()] = fmt.Sprintf("input=%q limit=%d bytes=%v", input, limit, bytesLimited)
				limits := source.DefaultLimits()
				if bytesLimited {
					limits.MaxBytes = limit
				} else {
					limits.MaxTokens = limit
				}
				program.WriteString("assert(scanner::report_limits(b\"")
				for _, b := range []byte(input) {
					fmt.Fprintf(&program, "\\x%02X", b)
				}
				fmt.Fprintf(&program, "\", false, %d, %d) == 0);\n", limits.MaxBytes, limits.MaxTokens)
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
	for _, input := range []string{"", "abc", "\x00\x00x", "\r\xff x", "1abc 2abc", "/*open", `"\q\q" tail`, `c"\0\0" tail`, `"\xff" tail`, "\"a\x00\xff\" tail", "\"line\nend", "\"open", "\"open\\", "\x00 \"\\q\""} {
		for _, limit := range []int{0, 1, 2, 4} {
			budgetNames[expected.Len()] = fmt.Sprintf("diagnostics input=%q limit=%d", input, limit)
			program.WriteString("assert(scanner::report_budget(b\"")
			for _, b := range []byte(input) {
				fmt.Fprintf(&program, "\\x%02X", b)
			}
			fmt.Fprintf(&program, "\", false, 4096, 1048576, %d) == 0);\n", limit)
			limits := source.DefaultLimits()
			limits.MaxDiagnostics = limit
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
	// Exercise every sort key independently of the naturally ordered scanner.
	records := []diagnostic.Diagnostic{
		{Code: "E0006", Span: diagnostic.Span{StartByte: 9, EndByte: 10}, Repair: diagnostic.RepairNone},
		{Code: "E0005", Span: diagnostic.Span{StartByte: 1, EndByte: 3}, Repair: diagnostic.RepairNone, Token: "b"},
		{Code: "E0005", Span: diagnostic.Span{StartByte: 1, EndByte: 3}, Repair: diagnostic.RepairNone, Token: "a"},
		{Code: "E0005", Span: diagnostic.Span{StartByte: 1, EndByte: 3}, Repair: diagnostic.RepairExit, Token: "a"},
		{Code: "E0004", Span: diagnostic.Span{StartByte: 1, EndByte: 3}, Repair: diagnostic.RepairNone},
		{Code: "E0001", Span: diagnostic.Span{StartByte: 1, EndByte: 2}, Repair: diagnostic.RepairNone},
		{Code: "E0001", Span: diagnostic.Span{StartByte: 1, EndByte: 2}, Repair: diagnostic.RepairNone},
	}
	program.WriteString("var v::Vec<diagnostics::Entry> entries = v::empty<diagnostics::Entry>();\n")
	for _, d := range records {
		fmt.Fprintf(&program, "assert(diagnostics::add(ref mut entries, %d, %d, %d, %t, b%q) == 0);\n", d.Code[4]-'0', d.Span.StartByte, d.Span.EndByte, d.Repair == diagnostic.RepairExit, d.Token)
	}
	program.WriteString("diagnostics::report(ref entries);\n")
	slices.SortFunc(records, diagnostic.Compare)
	fmt.Fprintf(&expected, "%d ", len(records))
	for _, d := range records {
		for _, value := range []string{d.Code, string(d.Repair), d.Token} {
			fmt.Fprintf(&expected, "%d ", len(value))
			for _, b := range []byte(value) {
				fmt.Fprintf(&expected, "%d ", b)
			}
		}
		fmt.Fprintf(&expected, "%d %d ", d.Span.StartByte, d.Span.EndByte)
	}
	program.WriteString("return 0; }\n")
	writeTestSource(t, root, "src/main.fy", program.String())
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("candidate release=%v code=%d stderr=%s", release, code, stderr)
		}
		if stdout != expected.String() {
			want := expected.String()
			i := 0
			for i < len(stdout) && i < len(want) && stdout[i] == want[i] {
				i++
			}
			nearest := -1
			for offset := range budgetNames {
				if offset <= i && offset > nearest {
					nearest = offset
				}
			}
			t.Logf("budget case: %s offset=%d", budgetNames[nearest], nearest)
			t.Fatalf("candidate release=%v mismatch at byte %d; lengths got=%d want=%d; got=%q want=%q", release, i, len(stdout), len(want), stdout[i:min(i+200, len(stdout))], want[i:min(i+200, len(want))])
		}
	}
}
