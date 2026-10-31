//go:build windows && amd64

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
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

func TestWindowsHandlerScopeData(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	input := windowsunwindprobe.ScopeFairySource
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
	runtimeSource, err := cruntime.WindowsRunSource(cruntime.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	companion := windowsunwindprobe.ScopeCompanionC + runtimeSource
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			a, err := cdriver.BuildWindowsHostWithLibraryLog(tc, profile, unit.Source, companion)
			if err != nil {
				t.Fatal(err)
			}
			var body pecoff.AMD64ScopeReport
			totalFields, totalRecords, rejected := 0, 0, 0
			for _, symbol := range []string{"fy_scope_body", "fy_scope_outer", "fy_scope_constant"} {
				var report pecoff.AMD64ScopeReport
				var spec pecoff.AMD64HandlerLinkSpec
				matches := 0
				var failures []string
				for _, handler := range []string{"__GSHandlerCheck_SEH", "__C_specific_handler"} {
					candidate := pecoff.AMD64HandlerLinkSpec{SourceSymbol: symbol, SourceObject: "runtime.obj", HandlerSymbol: handler}
					r, err := pecoff.InspectAMD64ScopeTable(a.Objects[1].Bytes, a.Bytes, a.LinkMap, candidate)
					if err == nil {
						report = r
						spec = candidate
						matches++
					} else {
						failures = append(failures, err.Error())
					}
				}
				if matches != 1 {
					t.Fatalf("scope %s matches=%d %v", symbol, matches, failures)
				}
				if symbol == "fy_scope_body" {
					body = report
					if profile == cdriver.LowProfile && report.GSWord == nil {
						t.Fatal("GS control lost its trailer")
					}
				}
				if symbol == "fy_scope_constant" {
					found := false
					for _, record := range report.Records {
						if record.Handler == 1 && record.Jump != 0 {
							found = true
						}
					}
					t.Logf("constant execute-handler source uses literal sentinel=%t", found)
				}
				totalRecords += len(report.Records)
				totalFields += report.RelocationFields
				pe, err := pecoff.InspectExecutable(a.Bytes)
				if err != nil {
					t.Fatal(err)
				}
				length := 4 + len(report.Records)*16
				if report.GSWord != nil {
					length += 4
				}
				for at := 0; at < length; at += 4 {
					mutated := bytes.Clone(a.Bytes)
					changed := false
					for _, section := range pe.Sections {
						if report.DataRVA >= section.VirtualAddress && uint64(report.DataRVA-section.VirtualAddress)+uint64(length) <= uint64(section.RawSize) {
							offset := section.RawOffset + report.DataRVA - section.VirtualAddress + uint32(at)
							mutated[offset] ^= 1
							changed = true
							break
						}
					}
					if !changed {
						t.Fatal("missing payload mutation site")
					}
					if _, err := pecoff.InspectAMD64HandlerLink(a.Objects[1].Bytes, mutated, a.LinkMap, spec); err != nil {
						t.Fatalf("payload mutation changed the already checked header: %v", err)
					}
					if _, err := pecoff.InspectAMD64ScopeTable(a.Objects[1].Bytes, mutated, a.LinkMap, spec); err == nil {
						t.Fatal("accepted independently changed scope payload")
					}
					rejected++
				}
				object := bytes.Clone(a.Objects[1].Bytes)
				object[report.ObjectDataOffset+4] ^= 1
				if _, err := pecoff.InspectAMD64ScopeTable(object, a.Bytes, a.LinkMap, spec); err == nil {
					t.Fatal("accepted changed scope source addend")
				}
				rejected++
				for _, count := range []uint32{0, 129} {
					object := bytes.Clone(a.Objects[1].Bytes)
					binary.LittleEndian.PutUint32(object[report.ObjectDataOffset:], count)
					if _, err := pecoff.InspectAMD64ScopeTable(object, a.Bytes, a.LinkMap, spec); err == nil {
						t.Fatal("accepted out-of-budget scope count")
					}
					rejected++
				}
				gs := "none"
				if report.GSWord != nil {
					gs = fmt.Sprintf("%#x", *report.GSWord)
				}
				t.Logf("%s handler=%s data=%#x records=%+v fields=%d GS=%s", symbol, spec.HandlerSymbol, report.DataRVA, report.Records, report.RelocationFields, gs)
			}
			run, err := cdriver.ExecuteWindowsMSVCTarget(a)
			if err != nil || run.ExitCode != 42 || run.Stderr != "" {
				t.Fatalf("scope execution %+v %v", run, err)
			}
			var live struct{ DataRVA, HandlerRVA, ControlRVA, Count, Normal, Inner, Outer, Abnormal uint32 }
			if err := json.Unmarshal([]byte(run.Stdout), &live); err != nil {
				t.Fatal(err)
			}
			if live.DataRVA != body.DataRVA || live.HandlerRVA != body.Link.LinkedHandlerRVA || live.Count != uint32(len(body.Records)) || live.Normal != 3 || live.Inner != 123 || live.Outer != 1435 || live.Abnormal != 1 {
				t.Fatalf("live scope mismatch %+v", live)
			}
			covering := 0
			for _, r := range body.Records {
				if live.ControlRVA >= r.Begin && live.ControlRVA < r.End {
					covering++
				}
			}
			if covering < 2 {
				t.Fatal("captured PC not in both nested scopes")
			}
			t.Logf("MSVC %s records=%d fields=%d rejected=%d live=%+v", tc.Version, totalRecords, totalFields, rejected, live)
			t.Logf("hashes fairy=%x unit=%x companion=%x object=%x image=%x", sha256.Sum256([]byte(input)), sha256.Sum256([]byte(unit.Source)), sha256.Sum256([]byte(companion)), sha256.Sum256(a.Objects[1].Bytes), sha256.Sum256(a.Bytes))
		})
	}
}
