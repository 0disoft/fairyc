package cbackend_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPartitionedZeroParameterInlineTraps(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC regression; Linux validation remains deferred")
	}
	for _, item := range []struct {
		name, expression string
		status           int
	}{
		{"divide", "10/0", int(cdriver.NativeTrapExitBase) + 3},
		{"signed", "-2147483648/-1", int(cdriver.NativeTrapExitBase) + 4},
		{"add", "2147483647+1", int(cdriver.NativeTrapExitBase) + 2},
		{"success", "40+2", 42},
	} {
		t.Run(item.name, func(t *testing.T) {
			program := lowerSource(t, "module main;@inline pub fn i32 value(){return "+item.expression+";}fn i32 main(){return value();}")
			parts, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", cbackend.Stage0WindowsMSVCProfile(), false, 128)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []string{"low", "optimized"} {
				t.Run(profile, func(t *testing.T) {
					defer func() {
						if t.Failed() {
							t.Log(parts.Header, parts.Sources)
						}
					}()
					compileSplitForTest(t, parts, profile, "", item.status)
				})
			}
		})
	}
}
