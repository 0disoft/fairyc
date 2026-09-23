package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// This opt-in comparison keeps the Fairy source identical across two separately
// built test binaries. The compact binary is an experiment, not a supported
// implementation of labeled calls.
func TestArgumentLabelsASTLayoutMeasure(t *testing.T) {
	lane := os.Getenv("FAIRYC_AST_LAYOUT_LANE")
	if lane == "" {
		t.Skip("opt-in AST layout measurement")
	}
	if lane != "current" && lane != "without-label-field" {
		t.Fatalf("unknown AST layout lane %q", lane)
	}
	root, sourceHashes := tokenIndexWorkspace(t, false)
	check := func() compiler.Result {
		t.Helper()
		workspace, err := workspaceloader.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		result, err := workspacecompiler.Check(workspace)
		if err != nil || !result.Verified || len(result.Semantic.Diagnostics) != 0 {
			t.Fatalf("workspace check: err=%v verified=%t diagnostics=%+v", err, result.Verified, result.Semantic.Diagnostics)
		}
		return result
	}
	_ = check() // Untimed warmup.
	const iterations = 4
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	var checked compiler.Result
	for i := 0; i < iterations; i++ {
		checked = check()
	}
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	encodedMIR, err := json.Marshal(checked.MIR)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encodedMIR)
	report := map[string]any{
		"lane":                         lane,
		"go":                           runtime.Version(),
		"gomaxprocs":                   runtime.GOMAXPROCS(0),
		"iterations":                   iterations,
		"source_sha256":                sourceHashes,
		"ast_expression_bytes":         reflect.TypeOf(ast.Expression{}).Size(),
		"mir_functions":                len(checked.MIR.Functions),
		"mir_json_sha256":              hex.EncodeToString(digest[:]),
		"check_ns_per_iteration":       elapsed.Nanoseconds() / iterations,
		"go_alloc_bytes_per_iteration": (after.TotalAlloc - before.TotalAlloc) / iterations,
		"go_mallocs_per_iteration":     (after.Mallocs - before.Mallocs) / iterations,
		"gc_cycles":                    after.NumGC - before.NumGC,
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("AST_LAYOUT_REPORT " + string(encoded))
}
