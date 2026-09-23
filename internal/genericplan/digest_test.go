package genericplan

import (
	"bytes"
	"testing"
)

func TestDigestChunksMatchesConcatenatedText(t *testing.T) {
	t.Parallel()
	large := bytes.Repeat([]byte("module-name-\x00\xce\xbb"), 4096)
	for _, test := range []struct {
		name   string
		domain string
		chunks [][]byte
	}{
		{name: "empty", domain: "source-nominal-module-v0"},
		{name: "module", domain: "source-nominal-module-v0", chunks: [][]byte{large, []byte("\n"), []byte("TokenIndex")}},
		{name: "package", domain: "source-nominal-package-v0", chunks: [][]byte{[]byte("closed-graph"), []byte("\n"), large, []byte("\n"), []byte("ByteOffset")}},
		{name: "binary", domain: "source-nominal-closed-graph-v0", chunks: [][]byte{[]byte{0, 0xff, '\n'}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var joined []byte
			for _, chunk := range test.chunks {
				joined = append(joined, chunk...)
			}
			want := DigestText(test.domain, string(joined))
			if got := DigestChunks(test.domain, test.chunks...); got != want {
				t.Fatalf("DigestChunks(%q) = %s, want %s", test.name, got, want)
			}
		})
	}
}
