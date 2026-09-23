package semantic

import (
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestByteForFlowRetainsOwnerBorrowThroughLoop(t *testing.T) {
	for _, sample := range []struct {
		name string
		body string
		want string
	}{
		{name: "mutate during iteration", body: "for (let u8 byte in view) { data[0] = byte; }", want: "E2101"},
		{name: "mutate after iteration", body: "for (let u8 byte in view) { assert(byte != 0); } data[0] = 9;"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			input := "module main; fn void run() { var u8[2] data = [7, 8]; let ref u8[] view = ref data; " + sample.body + " }"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
			if err != nil || parsed.AST == nil {
				t.Fatalf("parse: %v", err)
			}
			result := Analyze([]File{{Path: "main.fy", Module: parsed.AST}})
			if len(result.Unsupported) != 0 {
				t.Fatalf("unsupported: %+v", result.Unsupported)
			}
			if sample.want == "" {
				if len(result.Diagnostics) != 0 || result.CompletedThrough != PhaseFlow {
					t.Fatalf("after-loop mutation: %+v", result.Diagnostics)
				}
				return
			}
			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != sample.want || result.FailedPhase != PhaseFlow {
				t.Fatalf("in-loop mutation: %+v", result.Diagnostics)
			}
		})
	}
}
