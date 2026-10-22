package semantic_test

import (
	"fmt"
	"strings"
	"testing"
)

func TestCEnumBaseBoundaries(t *testing.T) {
	for _, attribute := range []string{"@repr(C)", "@repr(i32)", "@repr(c::bool)", "@repr(c::float)", "@repr(c::double)", "@repr(c::uint, c::int)", "@repr(c::uint) @repr(c::uint)"} {
		t.Run(attribute, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main; " + attribute + " enum State { Ready; }"})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("invalid C enum base accepted")
			}
		})
	}
	for _, body := range []string{"", "Ready { i32 code; };"} {
		r := analyzeText(t, map[string]string{"main.fy": "module main; @repr(c::uint) enum State {" + body + "}"})
		if len(r.Diagnostics)+len(r.Unsupported) == 0 {
			t.Fatal("empty or payload C enum accepted")
		}
	}
	for _, count := range []int{256, 257} {
		var variants strings.Builder
		for i := 0; i < count; i++ {
			fmt.Fprintf(&variants, "V%d;", i)
		}
		r := analyzeText(t, map[string]string{"main.fy": "module main; @repr(c::uchar) enum State {" + variants.String() + "}"})
		rejected := len(r.Diagnostics)+len(r.Unsupported) != 0
		if rejected != (count == 257) {
			t.Fatalf("%d variants: diagnostics=%+v unsupported=%+v", count, r.Diagnostics, r.Unsupported)
		}
	}
}
