package closureprep_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/closureprep"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func localClosure(t *testing.T, body string) *ast.Module {
	t.Helper()
	text := "module main;fn i32 main(){let i32 base=40;let c=fn [i32 base] i32(i32 x){" + body + "};return c(2);}"
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("parse closure locals: %v %+v", err, parsed)
	}
	return parsed.AST
}

func TestImmutableLocalsAdmissionBounds(t *testing.T) {
	var sixteen strings.Builder
	sixteen.WriteString("let i32 v0=x;")
	for i := 1; i < 16; i++ {
		fmt.Fprintf(&sixteen, "let v%d=v%d+1;", i, i-1)
	}
	for _, item := range []struct {
		name, body string
		accepted   bool
	}{
		{"zero", "return base+x;", true},
		{"tail", "let i32 a=base;let b=a+x;b", true},
		{"explicit return", "let a=base+x;return a;", true},
		{"sixteen", sixteen.String() + "return v15;", true},
		{"seventeen", sixteen.String() + "let v16=v15;return v16;", false},
		// 1 initializer + 63 returned nodes; one extra initializer makes 65.
		{"sixty four", "let unused=1;return " + strings.Repeat("x+", 31) + "x;", true},
		{"sixty five", "let unused=1;let another=1;return " + strings.Repeat("x+", 31) + "x;", false},
		{"group sixty four", "let unused=1;return ((" + strings.Repeat("x+", 30) + "x));", true},
		{"group sixty five", "let unused=1;return (" + strings.Repeat("x+", 31) + "x);", false},
		{"integer spellings", "let a=0xE;let b=0b10;let c=09;let d=-2147483648;return x;", true},
		{"integer separators", "let a=0xF_F;let b=0b1_0;let c=1_024;return x;", true},
		{"range belongs to typing", "let unused=" + strings.Repeat("9", 80) + ";return x;", true},
	} {
		t.Run(item.name, func(t *testing.T) {
			module := localClosure(t, item.body)
			before, err := json.Marshal(module)
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := closureprep.Prepare(module)
			if (err == nil) != item.accepted {
				t.Fatalf("accepted=%v: %v", item.accepted, err)
			}
			after, marshalErr := json.Marshal(module)
			if marshalErr != nil || !bytes.Equal(before, after) {
				t.Fatal("closure preparation mutated caller syntax")
			}
			if item.accepted {
				if len(prepared.Functions) != 2 || len(module.Functions) != 1 {
					t.Fatalf("one helper required: %+v", prepared.Functions)
				}
				written := module.Functions[0].Body.Statements[1].Binding.Expression.Closure.Function.Body
				got, _ := json.Marshal(prepared.Functions[1].Body)
				want, _ := json.Marshal(written)
				if !bytes.Equal(got, want) {
					t.Fatal("helper lost or reordered written initializers/return")
				}
			}
		})
	}
}

func TestImmutableLocalsRejectClosedFormsAtOriginalSpan(t *testing.T) {
	for name, body := range map[string]string{
		"forward":            "let a=b;let b=x;return a;",
		"self":               "let a=a;return a;",
		"duplicate":          "let a=x;let a=base;return a;",
		"parameter":          "let x=1;return x;",
		"capture":            "let base=1;return base;",
		"reserved":           "let _fy_closure_local=x;return x;",
		"explicit type":      "let i64 a=1;return x;",
		"borrow type":        "let ref i32 a=x;return x;",
		"inferred bool":      "let unused=true;return x;",
		"boolean arithmetic": "let unused=true+x;return x;",
		"inferred float":     "let unused=1.5;return x;",
		"inferred exponent":  "let unused=1e2;return x;",
		"inferred string":    "let unused=\"value\";return x;",
		"mutable":            "var i32 a=x;return a;",
		"assignment":         "let a=x;a=1;return a;",
		"call":               "let a=helper(x);return a;",
		"return call":        "let a=x;return helper(a);",
		"if":                 "let a=if(true){x}else{base};return a;",
		"while":              "let a=x;while(false){}return a;",
		"nested block":       "let a={x};return a;",
		"early return":       "return x;let a=base;return a;",
		"discard":            "let a=x;discard a;return x;",
		"expression":         "let a=x;a;return x;",
	} {
		t.Run(name, func(t *testing.T) {
			module := localClosure(t, body)
			want := module.Functions[0].Body.Statements[1].Span
			_, err := closureprep.Prepare(module)
			var refusal *closureprep.Error
			if !errors.As(err, &refusal) || refusal.Span != want {
				t.Fatalf("original closure binding span %v: %v", want, err)
			}
		})
	}
}
