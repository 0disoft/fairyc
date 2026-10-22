package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestFloatingFieldEmission(t *testing.T) {
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
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range []string{"#include <float.h>", "float single;", "double wide;", "offsetof(struct Packet, wide) == UINT64_C(8)", "FLT_MANT_DIG == 24", "DBL_MANT_DIG == 53", "fairyc_fixture_float_consume_v0(struct Packet"} {
			if !strings.Contains(unit.Source, fragment) {
				t.Fatalf("missing %q", fragment)
			}
		}
	}
}

func TestFloatingEmissionBoundaries(t *testing.T) {
	for name, text := range map[string]string{
		"header-macro":  strings.ReplaceAll(floatingTransportSource[len("module main;"):], "single", "FLT_MANT_DIG"),
		"private-field": `struct P{f32 x;} unsafe extern C fn P produce(); fn i32 main(){let P p=unsafe{produce()};return 0;}`,
		"bare-result":   `unsafe extern C fn f32 produce(); fn i32 main(){let f32 x=unsafe{produce()};return 0;}`,
		"bare-argument": `@repr(C) struct P{f32 x;} unsafe extern C fn P produce(); unsafe extern C fn i32 consume(f32 x); fn i32 main(){let P p=unsafe{produce()};return unsafe{consume(p.x)};}`,
	} {
		t.Run(name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+text)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile()); err == nil {
				t.Fatal("unadmitted floating transport emitted")
			}
		})
	}
}
