package cdriver

import (
	"bytes"
	"crypto/sha256"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/pecoff"
)

func TestWindowsDebugMapProbeEvidence(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const source = `
static __declspec(noinline) int triple(int value) { return value * 3; }
int main(void) { return triple(7) != 21; }
`
	release, err := BuildWindowsMSVCProbeWithMap(toolchain, OptimizedProfile, source)
	if err != nil {
		t.Fatal(err)
	}
	debug, err := BuildWindowsMSVCProbeWithDebugMap(toolchain, OptimizedProfile, source)
	if err != nil {
		t.Fatal(err)
	}
	if len(debug.ProgramDatabase) == 0 || len(debug.LinkMap) == 0 {
		t.Fatal("debug build omitted PDB or link map")
	}
	if _, err := pecoff.InspectProgramDatabaseAssociation(debug.Bytes, debug.ProgramDatabase); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []WindowsHostArtifact{release, debug} {
		outcome, runErr := ExecuteWindowsMSVCTarget(artifact)
		if runErr != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
			t.Fatalf("probe execution: %+v %v", outcome, runErr)
		}
	}
	text := func(artifact WindowsHostArtifact) []byte {
		report, inspectErr := pecoff.InspectExecutable(artifact.Bytes)
		if inspectErr != nil {
			t.Fatal(inspectErr)
		}
		for _, section := range report.Sections {
			if section.Name == ".text" {
				return artifact.Bytes[section.RawOffset : section.RawOffset+section.RawSize]
			}
		}
		t.Fatal("missing .text section")
		return nil
	}
	releaseText, debugText := text(release), text(debug)
	t.Logf("release EXE=%d .text=%d SHA256=%x; debug EXE=%d .text=%d SHA256=%x PDB=%d; identical .text=%t",
		len(release.Bytes), len(releaseText), sha256.Sum256(releaseText),
		len(debug.Bytes), len(debugText), sha256.Sum256(debugText), len(debug.ProgramDatabase),
		bytes.Equal(releaseText, debugText))
	releaseUnwind, err := pecoff.InspectAMD64Unwind(release.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	debugUnwind, err := pecoff.InspectAMD64Unwind(debug.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	matchingIntervals := 0
	for i := 0; i < len(releaseUnwind.Functions) && i < len(debugUnwind.Functions); i++ {
		left, right := releaseUnwind.Functions[i], debugUnwind.Functions[i]
		if left.BeginRVA != right.BeginRVA || left.EndRVA != right.EndRVA {
			break
		}
		matchingIntervals++
	}
	t.Logf("release unwind=%d debug unwind=%d matching-prefix intervals=%d",
		len(releaseUnwind.Functions), len(debugUnwind.Functions), matchingIntervals)
	if len(releaseUnwind.Functions) != len(debugUnwind.Functions) || matchingIntervals != len(releaseUnwind.Functions) {
		t.Fatal("debug symbols cannot be transferred to release unwind intervals")
	}
}
