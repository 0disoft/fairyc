package semantic_test

import "testing"

func TestImplicitMethodBorrowOrigins(t *testing.T) {
	const prefix = `module main;
resource Token{i32 value;cleanup{}}
fn ref Token Token::view(ref Token this){return this;}
fn ref mut Token Token::edit(ref mut Token this){return this;}
fn void Token::set(ref mut Token this,i32 n){this.value=n;}
`
	for _, test := range []struct{ name, source, code string }{
		{"shared returned", `fn i32 main(){var token=Token{.value=1};let view=token.view();assert(view.value==1);token.set(2);return token.value;}`, ""},
		{"mutable returned", `fn i32 main(){var token=Token{.value=1};let view=token.edit();view.value=2;return token.value;}`, ""},
		{"forwarded shared parameter", `fn ref Token forward(ref Token token){return token.view();}fn i32 main(){return 0;}`, ""},
		{"forwarded mutable parameter", `fn ref mut Token forward(ref mut Token token){return token.edit();}fn i32 main(){return 0;}`, ""},
		{"local cannot escape", `fn ref Token escape(){let token=Token{.value=1};return token.view();}fn i32 main(){return 0;}`, "E2102"},
		{"mutable local cannot escape", `fn ref mut Token escape(){var token=Token{.value=1};return token.edit();}fn i32 main(){return 0;}`, "E2102"},
		{"retained shared conflict", `fn i32 main(){var token=Token{.value=1};let view=token.view();token.set(2);return view.value;}`, "E2101"},
		{"retained mutable conflict", `fn i32 main(){var token=Token{.value=1};let view=token.edit();let copy=token.value;return view.value+copy;}`, "E2101"},
		{"immutable exclusive receiver", `fn i32 main(){let token=Token{.value=1};discard token.edit();return 0;}`, "E1008"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": prefix + test.source})
			if test.code == "" {
				assertCleanTypeResult(t, result)
				return
			}
			if len(result.Unsupported) != 0 || !hasDiagnosticCode(result, test.code) {
				t.Fatalf("want %s: diagnostics=%+v unsupported=%+v", test.code, result.Diagnostics, result.Unsupported)
			}
		})
	}
}
