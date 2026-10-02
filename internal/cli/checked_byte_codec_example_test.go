package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Fixed hex/error expectations are independent of the Fairy parser and codec.
// Build once per profile, then exercise real process argv.
func TestCheckedByteCodecExampleNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC")
	}
	root := filepath.Join("..", "..", "examples", "checked_byte_codec_quickstart")
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
	for _, p := range []string{"fairy.toml", "src/main.fy", "src/numbers.fy", "src/codec.fy"} {
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
		{"byte-zero", []string{"byte", "0"}, "00\n", "", 0},
		{"byte-max", []string{"byte", "255"}, "ff\n", "", 0},
		{"byte-outside", []string{"byte", "256"}, "", "VALUE: outside byte range 0..255", 1},
		{"byte-i64-max", []string{"byte", "9223372036854775807"}, "", "VALUE: outside byte range 0..255", 1},
		{"byte-negative", []string{"byte", "-1"}, "", "VALUE: negative value", 1},
		{"byte-minimum", []string{"byte", "-9223372036854775808"}, "", "VALUE: negative value", 1},
		{"little-256", []string{"le", "256", "2"}, "00 01\n", "", 0},
		{"little-order", []string{"le", "4660", "2"}, "34 12\n", "", 0},
		{"little-max", []string{"le", "9223372036854775807", "8"}, "ff ff ff ff ff ff ff 7f\n", "", 0},
		{"little-zero-padding", []string{"le", "0", "8"}, "00 00 00 00 00 00 00 00\n", "", 0},
		{"little-one-byte", []string{"le", "255", "1"}, "ff\n", "", 0},
		{"little-high-bits", []string{"le", "256", "1"}, "", "VALUE: does not fit WIDTH bytes", 1},
		{"little-max-seven", []string{"le", "9223372036854775807", "7"}, "", "VALUE: does not fit WIDTH bytes", 1},
		{"little-negative", []string{"le", "-1", "8"}, "", "VALUE: negative value", 1},
		{"little-minimum", []string{"le", "-9223372036854775808", "8"}, "", "VALUE: negative value", 1},
		{"width-zero", []string{"le", "1", "0"}, "", "WIDTH: expected 1..8", 1},
		{"width-nine", []string{"le", "1", "9"}, "", "WIDTH: expected 1..8", 1},
		{"width-negative", []string{"le", "1", "-1"}, "", "WIDTH: expected 1..8", 1},
		{"width-max", []string{"le", "1", "9223372036854775807"}, "", "WIDTH: expected 1..8", 1},
		{"error-order", []string{"le", "-1", "0"}, "", "WIDTH: expected 1..8", 1},
		{"above-i64", []string{"byte", "9223372036854775808"}, "", "VALUE: integer is outside i64 range", 1},
		{"below-i64", []string{"byte", "-9223372036854775809"}, "", "VALUE: integer is outside i64 range", 1},
		{"overflow-before-syntax", []string{"byte", "9223372036854775808x"}, "", "VALUE: integer is outside i64 range", 1},
		{"syntax-before-overflow", []string{"byte", "922337203685477580x8"}, "", "VALUE: expected signed decimal integer", 1},
		{"leading-zeros", []string{"le", "00256", "02"}, "00 01\n", "", 0},
		{"negative-zero", []string{"byte", "-0"}, "00\n", "", 0},
		{"empty", []string{"byte", ""}, "", "VALUE: empty integer", 1},
		{"bare-sign", []string{"byte", "-"}, "", "VALUE: expected signed decimal integer", 1},
		{"plus", []string{"byte", "+1"}, "", "VALUE: expected signed decimal integer", 1},
		{"space", []string{"byte", " 1"}, "", "VALUE: expected signed decimal integer", 1},
		{"decimal", []string{"byte", "1.0"}, "", "VALUE: expected signed decimal integer", 1},
		{"korean", []string{"byte", "한글"}, "", "VALUE: expected signed decimal integer", 1},
		{"width-syntax", []string{"le", "1", "x"}, "", "WIDTH: expected signed decimal integer", 1},
		{"width-overflow", []string{"le", "1", "9223372036854775808"}, "", "WIDTH: integer is outside i64 range", 1},
		{"too-long", []string{"byte", strings.Repeat("0", 21)}, "", "VALUE: argument exceeds 20 UTF-8 bytes", 1},
		{"max-length", []string{"byte", strings.Repeat("0", 20)}, "00\n", "", 0},
		{"missing", nil, "", "usage: checked-byte-codec byte VALUE | le VALUE WIDTH", 2},
		{"missing-width", []string{"le", "1"}, "", "usage: checked-byte-codec byte VALUE | le VALUE WIDTH", 2},
		{"extra", []string{"byte", "1", "2"}, "", "usage: checked-byte-codec byte VALUE | le VALUE WIDTH", 2},
		{"unknown-mode", []string{"other", "1"}, "", "usage: checked-byte-codec byte VALUE | le VALUE WIDTH", 2},
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", "checked-byte-codec-example")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	evidenceCwd, err := filepath.Abs(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	compilerPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	compilerBytes, err := os.ReadFile(compilerPath)
	if err != nil {
		t.Fatal(err)
	}
	buildInfo := "unavailable"
	if info, ok := debug.ReadBuildInfo(); ok {
		buildInfo = info.String()
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
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, filepath.Join(evidenceCwd, profile.Name+".exe"), tc.args...)
				command.Dir = evidenceCwd
				command.Stdin = strings.NewReader("")
				var stdout, stderr bytes.Buffer
				command.Stdout, command.Stderr = &stdout, &stderr
				runErr := command.Run()
				if ctx.Err() != nil {
					t.Fatal(ctx.Err())
				}
				if runErr != nil {
					if _, ok := runErr.(*exec.ExitError); !ok {
						t.Fatal(runErr)
					}
				}
				// Go captures bytes; decode as UTF-8 only after explicit validity checks.
				if !utf8.Valid(stdout.Bytes()) || !utf8.Valid(stderr.Bytes()) {
					t.Fatal("invalid UTF-8 output")
				}
				result := cdriver.WindowsHostOutcome{ExitCode: int32(command.ProcessState.ExitCode()), Stdout: stdout.String(), Stderr: stderr.String()}
				wantError := ""
				if tc.message != "" {
					wantError = fmt.Sprintf("checked-byte-codec: %s\n", tc.message)
				}
				executions = append(executions, map[string]any{"case": tc.name, "profile": profile.Name, "argv": command.Args, "cwd": command.Dir, "outputEncoding": "UTF-8", "stdout": result.Stdout, "stderr": result.Stderr, "exitCode": result.ExitCode, "expectedStdout": tc.stdout, "expectedStderr": wantError, "expectedExitCode": tc.code})
				if result.ExitCode != tc.code || result.Stdout != tc.stdout || result.Stderr != wantError {
					t.Fatalf("want code=%d stdout=%q stderr=%q; got %+v", tc.code, tc.stdout, wantError, result)
				}
			})
		}
	}
	if t.Failed() {
		return
	}
	data, err := json.MarshalIndent(map[string]any{"schema": "fairyc.checked-byte-codec-example/v1", "sdk": sdkstdlib.Version, "sdkSourceIdentity": definition.SourceIdentity, "language": sdkstdlib.LanguageRevision, "runtime": sdkstdlib.RuntimeABI, "target": target.X8664WindowsMSVC, "sourceSHA256": sources, "images": images, "executions": executions, "casesPerProfile": len(cases), "compilerIdentity": map[string]any{"kind": "in-process CLI compiler in Go test executable", "executableSHA256": hash(compilerBytes), "goVersion": runtime.Version(), "buildInfo": buildInfo}}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

// Exercise the maintained codec itself with untouched caller-owned sentinel bytes.
func TestCheckedByteCodecExamplePolicy(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	codec, err := os.ReadFile(filepath.Join("..", "..", "examples", "checked_byte_codec_quickstart", "src", "codec.fy"))
	if err != nil {
		t.Fatal(err)
	}
	source := `module main;
import codec as codec;
fn i32 error_code(codec::EncodeError error) {
    return switch(error) {
        case .Negative: { 1 } case .InvalidWidth: { 2 } case .Capacity: { 3 }
        case .HighBits: { 4 } case .Conversion: { 5 }
    };
}
fn void failure(i64 value, usize width, i32 expected) {
    var u8[8] output = [85; 8];
    let bool correct = switch (codec::encode_le(value, width, ref mut output)) {
        case .ok(_): { false } case .err(error): { error_code(error) == expected }
    };
    assert(correct);
    var usize at = 0;
    while (at < output.len) { assert(output[at] == 85); at += 1; }
}
fn i32 main() {
    failure(-1, 8, 1);
    failure(-9223372036854775808, 8, 1);
    failure(1, 0, 2);
    failure(1, 9, 2);
    failure(256, 1, 4);
    failure(9223372036854775807, 7, 4);
    { var u8[1] output = [85; 1];
      let bool capacity = switch (codec::encode_le(1, 2, ref mut output)) {
          case .ok(_): { false } case .err(error): { error_code(error) == 3 }
      }; assert(capacity && output[0] == 85); }
    { var u8[8] output = [85; 8];
      let bool encoded = switch (codec::encode_le(256, 2, ref mut output)) {
          case .ok(n): { n == 2 } case .err(_): { false }
      }; assert(encoded && output[0] == 0 && output[1] == 1);
      var usize at = 2; while (at < 8) { assert(output[at] == 85); at += 1; } }
    return 0;
}
`
	files := map[string][]byte{"fairy.toml": []byte("schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.byte-codec-policy\"\nname = \"byte-codec-policy\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"), "src/codec.fy": codec, "src/main.fy": []byte(source)}
	for p, data := range files {
		if err := os.WriteFile(filepath.Join(root, p), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("policy check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	outcome, err := interpreter.Execute(checked.MIR, tokenScannerEntry(t, checked))
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("policy execution: %v %+v", err, outcome)
	}
	t.Log("7 exact codec errors preserve every sentinel; successful encoding preserves all suffix bytes")
}
