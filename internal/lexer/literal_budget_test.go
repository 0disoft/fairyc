package lexer

import (
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"testing"
)

func TestDecimalExpansionChargesSharedBudget(t *testing.T) {
	for _, input := range []string{"1e100", "1e-100", "0e100", "0e-100"} {
		t.Run(input, func(t *testing.T) {
			limits := source.DefaultLimits()
			limits.MaxWorkUnits = 16
			result := Lex(source.NewFile("input.fy", []byte(input)), source.NewBudget(limits, len(input)))
			assertSingleExhaustion(t, result.Diagnostics)
			assertLosslessLexResult(t, []byte(input), result)
			for _, token := range result.Tokens {
				if token.Cooked != "" {
					t.Fatal("exhausted expansion published cooked data")
				}
			}
		})
	}
}

func TestDecimalExpansionExtremeExponentFailsBeforeAllocation(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, input := range []string{"1e" + strconv.Itoa(maxInt), "1e" + strconv.Itoa(-maxInt-1), "1.1e" + strconv.Itoa(-maxInt), "1e100 trailing"} {
		limits := source.DefaultLimits()
		limits.MaxWorkUnits = 64
		result := Lex(source.NewFile("input.fy", []byte(input)), source.NewBudget(limits, len(input)))
		assertSingleExhaustion(t, result.Diagnostics)
		assertLosslessLexResult(t, []byte(input), result)
	}
}

func TestDecimalExpansionExactWorkBoundary(t *testing.T) {
	for _, test := range []struct {
		input, cooked string
		work          int
	}{
		{"1e3", "1000", 7}, {"1e-3", "0.001", 9},
		{"1.25", "1.25", 8}, {"0e0", "0", 4},
		{"1.200", "1.2", 10},
	} {
		for _, enough := range []bool{false, true} {
			limits := source.DefaultLimits()
			limits.MaxWorkUnits = test.work
			if !enough {
				limits.MaxWorkUnits--
			}
			result := Lex(source.NewFile("input.fy", []byte(test.input)), source.NewBudget(limits, len(test.input)))
			assertLosslessLexResult(t, []byte(test.input), result)
			if !enough {
				assertSingleExhaustion(t, result.Diagnostics)
				continue
			}
			if len(result.Diagnostics) != 0 || len(result.Tokens) != 2 || result.Tokens[0].Cooked != test.cooked {
				t.Fatalf("%s boundary: %+v", test.input, result)
			}
		}
	}
}
