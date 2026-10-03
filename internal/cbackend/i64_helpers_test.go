package cbackend_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/target"
)

var i64HelperCases = []struct{ name, expression, result, left, right, expected, helpers string }{
	{"compare", "left == right", "bool", "42", "42", "true", ""},
	{"borrow", "*value", "i64", "42", "0", "42", ""},
	{"positive", "+left", "i64", "42", "0", "42", ""},
	{"negative", "-left", "i64", "42", "0", "-42", "bits"},
	{"complement", "~left", "i64", "41", "0", "-42", "bits"},
	{"add", "left + right", "i64", "40", "2", "42", "bits,add"},
	{"subtract", "left - right", "i64", "44", "2", "42", "bits,sub"},
	{"multiply", "left * right", "i64", "7", "6", "42", "bits,mul"},
	{"and", "left & right", "i64", "43", "42", "42", "bits"},
	{"xor", "left ^ right", "i64", "43", "1", "42", "bits"},
	{"or", "left | right", "i64", "40", "2", "42", "bits"},
	{"left-shift", "left << right", "i64", "21", "1", "42", "bits,shift"},
	{"negative-shift", "left >> right", "i64", "-84", "1", "-42", "bits,shift"},
	{"divide", "left / right", "i64", "84", "2", "42", ""},
	{"remainder", "left % right", "i64", "127", "85", "42", ""},
}

func i64HelperSource(index int) string {
	c := i64HelperCases[index]
	body := "return " + c.expression + ";"
	if c.name == "borrow" {
		body = "let ref i64 value = ref left; " + body
	}
	return fmt.Sprintf("module main; fn %s operation(i64 left, i64 right) { %s } fn i32 main() { let %s value = operation(%s, %s); return if (value == %s) { 42 } else { 1 }; }", c.result, body, c.result, c.left, c.right, c.expected)
}

func TestEmitI64HelpersDemand(t *testing.T) {
	for index, c := range i64HelperCases {
		t.Run(c.name, func(t *testing.T) {
			program := lowerSource(t, i64HelperSource(index))
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				unit, err := cbackend.Emit(program, "main::main", profile)
				if err != nil {
					t.Fatal(err)
				}
				for key, name := range map[string]string{"bits": "fy_bits_to_i64", "shift": "fy_sar_i64", "add": "fy_i64_add_overflow", "sub": "fy_i64_sub_overflow", "mul": "fy_i64_mul_overflow"} {
					want := strings.Contains(","+c.helpers+",", ","+key+",")
					if strings.Contains(unit.Source, name+"(") != want {
						t.Errorf("%s: helper %s demand differs", unit.Target, name)
					}
				}
				if strings.Contains(unit.Source, "fy_bits_to_i32(") {
					t.Fatal("i64 unary emitted an unused i32 helper")
				}
			}
		})
	}
}

// Both strict consumers must accept the actual Fairy-generated unit, including
// left-shift overflow checking's dependency on signed right-shift support.
func TestI64HelpersStrictLinuxConsumers(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux GNU amd64, GCC and Clang")
	}
	support, err := cruntime.LinuxRunSource(128)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gcc", "clang"} {
		t.Run(name, func(t *testing.T) {
			toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, name)
			if err != nil {
				t.Fatal(err)
			}
			for index, c := range i64HelperCases {
				t.Run(c.name, func(t *testing.T) {
					unit, err := cbackend.Emit(lowerSource(t, i64HelperSource(index)), "main::main", cbackend.Stage0Profile())
					if err != nil {
						t.Fatal(err)
					}
					for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
						artifact, err := cdriver.Build(toolchain, profile, unit.Source, support)
						if err != nil {
							t.Fatal(err)
						}
						outcome, err := cdriver.ExecuteLinuxTarget(artifact)
						if err != nil || outcome.ExitCode != 42 || outcome.TrapCategory != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
							t.Fatalf("%s: %+v %v", profile.Name, outcome, err)
						}
					}
				})
			}
		})
	}
}
