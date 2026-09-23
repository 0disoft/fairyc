package semantic

import (
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

// The public pipeline still rejects byte-for before Flow. Exercise its prepared
// scalar loop directly so admission cannot silently bypass owner liveness.
func TestByteForPreparedFlowRetainsOwnerBorrowThroughLoop(t *testing.T) {
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
			a := newAnalyzer([]File{{Path: "main.fy", Module: parsed.AST}}, nil)
			a.resolve()
			a.typeCheck()
			if len(a.diagnostics) != 0 || len(a.unsupported) != 1 || a.unsupported[0].Feature != "byte-view iteration awaits semantic and MIR lowering" {
				t.Fatalf("prepared typecheck: diagnostics = %+v, unsupported = %+v", a.diagnostics, a.unsupported)
			}
			a.flowCheck()
			if sample.want == "" {
				if len(a.diagnostics) != 0 {
					t.Fatalf("after-loop mutation: %+v", a.diagnostics)
				}
				return
			}
			if len(a.diagnostics) != 1 || a.diagnostics[0].Code != sample.want {
				t.Fatalf("in-loop mutation: %+v", a.diagnostics)
			}
		})
	}
}
