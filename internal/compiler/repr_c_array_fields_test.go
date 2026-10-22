package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestReprCArrayFieldsRoundTrip(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
@repr(C) struct Packet { u8 tag; u32[3] words; i64[2] signed_values; i32[2] codes; usize[2] sizes; u8[4] bytes; u8 end; }
@repr(C) struct Envelope { u8 head; Packet value; }
fn void replace(ref mut Packet packet) { packet.words = [9; 3]; }
fn i32 main() {
    var Packet p = Packet { .tag = 7, .words = [1,2,3], .signed_values = [4;2], .codes = [5;2], .sizes = [6;2], .bytes = [7;4], .end = 8 };
    let u32[3] copy = p.words;
    var Envelope e = Envelope { .head = 1, .value = p };
    e.value.words = [8;3];
    let ref Envelope view = ref e;
    let u32[3] observed = view.value.words;
    assert(observed[2] == 8);
    replace(ref mut p);
    assert(p.words[0] == 9 && copy[0] == 1 && e.value.words[0] == 8);
    assert(p.signed_values[1] == 4 && p.codes[1] == 5 && p.sizes[1] == 6 && p.bytes[3] == 7 && p.tag == 7 && p.end == 8);
    return 42;
}`)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v diagnostics=%+v unsupported=%+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	result, err := interpreter.Execute(program, "main::main")
	if err != nil || result.Kind != interpreter.OutcomeReturn || result.Value != 42 {
		t.Fatalf("interpreter: %+v %v", result, err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, required := range []string{"uint32_t words[UINT64_C(3)];", "int64_t signed_values[UINT64_C(2)];", "uint8_t bytes[UINT64_C(4)];", "fy_array_i"} {
			if !strings.Contains(unit.Source, required) {
				t.Fatalf("missing %q", required)
			}
		}
	}
}

func TestReprCArrayFieldBorrowMIRRejected(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
struct Packet { u8[2] bytes; }
fn i32 main() { let Packet p = Packet { .bytes = [1,2] }; let ref u8[] v = ref p.bytes; return if(v[0] == 1){42}else{0}; }
`)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, checked.Semantic)
	}
	checked.MIR.StructTypes[0].ReprC = true
	if len(mir.Verify(checked.MIR)) == 0 {
		t.Fatal("raw array field borrow accepted as a private wrapper reference")
	}
	if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile()); err == nil {
		t.Fatal("invalid borrow emitted")
	}
}
