package semantic_test

import (
	"fmt"
	"testing"
)

func TestIntegerTransportBoundaries(t *testing.T) {
	for _, typ := range []string{"i8", "i16", "u16", "u64", "isize"} {
		for name, body := range map[string]string{
			"literal":    `fn %[1]s bad(){return 1;}`,
			"arithmetic": `fn %[1]s bad(%[1]s a,%[1]s b){return a+b;}`,
			"comparison": `fn bool bad(%[1]s a,%[1]s b){return a==b;}`,
			"width":      `fn %[1]s bad(i32 a){return a;}`,
			"cast":       `fn %[1]s bad(i32 a){return a as %[1]s;}`,
			"array":      `@repr(C) struct P{%[1]s[2] values;}`,
		} {
			t.Run(typ+"/"+name, func(t *testing.T) {
				r := analyzeText(t, map[string]string{"main.fy": "module main;" + fmt.Sprintf(body, typ)})
				if len(r.Diagnostics)+len(r.Unsupported) == 0 {
					t.Fatal("unsupported integer transport operation admitted")
				}
			})
		}
	}
}
