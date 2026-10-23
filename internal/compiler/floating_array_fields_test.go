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

func TestFloatingArrayFieldEmission(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", floatingArrayTransportSource)})
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
		for _, fragment := range []string{"float single[UINT64_C(3)];", "double wide[UINT64_C(3)];", "sizeof(struct Packet) == UINT64_C(40)", "_Alignof(struct Packet) == UINT64_C(8)", "offsetof(struct Packet, wide) == UINT64_C(16)", "sizeof(float) == UINT64_C(4)", "sizeof(double) == UINT64_C(8)", "FLT_MANT_DIG == 24", "DBL_MANT_DIG == 53", "fy_array_i"} {
			if !strings.Contains(unit.Source, fragment) {
				t.Fatalf("missing %q", fragment)
			}
		}
	}
}

func TestFloatingArrayEmissionBoundaries(t *testing.T) {
	for _, typ := range []string{"f32", "f64"} {
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
