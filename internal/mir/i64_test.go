package mir

import (
	"math"
	"testing"
)

func TestI64LowerPreservesSignedImmediates(t *testing.T) {
	program := lowerTestSource(t, `module main;
fn i64 minimum() { return -9223372036854775808; }
fn i64 maximum() { return 9223372036854775807; }
fn i32 main() { return 0; }
`)
	for name, want := range map[string]int64{"main::minimum": math.MinInt64, "main::maximum": math.MaxInt64} {
		found := false
		for _, function := range program.Functions {
			if function.Name != name {
				continue
			}
			for _, block := range function.Blocks {
				operand := block.Terminator.ReturnValue
				if operand.Kind == OperandI64 && operand.I64 == want {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("lost literal %d in %s", want, name)
		}
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatal(failures)
	}
}
