package jsonenvelope

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type testHeader struct {
	Version string `json:"version"`
}

type testBody struct {
	Text string `json:"text"`
}

func TestEnvelopeMatchesWholeStructAtByteBoundaries(t *testing.T) {
	header := testHeader{Version: "<v1>&"}
	value := testBody{Text: "λ\"\\\n\x00<>&\u2028\u2029"}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(body)
	want, err := json.Marshal(struct {
		testHeader
		Body testBody `json:"body"`
	}{header, value})
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []uint64{0, uint64(len(want) - 1), uint64(len(want)), uint64(len(want) + 1), ^uint64(0)} {
		got, err := Marshal(header, "body", body, limit)
		if limit < uint64(len(want)) {
			if !errors.Is(err, ErrTooLarge) || got != nil {
				t.Fatalf("limit %d: output=%d error=%v", limit, len(got), err)
			}
			continue
		}
		if err != nil || !bytes.Equal(got, want) || cap(got) != len(got) {
			t.Fatalf("limit %d: output=%s error=%v", limit, got, err)
		}
		got[0] = '!'
	}
	if !bytes.Equal(body, original) || header.Version != "<v1>&" {
		t.Fatal("encoder changed its inputs")
	}
	got, err := Marshal(struct{}{}, "quoted\"key", []byte("null"), 100)
	if err != nil || string(got) != `{"quoted\"key":null}` {
		t.Fatalf("empty header and escaped key: %s, %v", got, err)
	}
}

func TestEnvelopeRejectsInvalidHeaders(t *testing.T) {
	for _, header := range []any{nil, []int{1}, "object"} {
		if got, err := Marshal(header, "body", []byte("null"), 100); err == nil || got != nil {
			t.Fatalf("non-object header %T accepted", header)
		}
	}
	_, err := Marshal(struct{ Channel chan int }{}, "body", []byte("null"), 100)
	var unsupported *json.UnsupportedTypeError
	if !errors.As(err, &unsupported) {
		t.Fatalf("lost encoding error: %v", err)
	}
}

func BenchmarkEnvelopeSerialization(b *testing.B) {
	value := testBody{Text: strings.Repeat("<>&λ", 1<<16)}
	header := testHeader{Version: "v1"}
	for _, reject := range []bool{false, true} {
		for _, reuse := range []bool{false, true} {
			name := "accepted/serialize-twice"
			if reject {
				name = "rejected/serialize-twice"
			}
			if reuse {
				name = strings.ReplaceAll(name, "serialize-twice", "reuse-body")
			}
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					body, err := json.Marshal(value)
					if err != nil {
						b.Fatal(err)
					}
					_ = sha256.Sum256(body)
					limit := uint64(len(body) + 100)
					if reject {
						limit = uint64(len(body) + 1)
					}
					var output []byte
					if reuse {
						output, err = Marshal(header, "body", body, limit)
					} else {
						output, err = json.Marshal(struct {
							testHeader
							Body testBody `json:"body"`
						}{header, value})
						if err == nil && uint64(len(output)) > limit {
							output, err = nil, ErrTooLarge
						}
					}
					if reject != errors.Is(err, ErrTooLarge) || (!reject && (err != nil || len(output) == 0)) {
						b.Fatalf("output=%d error=%v", len(output), err)
					}
				}
			})
		}
	}
}
