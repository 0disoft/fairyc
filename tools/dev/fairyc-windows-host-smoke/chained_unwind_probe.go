package main

import (
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
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

type chainedUnwindObservation struct {
	Profile                         string `json:"profile"`
	Toolchain                       string `json:"toolchain"`
	FairySHA256                     string `json:"fairySourceSHA256"`
	GeneratedSHA256                 string `json:"generatedCSHA256"`
	CompanionSHA256                 string `json:"companionSourceSHA256"`
	ImageSHA256                     string `json:"imageSHA256"`
	Functions                       int    `json:"functions"`
	UntestedFunctions               int    `json:"untestedFunctions"`
	Contexts                        int    `json:"contexts"`
	MaximumDepth                    int    `json:"maximumDepth"`
	IntegerRestores                 int    `json:"integerRestores"`
	VectorRestores                  int    `json:"vectorRestores"`
	RejectedMutations               int    `json:"rejectedMutations"`
	ZeroPrologFunctions             int    `json:"zeroPrologFunctions"`
	ChainedFunctions                int    `json:"chainedFunctions"`
	ZeroPrologEpilogs               int    `json:"zeroPrologEpilogs"`
	ZeroPrologContexts              int    `json:"zeroPrologContexts"`
	ZeroPrologPops                  int    `json:"zeroPrologPops"`
	ZeroPrologWideAdds              int    `json:"zeroPrologWideAdds"`
	ZeroPrologUnsupportedFrames     int    `json:"zeroPrologUnsupportedFrames"`
	ZeroPrologUnsupportedOperations int    `json:"zeroPrologUnsupportedOperations"`
	ZeroPrologCallTails             int    `json:"zeroPrologCallTails"`
	ZeroPrologOtherInstructions     int    `json:"zeroPrologOtherInstructions"`
	RejectedEpilogMutations         int    `json:"rejectedEpilogMutations"`
	RejectedEpilogContexts          int    `json:"rejectedEpilogContexts"`
}

func runWindowsChainedUnwindProbe(toolchain cdriver.WindowsHostToolchain) ([]chainedUnwindObservation, error) {
	input := windowsunwindprobe.ChainFairySource
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil {
		return nil, err
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return nil, fmt.Errorf("chained unwind parse: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil {
		return nil, err
	}
	if !checked.Verified || len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return nil, fmt.Errorf("chained unwind MIR: %+v", checked.Semantic.Diagnostics)
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return nil, err
	}
	runtimeSource, err := cruntime.WindowsRunSource(cruntime.NativeTrapExitBase)
	if err != nil {
		return nil, err
	}
	companion := windowsunwindprobe.ChainCompanionC + runtimeSource
	var rows []chainedUnwindObservation
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
			return nil, fmt.Errorf("chained unwind %s: %+v, %v", profile.Name, outcome, err)
		}
		row := chainedUnwindObservation{Profile: profile.Name, Toolchain: toolchain.Version}
		if err := json.Unmarshal([]byte(outcome.Stdout), &row); err != nil {
			return nil, err
		}
		if row.Functions < 10 || row.Contexts <= row.Functions || row.MaximumDepth < 2 || row.MaximumDepth > unwind.MaximumChainDepth || row.IntegerRestores == 0 {
			return nil, fmt.Errorf("chained unwind incomplete observation: %+v", row)
		}
		if row.ZeroPrologEpilogs == 0 || row.ZeroPrologContexts <= row.ZeroPrologEpilogs || row.ZeroPrologPops == 0 || row.ZeroPrologWideAdds == 0 ||
			row.ZeroPrologFunctions != row.ZeroPrologEpilogs+row.ZeroPrologUnsupportedFrames+row.ZeroPrologUnsupportedOperations+row.ZeroPrologCallTails+row.ZeroPrologOtherInstructions ||
			row.RejectedEpilogMutations != 6 || row.RejectedEpilogContexts != 2 {
			return nil, fmt.Errorf("zero-prolog incomplete observation: %+v", row)
		}
		if err := verifyChainedUnwindCoverage(artifact.Bytes, unwind, row); err != nil {
			return nil, err
		}
		row.RejectedMutations, err = rejectChainedUnwindMutations(artifact.Bytes, unwind)
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

// Cross-check the native receipt against the independently inspected file, so
// new linked fragments cannot silently disappear from the tested/untested split.
func verifyChainedUnwindCoverage(image []byte, report pecoff.UnwindReport, row chainedUnwindObservation) error {
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return err
	}
	defer file.Close()
	chained, zero := 0, 0
	for _, fn := range report.Functions {
		found := false
		for _, section := range file.Sections {
			if fn.UnwindInfoRVA < section.VirtualAddress || uint64(fn.UnwindInfoRVA-section.VirtualAddress)+4 > uint64(section.Size) {
				continue
			}
			offset := int(section.Offset + fn.UnwindInfoRVA - section.VirtualAddress)
			if image[offset]>>3&4 != 0 {
				chained++
				if image[offset+1] == 0 {
					zero++
				}
			}
			found = true
			break
		}
		if !found {
			return fmt.Errorf("unmapped chain coverage record %#x", fn.UnwindInfoRVA)
		}
	}
	if row.ChainedFunctions != chained || row.ZeroPrologFunctions != zero || row.Functions+row.ZeroPrologEpilogs+row.UntestedFunctions != chained {
		return fmt.Errorf("chain receipt does not partition %d fragments (%d zero-prolog): %+v", chained, zero, row)
	}
	return nil
}

