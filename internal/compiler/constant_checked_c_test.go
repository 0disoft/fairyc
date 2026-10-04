package compiler_test

import (
	"bytes"
	"encoding/json"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConstantCheckedTrapCEmission(t *testing.T) {
	data, err := os.ReadFile("testdata/constant_checked_trap.fy")
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(data))})
	if err != nil || !result.Verified {
		t.Fatalf("check: %v %+v", err, result.Semantic)
	}
	outcome, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != "integer-overflow" {
		t.Fatalf("execute: %v %+v", err, outcome)
	}
	before, _ := json.Marshal(result.MIR)
	unit, err := cbackend.Emit(result.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("..", "..", ".cache", "constant-checked-trap")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unit.c"), []byte(unit.Source), 0600); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(result.MIR)
	if !bytes.Equal(before, after) {
		t.Fatal("C reachability mutated MIR")
	}
	for _, helper := range []string{"fy_bits_to_i64(", "fy_i64_add_overflow("} {
		if strings.Contains(unit.Source, helper) {
			t.Fatal("unused integer helper retained")
		}
	}
	if strings.Contains(unit.Source, " = fy_fn_") || strings.Contains(unit.Source, "return fy_l_") || strings.Contains(unit.Source, ").f_0 = ") {
		t.Fatal("non-returning call continuation retained")
	}
	if !strings.Contains(unit.Source, "static _Noreturn") {
		t.Fatal("constant-trapping aggregate return not proven non-returning")
	}
}
