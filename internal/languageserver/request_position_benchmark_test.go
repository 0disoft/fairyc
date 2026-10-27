package languageserver

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func BenchmarkCompletionRequestPosition(b *testing.B) {
	for _, size := range []int{16 << 10, 1 << 20} {
		b.Run(fmt.Sprintf("bytes=%d", size), func(b *testing.B) {
			padding := strings.Repeat("// source padding for completion\n", size/33)
			text := []byte("module main;\n" + padding + "fn i32 main() { return 0; }\n")
			d := &document{URI: "file:///main.fy", Path: "main.fy", Text: text, Version: 1}
			s := &server{initialized: true, out: io.Discard, docs: map[string]*document{d.URI: d}}
			d.View = s.analyze(d, "")
			if !d.View.clean {
				b.Fatal("benchmark document did not analyze cleanly")
			}
			offset := strings.LastIndex(string(text), "return") + 3
			params, err := json.Marshal(documentParams{TextDocument: documentID{URI: d.URI}, Position: bytePosition(text, offset)})
			if err != nil {
				b.Fatal(err)
			}
			request := message{ID: json.RawMessage("1"), Method: "textDocument/completion", Params: params}
			if err := s.handle(request); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if err := s.handle(request); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
