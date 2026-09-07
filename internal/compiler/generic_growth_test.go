package compiler_test

import (
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
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
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/pecoff"
)

// This is a bounded calibration, not a timing regression gate or a representative workload.
func TestGenericGrowthMeasurementMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("generic growth calibration requires Windows amd64")
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
	coordinate := genericCoordinate(t)
	report := genericGrowthReport{
		Schema: "fairyc.generic-growth-calibration/v0", Decision: "observation-only-no-performance-threshold",
		Build: identity.String(), Go: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH,
		MSVC: toolchain.Version, Warmups: 0, Repetitions: 3, Cache: "no-compiler-artifact-cache; ambient-OS-and-toolchain-caches",
		Concurrency: 1, LogicalCPUs: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0), Clock: "go-monotonic-elapsed-nanoseconds",
		Order: "alternating-ascending-descending-shapes; low-then-optimized", Limits: genericplan.PlanLimits{MaxSpecializations: 8, MaxSpecializationsPerDeclaration: 8, MaxExpansionDepth: 1},
		Samples: make([]genericGrowthSample, 0, 24),
	}
	for repetition := 0; repetition < report.Repetitions; repetition++ {
		counts := []int{1, 2, 4, 8}
		if repetition%2 == 1 {
			counts = []int{8, 4, 2, 1}
		}
		for _, count := range counts {
			source := genericGrowthSource(count)
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)
			start := time.Now()
			result, plan, err := compiler.CheckGenericFunctionsForPackage(file, coordinate, report.Limits)
			compileNS := time.Since(start).Nanoseconds()
			if err != nil || !result.Verified {
				t.Fatalf("shape %d compile: %v; semantic: %+v", count, err, result.Semantic)
			}
			if len(plan.Specializations) != count || len(result.MIR.Functions) != count+1 {
				t.Fatal("duplicate calls were not deduplicated")
			}
			start = time.Now()
			unit, err := cbackend.Emit(result.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
			emitNS := time.Since(start).Nanoseconds()
			if err != nil {
				t.Fatal(err)
			}
			sourceHash := sha256.Sum256([]byte(source))
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				start = time.Now()
				artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalarRuntime)
				buildNS := time.Since(start).Nanoseconds()
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := cdriver.ExecuteWindowsHost(artifact)
				if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("shape %d %s: %+v %v", count, profile.Name, outcome, err)
				}
				sample := genericGrowthSample{Repetition: repetition, Specializations: count, Calls: count * 2, MIRFunctions: len(result.MIR.Functions), SourceSHA256: hex.EncodeToString(sourceHash[:]), SourceBytes: len(source), CUnitBytes: len(unit.Source), Profile: profile.Name, CompileNS: compileNS, EmitNS: emitNS, NativeBuildNS: buildNS, ExecutableBytes: len(artifact.Bytes), ExitCode: outcome.ExitCode}
				for _, object := range artifact.Objects {
					if object.Name != "unit.obj" {
						continue
					}
					sample.UnitObjectBytes = len(object.Bytes)
					inspected, err := pecoff.InspectObject(object.Bytes)
					if err != nil {
						t.Fatal(err)
					}
					for _, section := range inspected.Sections {
						if section.Characteristics&pe.IMAGE_SCN_CNT_CODE != 0 {
							sample.UnitCodeBytes += uint64(section.RawSize)
						}
					}
				}
				if sample.UnitObjectBytes == 0 || sample.UnitCodeBytes == 0 {
					t.Fatal("missing generated-unit code measurement")
				}
				report.Samples = append(report.Samples, sample)
			}
		}
	}
	// Both independent limits must fail before publishing MIR, not merely truncate the plan.
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", genericGrowthSource(8))
	for _, limits := range []genericplan.PlanLimits{{MaxSpecializations: 7, MaxSpecializationsPerDeclaration: 8, MaxExpansionDepth: 1}, {MaxSpecializations: 8, MaxSpecializationsPerDeclaration: 7, MaxExpansionDepth: 1}} {
		result, _, err := compiler.CheckGenericFunctionsForPackage(file, genericCoordinate(t), limits)
		if result.Verified || !reflect.DeepEqual(result.MIR, mir.Program{}) || genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
			t.Fatal("over-budget result published")
		}
	}
	report.RejectedLimits = 2
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(encoded))
}

func genericGrowthSource(count int) string {
	types := []string{"i32", "u32", "usize", "u8"}
	var source strings.Builder
	source.WriteString("module main; fn T id<T,U>(T value, U marker) { return value; } fn i32 main() {")
	for index := 0; index < count; index++ {
		name := types[index%len(types)]
		markerType, markerValue := "bool", "true"
		if index >= len(types) {
			markerType, markerValue = "i32", "7"
		}
		for repeat := 0; repeat < 2; repeat++ {
			fmt.Fprintf(&source, "assert(id<%s,%s>(42, %s) == 42);", name, markerType, markerValue)
		}
	}
	source.WriteString("return 42; }")
	return source.String()
}

type genericGrowthReport struct {
	Schema         string                 `json:"schema"`
	Decision       string                 `json:"decision"`
	Build          string                 `json:"test_executable_identity"`
	Go             string                 `json:"go_version"`
	OS             string                 `json:"os"`
	Arch           string                 `json:"arch"`
	MSVC           string                 `json:"msvc_version"`
	Warmups        int                    `json:"warmups"`
	Repetitions    int                    `json:"repetitions"`
	Cache          string                 `json:"cache_state"`
	Order          string                 `json:"sample_order"`
	Concurrency    int                    `json:"concurrency"`
	LogicalCPUs    int                    `json:"logical_cpus"`
	GOMAXPROCS     int                    `json:"gomaxprocs"`
	Clock          string                 `json:"clock"`
	Limits         genericplan.PlanLimits `json:"planner_limits"`
	RejectedLimits int                    `json:"rejected_limit_cases"`
	Samples        []genericGrowthSample  `json:"samples"`
}

type genericGrowthSample struct {
	Repetition      int    `json:"repetition"`
	Specializations int    `json:"specializations"`
	Calls           int    `json:"source_calls"`
	MIRFunctions    int    `json:"mir_functions"`
	SourceSHA256    string `json:"source_sha256"`
	SourceBytes     int    `json:"source_bytes"`
	CUnitBytes      int    `json:"c_unit_bytes"`
	Profile         string `json:"profile"`
	CompileNS       int64  `json:"specialization_to_verified_mir_ns"`
	EmitNS          int64  `json:"c_emission_ns"`
	NativeBuildNS   int64  `json:"native_build_and_inspection_ns"`
	UnitObjectBytes int    `json:"unit_object_bytes"`
	UnitCodeBytes   uint64 `json:"unit_code_section_bytes"`
	ExecutableBytes int    `json:"executable_bytes"`
	ExitCode        int32  `json:"exit_code"`
}
