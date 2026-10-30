package cbackend_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestEmitZeroIntegerArrays(t *testing.T) {
	program := lowerSource(t, `module main;
fn u8 zero() { return 0; }
fn i32 main() {
    var u8[4096] bytes = [0; 4096];
    let usize[4096] sizes = [0; 4096];
    let i32[4] signed_values = [0; 4];
    let i64[4] wide_values = [0; 4];
    let u32[4] unsigned_values = [0; 4];
    let u8[3] mixed = [0, 0, 7];
    let i32[4] repeated = [9; 4];
    let u8 z = zero();
    let u8[2] dynamic = [z, z];
    var usize i = 0;
    while (i < 4096) { assert(bytes[i] == 0 && sizes[i] == 0); i = i + 1; }
    i = 0;
    while (i < 4) {
        assert(signed_values[i] == 0 && wide_values[i] == 0 && unsigned_values[i] == 0);
        assert(repeated[i] == 9); i = i + 1;
    }
    bytes[4095] = 23;
    assert(bytes[0] == 0 && bytes[4095] == 23 && mixed[0] == 0 && mixed[2] == 7);
    assert(dynamic[0] == 0 && dynamic[1] == 0);
    return bytes[4095] as i32;
}
`)
	want, err := interpreter.Execute(program, "main::main")
	if err != nil || want.Kind != interpreter.OutcomeReturn || want.Value != 23 {
		t.Fatalf("interpreter: %+v %v", want, err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		t.Run(string(profile.Target), func(t *testing.T) {
			unit, err := cbackend.Emit(program, "main::main", profile)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(unit.Source, ".data = { 0 }") != 5 {
				t.Fatal("zero integer arrays were expanded instead of using C11 implicit initialization")
			}
			for _, explicit := range []string{
				".data = { UINT8_C(0), UINT8_C(0), UINT8_C(7) }",
				".data = { INT32_C(9), INT32_C(9), INT32_C(9), INT32_C(9) }",
				".data = { fy_l_",
			} {
				if !strings.Contains(unit.Source, explicit) {
					t.Fatalf("nonzero or local array initializer lost: %s", explicit)
				}
			}
			if len(unit.Source) > 30_000 {
				t.Fatalf("array emission grew to %d bytes", len(unit.Source))
			}
			t.Run("native", func(t *testing.T) {
				id, err := cdriver.HostNativeTarget()
				if err != nil || id != profile.Target {
					t.Skip("requires matching native host")
				}
				toolchain, err := cdriver.ResolveNativeToolchain(id, "")
				if err != nil {
					t.Fatal(err)
				}
				support, err := cruntime.ScalarSource(cdriver.NativeTrapExitBase)
				if err != nil {
					t.Fatal(err)
				}
				for _, optimization := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					t.Run(optimization.Name, func(t *testing.T) {
						artifact, err := toolchain.Build(cdriver.NativeBuildOptions{Profile: optimization, UnitSource: unit.Source, RuntimeSource: support})
						if err != nil {
							t.Fatal(err)
						}
						outcome, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{RejectOutput: true})
						if err != nil || outcome.ExitCode != int32(want.Value) || outcome.TrapCategory != "" {
							t.Fatalf("native: %+v %v", outcome, err)
						}
					})
				}
			})
		})
	}
}
