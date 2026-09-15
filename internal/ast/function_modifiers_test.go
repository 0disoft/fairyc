package ast

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestLowerFunctionModifiersStopAtDeclarationKeyword(t *testing.T) {
	for _, result := range []string{"i32", "unsafe extern C fn void()", "Box<unsafe extern C fn void()>"} {
		for mask := 0; mask < 4; mask++ {
			prefix := "@note(\"unsafe pub fn\") /* unsafe pub */\r\n"
			if mask&1 != 0 {
				prefix += "pub /* fn */ "
			}
			if mask&2 != 0 {
				prefix += "unsafe /* fn */ "
			}
			input := "module main;" + prefix + "fn " + result + " Owner::value(){return unsafe{value};}"
			tree := syntax.Parse(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
			if len(tree.Diagnostics) != 0 {
				t.Fatalf("parse %q: %+v", input, tree.Diagnostics)
			}
			module, err := Lower(tree)
			if err != nil || len(module.Functions) != 1 {
				t.Fatalf("lower: %v %+v", err, module)
			}
			fn := module.Functions[0]
			if fn.Public != (mask&1 != 0) || fn.Unsafe != (mask&2 != 0) || fn.Owner != "Owner" {
				t.Fatalf("modifiers for %q: public=%v unsafe=%v owner=%q", input, fn.Public, fn.Unsafe, fn.Owner)
			}
		}
	}
}
