//go:build windows && amd64

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"testing"

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

func TestWindowsFrameChainUnwind(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	input := windowsunwindprobe.ChainLinkFairySource
	p, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || p.AST == nil || len(p.Tree.Diagnostics) != 0 {
		t.Fatalf("parse %v", err)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: p.AST}})
	if err != nil || !checked.Verified {
		t.Fatalf("MIR %v", err)
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	rt, err := cruntime.WindowsRunSource(cruntime.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	companion := windowsunwindprobe.FrameChainCompanionC + rt
	object, err := pecoff.BuildAMD64FrameChainProbe()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			a, err := cdriver.BuildWindowsHostWithOptions(tc, profile, unit.Source, companion, cdriver.WindowsMSVCBuildOptions{CaptureMap: true, DisableICF: true, PrebuiltObjects: []cdriver.WindowsHostObject{{Name: "chain.obj", Bytes: object}}})
			if err != nil {
				t.Fatal(err)
			}
			link, err := pecoff.InspectAMD64ChainLink(object, a.Bytes, a.LinkMap, "chain.obj")
			if err != nil {
				t.Fatal(err)
			}
			if len(link.Records) != 2 || link.RelocationFields != 6 {
				t.Fatal("incomplete framed chain link")
			}
			run, err := cdriver.ExecuteWindowsMSVCTarget(a)
			if err != nil || run.ExitCode != 42 || run.Stderr != "" {
				t.Fatalf("framed execution %+v %v", run, err)
			}
			var row struct{ Contexts, FrameChecks, VectorRestores, RejectedContexts int }
			if err := json.Unmarshal([]byte(run.Stdout), &row); err != nil {
				t.Fatal(err)
			}
			if row.Contexts != 10 || row.FrameChecks != 7 || row.VectorRestores != 3 || row.RejectedContexts != 2 {
				t.Fatalf("incomplete frame observation %+v", row)
			}
			pe, err := pecoff.InspectExecutable(a.Bytes)
			if err != nil {
				t.Fatal(err)
			}
			// The first tail points at the primary record. Change only a valid primary
			// XMM save slot/register, leaving generic structure valid and never loaded.
			primary := link.Records[0].Linked[2]
			rejected := 0
			for _, s := range pe.Sections {
				if primary < s.VirtualAddress || uint64(primary-s.VirtualAddress)+16 > uint64(s.RawSize) {
					continue
				}
				at := s.RawOffset + primary - s.VirtualAddress
				for _, delta := range []uint32{5, 6} {
					image := bytes.Clone(a.Bytes)
					image[at+delta] ^= 0x10
					if _, err := pecoff.InspectAMD64Unwind(image); err != nil {
						t.Fatalf("framed mutation lost structure: %v", err)
					}
					if _, err := pecoff.InspectAMD64ChainLink(object, image, a.LinkMap, "chain.obj"); err == nil {
						t.Fatal("accepted different primary XMM payload")
					}
					rejected++
				}
				break
			}
			if rejected != 2 {
				t.Fatal("missing framed mutation sites")
			}
			t.Logf("MSVC %s contexts=%d frames=%d vectors=%d rejected=%d/%d fields=%d", tc.Version, row.Contexts, row.FrameChecks, row.VectorRestores, rejected, row.RejectedContexts, link.RelocationFields)
			t.Logf("hashes fixture=%x companion=%x object=%x image=%x", sha256.Sum256(object), sha256.Sum256([]byte(companion)), sha256.Sum256(a.Objects[1].Bytes), sha256.Sum256(a.Bytes))
		})
	}
}
