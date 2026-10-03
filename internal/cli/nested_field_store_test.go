package cli

import "testing"

const nestedFieldNativeSource = `module main;struct Span{i64 start;i64 end;usize size;bool ready;u8 byte;u32 word;i32 n;}struct Position{Span span;}struct Token{Position position;i32 calls;}
fn i64 next(ref mut Token s){s.calls+=1;s.position.span.start=99;return 2;}
fn void edit(ref mut Token s){s.position.span.start+=next(s);s.position.span.end=9;s.position.span.ready=true;}
fn i32 main(){var Token s=Token{.position=Position{.span=Span{.start=3,.end=7,.size=0,.ready=false,.byte=2,.word=10,.n=10}},.calls=0};var Token copy=s;edit(ref mut s);assert(s.position.span.start==5 && s.calls==1 && copy.position.span.start==3 && copy.calls==0);assert(s.position.span.end==9 && s.position.span.ready && !copy.position.span.ready);((s).position).span.byte+=2;s.position.span.byte*=3;s.position.span.word<<=2;s.position.span.word/=4;s.position.span.n-=2;s.position.span.n*=3;s.position.span.n/=4;s.position.span.n%=5;s.position.span.n<<=3;s.position.span.n>>=1;s.position.span.n|=3;s.position.span.n^=1;s.position.span.n&=6;assert(s.position.span.byte==12 && s.position.span.word==10 && s.position.span.n==6);return 0;}`

func TestNestedFieldStoreNative(t *testing.T) {
	cases := []struct{ name, source, trap string }{{"scalar-types-order-copy", nestedFieldNativeSource, ""}}
	cases = append(cases, struct{ name, source, trap string }{"reference-selection", `module main;struct Span{i64 start;}struct Token{Span position;}fn i32 main(){var Token first=Token{.position=Span{.start=3}};var Token second=Token{.position=Span{.start=7}};var ref mut Token selected=ref mut first;selected.position.start+={selected=ref mut second;2};assert(first.position.start==5 && second.position.start==7);selected.position.start=9;assert(second.position.start==9);return 0;}`, ""})
	cases = append(cases, struct{ name, source, trap string }{"owner-replacement", `module main;struct Span{i64 start;}struct Token{Span position;}fn i32 main(){var Token s=Token{.position=Span{.start=3}};s.position.start+={s=Token{.position=Span{.start=99}};2};assert(s.position.start==5);return 0;}`, ""})

	for _, item := range []struct{ name, typ, initial, action, trap string }{
		{"i64-overflow", "i64", "9223372036854775807", "+=1", "integer-overflow"},
		{"u8-overflow", "u8", "255", "+=1", "integer-overflow"},
		{"usize-underflow", "usize", "0", "-=1", "integer-overflow"},
		{"divide-zero", "i32", "42", "/=0", "division-by-zero"},
		{"signed-divide-overflow", "i32", "-2147483648", "/=-1", "signed-division-overflow"},
		{"bad-shift", "u32", "1", "<<=32", "invalid-shift"},
		{"negative-shift", "i32", "1", "<<=-1", "invalid-shift"},
		{"wide-shift", "i64", "1", "<<=64", "invalid-shift"},
	} {
		cases = append(cases, struct{ name, source, trap string }{item.name, "module main;struct Inner{" + item.typ + " value;}struct Outer{Inner inner;}fn i32 main(){var Outer s=Outer{.inner=Inner{.value=" + item.initial + "}};s.inner.value" + item.action + ";return 0;}", item.trap})
	}
	runI64ViewNativeCases(t, "nested-field-store", cases)
}

func TestNestedFieldStoreCache(t *testing.T) { checkStructVecViewCache(t, nestedFieldNativeSource) }
