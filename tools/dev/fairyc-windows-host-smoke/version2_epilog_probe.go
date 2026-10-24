package main

import (
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/json"
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

type version2EpilogObservation struct {
	Profile           string `json:"profile"`
	Toolchain         string `json:"toolchain"`
	FairySHA256       string `json:"fairySourceSHA256"`
	GeneratedSHA256   string `json:"generatedCSHA256"`
	CompanionSHA256   string `json:"companionSourceSHA256"`
	ImageSHA256       string `json:"imageSHA256"`
	Functions         int    `json:"functions"`
	Epilogs           int    `json:"epilogs"`
	Contexts          int    `json:"contexts"`
	Pops              int    `json:"pops"`
	ExplicitOffsets   int    `json:"explicitOffsets"`
	RejectedMutations int    `json:"rejectedMutations"`
}

func runWindowsVersion2EpilogProbe(toolchain cdriver.WindowsHostToolchain) ([]version2EpilogObservation, error) {
	input := windowsunwindprobe.EpilogFairySource
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil {
		return nil, err
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return nil, fmt.Errorf("v2 epilog parse: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil {
		return nil, err
	}
	if !checked.Verified || len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return nil, fmt.Errorf("v2 epilog MIR: %+v", checked.Semantic.Diagnostics)
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return nil, err
	}
	runtimeSource, err := cruntime.WindowsRunSource(cruntime.NativeTrapExitBase)
	if err != nil {
		return nil, err
	}
	companion := windowsunwindprobe.EpilogCompanionC + runtimeSource
	var rows []version2EpilogObservation
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCABIProbe(toolchain, profile, unit.Source, companion)
		if err != nil {
			return nil, err
		}
		unwind, err := pecoff.InspectAMD64Unwind(artifact.Bytes)
		if err != nil {
			return nil, err
		}
		outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
		if err != nil || outcome.ExitCode != 42 || outcome.Stderr != "" {
			return nil, fmt.Errorf("v2 epilog %s: %+v, %v", profile.Name, outcome, err)
		}
		row := version2EpilogObservation{Profile: profile.Name, Toolchain: toolchain.Version}
		if err := json.Unmarshal([]byte(outcome.Stdout), &row); err != nil {
			return nil, err
		}
		if row.Functions != unwind.Version2Infos || row.Functions == 0 || row.Epilogs < 3 || row.Contexts <= row.Epilogs || row.Pops < 2 || row.ExplicitOffsets == 0 {
			return nil, fmt.Errorf("v2 epilog incomplete observation: %+v", row)
		}
		row.RejectedMutations, err = rejectVersion2EpilogMutations(artifact.Bytes, unwind)
		if err != nil {
			return nil, err
		}
		row.FairySHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(input)))
		row.GeneratedSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(unit.Source)))
		row.CompanionSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(companion)))
		row.ImageSHA256 = fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes))
		rows = append(rows, row)
	}
	return rows, nil
}

// Mutate real final-image records. Invalid records never reach the OS unwinder.
func rejectVersion2EpilogMutations(image []byte, report pecoff.UnwindReport) (int, error) {
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	count := 0
	for _, fn := range report.Functions {
		for _, section := range file.Sections {
			if fn.UnwindInfoRVA < section.VirtualAddress || fn.UnwindInfoRVA-section.VirtualAddress+6 > section.Size {
				continue
			}
			offset := int(section.Offset + fn.UnwindInfoRVA - section.VirtualAddress)
			if image[offset]&7 != 2 {
				break
			}
			for _, invalidSize := range []byte{0, 255} {
				mutated := bytes.Clone(image)
				mutated[offset+4] = invalidSize
				if _, err := pecoff.InspectAMD64Unwind(mutated); err == nil || !strings.Contains(err.Error(), "version-2 epilog") {
					return count, fmt.Errorf("v2 epilog mutation accepted or unrelated rejection: %v", err)
				}
				count++
			}
			break
		}
	}
	if count != report.Version2Infos*2 {
		return count, fmt.Errorf("v2 mutation coverage: %d for %d records", count, report.Version2Infos)
	}
	return count, nil
}
