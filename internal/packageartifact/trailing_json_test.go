package packageartifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestTrailingJSONUsesValueOffsetAndExactWhitespace(t *testing.T) {
	for _, first := range []string{`{}`, ` {"text":"} [ \"","x":[0]} `} {
		for _, tail := range []string{"", " \t\r\n", "true", " \t{}", "\v", "\f", "\u00a0"} {
			encoded := []byte(first + tail)
			decoder := json.NewDecoder(bytes.NewReader(encoded))
			var value json.RawMessage
			if err := decoder.Decode(&value); err != nil {
				t.Fatal(err)
			}
			want := tail != "" && tail != " \t\r\n"
			if got := hasTrailingJSON(encoded, decoder); got != want {
				t.Fatalf("first %q, tail %q: got %v, want %v", first, tail, got, want)
			}
		}
	}
}

func TestArtifactDecodersRejectTrailingData(t *testing.T) {
	process := fixedProcessCodec(t, 1)
	transfers := *process
	transfers.storageTransfers = true
	decoders := map[string]func([]byte) error{
		"process":           func(data []byte) error { _, err := process.Decode(data); return err },
		"process-transfers": func(data []byte) error { _, err := transfers.Decode(data); return err },
		"body":              func(data []byte) error { _, err := DecodeStableBody(data, processArtifactLimit); return err },
		"context":           func(data []byte) error { _, err := DecodeContextDescriptor(data, processArtifactLimit); return err },
		"compilation": func(data []byte) error {
			_, err := DecodeStableCompilationArtifact(data, processArtifactLimit)
			return err
		},
	}
	for _, mode := range []bool{false, true} {
		decoders[fmt.Sprintf("stable-transfers=%v", mode)] = func(data []byte) error {
			_, err := decodeStableArtifact(data, processArtifactLimit, mode)
			return err
		}
	}
	tails := []string{"null", "true", "0", `"text"`, "{}", "[]", "[", "garbage", "\x00", "\u00a0"}
	largeTail := append([]byte("{}["), bytes.Repeat([]byte("0,"), 1<<19)...)
	largeTail = append(largeTail, '0', ']')
	for name, decode := range decoders {
		t.Run(name, func(t *testing.T) {
			for _, tail := range tails {
				// The empty envelope reaches the trailing check before schema validation.
				for _, space := range []string{"", " \t\r\n"} {
					if err := decode([]byte("{}" + space + tail)); err == nil || !strings.Contains(err.Error(), "trailing") {
						t.Fatalf("tail %q with space %q: %v", tail, space, err)
					}
				}
			}
			// A rejected suffix must not allocate an interface node for each number.
			allocations := testing.AllocsPerRun(1, func() {
				if err := decode(largeTail); err == nil || !strings.Contains(err.Error(), "trailing") {
					t.Fatalf("large trailing array: %v", err)
				}
			})
			if allocations > 128 {
				t.Fatalf("trailing array rejection allocated %.0f objects; limit 128", allocations)
			}
		})
	}
}

func BenchmarkRejectStableArtifactTrailingJSON(b *testing.B) {
	for _, elements := range []int{1 << 14, 1 << 19} {
		b.Run(fmt.Sprint(elements), func(b *testing.B) {
			data := append([]byte("{}["), bytes.Repeat([]byte("0,"), elements-1)...)
			data = append(data, '0', ']')
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := DecodeStableArtifact(data, uint64(len(data))); err == nil {
					b.Fatal("accepted trailing JSON array")
				}
			}
		})
	}
}
