package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

const nestedFieldStoreProgram = `module main;
struct Span {i64 start;usize end;bool ready;u8 byte;u32 word;i32 n;}
struct Position {Span span;}
struct Token {Position position;i32 calls;}
fn i64 next(ref mut Token s){s.calls+=1;s.position.span.start=99;return 2;}
fn void edit(ref mut Token s){s.position.span.start+=next(s);s.position.span.end=9;s.position.span.ready=true;}
fn i32 main(){
 var Token s=Token{.position=Position{.span=Span{.start=3,.end=7,.ready=false,.byte=2,.word=10,.n=10}},.calls=0};
 var Token copy=s;edit(ref mut s);
 assert(s.position.span.start==5 && s.calls==1 && copy.position.span.start==3 && copy.calls==0);
 assert(s.position.span.end==9 && s.position.span.ready && !copy.position.span.ready);
 ((s).position).span.byte+=2;s.position.span.byte*=3;
 s.position.span.word<<=2;s.position.span.word/=4;
 s.position.span.n-=2;s.position.span.n*=3;s.position.span.n/=4;s.position.span.n%=5;
 s.position.span.n<<=3;s.position.span.n>>=1;s.position.span.n|=3;s.position.span.n^=1;s.position.span.n&=6;
 assert(s.position.span.byte==12 && s.position.span.word==10 && s.position.span.n==6);
 return 42;
}`

func TestNestedFieldStoreRoundTrip(t *testing.T) {
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", nestedFieldStoreProgram)})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	encoded, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	stores := 0
	for _, fn := range decoded.Functions {
		for _, b := range fn.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationFieldStore && len(op.Fields) > 0 {
					stores++
					if !strings.Contains(strings.Join(op.Fields, "."), "position.span") {
						t.Fatalf("lost path: %+v", op)
					}
				}
			}
		}
	}
	if stores < 10 {
		t.Fatalf("lost stores: %d", stores)
	}
	outcome, err := interpreter.Execute(decoded, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("roundtrip: %+v %v", outcome, err)
	}
}

func TestNestedFieldStoreRejections(t *testing.T) {
	const prelude = `module main;struct Span {i64 start;i64 end;} struct Token {Span position;}
 fn Token make(){return Token{.position=Span{.start=0,.end=7}};}
 `
	for name, source := range map[string]string{
		"immutable":          "fn i32 main(){let Token s=make();s.position.start=1;return 0;}",
		"shared":             "fn void bad(ref Token s){s.position.start=1;}",
		"loan":               "fn i32 main(){var Token s=make();let ref Token view=ref s;s.position.start=1;assert(view.position.start==0);return 0;}",
		"field-loan":         "fn i32 main(){var Token s=make();let ref i64 view=ref s.position.end;s.position.start=1;assert(*view==7);return 0;}",
		"mutable-loan":       "fn i32 main(){var Token s=make();let ref mut Token view=ref mut s;s.position.start=1;view.position.end=9;return 0;}",
		"rhs-owner-write":    "fn i32 main(){var Token s=make();let ref mut Token selected=ref mut s;selected.position.start+={s.position.start=9;2};return 0;}",
		"rhs-owner-read":     "fn i32 main(){var Token s=make();let ref mut Token selected=ref mut s;selected.position.start+={s.position.start};return 0;}",
		"temporary":          "fn i32 main(){make().position.start=1;return 0;}",
		"aggregate-compound": "fn i32 main(){var Token s=make();s.position+=Span{.start=1,.end=2};return 0;}",
		"type-mismatch":      "fn i32 main(){var Token s=make();s.position.start=true;return 0;}",
		"borrow-field":       "struct Holder{ref mut Token inner;} fn void bad(ref mut Holder h){h.inner.position.start=1;}",
		"resource-root":      "resource Holder{Token inner;cleanup {}} fn void bad(ref mut Holder h){h.inner.position.start=1;}",
	} {
		t.Run(name, func(t *testing.T) {
			result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", prelude+source)})
			if err != nil || result.Verified || len(result.Semantic.Diagnostics)+len(result.Semantic.Unsupported) == 0 {
				t.Fatalf("accepted: %v %+v", err, result.Semantic)
			}
		})
	}
}

func TestNestedFieldStoreReferenceSelection(t *testing.T) {
	const source = `module main;struct Span{i64 start;}struct Token{Span position;}
 fn i32 main(){var Token first=Token{.position=Span{.start=3}};var Token second=Token{.position=Span{.start=7}};var ref mut Token selected=ref mut first;
 selected.position.start+={selected=ref mut second;2};
 assert(first.position.start==5 && selected.position.start==7);selected.position.start=9;assert(second.position.start==9);return 42;}`
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	outcome, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("selection: %+v %v", outcome, err)
	}
}
