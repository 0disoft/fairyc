package languageserver

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkCompletionUnchangedDocument(b *testing.B) {
	for _, size := range []int{16 << 10, 1 << 20} {
		b.Run(fmt.Sprintf("bytes=%d", size), func(b *testing.B) {
			padding := strings.Repeat("// source padding for completion\n", size/33)
			text := []byte("module main;\n" + padding + "fn i32 main() { return 0; }\n")
			d := &document{Path: "main.fy", Text: text, Version: 1}
			s := &server{}
			d.View = s.analyze(d, "")
			if !d.View.clean {
				b.Fatal("benchmark document did not analyze cleanly")
			}
			offset := strings.LastIndex(string(text), "return") + 3
			if len(s.completion(d, offset)) == 0 {
				b.Fatal("missing return completion")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if len(s.completion(d, offset)) == 0 {
					b.Fatal("missing return completion")
				}
			}
		})
	}
}
