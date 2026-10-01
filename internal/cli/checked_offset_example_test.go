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

// Fixed expected ranges are independent of the example's parser and arithmetic.
// Build once per profile, then exercise the actual argv path without allocating a file.
func TestCheckedOffsetExampleNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC")
	}
	root := filepath.Join("..", "..", "examples", "checked_offset_quickstart")
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	definition, err := sdkstdlib.Load()
	if err != nil {
		t.Fatal(err)
	}
	hash := func(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
	sources := map[string]string{}
	for _, p := range []string{"fairy.toml", "src/main.fy", "src/numbers.fy"} {
		data, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			t.Fatal(err)
		}
		sources[p] = hash(data)
	}
	cases := []struct {
		name            string
		args            []string
		stdout, message string
		code            int32
	}{
		{"interior", []string{"10", "3", "4"}, "length=10 offset=3 count=4 end=7\n", "", 0},
		{"whole-range", []string{"10", "0", "10"}, "length=10 offset=0 count=10 end=10\n", "", 0},
		{"empty-range", []string{"10", "3", "0"}, "length=10 offset=3 count=0 end=3\n", "", 0},
		{"empty-at-end", []string{"10", "10", "0"}, "length=10 offset=10 count=0 end=10\n", "", 0},
		{"zero-length", []string{"0", "0", "0"}, "length=0 offset=0 count=0 end=0\n", "", 0},
		{"last-byte", []string{"10", "9", "1"}, "length=10 offset=9 count=1 end=10\n", "", 0},
		{"wide-values", []string{"5000000000", "3000000000", "2000000000"}, "length=5000000000 offset=3000000000 count=2000000000 end=5000000000\n", "", 0},
		{"maximum-whole", []string{"9223372036854775807", "0", "9223372036854775807"}, "length=9223372036854775807 offset=0 count=9223372036854775807 end=9223372036854775807\n", "", 0},
		{"maximum-end", []string{"9223372036854775807", "9223372036854775807", "0"}, "length=9223372036854775807 offset=9223372036854775807 count=0 end=9223372036854775807\n", "", 0},
		{"maximum-last-byte", []string{"9223372036854775807", "9223372036854775806", "1"}, "length=9223372036854775807 offset=9223372036854775806 count=1 end=9223372036854775807\n", "", 0},
		{"negative-zero-leading-zeros", []string{"00010", "-0", "01"}, "length=10 offset=0 count=1 end=1\n", "", 0},
		{"negative-length", []string{"-1", "0", "0"}, "", "LENGTH: negative value", 1},
		{"negative-offset", []string{"10", "-1", "0"}, "", "OFFSET: negative value", 1},
		{"negative-count", []string{"10", "0", "-1"}, "", "COUNT: negative value", 1},
		{"minimum-conversion-error", []string{"10", "-9223372036854775808", "0"}, "", "OFFSET: negative value", 1},
		{"offset-after-end", []string{"10", "11", "0"}, "", "offset exceeds length", 1},
		{"end-with-data", []string{"10", "10", "1"}, "", "count exceeds remaining length", 1},
		{"count-after-end", []string{"10", "3", "8"}, "", "count exceeds remaining length", 1},
		{"empty-offset-after-end", []string{"0", "1", "0"}, "", "offset exceeds length", 1},
		{"empty-with-data", []string{"0", "0", "1"}, "", "count exceeds remaining length", 1},
		{"sum-beyond-i64", []string{"9223372036854775807", "1", "9223372036854775807"}, "", "count exceeds remaining length", 1},
		{"two-maxima", []string{"9223372036854775807", "9223372036854775807", "9223372036854775807"}, "", "count exceeds remaining length", 1},
		{"above-i64", []string{"9223372036854775808", "0", "0"}, "", "LENGTH: integer is outside i64 range", 1},
		{"below-i64", []string{"10", "-9223372036854775809", "0"}, "", "OFFSET: integer is outside i64 range", 1},
		{"usize-max-parse-error", []string{"10", "0", "18446744073709551615"}, "", "COUNT: integer is outside i64 range", 1},
		{"empty-number", []string{"10", "", "0"}, "", "OFFSET: empty integer", 1},
		{"bare-sign", []string{"10", "-", "0"}, "", "OFFSET: expected signed decimal integer", 1},
		{"plus-sign", []string{"10", "+1", "0"}, "", "OFFSET: expected signed decimal integer", 1},
		{"whitespace", []string{"10", " 1", "0"}, "", "OFFSET: expected signed decimal integer", 1},
		{"decimal-point", []string{"10", "1.0", "0"}, "", "OFFSET: expected signed decimal integer", 1},
		{"korean", []string{"10", "한글", "0"}, "", "OFFSET: expected signed decimal integer", 1},
		{"too-long", []string{strings.Repeat("0", 21), "0", "0"}, "", "LENGTH: argument exceeds 20 UTF-8 bytes", 1},
		{"max-argument-bytes", []string{strings.Repeat("0", 20), "0", "0"}, "length=0 offset=0 count=0 end=0\n", "", 0},
		{"missing", nil, "", "usage: checked-offset LENGTH OFFSET COUNT", 2},
		{"extra", []string{"10", "0", "0", "extra"}, "", "usage: checked-offset LENGTH OFFSET COUNT", 2},
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", "checked-offset-example")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	var images, executions []map[string]any
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var diagnostics bytes.Buffer
		artifact, code := compileWindowsProgram(checked.MIR, tokenScannerEntry(t, checked), target.X8664WindowsMSVC, profile, &diagnostics)
		if code != ExitSuccess {
			t.Fatalf("native build: %s", diagnostics.String())
		}
		if err := os.WriteFile(filepath.Join(evidenceRoot, profile.Name+".exe"), artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		images = append(images, map[string]any{"profile": profile.Name, "exeSHA256": hash(artifact.Bytes), "exeBytes": len(artifact.Bytes), "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version})
		for _, tc := range cases {
			t.Run(profile.Name+"/"+tc.name, func(t *testing.T) {
				result, err := cdriver.ExecuteWindowsMSVCTargetWithInputAndArgs(artifact, strings.NewReader(""), tc.args)
				if err != nil {
					t.Fatal(err)
				}
				wantError := ""
				if tc.message != "" {
					wantError = fmt.Sprintf("checked-offset: %s\n", tc.message)
				}
				executions = append(executions, map[string]any{"case": tc.name, "profile": profile.Name, "argv": tc.args, "stdout": result.Stdout, "stderr": result.Stderr, "exitCode": result.ExitCode, "expectedStdout": tc.stdout, "expectedStderr": wantError, "expectedExitCode": tc.code})
				if result.ExitCode != tc.code || result.Stdout != tc.stdout || result.Stderr != wantError {
					t.Fatalf("want code=%d stdout=%q stderr=%q; got %+v", tc.code, tc.stdout, wantError, result)
				}
			})
		}
	}
	if t.Failed() {
		return
	}
	data, err := json.MarshalIndent(map[string]any{"schema": "fairyc.checked-offset-example/v1", "sdk": sdkstdlib.Version, "sdkSourceIdentity": definition.SourceIdentity, "language": sdkstdlib.LanguageRevision, "runtime": sdkstdlib.RuntimeABI, "target": target.X8664WindowsMSVC, "sourceSHA256": sources, "images": images, "executions": executions}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
