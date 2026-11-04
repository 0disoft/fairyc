package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestCheckPreservesSemanticReportExhaustion(t *testing.T) {
	var source strings.Builder
	source.WriteString("module main;fn i32 main(){")
	for index := range 512 {
		fmt.Fprintf(&source, "missing%d();", index)
	}
	source.WriteString("return 0;}")
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", source.String())
	result, err := compiler.Check([]semantic.File{file})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Verified || len(result.MIR.Functions) != 0 {
		t.Fatalf("bounded errors published verified MIR: %+v", result)
	}
	if len(result.Semantic.Diagnostics) != 257 || result.Semantic.Diagnostics[256].Code != semantic.SemanticReportExhaustedCode {
		t.Fatalf("compiler diagnostics = %+v", result.Semantic.Diagnostics)
	}
}
