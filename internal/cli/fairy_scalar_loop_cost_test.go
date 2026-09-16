package cli

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

// Diagnostic observations, not wall-time regression thresholds. The dedicated
// command adds a process-level timeout; input functions are never executed.
func TestWindowsFairyParserLoopCost(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	runWindowsFairyFunctionTree(t, testFairyScalarLoopCost)
}

func testFairyScalarLoopCost(t *testing.T, native func(string) string) {
	t.Logf("runtime=%s platform=%s/%s samples=3 max_depth=16 stop_after_semantic=500ms", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	for _, depth := range []int{0, 4, 8, 12, 16} {
		input := "module main;fn i32 f(){" + strings.Repeat("while(false){", depth) + "let i32 x=1;" + strings.Repeat("}", depth) + "return 0;}"
		stop := false
		for sample := 0; sample < 3; sample++ {
			start := time.Now()
			wire := native(input)
			nativeTime := time.Since(start)
			start = time.Now()
			module, err := adaptFairyScalar(input, wire)
			adapterTime := time.Since(start)
			if err != nil {
				t.Fatal(err)
			}
			start = time.Now()
			ref, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
			frontendTime := time.Since(start)
			if err != nil || ref.AST == nil {
				t.Fatalf("frontend: %v", err)
			}
			start = time.Now()
			got := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: module}})
			adaptedTime := time.Since(start)
			start = time.Now()
			want := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: ref.AST}})
			referenceTime := time.Since(start)
			if len(want.Diagnostics) != 0 || len(want.Unsupported) != 0 || !reflect.DeepEqual(got, want) {
				t.Fatalf("semantic divergence at depth %d", depth)
			}
			t.Logf("depth=%d bytes=%d sample=%d native_us=%d adapter_us=%d frontend_us=%d semantic_adapted_us=%d semantic_reference_us=%d", depth, len(input), sample+1, nativeTime.Microseconds(), adapterTime.Microseconds(), frontendTime.Microseconds(), adaptedTime.Microseconds(), referenceTime.Microseconds())
			if adaptedTime >= 500*time.Millisecond || referenceTime >= 500*time.Millisecond {
				stop = true
				break
			}
		}
		if stop {
			t.Log("stopped sampling/depth escalation after slow semantic observation; this is not a performance pass/fail threshold")
			break
		}
	}
}
