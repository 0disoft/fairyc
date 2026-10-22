package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

const integerTransportSource = `module main;
@repr(C) struct Packet{i8 tiny;i16 small;u16 word;u64 wide;isize offset;}
@link_name("fairyc_fixture_integer_make_v0") unsafe extern C fn Packet produce();
@link_name("fairyc_fixture_integer_consume_v0") unsafe extern C fn i32 consume(Packet value);
fn i8 copy_tiny(i8 value){return value;}
fn i16 copy_small(i16 value){return value;}
fn u16 copy_word(u16 value){return value;}
fn u64 copy_wide(u64 value){return value;}
fn isize copy_offset(isize value){return value;}
fn i32 main(){
    let Packet p=unsafe{produce()};
    let Packet copy=Packet{.tiny=copy_tiny(p.tiny),.small=copy_small(p.small),
        .word=copy_word(p.word),.wide=copy_wide(p.wide),.offset=copy_offset(p.offset)};
    return unsafe{consume(copy)};
}`

func TestIntegerTransportMIRRoundTrip(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", integerTransportSource)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for i, typ := range []hir.Type{hir.TypeI8, hir.TypeI16, hir.TypeU16, hir.TypeU64, hir.TypeISize} {
		if program.StructTypes[0].Fields[i].Type != typ {
			t.Fatalf("integer field identity lost: %s", typ)
		}
	}
	if _, err = interpreter.Execute(program, "main::main"); err == nil {
		t.Fatal("interpreter fabricated a foreign integer producer")
	}
}
