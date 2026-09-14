package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// Positive witnesses for the production families in spec/grammar/stage0.ebnf.
// These supplement, rather than replace, tree-shape, invalid-input and budget tests.
func testParserStage0Grammar(t *testing.T) {
	cases := []struct{ name, body string }{
		{"module/import/path", `module main; import std::io as io; import a::b::c as other;`},
		{"attributes/items", `@a @pkg::b() @args(path::value,0xA,"text",) pub const i32 N=1; @check pub static_assert(N==1);`},
		{"state/fields", `@contract_uid("x") pub state {cell u8 counter; map pkg::Key ref u8[] values;}`},
		{"struct/parameters", `struct Empty{} @repr(C) pub struct Pair<T,U,>{pub T left; U right;} struct One<T>{T value;}`},
		{"resource/cleanup", `resource Empty{} resource Bare{cleanup{}} pub resource Owner<T>{pub T value; @a @b(1,"x") cleanup {burn value;}}`},
		{"enum/payload", `pub enum Item {Empty; Unit{}; Pair{i32 left;ref u8[] right;};} enum Void{}`},
		{"function/owner", `fn void plain(){} pub unsafe fn T Box::take<T,>(T value,){return value;} fn void Pair::clear(){}`},
		{"foreign/functions", `unsafe extern C fn void empty(); @link_name("other") pub unsafe extern C fn i32 foreign(i32 value,ref u8[] data,);`},
		{"type/prefix/postfix", `fn (ptr mut ref u8[])[cfg::N][0b10] f(ref mut pkg::Box<(ptr u8)[16],u8[]> a,ptr ptr u8 p,){let ref u8[] v=a;}`},
		{"type/function", `fn void f(unsafe extern C fn void() cb,Box<unsafe extern C fn ptr u8(ref u8[],i32,)> nested){}`},
		{"binding/return", `fn void f(){let x=1;var y=2;let i32 z=3;var ptr u8 p=null;return;} fn i32 g(){return x;}`},
		{"assignment/operators", `fn void f(){x=1;x+=2;x-=3;x*=4;x/=5;x%=6;x<<=1;x>>=1;x&=2;x^=3;x|=4;}`},
		{"operand/statements", `fn void f(){assert(a);defer close(a);discard f();burn value;f();}`},
		{"block/if/statements", `fn void f(){{} {1}; if(a){1}else if(b){2}else{3} if(a){} {x}.field; if(a){x}else{y}+z;}`},
		{"while/for/jumps", `fn void f(){while(a){break;continue;}for(;;){}for(let i=0;i<4;i+=1){}for(var i32 i=0;;i=i+1){}for(i=0;a;){} }`},
		{"region", `fn void f(){region r in pool {let x=1;} region s in (Pool{.size=2}) {discard s;}}`},
		{"precedence/operators", `fn void f(){return a||b&&c==d|e^f&g<<h+i*j/k%l-m>>n;}`},
		{"comparison/operators", `fn void f(){a==b;a!=b;a<b;a<=b;a>b;a>=b;}`},
		{"cast/prefix", `fn void f(){-x;+x;!x;~x;ref x;ref mut x;*p;x as i32 as u8;}`},
		{"postfix/calls", `fn void f(){pkg::f();pkg::f(a,b,)(c).field?;pkg::f<u8,Box<i32>,>(a);}`},
		{"postfix/index", `fn void f(){a[0];a[1..2];a[1..];a[..2];a[..];}`},
		{"primary/arrays", `fn void f(){[];[1];[1,2,];[1;0x10];[f();cfg::N];(a);a::b;}`},
		{"primary/records", `fn void f(){R{};pkg::R<u8>{.x=1,.y=f(),};.Empty;.Unit{};.Pair{.left=1,.right=2,};}`},
		{"primary/blocks", `fn void f(){let a={let x=1;x};let b=if(a){1}else{2};let c=unsafe{*p};unsafe{discard f();}}`},
		{"switch/patterns", `enum E{Empty;Pair{i32 left;i32 right;};} fn void f(){let x=.Pair{.left=1,.right=2};return switch(x){case .Empty:{}case .ok(value):{value}case .Pair{.left=a,.right=b,}:{a+b}case .Unit{}:{0}};}`},
		{"literals/numbers", `fn void f(){true;false;null;0;1_234;0xAb_CD;0b10_01;1.25;1e2;1E+2;1.2e-3;1_2.3_4e5_6;}`},
		{"literals/strings", `fn void f(){"";"한글😀";"\\\"\n\r\t\0\x41\u{1F600}";b"\xFF\0";c"text\x41";}`},
	}
	var p, main strings.Builder
	main.WriteString("fn i32 main(){")
	for index, c := range cases {
		input := c.body
		if !strings.HasPrefix(input, "module ") {
			input = "module main; /* 한글 */\r\n" + input
		}
		want := parserDeclarationRecords(t, "main.fy", []byte(input))
		if len(want) == 0 {
			t.Fatalf("empty production witness %s", c.name)
		}
		fmt.Fprintf(&p, "fn i32 grammar_%d(ref syntax::Parsed parsed){assert(syntax::problem_count(parsed)==0 && integrity::valid(parsed) && syntax::declaration_count(parsed)==%d && syntax::source_view(parsed).len==%d);", index, len(want)/5, len(input))
		for i := 0; i < len(want); i += 5 {
			fmt.Fprintf(&p, "{let d=syntax::declaration_at(parsed,%d);assert(d.kind==%d && d.start==%d && d.end==%d && d.name_start==%d && d.name_end==%d);}", i/5, want[i], want[i+1], want[i+2], want[i+3], want[i+4])
		}
		p.WriteString("return 0;}\n")
		fmt.Fprintf(&main, "{let r=switch(make(b%s)){case .ok(parsed):{grammar_%d(ref parsed)}case .err(_):{1}};assert(r==0);}", strconv.Quote(input), index)
		t.Logf("production witness %02d: %s", index, c.name)
	}
	main.WriteString("return 0;}")
	runParserProbe(t, p.String()+main.String())
}
