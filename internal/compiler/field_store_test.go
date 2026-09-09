package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
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

func TestFieldStoreAdmissionBoundaries(t *testing.T) {
	for name, source := range map[string]string{
		"shared owner":         `struct S { i32 x; } fn void edit(ref S s) { s.x = 1; }`,
		"immutable owner":      `struct S { i32 x; } fn void edit() { let S s = S { .x = 0 }; s.x = 1; }`,
		"moved owner":          `resource R { i32 x; cleanup {} } fn void consume(R r) {} fn void edit() { var R r = R { .x = 0 }; consume(r); r.x = 1; }`,
		"resource replacement": `resource R { i32 x; cleanup {} } resource H { R r; cleanup {} } fn void edit() { var H h = H { .r = R { .x = 0 } }; h.r = R { .x = 1 }; }`,
		"nested path":          `struct S { i32 x; } struct H { S s; } fn void edit() { var H h = H { .s = S { .x = 0 } }; h.s.x = 1; }`,
		"compound":             `struct S { i32 x; } fn void edit(ref mut S s) { s.x += 1; }`,
		"u8":                   `struct S { u8 x; } fn void edit(ref mut S s) { s.x = 1; }`,
		"u32":                  `struct S { u32 x; } fn void edit(ref mut S s) { s.x = 1; }`,
	} {
		t.Run(name, func(t *testing.T) {
			result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main; "+source)})
			if err != nil || result.Verified || len(result.MIR.Functions) != 0 || len(result.Semantic.Diagnostics)+len(result.Semantic.Unsupported) == 0 {
				t.Fatalf("invalid field store: %+v %v", result, err)
			}
		})
	}
}
