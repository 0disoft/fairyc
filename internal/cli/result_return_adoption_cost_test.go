package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
)

// Opt-in source comparison; the old variant is the checked-in ADR 0398 form.
func TestResultBranchReviewReturnAdoptionCost(t *testing.T) {
	if os.Getenv("FAIRYC_RESULT_RETURN_COST") != "1" {
		t.Skip("opt-in Result return-arm comparison")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC evidence")
	}
	root, _, _ := parserLabelWorkspace(t)
	data, err := os.ReadFile(filepath.Join(root, "src", "diagnostics.fy"))
	if err != nil {
		t.Fatal(err)
	}
	changed := string(data)
	start := strings.Index(changed, "pub fn i32 add(ref mut v::Vec<Entry> entries,")
	end := strings.Index(changed, "\nfn bool less(")
	if start < 0 || end <= start || strings.Count(changed, "pub fn i32 add(ref mut v::Vec<Entry> entries,") != 1 {
		t.Fatal("cannot isolate the adopted diagnostics::add body")
	}
	previous := `pub fn i32 add(ref mut v::Vec<Entry> entries, u32 code, usize start, usize end, bool exit_repair, ref u8[] token) {
    return switch (bytes::copy(token)) {
        case .ok(owned) : {
            let Entry entry = Entry { .code = code, .start = start, .end = end, .exit_repair = exit_repair, .token = owned };
            // Push returns the exact new length without converting a mutable borrow.
            // One overflow record is retained only until the failed scan drops its owner.
            switch (v::push<Entry>(entries, entry)) {
                case .ok(length) : { if (length > 256 && !(code == 10 && length == 257)) { 4 } else { 0 } }
                case .err(_) : { 3 }
            }
        }
        case .err(_) : { 3 }
    };
}
`
	plain := changed[:start] + previous + changed[end:]
	type sample struct {
		Repetition       int    `json:"repetition"`
		Variant          string `json:"variant"`
		WorkspaceLowerNS int64  `json:"workspace_lower_ns"`
		CEmitNS          int64  `json:"c_emit_ns"`
		CBytes           int    `json:"c_bytes"`
		MIRFunctions     int    `json:"mir_functions"`
	}
	var samples []sample
	for repetition := -1; repetition < 5; repetition++ {
		variants := []struct{ name, source string }{{"plain", plain}, {"changed", changed}}
		if repetition%2 == 0 {
			variants[0], variants[1] = variants[1], variants[0]
		}
		for _, variant := range variants {
			writeTestSource(t, root, "src/diagnostics.fy", variant.source)
			var diagnostics strings.Builder
			start := time.Now()
			lowered, code := lowerWorkspaceExecutable(root, false, workspaceCompilePolicy{stableDiagnostics: true}, "Result return adoption cost", &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("%s repetition %d: %s", variant.name, repetition, diagnostics.String())
			}
			workspaceLowerNS := time.Since(start).Nanoseconds()
			start = time.Now()
			unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			if repetition >= 0 {
				samples = append(samples, sample{repetition, variant.name, workspaceLowerNS, time.Since(start).Nanoseconds(), len(unit.Source), len(lowered.program.Functions)})
			}
		}
	}
	if len(samples) != 10 {
		t.Fatalf("incomplete samples: %d", len(samples))
	}
	report, err := json.Marshal(map[string]any{
		"schema": "fairyc.result-return-adoption/v1", "go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0),
		"plain_sha256":   fmt.Sprintf("%x", sha256.Sum256([]byte(plain))),
		"changed_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(changed))),
		"samples":        samples,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("RESULT_RETURN_CHECK_SAMPLES %s", report)
	compareParserNativeEdit(t, root, "src/diagnostics.fy", plain, changed, "RESULT_RETURN_NATIVE_SIZES")
}
