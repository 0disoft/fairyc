package languageserver

import (
	"strings"
	"testing"
)

func BenchmarkDiagnosticPositions1MiB256(b *testing.B) {
	text := []byte(strings.Repeat("// source padding for positions\n", (1<<20)/32))
	if len(text) != 1<<20 {
		b.Fatal("benchmark source must be exactly 1 MiB")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		view := &analysis{diagnostics: map[string][]lspDiagnostic{}}
		for n := range 256 {
			start := len(text) - 2 - (n*7)%1024
			view.add("main.fy", text, start, start+1, "E0007", "expected token")
		}
		if len(view.diagnostics["main.fy"]) != 256 {
			b.Fatal("missing diagnostic ranges")
		}
	}
}
