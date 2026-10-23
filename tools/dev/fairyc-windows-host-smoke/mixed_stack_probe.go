package main

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/windowsunwindprobe"
)

type mixedStackObservation struct {
	Case              string `json:"case"`
	Profile           string `json:"profile"`
	Toolchain         string `json:"toolchain"`
	FairySourceSHA256 string `json:"fairySourceSHA256"`
	GeneratedCSHA256  string `json:"generatedCSHA256"`
	CompanionSHA256   string `json:"companionSourceSHA256"`
	ImageSHA256       string `json:"imageSHA256"`
	CallerSHA256      string `json:"callerSHA256"`
	CalleeSHA256      string `json:"calleeSHA256"`
	RuntimeFunctions  int    `json:"runtimeFunctions"`
	Exit              int32  `json:"exit"`
	TrapCode          uint8  `json:"trapCode"`
}

func runWindowsMixedStackProbe(toolchain cdriver.WindowsHostToolchain) ([]mixedStackObservation, error) {
	runtimeSource, err := cruntime.WindowsRunSource(cruntime.NativeTrapExitBase)
	if err != nil {
		return nil, err
	}
	var observations []mixedStackObservation
	for _, item := range []struct {
		name, value string
		trap        bool
		prefix      string
	}{
		{"walk-and-c-boundary-catch", "40", false, ""},
		{"callback-trap-no-unwind", "13", true, ""},
		{"reject-handler-address", "40", true, "#define FY_MIXED_HANDLER_SHIFT 1\n"},
		{"reject-handler-data", "40", true, "#define FY_MIXED_DATA_SHIFT 4\n"},
		{"reject-caller-stack", "40", true, "#define FY_MIXED_STACK_SHIFT 8\n"},
	} {
		input := strings.ReplaceAll(windowsunwindprobe.MixedFairySource, "INPUT_VALUE", item.value)
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
			return nil, fmt.Errorf("mixed stack parse: %v %+v", err, parsed.Tree.Diagnostics)
		}
		checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
		if err != nil || !checked.Verified || len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
			return nil, fmt.Errorf("mixed stack MIR: %v %+v", err, checked.Semantic.Diagnostics)
		}
		unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			return nil, err
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			companion := item.prefix + windowsunwindprobe.MixedCompanionC + runtimeSource
			artifact, err := cdriver.BuildWindowsMSVCABIProbe(toolchain, profile, unit.Source, companion)
			if err != nil {
				return nil, fmt.Errorf("mixed stack %s/%s: %w", item.name, profile.Name, err)
			}
			if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != "callee.obj" {
				return nil, fmt.Errorf("mixed stack missing independent objects")
			}
			unwind, err := pecoff.InspectAMD64Unwind(artifact.Bytes)
			if err != nil {
				return nil, err
			}
			artifact.TrapControl = true
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
			wantExit := int32(42)
			code := uint8(0)
			if item.trap {
				category, ok := cbackend.TrapCode("assertion-failed")
				if !ok {
					return nil, fmt.Errorf("missing assertion trap identity")
				}
				code = uint8(category)
				wantExit = int32(cruntime.NativeTrapExitBase) + int32(code)
			}
			controlOK := outcome.TrapControl == ""
			if item.trap {
				controlOK = len(outcome.TrapControl) > 0 && outcome.TrapControl[0] == code
			}
			if err != nil || outcome.ExitCode != wantExit || outcome.Stdout != "" || outcome.Stderr != "" || !controlOK {
				return nil, fmt.Errorf("mixed stack %s/%s result: %+v, %v", item.name, profile.Name, outcome, err)
			}
			observations = append(observations, mixedStackObservation{
				Case: item.name, Profile: profile.Name, Toolchain: toolchain.Version,
				FairySourceSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(input))),
				GeneratedCSHA256:  fmt.Sprintf("%x", sha256.Sum256([]byte(unit.Source))),
				CompanionSHA256:   fmt.Sprintf("%x", sha256.Sum256([]byte(companion))),
				ImageSHA256:       fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes)),
				CallerSHA256:      fmt.Sprintf("%x", sha256.Sum256(artifact.Objects[0].Bytes)),
				CalleeSHA256:      fmt.Sprintf("%x", sha256.Sum256(artifact.Objects[1].Bytes)),
				RuntimeFunctions:  unwind.RuntimeFunctions, Exit: outcome.ExitCode, TrapCode: code,
			})
		}
	}
	return observations, nil
}