func rejectChainedUnwindMutations(image []byte, report pecoff.UnwindReport) (int, error) {
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	mapRVA := func(rva uint32) (int, error) {
		for _, section := range file.Sections {
			if rva >= section.VirtualAddress && uint64(rva-section.VirtualAddress)+4 <= uint64(section.Size) {
				return int(section.Offset + rva - section.VirtualAddress), nil
			}
		}
		return 0, fmt.Errorf("unmapped mutation RVA %#x", rva)
	}
	for _, fn := range report.Functions {
		offset, err := mapRVA(fn.UnwindInfoRVA)
		if err != nil {
			return 0, err
		}
		if image[offset]>>3 != 4 || image[offset+3] != 0 || image[offset+2] < 2 {
			continue
		}
		tail := offset + 4 + ((int(image[offset+2])+1)&^1)*2
		parent, err := mapRVA(binary.LittleEndian.Uint32(image[tail+8:]))
		if err != nil {
			return 0, err
		}
		for index, mutation := range []struct {
			want string
			edit func([]byte)
		}{
			{"cycle", func(b []byte) { binary.LittleEndian.PutUint32(b[tail+8:], fn.UnwindInfoRVA) }},
			{"map UNWIND_INFO", func(b []byte) { binary.LittleEndian.PutUint32(b[tail+8:], 0xfffffff0) }},
			{"frame register/offset mismatch", func(b []byte) { b[offset+3] = 5 }},
			{"invalid version/flags", func(b []byte) { b[offset] |= 1 << 3 }},
			{"invalid version/flags", func(b []byte) { b[parent] &^= 7 }},
			{"changes fixed stack/frame", func(b []byte) {
				b[offset+2] = 1
				b[offset+4], b[offset+5] = 0, 2
				copy(b[offset+8:offset+20], image[tail:tail+12])
			}},
		} {
			mutated := bytes.Clone(image)
			mutation.edit(mutated)
			if _, err := pecoff.InspectAMD64Unwind(mutated); err == nil || !strings.Contains(err.Error(), mutation.want) {
				return index, fmt.Errorf("chain mutation %d accepted or unrelated rejection: %v", index, err)
			}
		}
		return 6, nil
	}
	return 0, fmt.Errorf("no linked chained record available for mutations")
}
