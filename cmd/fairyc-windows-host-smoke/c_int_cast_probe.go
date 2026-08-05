package main

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const cIntCastFairySource = `module main;

@link_name("fairyc_c_int_source_v0")
unsafe extern C fn c::int source() ;

@link_name("fairyc_c_int_sink_v0")
unsafe extern C fn i32 sink(c::int value) ;

fn i32 main() {
    let c::int input = unsafe { source() };
    let i32 value = input as i32;
    let c::int output = value as c::int;
    return unsafe { sink(output) };
}
`

const cIntCastCompanionSource = `#include <stdint.h>

int fairyc_c_int_source_v0(void) {
    return 42;
}

int32_t fairyc_c_int_sink_v0(int value) {
    return (int32_t)value;
}
`

func runCIntCastProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(cIntCastFairySource)), source.DefaultLimits())
	if err != nil {
		return 0, fmt.Errorf("parse generated-Fairy c::int cast source: %w", err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return 0, fmt.Errorf("generated-Fairy c::int cast source is not parser-valid: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil {
		return 0, fmt.Errorf("compile generated-Fairy c::int cast source: %w", err)
	}
	semanticResult := checked.Semantic
	if semanticResult.CompletedThrough != semantic.PhaseFlow || semanticResult.FailedPhase != semantic.PhaseNone ||
		len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return 0, fmt.Errorf("generated-Fairy c::int cast source did not complete Flow: phase=%q failed=%q diagnostics=%+v unsupported=%+v", semanticResult.CompletedThrough, semanticResult.FailedPhase, semanticResult.Diagnostics, semanticResult.Unsupported)
	}
	if !checked.Verified {
		return 0, fmt.Errorf("generated-Fairy c::int cast source did not produce verified MIR")
	}
	caller, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, fmt.Errorf("emit generated-Fairy Windows c::int cast C: %w", err)
	}
	for _, required := range []string{
		"extern int fairyc_c_int_source_v0(void);",
		"extern int32_t fairyc_c_int_sink_v0(int fy_a_0);",
		"= (int32_t)(fy_l_",
		"= (int)(fy_l_",
	} {
		if !strings.Contains(caller.Source, required) {
			return 0, fmt.Errorf("generated-Fairy Windows c::int cast C lost required fragment %q", required)
		}
	}
	return runWindowsSeparateTranslationUnitProbe(
		toolchain,
		"generated-Fairy c::int cast",
		caller.Source,
		cIntCastCompanionSource,
		42,
		cdriver.C11ConsumerMode,
	)
}
