package compiler_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
)

// Cross-target emission does not replace the still-missing Linux native run.
// It checks that the private source form needs no extra Linux runtime symbol
// or generated function compared with the equivalent explicit-index loop.
func TestPrivateVectorSDKCandidateLinuxVectorLoopEmission(t *testing.T) {
	const header = `module main;
import std::vec as v;
import std::bytes as bytes;
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
fn i32 exercise() {
    var v::Vec<i32> values = v::empty<i32>();
    assert(switch (v::push<i32>(ref mut values, 7)) {
        case .ok(_): { true }
        case .err(_): { false }
    });
    var i32 sum = 0;
`
	const body = `        sum += *item;`
	const footer = `    return sum;
}
fn i32 main() {
    let i32 total = exercise();
    assert(unsafe { clean() } == 1);
    return total;
}`
	var units [2]cbackend.Unit
	for index, form := range []struct {
		name, loop string
		candidate  bool
	}{
		{"explicit index", `    let usize bound = v::len<i32>(ref values);
    for (var usize index = 0; index < bound; index += 1) {
        let ref i32 item = v::at<i32>(ref values, index);
` + body + `
    }
`, false},
		{"vector for", `    for (let ref i32 item in ref values) {
` + body + `
    }
`, true},
	} {
		t.Run(form.name, func(t *testing.T) {
			input, coordinates := vectorSDKInputWithCandidate(t, header+form.loop+footer, true)
			owners := []hir.PackageID{3}
			var analysis compiler.GenericAnalysis
			var err error
			if form.candidate {
				analysis, err = compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates,
					genericplan.DefaultPlanLimits(), owners)
			} else {
				analysis, err = compiler.AnalyzeSDKStoragePackages(input, coordinates,
					genericplan.DefaultPlanLimits(), owners)
			}
			if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow ||
				len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
				t.Fatalf("Linux vector source and Flow: %v diagnostics=%+v unsupported=%+v", err,
					analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("Linux vector verified MIR: %v diagnostics=%+v", err, result.Semantic.Diagnostics)
			}
			unit, err := cbackend.EmitStorageTransfers(result.MIR,
				hir.QualifiedModulePath(1, "main")+"::main", cbackend.Stage0Profile())
			if err != nil || unit.Target != string(target.X8664LinuxGNU) || len(unit.Source) == 0 {
				t.Fatalf("Linux vector C emission: %v target=%q bytes=%d", err, unit.Target, len(unit.Source))
			}
			units[index] = unit
		})
	}
	if len(units[0].Functions) != len(units[1].Functions) ||
		!slices.Equal(units[0].ForeignSymbols, units[1].ForeignSymbols) {
		t.Fatalf("Linux vector source added generated functions or runtime symbols: indexed=%d/%v candidate=%d/%v",
			len(units[0].Functions), units[0].ForeignSymbols,
			len(units[1].Functions), units[1].ForeignSymbols)
	}
	t.Run("native", func(t *testing.T) {
		if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
			t.Skip("requires Linux amd64 glibc and GCC")
		}
		toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, "gcc")
		if err != nil {
			t.Fatal(err)
		}
		scalar, err := cruntime.ScalarSource(192)
		if err != nil {
			t.Fatal(err)
		}
		symbols := []string{"fy_std_system_heap_allocate_v0"}
		for _, symbol := range units[0].ForeignSymbols {
			if strings.HasPrefix(symbol, "fy_std_") {
				symbols = append(symbols, symbol)
			}
		}
		sdk, err := cstdlib.SourceForSymbols(symbols)
		if err != nil {
			t.Fatal(err)
		}
		heap, err := cstdlib.SourceForSymbols(symbols[:1])
		if err != nil {
			t.Fatal(err)
		}
		storage, err := cstdlib.LinuxGNUVectorStorageSource()
		if err != nil || !strings.HasPrefix(storage, heap) {
			t.Fatalf("Linux vector storage heap boundary: %v", err)
		}
		const cleanupProbe = `
int32_t fairyc_fixture_vector_clean(void) {
    return vector_live == 0 && vector_allocations == vector_frees ? 1 : 0;
}
`
		support := scalar + vectorStorageAllocator + sdk + strings.TrimPrefix(storage, heap) + cleanupProbe
		for index, name := range []string{"explicit index", "vector for"} {
			t.Run(name, func(t *testing.T) {
				for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					t.Run(profile.Name, func(t *testing.T) {
						artifact, err := cdriver.Build(toolchain, profile, units[index].Source, support)
						if err != nil {
							t.Fatal(err)
						}
						path := filepath.Join(t.TempDir(), "vector-loop")
						if err := os.WriteFile(path, artifact.Bytes, 0o700); err != nil {
							t.Fatal(err)
						}
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						output, err := exec.CommandContext(ctx, path).CombinedOutput()
						var exit *exec.ExitError
						if !errors.As(err, &exit) || exit.ExitCode() != 7 || len(output) != 0 || ctx.Err() != nil {
							t.Fatalf("Linux vector loop: %v output=%q", err, output)
						}
					})
				}
			})
		}
	})
}
