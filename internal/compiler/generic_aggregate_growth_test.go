package compiler_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

// Compare the same nested ownership workload with and without source specialization.
// Timings are observations, never pass/fail thresholds or production-workload claims.
func TestGenericAggregateGrowthComparisonMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("aggregate growth comparison requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	scalarRuntime, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	limits := genericplan.PlanLimits{MaxSpecializations: 20, MaxSpecializationsPerDeclaration: 5, MaxExpansionDepth: 4}
	type sample struct {
		Repetition      int    `json:"repetition"`
		Types           int    `json:"scalar_types"`
		Mode            string `json:"mode"`
		Profile         string `json:"profile"`
		SourceSHA256    string `json:"source_sha256"`
		SourceBytes     int    `json:"source_bytes"`
		Specializations int    `json:"specializations"`
		MIRFunctions    int    `json:"mir_functions"`
		CBytes          int    `json:"c_bytes"`
		ExecutableBytes int    `json:"executable_bytes"`
		CheckNS         int64  `json:"parsed_source_to_verified_mir_ns"`
		EmitNS          int64  `json:"c_emission_ns"`
		NativeBuildNS   int64  `json:"native_build_ns"`
	}
	var samples []sample
	for repetition := 0; repetition < 2; repetition++ {
		counts, modes := []int{1, 3, 5}, []string{"generic", "manual"}
		if repetition == 1 {
			counts, modes = []int{5, 3, 1}, []string{"manual", "generic"}
		}
		for _, count := range counts {
			for _, mode := range modes {
				source := genericAggregateGrowthSource(count, mode == "generic")
				file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)
				coordinate := genericCoordinate(t)
				var result compiler.Result
				specializations := 0
				start := time.Now()
				if mode == "generic" {
					var plan genericplan.Plan
					result, plan, err = compiler.CheckGenericFunctionsForPackage(file, coordinate, limits)
					specializations = len(plan.Specializations)
				} else {
					result, err = compiler.Check([]semantic.File{file})
				}
				checkNS := time.Since(start).Nanoseconds()
				if err != nil || !result.Verified {
					t.Fatalf("%s/%d: %v %+v", mode, count, err, result.Semantic)
				}
				if mode == "generic" && specializations != 4*count {
					t.Fatalf("specialization count %d, want %d", specializations, 4*count)
				}
				if len(result.MIR.StructTypes) != count || len(result.MIR.ResourceTypes) != count {
					t.Fatal("aggregate shape mismatch")
				}
				outcome, err := interpreter.Execute(result.MIR, "main::main")
				if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != 2*count {
					t.Fatalf("%s/%d interpreter: %+v %v", mode, count, outcome, err)
				}
				start = time.Now()
				unit, err := cbackend.Emit(result.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
				emitNS := time.Since(start).Nanoseconds()
				if err != nil {
					t.Fatal(err)
				}
				digest := sha256.Sum256([]byte(source))
				for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					start = time.Now()
					artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalarRuntime)
					buildNS := time.Since(start).Nanoseconds()
					if err != nil {
						t.Fatal(err)
					}
					native, err := cdriver.ExecuteWindowsHost(artifact)
					if err != nil || native.ExitCode != 42 || native.Stdout != "" || native.Stderr != "" {
						t.Fatalf("%s/%d/%s native: %+v %v", mode, count, profile.Name, native, err)
					}
					samples = append(samples, sample{repetition, count, mode, profile.Name, hex.EncodeToString(digest[:]), len(source), specializations, len(result.MIR.Functions), len(unit.Source), len(artifact.Bytes), checkNS, emitNS, buildNS})
				}
			}
		}
	}
	report := struct {
		Schema  string                 `json:"schema"`
		Build   string                 `json:"test_executable_identity"`
		Go      string                 `json:"go_version"`
		MSVC    string                 `json:"msvc_version"`
		Limits  genericplan.PlanLimits `json:"planner_limits"`
		Samples []sample               `json:"samples"`
	}{"fairyc.aggregate-growth-comparison/v0", identity.String(), runtime.Version(), toolchain.Version, limits, samples}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(encoded))
}

func genericAggregateGrowthSource(count int, generic bool) string {
	types := []string{"bool", "u8", "i32", "u32", "usize"}
	var source, body strings.Builder
	source.WriteString("module main; ")
	declaration := `struct CELL { TYPE value; }
resource TOKEN { CELL payload; cleanup {} }
fn TOKEN MAKE(TYPE value) { return TOKEN { .payload = CELL { .value = value } }; }
fn i32 CONSUME(TOKEN owned, TYPE expected) { if (owned.payload.value == expected) { return 1; } return 0; }
`
	if generic {
		source.WriteString(strings.NewReplacer("CELL", "Cell<T>", "TOKEN", "Token<T>", "MAKE", "make<T>", "CONSUME", "consume<T>", "TYPE", "T").Replace(declaration))
	}
	for index := 0; index < count; index++ {
		scalar, value := types[index], "42"
		if scalar == "bool" {
			value = "true"
		}
		makeName, consumeName := fmt.Sprintf("make%d", index), fmt.Sprintf("consume%d", index)
		if generic {
			makeName, consumeName = "make<"+scalar+">", "consume<"+scalar+">"
		} else {
			source.WriteString(strings.NewReplacer("CELL", fmt.Sprintf("Cell%d", index), "TOKEN", fmt.Sprintf("Token%d", index), "MAKE", makeName, "CONSUME", consumeName, "TYPE", scalar).Replace(declaration))
		}
		for repeat := 0; repeat < 2; repeat++ {
			fmt.Fprintf(&body, "assert(%s(%s(%s), %s) == 1);", consumeName, makeName, value, value)
		}
	}
	source.WriteString("fn i32 main() { " + body.String() + "return 42; }")
	return source.String()
}
