package semantic_test

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

// Local diagnostic observations, not timing assertions. The configured intent
// bounds the process; the between-sample stop cannot cancel one Analyze call.
func TestStatefulLoopCost(t *testing.T) {
	t.Logf("runtime=%s platform=%s/%s depths=1,4,8 samples=3 stop_after_semantic=250ms", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	for _, fixture := range []struct {
		name, prelude, before, body, after string
		unhandled, borrow, handling        bool
	}{
		{"borrow-live", "fn void observe(ref i32 p){}", "var i32 x=1;let ref i32 p=ref x;", "let i32 y=x+1;", "observe(p);", false, true, false},
		{"result-reassign-handled", "", "var Result<i32,i32> p=.ok(0);discard p;", "p=.ok(1);discard p;", "", false, false, true},
		{"result-continue-overwrite", "", "var Result<i32,i32> p=.ok(0);discard p;", "p=.ok(1);continue;", "discard p;", true, false, true},
		{"result-break-pending", "", "", "let Result<i32,i32> p=.ok(1);break;", "", true, false, false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			for _, depth := range []int{1, 4, 8} {
				input := "module main;" + fixture.prelude + "fn i32 f(bool flag){" + fixture.before + strings.Repeat("while(flag){", depth) + fixture.body + strings.Repeat("}", depth) + fixture.after + "return 0;}"
				var baseline semantic.Result
				stop := false
				for sample := 0; sample < 3; sample++ {
					start := time.Now()
					parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
					frontendTime := time.Since(start)
					if err != nil || parsed.AST == nil {
						t.Fatalf("frontend: %v", err)
					}
					start = time.Now()
					got := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
					elapsed := time.Since(start)
					if len(got.Unsupported) != 0 {
						t.Fatalf("unsupported: %+v", got.Unsupported)
					}
					if (len(got.Diagnostics) > 0) != fixture.unhandled {
						t.Fatalf("unexpected diagnostics: %+v", got.Diagnostics)
					}
					for _, d := range got.Diagnostics {
						if d.Code != "E4101" {
							t.Fatalf("unexpected diagnostic: %+v", d)
						}
					}
					if fixture.borrow && len(got.Borrows) == 0 {
						t.Fatal("missing borrow facts")
					}
					if fixture.handling && len(got.ResultHandling) == 0 {
						t.Fatal("missing Result handling facts")
					}
					if sample == 0 {
						baseline = got
					} else if !reflect.DeepEqual(baseline, got) {
						t.Fatal("nondeterministic semantic result")
					}
					t.Logf("depth=%d bytes=%d sample=%d frontend_us=%d semantic_us=%d diagnostics=%d borrows=%d handling=%d", depth, len(input), sample+1, frontendTime.Microseconds(), elapsed.Microseconds(), len(got.Diagnostics), len(got.Borrows), len(got.ResultHandling))
					if elapsed >= 250*time.Millisecond {
						stop = true
						break
					}
				}
				if stop {
					t.Log("stopped deeper sampling after slow observation; not a timing assertion")
					break
				}
			}
		})
	}
}
