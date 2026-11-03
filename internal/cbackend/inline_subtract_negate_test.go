package cbackend_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPartitionedInlineSubtractNegate(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC regression; native Linux validation remains deferred")
	}
	for _, item := range []struct {
		name, expression, constant, args, expected string
		unary, trap                                bool
	}{
		{"subtract_low", "a-b", "-2147483648-1", "-2147483648,1", "0", false, true},
		{"subtract_high", "a-b", "2147483647-(-1)", "2147483647,-1", "0", false, true},
		{"negate", "-a", "-(-2147483648)", "-2147483648", "0", true, true},
		{"minimum", "a-b", "-2147483648-0", "-2147483648,0", "-2147483648", false, false},
		{"maximum", "a-b", "2147483647-0", "2147483647,0", "2147483647", false, false},
		{"negation", "-a", "-(-42)", "-42", "42", true, false},
		{"max_negation", "-a", "-(2147483647)", "2147483647", "-2147483647", true, false},
		{"negate_zero", "-a", "-0", "0", "0", true, false},
	} {
		for _, zero := range []bool{false, true} {
			name := "parameterized/" + item.name
			parameters, expression, args := "i32 a,i32 b", item.expression, item.args
			if item.unary {
				parameters = "i32 a"
			}
			if zero {
				name = "zero/" + item.name
				parameters, expression, args = "", item.constant, ""
			}
			t.Run(name, func(t *testing.T) {
				program := lowerSource(t, "module main;@inline pub fn i32 value("+parameters+"){return "+expression+";}fn i32 main(){assert(value("+args+")=="+item.expected+");return 42;}")
				parts, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", cbackend.Stage0WindowsMSVCProfile(), false, 128)
				if err != nil {
					t.Fatal(err)
				}
				status := 42
				if item.trap {
					status = int(cdriver.NativeTrapExitBase) + 2
				}
				for _, profile := range []string{"low", "optimized"} {
					t.Run(profile, func(t *testing.T) {
						defer func() {
							if t.Failed() {
								t.Log(parts.Header, parts.Sources)
							}
						}()
						compileSplitForTest(t, parts, profile, "", status)
					})
				}
			})
		}
	}
}
