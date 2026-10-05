package compiler_test

import (
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"os"
	"testing"
)

func TestPlainStructFieldStoreRoundTrip(t *testing.T) {
	source, err := os.ReadFile("testdata/plain_struct_field_store.fy")
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if err != nil || !result.Verified {
		t.Fatalf("check: %v %+v", err, result.Semantic)
	}
	data, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(data, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	structTypes := map[hir.Type]bool{}
	for _, definition := range decoded.StructTypes {
		structTypes[definition.Type] = true
	}
	direct, nested := 0, 0
	for _, fn := range decoded.Functions {
		for _, block := range fn.Blocks {
			for _, op := range block.Operations {
				if op.Kind == mir.OperationFieldStore && structTypes[op.Type] {
					if len(op.Fields) == 0 {
						direct++
					} else {
						nested++
					}
				}
			}
		}
	}
	if direct < 3 || nested < 3 {
		t.Fatalf("missing aggregate stores direct=%d nested=%d", direct, nested)
	}
	for _, program := range []mir.Program{result.MIR, decoded} {
		outcome, err := interpreter.Execute(program, "main::main")
		if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
			t.Fatalf("execute: %+v %v", outcome, err)
		}
	}
}

func TestPlainStructFieldStoreRejects(t *testing.T) {
	const prelude = `module main;struct Span{i64 start;i64 end;}struct Position{Span span;}struct Token{Position position;}
 fn Token make(){return Token{.position=Position{.span=Span{.start=0,.end=7}}};}`
	for name, source := range map[string]string{
		"immutable":        `fn i32 main(){let Token s=make();s.position=Position{.span=Span{.start=1,.end=2}};return 0;}`,
		"shared":           `fn void bad(ref Token s,Position p){s.position=p;}`,
		"owner-loan":       `fn i32 main(){var Token s=make();let ref Token view=ref s;s.position=Position{.span=Span{.start=1,.end=2}};assert(view.position.span.start==0);return 0;}`,
		"field-loan":       `fn i32 main(){var Token s=make();let ref i64 view=ref s.position.span.end;s.position.span=Span{.start=1,.end=2};assert(*view==7);return 0;}`,
		"mutable-loan":     `fn i32 main(){var Token s=make();let ref mut Token view=ref mut s;s.position=Position{.span=Span{.start=1,.end=2}};view.position.span.start=9;return 0;}`,
		"rhs-owner-write":  `fn i32 main(){var Token s=make();let ref mut Token selected=ref mut s;selected.position={s=make();Position{.span=Span{.start=1,.end=2}}};return 0;}`,
		"rhs-owner-read":   `fn i32 main(){var Token s=make();let ref mut Token selected=ref mut s;selected.position={s.position};return 0;}`,
		"nominal-mismatch": `struct Other{i64 start;i64 end;}fn void bad(ref mut Token s){s.position.span=Other{.start=1,.end=2};}`,
		"compound":         `fn void bad(ref mut Token s,Position p){s.position+=p;}`,
		"temporary":        `fn void bad(Position p){make().position=p;}`,
		"immutable-array":  `fn void bad(Position p){let Token[1] a=[make()];a[0].position=p;}`,
		"resource-root":    `resource Owner{Position position;cleanup{}}fn void bad(ref mut Owner s,Position p){s.position=p;}`,
		"nonplain-root":    `struct Owner{Position position;Option<i32> other;}fn void bad(ref mut Owner s,Position p){s.position=p;}`,
		"nonplain-leaf":    `struct Inner{Option<i32> values;}struct Owner{Inner inner;}fn void bad(ref mut Owner s,Inner v){s.inner=v;}`,
	} {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", prelude+source)})
			if err != nil || r.Verified || len(r.Semantic.Diagnostics)+len(r.Semantic.Unsupported) == 0 {
				t.Fatalf("accepted: %v %+v", err, r.Semantic)
			}
		})
	}
}
