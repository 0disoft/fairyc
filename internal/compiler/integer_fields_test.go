package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestIntegerFieldEmission(t *testing.T) {
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
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range []string{"int8_t tiny;", "int16_t small;", "uint16_t word;", "uint64_t wide;", "intptr_t offset;",
			"offsetof(struct Packet, wide) == UINT64_C(8)", "offsetof(struct Packet, offset) == UINT64_C(16)",
			"sizeof(intptr_t) == UINT64_C(8)", "_Alignof(int16_t) == UINT64_C(2)", "((uint64_t)-1) > (uint64_t)0",
			"((int8_t)-1) < (int8_t)0", "fairyc_fixture_integer_consume_v0(struct Packet"} {
			if !strings.Contains(unit.Source, fragment) {
				t.Fatalf("missing %q", fragment)
			}
		}
	}
}

func TestIntegerTransportEmissionBoundaries(t *testing.T) {
	for _, typ := range []string{"i8", "i16", "u16", "u64", "isize"} {
		for name, text := range map[string]string{
			"private-field": `struct P{%[1]s x;} unsafe extern C fn P produce(); fn i32 main(){let P p=unsafe{produce()};return 0;}`,
			"bare-result":   `unsafe extern C fn %[1]s produce(); fn i32 main(){let %[1]s x=unsafe{produce()};return 0;}`,
			"bare-argument": `@repr(C) struct P{%[1]s x;} unsafe extern C fn P produce(); unsafe extern C fn i32 consume(%[1]s x); fn i32 main(){let P p=unsafe{produce()};return unsafe{consume(p.x)};}`,
			"raw-pointer":   `unsafe extern C fn i32 consume(ptr %[1]s x); fn i32 main(){return unsafe{consume(null)};}`,
		} {
			t.Run(typ+"/"+name, func(t *testing.T) {
				checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+fmt.Sprintf(text, typ))})
				if err != nil || !checked.Verified {
					t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
				}
				for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					if _, err := cbackend.Emit(checked.MIR, "main::main", profile); err == nil {
						t.Fatal("unadmitted integer transport emitted")
					}
				}
			})
		}
	}
	for _, name := range []string{"int8_t", "int16_t", "uint16_t", "intptr_t"} {
		t.Run("provider-name/"+name, func(t *testing.T) {
			text := fmt.Sprintf(`module main; @link_name(%q) unsafe extern C fn i32 bad(); fn i32 main(){return unsafe{bad()};}`, name)
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile()); err == nil {
				t.Fatal("C provider typedef name admitted as a foreign symbol")
			}
		})
	}
}

func TestIntegerTransportFieldStoresStayClosed(t *testing.T) {
	for _, typ := range []string{"i8", "i16", "u16", "u64", "isize"} {
		t.Run(typ, func(t *testing.T) {
			text := fmt.Sprintf(`module main; @repr(C) struct P{%[1]s x;}
fn void bad(ref mut P p,P q){p.x=q.x;} fn i32 main(){return 0;}`, typ)
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil || checked.Verified || len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 1 ||
				checked.Semantic.Unsupported[0].Feature != "field assignment without executable field storage" {
				t.Fatalf("want MIR field-store rejection, got: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
		})
	}
}
