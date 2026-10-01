package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

// Literal byte expectations and encoding/binary are independent of the Fairy loop.
// This maintained fixture owns width/high-bit policy separately from SDK conversion.
func checkedByteConversionSource() string {
	var source strings.Builder
	source.WriteString(`module main;
import std::num as num;
fn Result<u8, num::ConversionError> forward(i64 value) {
    let u8 converted = num::u8_from_i64(value)?;
    return .ok(converted);
}
fn void pair(i64 input, u8 expected) {
    let bool direct = switch (num::u8_from_i64(input)) {
        case .ok(v): { v == expected && v as i64 == input } case .err(_): { false }
    };
    let bool propagated = switch (forward(input)) {
        case .ok(v): { v == expected } case .err(_): { false }
    };
    assert(direct && propagated);
}
fn bool expected_error(Result<u8, num::ConversionError> result, bool negative) {
    return switch (result) {
        case .ok(_): { false }
        case .err(e): { switch(e) {
            case .Negative: { negative } case .OutOfRange: { !negative }
        } }
    };
}
fn void failure(i64 input, bool negative) {
    assert(expected_error(num::u8_from_i64(input), negative));
    assert(expected_error(forward(input), negative));
}
// Codes: 1 negative, 2 invalid width/capacity, 3 excess high bits, 4 conversion.
// Validate the full field before writing any byte; no wrap/truncation policy.
fn i32 encode(i64 input, usize width, ref mut u8[] output) {
    if (input < 0) { return 1; }
    if (width == 0 || width > 8 || width > output.len) { return 2; }
    var i64 excess = input; var usize at = 0;
    while (at < width) { excess /= 256; at += 1; }
    if (excess != 0) { return 3; }
    var i64 remaining = input; at = 0;
    while (at < width) {
        let u8 digit = switch (num::u8_from_i64(remaining % 256)) {
            case .ok(v): { v } case .err(_): { return 4; }
        };
        output[at] = digit; remaining /= 256; at += 1;
    }
    return 0;
}
fn void untouched(ref u8[] output) {
    var usize at = 0;
    while (at < output.len) { assert(output[at] == 85); at += 1; }
}
fn i32 main() {
`)
	for value := 0; value <= 255; value++ {
		fmt.Fprintf(&source, "    pair(%d, %d);\n", value, value)
	}
	source.WriteString(`    failure(-1, true);
    failure(-9223372036854775808, true);
    failure(256, false);
    failure(9223372036854775807, false);
    {
        var u8[8] output = [85; 8];
        assert(encode(256, 2, ref mut output) == 0);
        assert(output[0] == 0 && output[1] == 1);
        assert(output[2] == 85 && output[7] == 85);
    }
    {
        var u8[8] output = [85; 8];
        assert(encode(9223372036854775807, 8, ref mut output) == 0);
`)
	var expected [8]byte
	binary.LittleEndian.PutUint64(expected[:], math.MaxInt64)
	for index, value := range expected {
		fmt.Fprintf(&source, "        assert(output[%d] == %d);\n", index, value)
	}
	source.WriteString("    }\n")
	for _, tc := range []struct {
		value string
		width int
		code  int
	}{
		{"-1", 8, 1}, {"-9223372036854775808", 8, 1}, {"1", 0, 2},
		{"1", 9, 2}, {"256", 1, 3}, {"9223372036854775807", 7, 3},
	} {
		fmt.Fprintf(&source, "    { var u8[8] output = [85; 8]; assert(encode(%s, %d, ref mut output) == %d); untouched(ref output); }\n", tc.value, tc.width, tc.code)
	}
	source.WriteString(`    { var u8[1] output = [85; 1]; assert(encode(1, 2, ref mut output) == 2); untouched(ref output); }
    return 0;
}
`)
	return source.String()
}

