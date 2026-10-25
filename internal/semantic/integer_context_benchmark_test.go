package semantic_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func BenchmarkContextualIntegerChain(b *testing.B) {
	for _, count := range []int{2000, 8000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			text := "module main; fn u32 sum(u32 peer) { return " + strings.Repeat("1 + ", count) + "peer; }"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
				b.Fatalf("parse integer chain: %v", err)
			}
			files := []semantic.File{{Path: "main.fy", Module: parsed.AST}}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				result := semantic.Analyze(files)
				if result.CompletedThrough != semantic.PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
					b.Fatalf("integer chain did not type-check: %+v %+v", result.Diagnostics, result.Unsupported)
				}
			}
		})
	}
}
