package formatter

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestGreenHashesPreserveEncoding(t *testing.T) {
	for _, text := range []string{"", "identifier", strings.Repeat("가🙂\x00", 128), strings.Repeat("x", 4097)} {
		tree := &syntax.Tree{
			Source: []byte(text), Root: 1,
			Tokens: []lexer.Token{{Kind: lexer.String, Span: diagnostic.Span{EndByte: len(text)}, Cooked: text}, {Kind: lexer.EOF}},
			Nodes:  []syntax.Node{{Kind: syntax.KindLiteral, Text: text, Cooked: text}, {Kind: syntax.KindSource, Children: []syntax.NodeID{0}}},
		}
		// Independently encode the original length-prefixed format in memory.
		appendText := func(data []byte, value string) []byte {
			data = binary.LittleEndian.AppendUint64(data, uint64(len(value)))
			return append(data, value...)
		}
		shape := binary.LittleEndian.AppendUint64(nil, uint64(tree.Root))
		for _, node := range tree.Nodes {
			shape = appendText(shape, string(node.Kind))
			shape = appendText(shape, node.Text)
			shape = appendText(shape, node.Cooked)
			shape = binary.LittleEndian.AppendUint64(shape, uint64(len(node.Children)))
			for _, child := range node.Children {
				shape = binary.LittleEndian.AppendUint64(shape, uint64(child))
			}
		}
		wantShape := sha256.Sum256(shape)
		green, err := BuildGreen(tree)
		if err != nil {
			t.Fatal(err)
		}
		if green.ShapeIdentity() != wantShape {
			t.Fatalf("%d-byte text changed shape encoding", len(text))
		}
		identity := append([]byte("fairyc-green-tree/v0\x00"), green.sourceHash[:]...)
		identity = append(identity, wantShape[:]...)
		for _, token := range green.tokens {
			identity = append(identity, byte(token.Kind), byte(token.Trivia))
			identity = appendText(identity, token.Text)
			identity = appendText(identity, token.Cooked)
		}
		if green.Identity() != sha256.Sum256(identity) {
			t.Fatalf("%d-byte text changed green identity encoding", len(text))
		}
	}
}

func BenchmarkGreenStringHash(b *testing.B) {
	for _, test := range []struct {
		name, value string
		count       int
	}{{"512-identifiers", "identifier", 512}, {"1MiB", strings.Repeat("x", 1<<20), 1}} {
		b.Run(test.name, func(b *testing.B) {
			b.Run("previous", func(b *testing.B) {
				hash := sha256.New()
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					hash.Reset()
					for range test.count {
						previousHashString(hash, test.value)
					}
				}
			})
			b.Run("bounded", func(b *testing.B) {
				hash := sha256.New()
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					hash.Reset()
					var scratch [256]byte
					for range test.count {
						writeString(hash, test.value, &scratch)
					}
				}
			})
		})
	}
}

// Retain the old path only as a benchmark control for host timing variation.
func previousHashString(writer byteWriter, value string) {
	var length [8]byte
	binary.LittleEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = writer.Write(length[:])
	_, _ = writer.Write([]byte(value))
}
