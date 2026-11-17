package cbackend_test

import (
	"os"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestEmitWindowsTargetSmokeSource(t *testing.T) {
	text, err := os.ReadFile("../../examples/windows_target/main.fy")
	if err != nil {
		t.Fatal(err)
	}
	program := lowerSources(t, map[string]string{"examples/windows_target/main.fy": string(text)})
	if failures := mir.Verify(program); len(failures) != 0 {
		t.Fatalf("invalid smoke MIR: %+v", failures)
	}
	if _, err := cbackend.Emit(program, "examples::windows_target::main::main", cbackend.Stage0WindowsMSVCProfile()); err != nil {
		t.Fatal(err)
	}
}
