package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

const floatingTransportSource = `module main;
@repr(C) struct Packet{f32 single;f64 wide;u32 marker;}
@link_name("fairyc_fixture_float_make_v0") unsafe extern C fn Packet produce();
@link_name("fairyc_fixture_float_consume_v0") unsafe extern C fn i32 consume(Packet value);
fn f32 copy_single(f32 value){return value;}
fn f64 copy_wide(f64 value){return value;}
fn i32 main(){
    let Packet p=unsafe{produce()};
    let f32 single=copy_single(p.single);
    let f64 wide=copy_wide(p.wide);
    let Packet copy=Packet{.single=single,.wide=wide,.marker=p.marker};
    return unsafe{consume(copy)};
}`

func TestFloatingTransportMIRRoundTrip(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", floatingTransportSource)})
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
	if program.StructTypes[0].Fields[0].Type != hir.TypeF32 || program.StructTypes[0].Fields[1].Type != hir.TypeF64 {
		t.Fatal("floating field identities lost")
	}
	if _, err = interpreter.Execute(program, "main::main"); err == nil {
		t.Fatal("interpreter fabricated a foreign float producer")
	}
}
