package sdkstdlib

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
)

func TestI64ConsoleOutputSurface(t *testing.T) {
	d, e := Load()
	if e != nil {
		t.Fatal(e)
	}
	for _, src := range d.Sources {
		if src.Path != "std/io.fy" {
			continue
		}
		found := map[string]bool{}
		for _, fn := range src.Module.Functions {
			if fn.Name != "write_i64" && fn.Name != "write_stderr_i64" {
				continue
			}
			found[fn.Name] = true
			if !fn.Public || len(fn.Parameters) != 1 || fn.Parameters[0].Type.Path != "i64" || fn.Parameters[0].Type.Kind != ast.TypeKindNamed {
				t.Fatalf("unexpected i64 console signature: %+v", fn)
			}
		}
		if !found["write_i64"] || !found["write_stderr_i64"] {
			t.Fatal("missing public i64 output")
		}
		text := string(src.Bytes)
		if strings.Contains(text, "fy_std_stdout_write_i64") || strings.Contains(text, "fy_std_stderr_write_i64") {
			t.Fatal("i64 output must use existing byte writers")
		}
		return
	}
	t.Fatal("missing std/io.fy")
}
