package languageserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequestPositionIndexMatchesLinearAdmission(t *testing.T) {
	for name, text := range map[string][]byte{
		"empty":        nil,
		"unicode":      []byte(strings.Repeat("한😀x\r\nsecond\n", 180)),
		"long-line":    []byte(strings.Repeat("x", 1022) + "😀" + strings.Repeat("한x", 600)),
		"lone-cr":      []byte(strings.Repeat("x", 1023) + "\r" + strings.Repeat("ignored", 500) + "\n한😀\r\nz"),
		"empty-lines":  []byte(strings.Repeat("\r\r\n\n", 600)),
		"invalid-utf8": append([]byte(strings.Repeat("x", 1023)+"\n"), 0xff, 0xc0, 0x80, 0xf0, 0x9f),
	} {
		t.Run(name, func(t *testing.T) {
			index := newPositionIndex(text)
			check := func(p Position) {
				t.Helper()
				got, ok := index.positionByte(p)
				want, valid := positionByte(text, p)
				if ok != valid || valid && got != want {
					t.Fatalf("position %+v: got %d/%t, want %d/%t", p, got, ok, want, valid)
				}
			}
			check(Position{-1, 0})
			check(Position{0, -1})
			check(Position{})
			if index.scanned != 0 {
				t.Fatal("invalid or byte-zero request eagerly scanned the source")
			}
			for offset := 0; offset <= len(text); offset++ {
				p := bytePosition(text, offset)
				check(p)
				check(Position{p.Line, p.Character + 1})
			}
			// Include checkpoints past lone CR and partial UTF-8 boundaries, then
			// query earlier lines. Invalid UTF-16 surrogate midpoints stay invalid.
			index.bytePosition(len(text))
			for offset := len(text); offset >= 0; offset-- {
				check(bytePosition(text, offset))
			}
			check(Position{index.position.Line + 1, 0})
			if len(index.checkpoints) > len(text)/positionCheckpointBytes+1 {
				t.Fatal("request index exceeded sparse checkpoint bound")
			}
		})
	}
}

func TestRequestPositionIndexInvalidatesBeforeCanceledAnalysis(t *testing.T) {
	uri := fileURI(filepath.Join(t.TempDir(), "main.fy"))
	s := &server{initialized: true, out: io.Discard, docs: map[string]*document{}}
	code := "module main;\n" + strings.Repeat("// padding\n", 256) + "fn i32 main(){return 0;}"
	send(t, s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": code}})
	d := s.docs[uri]
	if d == nil {
		t.Fatal("test document was not opened")
	}
	if got, ok := d.positionByte(bytePosition(d.Text, len(code))); !ok || got != len(code) {
		t.Fatal("initial request position mismatch")
	}
	previous := d.positionIndex
	change := func(version int, text string, partial bool) message {
		t.Helper()
		item := map[string]any{"text": text}
		if partial {
			item["range"] = Range{}
		}
		params, err := json.Marshal(map[string]any{"textDocument": documentID{URI: uri, Version: version}, "contentChanges": []any{item}})
		if err != nil {
			t.Fatal(err)
		}
		return message{Method: "textDocument/didChange", Params: params}
	}
	for _, rejected := range []message{change(1, "stale", false), change(2, "partial", true)} {
		if err := s.handle(rejected); err != nil || d.positionIndex != previous || string(d.Text) != code {
			t.Fatalf("rejected change altered position snapshot: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.ctx = ctx
	updated := "module main;\n// 한😀\nfn i32 main(){return 0;}"
	if err := s.handle(change(2, updated, false)); !errors.Is(err, context.Canceled) || d.positionIndex != nil {
		t.Fatalf("accepted change retained stale index after cancellation: %v", err)
	}
	want := strings.Index(updated, "\nfn")
	if got, ok := d.positionByte(bytePosition(d.Text, want)); !ok || got != want || d.positionIndex == previous {
		t.Fatal("new source positions reused stale analysis or document index")
	}
}

func TestRequestPositionIndexExtendsOnlyAsFarAsNeeded(t *testing.T) {
	text := []byte("first\nsecond\n" + strings.Repeat("// padding\n", 100000))
	d := &document{Text: text}
	if got, ok := d.positionByte(Position{1, 3}); !ok || got != len("first\nsec") {
		t.Fatal("incorrect early-line request position")
	}
	index := d.positionIndex
	if index.scanned > positionCheckpointBytes || len(index.checkpoints) != 2 {
		t.Fatal("early-line request indexed the full document")
	}
	scanned := index.scanned
	if _, ok := d.positionByte(Position{0, 2}); !ok || d.positionIndex != index || index.scanned != scanned {
		t.Fatal("backward request rebuilt or extended the snapshot index")
	}
}
