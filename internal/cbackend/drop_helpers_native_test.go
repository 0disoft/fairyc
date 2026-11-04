package cbackend_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestPartitionedNestedDropHelpers(t *testing.T) {
	program := lowerSource(t, `module main;
resource Leaf { i32 value; cleanup { assert(this.value == 7); } }
resource Box { Leaf first; Leaf last; cleanup {} }
resource Outer { Box first; Box last; cleanup {} }
fn Box make() { return Box { .first = Leaf { .value = 7 }, .last = Leaf { .value = 7 } }; }
fn i32 main() { let Outer tree = Outer { .first = make(), .last = make() }; return 42; }
`)
	profile := cbackend.Stage0Profile()
	if runtime.GOOS == "windows" {
		profile = cbackend.Stage0WindowsMSVCProfile()
	}
	parts, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", profile, false, 256)
	if err != nil {
		t.Fatal(err)
	}
	for _, optimization := range []string{"low", "optimized"} {
		t.Run(optimization, func(t *testing.T) { compileSplitForTest(t, parts, optimization, "", 42) })
	}
}