func TestCheckedByteConversionExecution(t *testing.T) {
	text := checkedByteConversionSource()
	checked, err := workspacecompiler.Check(checkedIntegerConversionWorkspace(t, text))
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	entry := tokenScannerEntry(t, checked)
	encoded, err := mircodec.Encode(checked.MIR, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(decoded, entry)
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("interpreter: %v %+v", err, outcome)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		if _, err := cbackend.Emit(decoded, entry, profile); err != nil {
			t.Fatalf("C emission: %v", err)
		}
	}
	t.Log("MIR round trip/interpreter/C: 256 exact byte pairs, 4 exact errors, 9 caller codec cases")
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC native evidence; interpreter already checked")
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", "checked-byte-conversion")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	hash := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	definition, err := sdkstdlib.Load()
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
	var images []map[string]any
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			var diagnostics bytes.Buffer
			artifact, code := compileWindowsProgram(decoded, entry, target.X8664WindowsMSVC, profile, &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("native build: %s", diagnostics.String())
			}
			result, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
			if err != nil {
				t.Fatal(err)
			}
			if result.ExitCode != 0 || result.Stdout != "" || result.Stderr != "" {
				t.Fatalf("native execution: %+v", result)
			}
			if err := os.WriteFile(filepath.Join(evidenceRoot, profile.Name+".exe"), artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			images = append(images, map[string]any{"profile": profile.Name, "exeSHA256": hash(artifact.Bytes), "exeBytes": len(artifact.Bytes), "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version, "exitCode": result.ExitCode, "stdout": result.Stdout, "stderr": result.Stderr})
		})
	}
	if t.Failed() {
		return
	}
	values := make([]int, 256)
	for index := range values {
		values[index] = index
	}
	receipt := map[string]any{"schema": "fairyc.checked-byte-conversion/v1", "sdk": sdkstdlib.Version, "sdkSourceIdentity": definition.SourceIdentity, "language": sdkstdlib.LanguageRevision, "runtime": sdkstdlib.RuntimeABI, "target": target.X8664WindowsMSVC, "sourceSHA256": hash([]byte(text)), "successValues": values, "errorValues": []string{"-1", "-9223372036854775808", "256", "9223372036854775807"}, "codecCases": 9, "interpreter": "passed after MIR codec round trip", "cEmission": []string{"x86_64-linux-gnu", "x86_64-windows-msvc"}, "compilerIdentity": map[string]any{"kind": "in-process CLI compiler in Go test executable", "executableSHA256": hash(compilerBytes), "goVersion": runtime.Version(), "buildInfo": buildInfo}, "images": images}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "main.fy"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCheckedByteConversionRetainsRejections(t *testing.T) {
	for _, tc := range []struct{ name, source, feature string }{
		{"i64-as-u8", "fn i32 main() { let i64 v = 1; let u8 n = v as u8; return 0; }", "cast from i64 to u8"},
		{"i64-as-u8-unsafe", "fn i32 main() { let i64 v = 1; let u8 n = unsafe { v as u8 }; return 0; }", "cast from i64 to u8"},
		{"u32-as-u8", "fn i32 main() { let u32 v = 1; let u8 n = v as u8; return 0; }", "cast from u32 to u8"},
		{"u32-as-u8-unsafe", "fn i32 main() { let u32 v = 1; let u8 n = unsafe { v as u8 }; return 0; }", "cast from u32 to u8"},
		{"unhandled-result", "import std::num as num; fn i32 main() { num::u8_from_i64(1); return 0; }", ""},
		{"nonexhaustive-error", "import std::num as num; fn i32 main() { let i32 n = switch(num::u8_from_i64(-1)) { case .ok(_): { 0 } case .err(e): { switch(e) { case .Negative: { 1 } } } }; return n; }", "non-exhaustive enum match"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checked, err := workspacecompiler.Check(checkedIntegerConversionWorkspace(t, "module main; "+tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified {
				t.Fatal("excluded source was accepted")
			}
			if tc.feature != "" {
				found := false
				for _, unsupported := range checked.Semantic.Unsupported {
					if unsupported.Feature == tc.feature {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing cast rejection: %+v", checked.Semantic.Unsupported)
				}
			} else if len(checked.Semantic.Diagnostics) == 0 {
				t.Fatalf("missing Result/exhaustiveness diagnostic: %+v", checked.Semantic.Unsupported)
			}
		})
	}
}
