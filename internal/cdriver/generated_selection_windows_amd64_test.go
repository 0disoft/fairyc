//go:build windows && amd64

package cdriver_test

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestGeneratedCOMDATSelection(t *testing.T) {
	workspace, err := workspaceloader.Load(filepath.Join("..", "..", "examples", "std_bytes_quickstart"))
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("source check: %v verified=%v", err, checked.Verified)
	}
	id, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.EmitStorageTransfers(checked.MIR, hir.QualifiedModulePath(id, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := nativesupport.Source(target.X8664WindowsMSVC, unit.ForeignSymbols, nativesupport.Options{SDKStorage: true})
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsHostWithLibraryLog(toolchain, profile, unit.Source, runtime)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("native bytes example: %+v %v", outcome, err)
			}
			if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "unit.obj" || artifact.Objects[1].Name != "runtime.obj" {
				t.Fatal("missing production objects")
			}
			first, second := artifact.Objects[0].Bytes, artifact.Objects[1].Bytes
			inspect := func(image, linkMap, log []byte) (pecoff.GeneratedCOMDATSelection, error) {
				return pecoff.InspectGeneratedCOMDATSelection(first, second, image, linkMap, log)
			}
			report, err := inspect(artifact.Bytes, artifact.LinkMap, artifact.LinkerVerbose)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("parents=%d retained=%d discarded=%d associative-inventory=%d", len(report.Parents), report.Retained, report.Discarded, report.AssociativeSectionInventory)
			if profile != cdriver.OptimizedProfile {
				return
			}
			if report.Retained == 0 || report.Discarded == 0 || report.AssociativeSectionInventory == 0 {
				t.Fatal("optimized source did not exercise selection and discard")
			}
			var retained, discarded pecoff.GeneratedCOMDATParent
			for _, parent := range report.Parents {
				if parent.Retained && retained.Symbol == "" {
					retained = parent
				}
				if !parent.Retained && discarded.Symbol == "" {
					discarded = parent
				}
			}
			withoutMap := filterSelectionLines(artifact.LinkMap, func(line string) bool {
				fields := strings.Fields(line)
				return len(fields) >= 4 && fields[1] == retained.Symbol && fields[len(fields)-1] == retained.Object
			})
			if bytes.Equal(withoutMap, artifact.LinkMap) {
				t.Fatal("map mutation removed no row")
			}
			if _, err = inspect(artifact.Bytes, withoutMap, artifact.LinkerVerbose); err == nil {
				t.Fatal("accepted missing retained map row")
			}
			wanted := "Discarded " + discarded.Symbol + " from " + discarded.Object
			withoutDiscard := filterSelectionLines(artifact.LinkerVerbose, func(line string) bool { return strings.TrimSpace(line) == wanted })
			if bytes.Equal(withoutDiscard, artifact.LinkerVerbose) {
				t.Fatal("discard mutation removed no row")
			}
			if _, err = inspect(artifact.Bytes, artifact.LinkMap, withoutDiscard); err == nil {
				t.Fatal("accepted missing discard record")
			}
			contradictory := append(slices.Clone(artifact.LinkerVerbose), []byte("\nDiscarded "+retained.Symbol+" from "+retained.Object+"\n")...)
			if _, err = inspect(artifact.Bytes, artifact.LinkMap, contradictory); err == nil {
				t.Fatal("accepted simultaneous retention and discard")
			}
			mutated := slices.Clone(artifact.Bytes)
			pe, err := pecoff.InspectExecutable(mutated)
			if err != nil {
				t.Fatal(err)
			}
			var changed bool
			for _, parent := range report.Parents {
				if !parent.Retained {
					continue
				}
				object := first
				if parent.Object == "runtime.obj" {
					object = second
				}
				coff, err := pecoff.InspectObject(object)
				if err != nil {
					t.Fatal(err)
				}
				if coff.Sections[parent.SectionIndex-1].Characteristics&0x20 == 0 {
					continue
				}
				// A function's first opcode must not be a relocated operand.
				for _, r := range coff.RelocationRecords {
					if r.SectionIndex == parent.SectionIndex && r.VirtualAddress == 0 {
						t.Fatal("first opcode is relocated")
					}
				}
				for _, section := range pe.Sections {
					if parent.ImageRVA >= section.VirtualAddress && parent.ImageRVA-section.VirtualAddress < section.RawSize {
						at := section.RawOffset + parent.ImageRVA - section.VirtualAddress
						mutated[at] ^= 1
						changed = true
						break
					}
				}
				if changed {
					break
				}
			}
			if !changed {
				t.Fatal("no retained function to mutate")
			}
			if _, err = inspect(mutated, artifact.LinkMap, artifact.LinkerVerbose); err == nil {
				t.Fatal("accepted changed final machine code")
			}
		})
	}
}

func filterSelectionLines(raw []byte, remove func(string) bool) []byte {
	var kept []string
	for _, line := range strings.Split(string(raw), "\n") {
		if !remove(line) {
			kept = append(kept, line)
		}
	}
	return []byte(strings.Join(kept, "\n"))
}
