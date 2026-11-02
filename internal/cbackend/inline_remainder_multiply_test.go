package cbackend_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPartitionedInlineRemainderMultiply(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC regression; native Linux validation remains deferred")
	}
	for _, item := range []struct {
		name, expression, constant, args string
		status                           int
	}{
		{"remzero", "a%b", "10%0", "10,0", int(cdriver.NativeTrapExitBase) + 3},
		{"remsigned", "a%b", "-2147483648%-1", "-2147483648,-1", int(cdriver.NativeTrapExitBase) + 4},
		{"multiply", "a*b", "46341*46341", "46341,46341", int(cdriver.NativeTrapExitBase) + 2},
		{"remainder_value", "a%b", "85%43", "85,43", 42},
		{"negative_remainder", "(a%b)+84", "(-85%43)+84", "-85,43", 42},
		{"product", "a*b", "6*7", "6,7", 42},
	} {
		for _, zero := range []bool{false, true} {
			name := "parameterized/" + item.name
			parameters, expression, args := "i32 a,i32 b", item.expression, item.args
			if zero {
				name = "zero/" + item.name
				expression = item.constant
				parameters, args = "", ""
			}
			t.Run(name, func(t *testing.T) {
				program := lowerSource(t, "module main;@inline pub fn i32 value("+parameters+"){return "+expression+";}fn i32 main(){return value("+args+");}")
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
