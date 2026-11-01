package compiler_test

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
	"runtime"
	"strings"
	"testing"
)

func broaderScalarProgram() (string, string) {
	var src, observer strings.Builder
	src.WriteString(`module main;fn T id<T>(T value){return value;}fn T relay<T>(T value){return id<T>(value);}`)
	var body strings.Builder
	for _, v := range []struct{ fy, c, value string }{{"i8", "int8_t", "INT8_MIN"}, {"i16", "int16_t", "INT16_MIN"}, {"isize", "intptr_t", "INT64_C(-2147483649)"}, {"u16", "uint16_t", "UINT16_MAX"}, {"u64", "uint64_t", "UINT64_MAX"}, {"f32", "float", "1.25f"}, {"f64", "double", "-3.5"}} {
		fmt.Fprintf(&src, "@repr(C) struct P_%s{%s value;}unsafe extern C fn P_%s seed_%s();unsafe extern C fn i32 observe_%s(P_%s value);", v.fy, v.fy, v.fy, v.fy, v.fy, v.fy)
		fmt.Fprintf(&body, "let P_%s input_%s=unsafe{seed_%s()};let %s output_%s=relay<%s>(input_%s.value);assert(unsafe{observe_%s(P_%s{.value=output_%s})}==1);", v.fy, v.fy, v.fy, v.fy, v.fy, v.fy, v.fy, v.fy, v.fy, v.fy)
		fmt.Fprintf(&observer, "typedef struct{%s value;} P_%s;P_%s seed_%s(void){P_%s p={%s};return p;}int32_t observe_%s(P_%s p){return p.value==%s;}\n", v.c, v.fy, v.fy, v.fy, v.fy, v.value, v.fy, v.fy, v.value)
	}
	fmt.Fprintf(&src, "fn i32 main(){%s return 42;}", body.String())
	return src.String(), observer.String()
}

func TestBroaderGenericsScalarTransport(t *testing.T) {
	text, extra := broaderScalarProgram()
	result, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
	if err != nil || !result.Verified {
		t.Fatalf("scalar expansion: %v %+v", err, result.Semantic)
	}
	runExpandedLanguageNative(t, result, "main::main", extra)
}

func runExpandedLanguageNative(t *testing.T, result compiler.Result, entry, extra string) {
	t.Helper()
	for _, p := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		if _, err := cbackend.Emit(result.MIR, entry, p); err != nil {
			t.Fatal(err)
		}
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	support := scalar + "\n" + extra
	if runtime.GOOS == "windows" {
		tc, err := cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
		unit, err := cbackend.Emit(result.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []cdriver.ConsumerMode{cdriver.C11ConsumerMode, cdriver.C17ConsumerMode} {
			for _, p := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(string(mode)+"/"+p.Name, func(t *testing.T) {
					a, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(tc, p, mode, unit.Source, support)
					if err != nil {
						t.Fatal(err)
					}
					out, err := cdriver.ExecuteWindowsMSVCABIProbe(a)
					if err != nil || out.ExitCode != 42 || out.Stdout != "" || out.Stderr != "" {
						t.Fatalf("execute: %+v %v", out, err)
					}
					t.Logf("MSVC %s: exit42 empty streams", tc.Version)
				})
			}
		}
	} else if runtime.GOOS == "linux" {
		unit, err := cbackend.Emit(result.MIR, entry, cbackend.Stage0Profile())
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"gcc", "clang"} {
			tc, err := cdriver.Resolve(target.X8664LinuxGNU, name)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(name+"/"+p.Name, func(t *testing.T) {
					a, err := cdriver.Build(tc, p, unit.Source, support)
					if err != nil {
						t.Fatal(err)
					}
					out, err := cdriver.ExecuteLinuxTarget(a)
					if err != nil || out.ExitCode != 42 || out.Stdout != "" || out.Stderr != "" {
						t.Fatalf("execute: %+v %v", out, err)
					}
				})
			}
		}
	} else {
		t.Skip("native expansion requires Windows or Linux amd64")
	}
}
