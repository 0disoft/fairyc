//go:build windows && amd64

package cdriver_test

import (
	"bytes"
	"encoding/binary"
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

func TestGeneratedUnwindSelection(t *testing.T) {
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
				t.Fatalf("native example: %+v %v", outcome, err)
			}
			if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "unit.obj" || artifact.Objects[1].Name != "runtime.obj" {
				t.Fatal("missing generated objects")
			}
			first, second := artifact.Objects[0].Bytes, artifact.Objects[1].Bytes
			report, err := pecoff.InspectGeneratedUnwindSelection(first, second, artifact.Bytes, artifact.LinkMap, artifact.LinkerVerbose)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("children=%d retained=%d discarded=%d RVA-fields=%d pdata=%d xdata-bytes=%d", len(report.Children), report.Retained, report.Discarded, report.RelocationFields, report.MatchedPDataEntries, report.MatchedXDataBytes)
			if profile != cdriver.OptimizedProfile {
				return
			}
			if report.Retained == 0 || report.Discarded == 0 || report.RelocationFields == 0 || report.MatchedPDataEntries == 0 {
				t.Fatal("missing retained/discarded unwind coverage")
			}
			var retained, discarded pecoff.GeneratedUnwindChild
			for _, child := range report.Children {
				if child.Retained && child.Kind == ".xdata" && retained.Symbol == "" {
					retained = child
				}
				if !child.Retained && discarded.Symbol == "" {
					discarded = child
				}
			}
			withoutMap := filterSelectionLines(artifact.LinkMap, func(line string) bool {
				fields := strings.Fields(line)
				return len(fields) >= 4 && fields[1] == retained.Symbol && fields[len(fields)-1] == retained.Object
			})
			if bytes.Equal(withoutMap, artifact.LinkMap) {
				t.Fatal("unwind map mutation removed no row")
			}
			if _, err = pecoff.InspectGeneratedUnwindSelection(first, second, artifact.Bytes, withoutMap, artifact.LinkerVerbose); err == nil {
				t.Fatal("accepted missing unwind map row")
			}
			withoutDiscard := filterSelectionLines(artifact.LinkerVerbose, func(line string) bool {
				return strings.TrimSpace(line) == "Discarded "+discarded.Symbol+" from "+discarded.Object
			})
			if bytes.Equal(withoutDiscard, artifact.LinkerVerbose) {
				t.Fatal("unwind discard mutation removed no row")
			}
			if _, err = pecoff.InspectGeneratedUnwindSelection(first, second, artifact.Bytes, artifact.LinkMap, withoutDiscard); err == nil {
				t.Fatal("accepted missing child discard record")
			}
			pe, err := pecoff.InspectExecutable(artifact.Bytes)
			if err != nil {
				t.Fatal(err)
			}
			var changedImage bool
			for _, child := range report.Children {
				if !child.Retained || child.Kind != ".xdata" {
					continue
				}
				for _, section := range pe.Sections {
					if child.ImageRVA < section.VirtualAddress || uint64(child.ImageRVA-section.VirtualAddress)+2 > uint64(section.RawSize) {
						continue
					}
					at := section.RawOffset + child.ImageRVA - section.VirtualAddress + 1
					if artifact.Bytes[at] == 255 {
						continue
					}
					mutated := slices.Clone(artifact.Bytes)
					mutated[at]++
					// Keep the record structurally valid: the new check must detect its
					// disagreement with the actual source unwind information.
					if _, err = pecoff.InspectAMD64Unwind(mutated); err != nil {
						continue
					}
					if _, err = pecoff.InspectGeneratedUnwindSelection(first, second, mutated, artifact.LinkMap, artifact.LinkerVerbose); err == nil {
						t.Fatal("accepted changed final unwind information")
					}
					changedImage = true
					break
				}
				if changedImage {
					break
				}
			}
			if !changedImage {
				t.Fatal("no structurally valid unwind mutation")
			}
			var changedPData bool
			for _, child := range report.Children {
				if !child.Retained || child.Kind != ".pdata" {
					continue
				}
				object := first
				if child.Object == "runtime.obj" {
					object = second
				}
				coff, err := pecoff.InspectObject(object)
				if err != nil {
					t.Fatal(err)
				}
				at := coff.Sections[child.SectionIndex-1].RawOffset
				begin, end := binary.LittleEndian.Uint32(object[at:]), binary.LittleEndian.Uint32(object[at+4:])
				if uint64(begin)+1 >= uint64(end) {
					continue
				}
				mutated := slices.Clone(object)
				binary.LittleEndian.PutUint32(mutated[at:], begin+1)
				if _, err = pecoff.InspectObject(mutated); err != nil {
					t.Fatal(err)
				}
				a, b := first, second
				if child.Object == "unit.obj" {
					a = mutated
				} else {
					b = mutated
				}
				if _, err = pecoff.InspectGeneratedUnwindSelection(a, b, artifact.Bytes, artifact.LinkMap, artifact.LinkerVerbose); err == nil {
					t.Fatal("accepted changed source function range")
				}
				changedPData = true
				break
			}
			if !changedPData {
				t.Fatal("no bounded .pdata range mutation")
			}
		})
	}
}
