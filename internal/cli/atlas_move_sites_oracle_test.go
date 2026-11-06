package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

// Freeze actual Go MIR observations before deriving the Fairy collector. In
// particular, a consumed argument is observable before a later argument traps;
// scanning only completed call operations would silently lose that move.
type atlasMoveOracleSite struct{ function, context, operand string }
type atlasMoveOracleFixture struct {
	name, source string
	sites        []atlasMoveOracleSite
}

func atlasMoveSiteOracleCases() []atlasMoveOracleFixture {
	type site = atlasMoveOracleSite
	const resource = "module main;resource Token{i32 value;cleanup{}}\n"
	return []atlasMoveOracleFixture{
		{"copy", "module main;fn i32 main(){let i32 value=1;let copy=value;return copy;}", nil},
		{"discard", resource + "fn i32 main(){let token=Token{.value=1};discard token;return 0;}", nil},
		{"binding", resource + "fn i32 main(){let token=Token{.value=1};let other=token;return other.value;}", []site{{"main", "other=token;", "token"}}},
		{"return", resource + "fn Token forward(Token token){return token;}fn i32 main(){let token=Token{.value=1};let other=forward(token);return other.value;}", []site{{"forward", "return token;", "token"}, {"main", "forward(token)", "token"}}},
		{"call", resource + "fn i32 take(Token token){return token.value;}fn i32 main(){let token=Token{.value=1};return take(token);}", []site{{"main", "take(token)", "token"}}},
		{"borrow", resource + "fn i32 read(ref Token token){return token.value;}fn i32 main(){let token=Token{.value=1};return read(ref token);}", nil},
		{"temporary", resource + "fn i32 take(Token token){return token.value;}fn i32 main(){return take(Token{.value=1});}", nil},
		{"before-later-trap", resource + "fn i32 take(Token token,i32 n){return token.value+n;}fn i32 main(){let token=Token{.value=1};return take(token,{assert(false);0});}", []site{{"main", "take(token,", "token"}}},
		{"after-earlier-trap", resource + "fn i32 take(i32 n,Token token){return token.value+n;}fn i32 main(){let token=Token{.value=1};return take({assert(false);0},token);}", nil},
		{"resource-field", resource + "resource Wrap{Token inner;cleanup{}}fn i32 main(){let token=Token{.value=1};let wrapped=Wrap{.inner=token};return wrapped.inner.value;}", []site{{"main", ".inner=token}", "token"}}},
		{"option-payload", resource + "fn i32 main(){let token=Token{.value=1};let Option<Token> maybe=.some(token);discard maybe;return 0;}", []site{{"main", ".some(token)", "token"}}},
		{"result-payload", resource + "fn i32 main(){let token=Token{.value=1};let Result<Token,i32> result=.ok(token);discard result;return 0;}", []site{{"main", ".ok(token)", "token"}}},
		{"propagation", resource + "fn Result<i32,i32> work(){let Result<Token,i32> result=.ok(Token{.value=1});let token=result?;return .ok(token.value);}fn i32 main(){discard work();return 0;}", nil},
		{"switch", resource + "fn i32 main(){let Option<Token> maybe=.some(Token{.value=1});switch(maybe){case .some(value):{discard value;}case .none:{}};return 0;}", []site{{"main", "switch(maybe)", "maybe"}}},
	}
}

func TestAtlasMoveSitesOracle(t *testing.T) {
	for _, test := range atlasMoveSiteOracleCases() {
		t.Run(test.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, test.source)
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("move oracle source rejected: %d %s", code, &diagnostic)
			}
			seenMain, observations := false, 0
			for _, partition := range prepared.partitions {
				encoded, err := atlasmap.EncodeV3(partition, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				for _, summary := range decoded.Functions() {
					if summary.Function.Module != "main" {
						continue
					}
					seenMain = seenMain || summary.Function.Name == "main"
					var expected []semanticsnapshot.Span
					for _, s := range test.sites {
						if s.function != summary.Function.Name {
							continue
						}
						if strings.Count(test.source, s.context) != 1 || strings.Count(s.context, s.operand) != 1 {
							t.Fatalf("ambiguous original source selector: %+v", s)
						}
						start := strings.Index(test.source, s.context) + strings.Index(s.context, s.operand)
						expected = append(expected, semanticsnapshot.Span{PackageInstance: summary.Function.PackageInstance, Source: "main.fy", StartByte: start, EndByte: start + len(s.operand)})
					}
					if !slices.Equal(summary.MoveSites, expected) {
						t.Fatalf("%s move sites = %+v, want original source operands %+v", summary.Function.Name, summary.MoveSites, expected)
					}
					observations += len(summary.MoveSites)
				}
			}
			if !seenMain || observations != len(test.sites) {
				t.Fatalf("incomplete oracle: main=%v, moves=%d want %d", seenMain, observations, len(test.sites))
			}
		})
	}
}
