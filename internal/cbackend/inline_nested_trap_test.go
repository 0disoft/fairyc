package cbackend_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPartitionedInlineNestedTraps(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC nested inline traps; native Linux remains deferred")
	}
	for _, item := range []struct {
		name, expression, constant, args string
		status                           int
	}{
		{"inner", "(a+1)/b+(2147483647+1)", "(2147483647+1)/0+(2147483647+1)", "2147483647,0", int(cdriver.NativeTrapExitBase) + 2},
		{"division", "(a+1)/b+(2147483647+1)", "(9+1)/0+(2147483647+1)", "9,0", int(cdriver.NativeTrapExitBase) + 3},
		{"outer", "(a+1)/b+(2147483647+1)", "(9+1)/2+(2147483647+1)", "9,2", int(cdriver.NativeTrapExitBase) + 2},
		{"success", "-(-((a+1)/b))+(39-2)", "-(-((9+1)/2))+(39-2)", "9,2", 42},
	} {
		for _, zero := range []bool{false, true} {
			name := "parameterized/" + item.name
			parameters, expression, args := "i32 a,i32 b", item.expression, item.args
			if zero {
				name = "zero/" + item.name
				parameters, expression, args = "", item.constant, ""
			}
			t.Run(name, func(t *testing.T) {
				program := lowerSource(t, "module main;@inline pub fn i32 value("+parameters+"){return "+expression+";}fn i32 main(){assert(value("+args+")==42);return 42;}")
				parts, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", cbackend.Stage0WindowsMSVCProfile(), false, 128)
				if err != nil {
					t.Fatal(err)
				}
				for _, profile := range []string{"low", "optimized"} {
					t.Run(profile, func(t *testing.T) { compileSplitForTest(t, parts, profile, "", item.status) })
				}
			})
		}
	}
}
