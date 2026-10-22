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

func TestIntegerArrayFieldEmission(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", integerArrayTransportSource)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, checked.Semantic)
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
		for _, fragment := range []string{"int8_t tiny[UINT64_C(3)];", "int16_t small[UINT64_C(3)];", "uint16_t word[UINT64_C(3)];", "uint64_t wide[UINT64_C(3)];", "intptr_t offset[UINT64_C(3)];", "sizeof(struct Packet) == UINT64_C(64)", "offsetof(struct Packet, wide) == UINT64_C(16)", "offsetof(struct Packet, offset) == UINT64_C(40)", "sizeof(intptr_t) == UINT64_C(8)", "_Alignof(int16_t) == UINT64_C(2)", "fy_array_i"} {
			if !strings.Contains(unit.Source, fragment) {
				t.Fatalf("missing %q", fragment)
			}
		}
	}
}

func TestIntegerArrayEmissionBoundaries(t *testing.T) {
	for _, typ := range []string{"i8", "i16", "u16", "u64", "isize"} {
		for name, body := range map[string]string{
			"foreign-result":   `unsafe extern C fn %[1]s[3] produce(); fn i32 main(){let %[1]s[3] a=unsafe{produce()};return 0;}`,
			"foreign-argument": `@repr(C) struct P{%[1]s[3] values;} unsafe extern C fn P produce(); unsafe extern C fn i32 consume(%[1]s[3] a); fn i32 main(){let P p=unsafe{produce()};return unsafe{consume(p.values)};}`,
			"option":           `fn Option<%[1]s[3]> bad(%[1]s[3] a){return Some(a);} fn i32 main(){return 0;}`,
			"result":           `fn Result<%[1]s[3],i32> bad(%[1]s[3] a){return Ok(a);} fn i32 main(){return 0;}`,
			"enum":             `enum E{V{%[1]s[3] values;};} fn i32 main(){return 0;}`,
			"resource":         `resource R{%[1]s[3] values;cleanup{}} fn i32 main(){return 0;}`,
		} {
			t.Run(typ+"/"+name, func(t *testing.T) {
				checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+fmt.Sprintf(body, typ))})
				if err != nil {
					t.Fatal(err)
				}
				if !checked.Verified {
					return
				}
				for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					if _, err := cbackend.Emit(checked.MIR, "main::main", profile); err == nil {
						t.Fatal("unadmitted array transport emitted")
					}
				}
			})
		}
		for _, length := range []int{1, 4096} {
			t.Run(fmt.Sprintf("%s/length-%d", typ, length), func(t *testing.T) {
				src := fmt.Sprintf("module main; @repr(C) struct P{%s[%d] values;} unsafe extern C fn P produce(); fn i32 main(){let P p=unsafe{produce()};return 0;}", typ, length)
				checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", src)})
				if err != nil || !checked.Verified {
					t.Fatalf("check: %v %+v", err, checked.Semantic)
				}
				for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					if _, err := cbackend.Emit(checked.MIR, "main::main", profile); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
