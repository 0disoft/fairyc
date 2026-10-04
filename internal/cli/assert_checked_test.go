package cli

import "testing"

func TestAssertCheckedTrapNative(t *testing.T) {
	runI64ViewNativeCases(t, "assert-checked-trap", []struct{ name, source, trap string }{
		{"shared-call", `module main;struct Span{i64 start;}struct Token{Span position;}fn ref i64 number(ref Token t){return ref t.position.start;}fn void edit(ref mut Token t){let ref i64 before=number(t);assert(*before==9223372036854775807);t.position.start+=1;}fn i32 main(){var Token t=Token{.position=Span{.start=9223372036854775807}};edit(ref mut t);return 0;}`, "integer-overflow"},

		{"shared-field", `module main;struct Token{i64 value;}fn void edit(ref mut Token t){let ref i64 n=ref t.value;assert(*n==9223372036854775807);t.value+=1;}fn i32 main(){var Token t=Token{.value=9223372036854775807};edit(ref mut t);return 0;}`, "integer-overflow"},
		{"assertion-first", `module main;struct Token{i64 value;}fn void edit(ref mut Token t){let ref i64 n=ref t.value;assert(*n==9223372036854775807);t.value+=1;}fn i32 main(){var Token t=Token{.value=41};edit(ref mut t);return 0;}`, "assertion-failed"},
		{"scalar-rebind", `module main;fn void edit(i64 input){var i64 n=input;assert(n==9223372036854775807);n=40;let ref mut i64 slot=ref mut n;*slot=*slot+2;assert(n==42);}fn i32 main(){edit(9223372036854775807);return 0;}`, ""},
		{"unknown-call", `module main;struct Token{i64 value;}fn void reset(ref mut Token t){t.value=40;}fn void edit(ref mut Token t){assert(t.value==9223372036854775807);reset(t);t.value+=2;assert(t.value==42);}fn i32 main(){var Token t=Token{.value=9223372036854775807};edit(ref mut t);return 0;}`, ""},
		{"field-write", `module main;struct Token{i64 value;}fn void edit(ref mut Token t){assert(t.value==9223372036854775807);t.value=40;t.value+=2;assert(t.value==42);}fn i32 main(){var Token t=Token{.value=9223372036854775807};edit(ref mut t);return 0;}`, ""},
	})
}
