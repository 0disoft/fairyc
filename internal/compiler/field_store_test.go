package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestFieldStoreArtifactRoundTrip(t *testing.T) {
	const source = `module main; struct S { i32 x; }
fn i32 next(ref mut S s) { s.x = s.x + 1; return s.x; }
fn i32 main() { var S a = S { .x = 40 }; var S b = a;
b.x = next(ref mut a); assert(a.x == 41); assert(b.x == 41); b.x = b.x + 1; return b.x; }`
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %+v %v", result, err)
	}
	encoded, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(decoded, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("round trip: %+v %v", outcome, err)
	}
}

func TestOneWordStructFieldStoreArtifactRoundTrip(t *testing.T) {
	const source = `module main;
struct TokenIndex { usize raw; }
struct Cursor { TokenIndex index; }
fn void advance(ref mut Cursor cursor) { cursor.index = TokenIndex { .raw = cursor.index.raw + 1 }; }
fn i32 main() {
    var Cursor cursor = Cursor { .index = TokenIndex { .raw = 4 } };
    var Cursor copy = cursor;
    advance(ref mut copy);
    assert(cursor.index.raw == 4 && copy.index.raw == 5);
    return 42;
}`
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %+v %v", result, err)
	}
	encoded, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(decoded, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("one-word struct store round trip: %+v %v", outcome, err)
	}
}

func TestFieldStoreScalarDerefLowersWithoutTargetRead(t *testing.T) {
	const source = `module main;
fn void set(ref mut i32 slot) { *slot = 42; }
fn i32 main() { var i32 value = 1; set(ref mut value); return value; }`
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %+v %v", result, err)
	}
	stores := 0
	for _, function := range result.MIR.Functions {
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind == mir.OperationDerefStore {
					stores++
				}
				if operation.Kind == mir.OperationUnary && operation.Operator == "*" && function.Name == "main::set" {
					t.Fatal("store loaded the old value")
				}
			}
		}
	}
	if stores != 1 {
		t.Fatalf("want one dereference store, got %d", stores)
	}
	encoded, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(decoded, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("dereference store artifact round trip: %+v %v", outcome, err)
	}
}

func TestFieldStoreScalarDerefRejectsMutableFieldProjection(t *testing.T) {
	const source = `module main; struct State { i32 count; }
fn i32 main() {
    var State state = State { .count = 1 };
    let ref mut i32 slot = ref mut state.count;
    *slot = 42;
    return state.count;
}`
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verified {
		t.Fatal("mutable field projection must remain unsupported")
	}
}

func TestFieldStoreAdmissionBoundaries(t *testing.T) {
	for name, source := range map[string]string{
		"shared owner":         `struct S { i32 x; } fn void edit(ref S s) { s.x = 1; }`,
		"immutable owner":      `struct S { i32 x; } fn void edit() { let S s = S { .x = 0 }; s.x = 1; }`,
		"moved owner":          `resource R { i32 x; cleanup {} } fn void consume(R r) {} fn void edit() { var R r = R { .x = 0 }; consume(r); r.x = 1; }`,
		"resource replacement": `resource R { i32 x; cleanup {} } resource H { R r; cleanup {} } fn void edit() { var H h = H { .r = R { .x = 0 } }; h.r = R { .x = 1 }; }`,
		"nested path":          `struct S { i32 x; } struct H { S s; } fn void edit() { var H h = H { .s = S { .x = 0 } }; h.s.x = 1; }`,
		"boolean compound":     `struct S { bool x; } fn void edit(ref mut S s) { s.x += true; }`,
		"u8 overflow":          `struct S { u8 x; } fn void edit(ref mut S s) { s.x = 256; }`,
		"u32 overflow":         `struct S { u32 x; } fn void edit(ref mut S s) { s.x = 4294967296; }`,
		"unsigned mismatch":    `struct S { u8 x; } fn void edit(ref mut S s, u32 value) { s.x = value; }`,
	} {
		t.Run(name, func(t *testing.T) {
			result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main; "+source)})
			if err != nil || result.Verified || len(result.MIR.Functions) != 0 || len(result.Semantic.Diagnostics)+len(result.Semantic.Unsupported) == 0 {
				t.Fatalf("invalid field store: %+v %v", result, err)
			}
		})
	}
}

func TestUnsignedFieldStoreArtifactRoundTrip(t *testing.T) {
	const source = `module main;
struct S { u8 byte; u32 word; }
resource R { u8 byte; u32 word; cleanup {} }
fn void fill(ref mut S s) { s.byte = 255; s.word = 4294967295; }
fn i32 main() {
    var S original = S { .byte = 0, .word = 0 };
    var S copy = original; fill(ref mut copy);
    assert(copy.byte == 255 && copy.word == 4294967295);
    assert(original.byte == 0 && original.word == 0);
    var R owned = R { .byte = 0, .word = 0 };
    owned.byte = copy.byte; owned.word = copy.word;
    assert(owned.byte == 255 && owned.word == 4294967295);
    copy.byte = 0; copy.word = 0;
    assert(owned.byte == 255 && owned.word == 4294967295);
    return 42;
}`
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %+v %v", result, err)
	}
	encoded, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(decoded, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("unsigned round trip: %+v %v", outcome, err)
	}
}
