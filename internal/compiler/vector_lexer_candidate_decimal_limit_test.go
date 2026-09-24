package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// The benchmark C image narrows only decimal materialization. Its Go-oracle
// reference still accepts larger expansions when built without this define.
func TestPrivateVectorSDKCandidateDecimalLimitCParity(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	core, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "scanner_reference.c"))
	if err != nil {
		t.Fatal(err)
	}
	const define = "#define FC_CANDIDATE_DECIMAL_LIMIT\n"
	if !strings.Contains(scannerCBenchmarkSource(t), define) {
		t.Fatal("C budget image omitted the candidate decimal limit")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" {
		t.Fatal("real scanner source changed")
	}
	program, entry, _ := scannerBudgetFairyProgramWithMain(t, variants[0], candidateDecimalLimitMain)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			cImage, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, define+string(core))
			if err != nil {
				t.Fatal(err)
			}
			for _, spelling := range []string{"1e16383", "1e-16382"} {
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(cImage, scannerCInput([]byte(spelling)))
				if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" ||
					outcome.Stdout != scannerGoObservation([]byte(spelling)) {
					t.Fatalf("C admitted boundary %q: %+v %v", spelling, outcome, err)
				}
			}
			for _, spelling := range []string{"1e16384", "1e-16383"} {
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(cImage, scannerCInput([]byte(spelling)))
				if err != nil || outcome.ExitCode != 4 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
					t.Fatalf("C rejected boundary %q: %+v %v", spelling, outcome, err)
				}
			}
			fairyImage, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
			outcome, err := cdriver.ExecuteWindowsHost(fairyImage)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("Fairy candidate boundary matrix: %+v %v", outcome, err)
			}
		})
	}
}

const candidateDecimalLimitMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
fn i32 scan_status(ref u8[] input) {
    let bytes::Buffer owned = switch (bytes::copy(input)) {
        case .ok(value): { value }
        case .err(_): { return 3; }
    };
    return switch (scanner::collect(owned, false, 32768, 1048576, 256, 16777216)) {
        case .ok(stream): {
            let scanner::Storage usage = scanner::storage(ref stream);
            if (usage.source_bytes == input.len) { 0 } else { 9 }
        }
        case .err(code): { code }
    };
}
fn i32 main() {
    if (scan_status(b"1e16383"[..]) != 0) { return 11; }
    if (scan_status(b"1e-16382"[..]) != 0) { return 12; }
    if (scan_status(b"1e16384"[..]) != 4) { return 13; }
    if (scan_status(b"1e-16383"[..]) != 4) { return 14; }
    return 0;
}`
