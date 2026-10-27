package languageserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

func TestCompletionContextPreservesLexerBoundaries(t *testing.T) {
	for _, text := range []string{
		`module main; fn i32 main() { return 0; }`,
		"// 주석 😀\n/// documentation\n/* outer /* nested */ end */ value",
		`"한😀"b"bytes"c"C string" "escaped\"quote"`,
		"\"unterminated\n/* unterminated",
		"",
	} {
		d := &document{Text: []byte(text)}
		scanned := lexer.Lex(source.NewFile("main.fy", d.Text), source.NewBudget(source.DefaultLimits(), len(d.Text)))
		for offset := 0; offset <= len(text); offset++ {
			want := false
			for _, token := range scanned.Tokens {
				if token.Span.StartByte < offset && offset < token.Span.EndByte &&
					(token.Kind == lexer.String || token.Kind == lexer.ByteString || token.Kind == lexer.CString ||
						token.Trivia == lexer.TriviaLineComment || token.Trivia == lexer.TriviaDocumentationLine || token.Trivia == lexer.TriviaBlockComment) {
					want = true
				}
			}
			if got := d.blocksCompletion(offset); got != want {
				t.Fatalf("%q at byte %d: blocked=%v, want %v", text, offset, got, want)
			}
		}
		cached := d.completionIndex
		d.blocksCompletion(0)
		if cached == nil || d.completionIndex != cached {
			t.Fatal("unchanged document did not reuse its index")
		}
	}
}

func TestCompletionContextInvalidatesBeforeCanceledAnalysis(t *testing.T) {
	uri := fileURI(filepath.Join(t.TempDir(), "main.fy"))
	s := &server{initialized: true, out: io.Discard, docs: map[string]*document{}}
	code := "module main; fn i32 main() { return 0; }"
	send(t, s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": code}})
	d := s.docs[uri]
	if len(s.completion(d, strings.Index(code, "return")+3)) == 0 {
		t.Fatal("missing code completion")
	}
	change := func(version int, text string) message {
		t.Helper()
		params, err := json.Marshal(map[string]any{"textDocument": documentID{URI: uri, Version: version}, "contentChanges": []any{map[string]any{"text": text}}})
		if err != nil {
			t.Fatal(err)
		}
		return message{Method: "textDocument/didChange", Params: params}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.ctx = ctx
	comment := "// " + code
	if err := s.handle(change(2, comment)); !errors.Is(err, context.Canceled) {
		t.Fatalf("analysis cancellation: %v", err)
	}
	if d.completionIndex != nil {
		t.Fatal("accepted change retained the stale code index")
	}
	if len(s.completion(d, strings.Index(comment, "return")+3)) != 0 {
		t.Fatal("comment completion used the previous code snapshot")
	}
	s.ctx = context.Background()
	if err := s.handle(change(3, code)); err != nil {
		t.Fatal(err)
	}
	if len(s.completion(d, strings.Index(code, "return")+3)) == 0 {
		t.Fatal("removed comment still blocked code completion")
	}
	cached := d.completionIndex
	if err := s.handle(change(2, comment)); err != nil {
		t.Fatal(err)
	}
	if d.completionIndex != cached || string(d.Text) != code {
		t.Fatal("stale version changed the completion snapshot")
	}
	send(t, s, 0, "textDocument/didClose", map[string]any{"textDocument": documentID{URI: uri}})
	if s.docs[uri] != nil {
		t.Fatal("closed document retained its index")
	}
}
