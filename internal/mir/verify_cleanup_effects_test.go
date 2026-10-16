package mir

import (
	"strings"
	"testing"
)

const cleanupEffectSource = `module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}`

func TestVerifyRejectsMissingTransitiveCleanupAuthority(t *testing.T) {
	for _, body := range []string{
		`discard make();`,
		`make();`,
		`let R value=R{.n=1};`,
		`let Box value=make();`,
		`let Option<Box> value=.some(make());`,
		`let Result<Box,i32> value=.ok(make());discard value.is_ok;`,
	} {
		program := lowerTestSource(t, cleanupEffectSource+`@effects(console)fn i32 main(){`+body+`return 0;}`)
		if failures := Verify(program); len(failures) != 0 {
			t.Fatalf("valid MIR: %+v", failures)
		}
		for i := range program.Functions {
			if program.Functions[i].Name == "main::main" {
				program.Functions[i].Effects = nil
			}
		}
		failures := Verify(program)
		if len(failures) == 0 {
			t.Fatalf("missing authority accepted: %s", body)
		}
		for _, failure := range failures {
			if failure.Code != "V1001" || !strings.Contains(failure.Detail, "cleanup exceeds function effect authority") {
				t.Fatalf("unexpected failure: %+v", failure)
			}
		}
	}
}

func TestVerifyWholeRootTransferDoesNotRequireCleanupAuthority(t *testing.T) {
	program := lowerTestSource(t, cleanupEffectSource+`fn Box transfer(Box value){return value;}`)
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("transfer MIR: %+v", failures)
	}
}
