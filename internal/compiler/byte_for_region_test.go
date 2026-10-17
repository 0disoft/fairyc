package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

const byteForRegionSource = `module std::alloc;
pub resource Arena { i32 opaque; cleanup {} }
pub struct AllocError { u32 status; }
resource Token { i32 value; cleanup {} }
fn i32 main() {
    var Arena arena = Arena { .opaque = 0 };
    let ref u8[] view = b"AB"[..];
    for (let u8 byte in view) {
        let Token body = Token { .value = byte as i32 };
        region scratch in ref mut arena {
            let Token inner = Token { .value = byte as i32 };
            discard scratch.place(byte as i32);
            if (byte == 65) { continue; }
            break;
        }
    }
    return 42;
}`

func TestByteForRegionResetOnContinueAndBreak(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "std/alloc.fy", byteForRegionSource)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 0 {
		t.Fatalf("compile byte-for Region exits: %v diagnostics=%+v unsupported=%+v", err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	outcome, err := interpreter.Execute(result.MIR, "std::alloc::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("execute byte-for Region exits: %+v %v", outcome, err)
	}
	if len(outcome.RegionTrace) != 2 || len(outcome.DropTrace) != 5 {
		t.Fatalf("cleanup counts: regions=%+v drops=%+v", outcome.RegionTrace, outcome.DropTrace)
	}
	for index, expected := range []struct {
		exit  string
		drops int
	}{{"continue", 1}, {"break", 3}} {
		event := outcome.RegionTrace[index]
		if event.ExitKind != expected.exit || event.Placements != 1 || event.DropCountBeforeReset != expected.drops {
			t.Fatalf("Region reset %d: %+v", index, event)
		}
	}
	for index, expected := range []struct{ resourceType, binding string }{
		{"std::alloc::Token", "inner"}, {"std::alloc::Token", "body"},
		{"std::alloc::Token", "inner"}, {"std::alloc::Token", "body"},
		{"std::alloc::Arena", "arena"},
	} {
		drop := outcome.DropTrace[index]
		start := strings.Index(byteForRegionSource, expected.binding+" =")
		if start < 0 || drop.ResourceType.String() != expected.resourceType || drop.Binding.Source != "std/alloc.fy" || drop.Binding.StartByte != start {
			t.Fatalf("drop %d: %+v", index, outcome.DropTrace[index])
		}
	}
}

func TestByteForRegionResetOnReturnAndPropagation(t *testing.T) {
	const source = `module std::alloc;
pub resource Arena { i32 opaque; cleanup {} }
pub struct AllocError { u32 status; }
resource Token { i32 value; cleanup {} }
fn Result<i32, i32> step(u8 byte) { return .err(byte as i32); }
fn Result<i32, i32> route() {
    var Arena arena = Arena { .opaque = 0 };
    let ref u8[] view = b"B"[..];
    for (let u8 byte in view) {
        let Token body = Token { .value = byte as i32 };
        region scratch in ref mut arena {
            let Token inner = Token { .value = byte as i32 };
            discard scratch.place(byte as i32);
            EXIT
        }
    }
    return .ok(0);
}
fn i32 main() {
    return switch (route()) {
        case .ok(value): { value }
        case .err(code): { code - 24 }
    };
}`
	for _, test := range []struct {
		name, exit, exitKind string
	}{
		{"return", "return .ok(42);", "return"},
		{"propagate", "let i32 value = step(byte)?;", "propagate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := strings.Replace(source, "EXIT", test.exit, 1)
			file := parseCompilerFile(t, hir.DefaultPackage, "std/alloc.fy", input)
			result, err := compiler.CheckExplicitFiles([]semantic.File{file})
			if err != nil || !result.Verified || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 0 {
				t.Fatalf("compile byte-for Region %s: %v diagnostics=%+v unsupported=%+v", test.exitKind, err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
			}
			outcome, err := interpreter.Execute(result.MIR, "std::alloc::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
				t.Fatalf("execute byte-for Region %s: %+v %v", test.exitKind, outcome, err)
			}
			if len(outcome.RegionTrace) != 1 || len(outcome.DropTrace) != 3 {
				t.Fatalf("cleanup counts: regions=%+v drops=%+v", outcome.RegionTrace, outcome.DropTrace)
			}
			event := outcome.RegionTrace[0]
			if event.ExitKind != test.exitKind || event.Placements != 1 || event.DropCountBeforeReset != 1 {
				t.Fatalf("Region reset: %+v", event)
			}
			for index, binding := range []string{"inner", "body", "arena"} {
				drop := outcome.DropTrace[index]
				start := strings.Index(input, binding+" =")
				if drop.Binding.Source != "std/alloc.fy" || drop.Binding.StartByte != start {
					t.Fatalf("drop %d: %+v", index, drop)
				}
			}
		})
	}
}
