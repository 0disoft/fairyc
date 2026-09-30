package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Fixed decimal vectors are independent of the Fairy parser and formatter.
// Errors must produce no partial summary and must return 1 rather than trap.
func TestI64CSV(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC")
	}
	root := filepath.Join("..", "..", "examples", "csv_aggregator_i64")
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", "i64-csv")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	sources := make(map[string]string)
	for _, name := range []string{"fairy.toml", "src/main.fy", "src/csv.fy", "src/numbers.fy", "src/diagnostics.fy"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		sources[name] = hex.EncodeToString(sum[:])
	}
	cases := []struct {
		name, stdout, message string
		record                int
	}{
		{"wide-total", "count=2 sum=4294967294 min=2147483647 max=2147483647\n", "", 0},
		{"wide-values", "count=3 sum=10000000000 min=-5000000000 max=10000000000\n", "", 0},
		{"minimum", "count=1 sum=-9223372036854775808 min=-9223372036854775808 max=-9223372036854775808\n", "", 0},
		{"maximum", "count=1 sum=9223372036854775807 min=9223372036854775807 max=9223372036854775807\n", "", 0},
		{"edges", "count=2 sum=-1 min=-9223372036854775808 max=9223372036854775807\n", "", 0},
		{"signed-zero", "count=3 sum=0 min=0 max=0\n", "", 0},
		{"quoted", "count=2 sum=0 min=-5000000000 max=5000000000\n", "", 0},
		{"above-maximum", "", "selected integer is outside i64 range", 2},
		{"below-minimum", "", "selected integer is outside i64 range", 2},
		{"too-many-digits", "", "selected integer is outside i64 range", 2},
		{"positive-overflow", "", "sum exceeds i64 range", 3},
		{"negative-overflow", "", "sum exceeds i64 range", 3},
		{"late-overflow", "", "sum exceeds i64 range", 4},
		{"syntax", "", "selected value must be a signed decimal integer", 2},
		{"empty-number", "", "empty numeric value", 2},
	}
	var evidence []map[string]any
	var images []map[string]any
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var diagnostics bytes.Buffer
		artifact, code := compileWindowsProgram(checked.MIR, tokenScannerEntry(t, checked), target.X8664WindowsMSVC, profile, &diagnostics)
		if code != ExitSuccess {
			t.Fatalf("%s: %s", profile.Name, diagnostics.String())
		}
		sum := sha256.Sum256(artifact.Bytes)
		if err := os.WriteFile(filepath.Join(evidenceRoot, profile.Name+".exe"), artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		images = append(images, map[string]any{"profile": profile.Name, "exeSHA256": hex.EncodeToString(sum[:]), "exeBytes": len(artifact.Bytes), "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version})
		for _, tc := range cases {
			t.Run(profile.Name+"/"+tc.name, func(t *testing.T) {
				inputPath, err := filepath.Abs(filepath.Join(root, "samples", tc.name+".csv"))
				if err != nil {
					t.Fatal(err)
				}
				input, err := os.ReadFile(inputPath)
				if err != nil {
					t.Fatal(err)
				}
				inputSum := sha256.Sum256(input)
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInputAndArgs(artifact, strings.NewReader(""), []string{inputPath, "value"})
				if err != nil {
					t.Fatal(err)
				}
				wantCode, wantError := 0, ""
				if tc.message != "" {
					wantCode = 1
					wantError = fmt.Sprintf("%s:record=%d:line=%d: error: %s\n", inputPath, tc.record, tc.record, tc.message)
				}
				evidence = append(evidence, map[string]any{"case": tc.name, "profile": profile.Name, "inputSHA256": hex.EncodeToString(inputSum[:]), "stdout": outcome.Stdout, "stderr": outcome.Stderr, "exitCode": outcome.ExitCode, "expectedStdout": tc.stdout, "expectedStderr": wantError, "expectedExitCode": wantCode})
				if outcome.ExitCode != int32(wantCode) || outcome.Stdout != tc.stdout || outcome.Stderr != wantError {
					t.Fatalf("want code=%d stdout=%q stderr=%q; got %+v", wantCode, tc.stdout, wantError, outcome)
				}
			})
		}
	}
	if t.Failed() {
		return
	}
	encoded, err := json.MarshalIndent(map[string]any{"schema": "fairyc.i64-csv/v1", "sdk": sdkstdlib.Version, "sourcesSHA256": sources, "images": images, "cases": evidence}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
