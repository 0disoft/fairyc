package main

import (
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
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

type chainedLinkObservation struct {
	Profile           string                      `json:"profile"`
	Toolchain         string                      `json:"toolchain"`
	FairySHA256       string                      `json:"fairySourceSHA256"`
	GeneratedSHA256   string                      `json:"generatedCSHA256"`
	CompanionSHA256   string                      `json:"companionSourceSHA256"`
	ObjectSHA256      string                      `json:"fixtureObjectSHA256"`
	ImageSHA256       string                      `json:"imageSHA256"`
	MapSHA256         string                      `json:"linkMapSHA256"`
	Contexts          int                         `json:"contexts"`
	RejectedMutations int                         `json:"rejectedMutations"`
	Link              pecoff.AMD64ChainLinkReport `json:"link"`
}

func runWindowsChainedLinkProbe(toolchain cdriver.WindowsHostToolchain) ([]chainedLinkObservation, error) {
	input := windowsunwindprobe.ChainLinkFairySource
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil {
		return nil, err
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return nil, fmt.Errorf("chain link parse: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil {
		return nil, err
	}
	if !checked.Verified || len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return nil, fmt.Errorf("chain link MIR rejected")
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return nil, err
	}
	runtimeSource, err := cruntime.WindowsRunSource(cruntime.NativeTrapExitBase)
	if err != nil {
		return nil, err
	}
	companion := windowsunwindprobe.ChainLinkCompanionC + runtimeSource
	object, err := pecoff.BuildAMD64ChainLinkProbe()
	if err != nil {
		return nil, err
	}
	var rows []chainedLinkObservation
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsHostWithOptions(toolchain, profile, unit.Source, companion, cdriver.WindowsMSVCBuildOptions{
			CaptureMap: true, DisableICF: true, PrebuiltObjects: []cdriver.WindowsHostObject{{Name: "chain.obj", Bytes: object}},
		})
		if err != nil {
			return nil, err
		}
		link, err := pecoff.InspectAMD64ChainLink(object, artifact.Bytes, artifact.LinkMap, "chain.obj")
		if err != nil {
			return nil, err
		}
		if len(link.Records) != 2 || link.RelocationFields != 6 {
			return nil, fmt.Errorf("chain link incomplete: %+v", link)
		}
		for _, record := range link.Records {
			for _, addend := range record.Addends {
				if addend == 0 {
					return nil, fmt.Errorf("chain link lost nonzero addends")
				}
			}
		}
		outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
		if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
			return nil, fmt.Errorf("chain link %s: %+v, %v", profile.Name, outcome, err)
		}
		rejected, err := rejectChainedLinkMutations(object, artifact, link)
		if err != nil {
			return nil, err
		}
		hash := func(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
		rows = append(rows, chainedLinkObservation{Profile: profile.Name, Toolchain: toolchain.Version,
			FairySHA256: hash([]byte(input)), GeneratedSHA256: hash([]byte(unit.Source)), CompanionSHA256: hash([]byte(companion)),
			ObjectSHA256: hash(object), ImageSHA256: hash(artifact.Bytes), MapSHA256: hash(artifact.LinkMap), Contexts: 3, RejectedMutations: rejected, Link: link})
	}
	return rows, nil
}

// Keep all three mutations structurally valid. The source object's symbols and
// addends, rather than mere executable ranges, must reject the wrong targets.
func rejectChainedLinkMutations(object []byte, artifact cdriver.WindowsHostArtifact, link pecoff.AMD64ChainLinkReport) (int, error) {
	file, err := pe.NewFile(bytes.NewReader(artifact.Bytes))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	last := link.Records[len(link.Records)-1]
	for _, section := range file.Sections {
		if last.UnwindRVA < section.VirtualAddress || uint64(last.UnwindRVA-section.VirtualAddress)+16 > uint64(section.Size) {
			continue
		}
		tail := int(section.Offset+last.UnwindRVA-section.VirtualAddress) + 4
		for field, value := range []uint32{last.Linked[0] + 1, last.Linked[1] - 1, link.Records[0].Linked[2]} {
			image := bytes.Clone(artifact.Bytes)
			binary.LittleEndian.PutUint32(image[tail+field*4:], value)
			if _, err := pecoff.InspectAMD64Unwind(image); err != nil {
				return field, fmt.Errorf("chain link mutation is not structurally valid: %w", err)
			}
			if _, err := pecoff.InspectAMD64ChainLink(object, image, artifact.LinkMap, "chain.obj"); err == nil || !strings.Contains(err.Error(), "chain tail field") {
				return field, fmt.Errorf("chain link mutation accepted or unrelated rejection: %v", err)
			}
		}
		return 3, nil
	}
	return 0, fmt.Errorf("chain link mutation record unmapped")
}
