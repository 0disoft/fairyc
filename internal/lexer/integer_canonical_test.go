package lexer

import (
	"math/big"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestDecimalIntegerCanonicalMatchesBigInt(t *testing.T) {
	for length := 1; length <= 256; length++ {
		digits := strings.Repeat("0123456789", (length+9)/10)[:length]
		for _, raw := range []string{digits, "000" + digits, strings.Join(strings.Split(digits, ""), "_")} {
			value, ok := new(big.Int).SetString(strings.ReplaceAll(raw, "_", ""), 10)
			if !ok {
				t.Fatal("invalid test integer")
			}
			got, valid := cookNumber(raw, Integer, source.NewBudget(source.DefaultLimits(), len(raw)))
			if !valid || got != value.String() {
				t.Fatalf("%q: got (%q, %v), want %q", raw, got, valid, value.String())
			}
		}
	}
	for _, raw := range []string{"", "_1", "1_", "1__0", "12a", "1.2", "1e2"} {
		if got, valid := cookNumber(raw, Integer, source.NewBudget(source.DefaultLimits(), len(raw))); valid || got != "" {
			t.Fatalf("invalid integer %q accepted as %q", raw, got)
		}
	}
}

func TestIntegerCanonicalBothLexerRevisions(t *testing.T) {
	for _, lex := range []func(source.File, *source.Budget) Result{Lex, LexPreviousRevision} {
		for _, test := range []struct{ raw, want string }{
			{"0", "0"}, {"0_0_0", "0"}, {"000_042", "42"},
			{"18446744073709551616", "18446744073709551616"},
			{"0x00_ff", "255"}, {"0b001_010", "10"},
			{strings.Repeat("9", 65536), strings.Repeat("9", 65536)},
		} {
			file := source.NewFile("main.fy", []byte(test.raw))
			limits := source.DefaultLimits()
			limits.MaxWorkUnits = len(file.Bytes)
			got := lex(file, source.NewBudget(limits, len(file.Bytes)))
			if len(got.Diagnostics) != 0 || len(got.Tokens) != 2 || got.Tokens[0].Kind != Integer || got.Tokens[0].Cooked != test.want {
				t.Fatalf("%d-byte integer: unexpected tokens or diagnostics", len(test.raw))
			}
			assertLosslessLexResult(t, file.Bytes, got)
			limits.MaxWorkUnits--
			exhausted := lex(file, source.NewBudget(limits, len(file.Bytes)))
			assertSingleExhaustion(t, exhausted.Diagnostics)
			assertLosslessLexResult(t, file.Bytes, exhausted)
			for _, token := range exhausted.Tokens {
				if token.Cooked != "" {
					t.Fatal("exhausted integer published cooked data")
				}
			}
		}
	}
}

func BenchmarkCookDecimalInteger(b *testing.B) {
	for _, test := range []struct{ name, raw string }{
		{"typical", "1234567890"},
		{"separated", "000_123_456_789"},
		{"64KiB", strings.Repeat("9", 65536)},
	} {
		b.Run(test.name, func(b *testing.B) {
			budget := source.NewBudget(source.DefaultLimits(), len(test.raw))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, valid := cookNumber(test.raw, Integer, budget); !valid {
					b.Fatal("valid integer rejected")
				}
			}
		})
	}
}

func BenchmarkLexDecimalIntegers(b *testing.B) {
	file := source.NewFile("main.fy", []byte(strings.Repeat("let n = 1234567890;\n", 512)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		got := Lex(file, source.NewBudget(source.DefaultLimits(), len(file.Bytes)))
		if len(got.Diagnostics) != 0 {
			b.Fatal(got.Diagnostics)
		}
	}
}
