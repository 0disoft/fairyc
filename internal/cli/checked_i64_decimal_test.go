package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

type decimalOracleCase struct {
	InputHex string `json:"inputHex"`
	Value    string `json:"value"`
	Error    string `json:"error"`
}

// Exact positive big.Int magnitudes, not the SDK's negative accumulator guard.
// Compare after each digit to independently preserve first-failure ordering.
func decimalOracle(raw []byte) decimalOracleCase {
	result := decimalOracleCase{InputHex: hex.EncodeToString(raw), Value: "0"}
	if len(raw) == 0 {
		result.Error = "Empty"
		return result
	}
	negative, at := raw[0] == '-', 0
	if negative {
		at = 1
	}
	if at == len(raw) {
		result.Error = "InvalidDecimal"
		return result
	}
	limit := new(big.Int).Lsh(big.NewInt(1), 63)
	if !negative {
		limit.Sub(limit, big.NewInt(1))
	}
	value := new(big.Int)
	for ; at < len(raw); at++ {
		b := raw[at]
		if b < '0' || b > '9' {
			result.Error = "InvalidDecimal"
			return result
		}
		value.Mul(value, big.NewInt(10)).Add(value, big.NewInt(int64(b-'0')))
		if value.Cmp(limit) > 0 {
			result.Error = "OutOfRange"
			return result
		}
	}
	if negative {
		value.Neg(value)
	}
	result.Value = value.String()
	return result
}

func decimalOracleCases() []decimalOracleCase {
	inputs := []string{"", "-", "0", "-0", "000256", "-0001", "+1", "--1", "-+1", "1-", " 1", "1 ", "1\t0", "1\n", "1\r", "0x10", "1_0", "1.0", "1,0", "1\x00", "\x7f", "\x80", "\xc0\xaf", "\xff", "\xef\xbb\xbf0", "１２", "한글", "9223372036854775807", "-9223372036854775808", "9223372036854775808", "-9223372036854775809", "9223372036854775808x", "922337203685477580x8", "-9223372036854775809x", "-922337203685477580x9", strings.Repeat("0", 96) + "123", "-" + strings.Repeat("0", 96) + "9223372036854775808", strings.Repeat("9", 80), "1" + strings.Repeat("0", 40)}
	power := big.NewInt(1)
	for exponent := 0; exponent <= 19; exponent++ {
		for _, delta := range []int64{-1, 0, 1} {
			n := new(big.Int).Add(power, big.NewInt(delta))
			inputs = append(inputs, n.String(), new(big.Int).Neg(n).String())
		}
		power.Mul(power, big.NewInt(10))
	}
	seen := map[string]bool{}
	var cases []decimalOracleCase
	for _, input := range inputs {
		if !seen[input] {
			seen[input] = true
			cases = append(cases, decimalOracle([]byte(input)))
		}
	}
	return cases
}

func checkedI64DecimalSource(cases []decimalOracleCase) string {
	var source strings.Builder
	source.WriteString(`module main;
import std::num as num;
fn Result<i64, num::ParseI64Error> forward(ref u8[] raw) {
    let i64 value = num::parse_i64_decimal(raw)?;
    return .ok(value);
}
fn bool expected(Result<i64, num::ParseI64Error> result, i32 code, i64 value) {
    return switch (result) {
        case .ok(v): { code == 0 && v == value }
        case .err(e): { switch (e) {
            case .Empty: { code == 1 }
            case .InvalidDecimal: { code == 2 }
            case .OutOfRange: { code == 3 }
        } }
    };
}
fn void check(ref u8[] raw, i32 code, i64 value) {
    assert(expected(num::parse_i64_decimal(raw), code, value));
    assert(expected(forward(raw), code, value));
}
fn void same(ref u8[] left, ref u8[] right) {
    assert(left.len == right.len);
    var usize at = 0;
    while (at < left.len) { assert(left[at] == right[at]); at += 1; }
}
fn i32 main() {
`)
	for _, tc := range cases {
		raw, _ := hex.DecodeString(tc.InputHex)
		padded := append(append([]byte{85}, raw...), 170)
		literal := func(b []byte) string {
			parts := make([]string, len(b))
			for i, v := range b {
				parts[i] = fmt.Sprint(v)
			}
			return "[" + strings.Join(parts, ", ") + "]"
		}
		code := map[string]int{"": 0, "Empty": 1, "InvalidDecimal": 2, "OutOfRange": 3}[tc.Error]
		// Empty full view uses a zero-length subview of a real owner; no zero-sized array.
		if len(raw) > 0 {
			fmt.Fprintf(&source, "    { var u8[%d] raw = %s; let u8[%d] saved = %s; check(ref raw, %d, %s); same(ref raw, ref saved); }\n", len(raw), literal(raw), len(raw), literal(raw), code, tc.Value)
		}
		fmt.Fprintf(&source, "    { var u8[%d] owner = %s; let u8[%d] saved = %s; let ref u8[] full = ref owner; check(full[1..%d], %d, %s); same(full, ref saved); }\n", len(padded), literal(padded), len(padded), literal(padded), len(raw)+1, code, tc.Value)
	}
	source.WriteString("    return 0;\n}\n")
	return source.String()
}

func TestCheckedI64DecimalExecution(t *testing.T) {
	cases := decimalOracleCases()
	// Pin decisive oracle contracts as well as comparing generated programs.
	for input, want := range map[string]string{"9223372036854775808x": "OutOfRange", "922337203685477580x8": "InvalidDecimal", "-9223372036854775809x": "OutOfRange"} {
		if got := decimalOracle([]byte(input)); got.Error != want {
			t.Fatalf("oracle %q: %+v", input, got)
		}
	}
	text := checkedI64DecimalSource(cases)
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
	var linuxUnit cbackend.Unit
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(decoded, entry, profile)
		if err != nil {
			t.Fatal(err)
		}
		if profile.Target == target.X8664LinuxGNU {
			linuxUnit = unit
		}
	}
	t.Logf("%d independent oracle inputs; full/subviews, unchanged owners, direct and propagated Results passed after MIR round trip", len(cases))
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		runLinuxSDKHelperExecution(t, linuxUnit)
		return
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux GNU or Windows MSVC amd64; interpreter and emission already checked")
	}
	evidenceRoot, err := filepath.Abs(filepath.Join("..", "..", ".cache", "checked-i64-decimal"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	hash := func(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
	definition, err := sdkstdlib.Load()
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	compilerBytes, err := os.ReadFile(executable)
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
			exe := filepath.Join(evidenceRoot, profile.Name+".exe")
			if err := os.WriteFile(exe, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, exe)
			command.Dir = evidenceRoot
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			runErr := command.Run()
			// Native streams are byte pipes explicitly decoded as UTF-8, no codepage fallback.
			if !utf8.Valid(stdout.Bytes()) || !utf8.Valid(stderr.Bytes()) {
				t.Fatal("invalid UTF-8 native output")
			}
			if runErr != nil || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("native run: %v stdout=%q stderr=%q", runErr, stdout.String(), stderr.String())
			}
			images = append(images, map[string]any{"profile": profile.Name, "exeSHA256": hash(artifact.Bytes), "exeBytes": len(artifact.Bytes), "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version, "argv": command.Args, "cwd": command.Dir, "stdin": "closed", "outputEncoding": "UTF-8", "stdout": stdout.String(), "stderr": stderr.String(), "stdoutHex": hex.EncodeToString(stdout.Bytes()), "stderrHex": hex.EncodeToString(stderr.Bytes()), "exitCode": command.ProcessState.ExitCode()})
		})
	}
	if t.Failed() {
		return
	}
	receipt := map[string]any{"schema": "fairyc.checked-i64-decimal/v1", "sdk": sdkstdlib.Version, "sdkSourceIdentity": definition.SourceIdentity, "language": sdkstdlib.LanguageRevision, "runtime": sdkstdlib.RuntimeABI, "sourceSHA256": hash([]byte(text)), "cases": cases, "views": "full and bounded subviews; empty uses empty subview", "ownerPreservation": "all bytes compared after direct and propagated calls", "oracle": "positive arbitrary-precision math/big prefix magnitudes", "interpreter": "passed after MIR codec round trip", "cEmission": []string{"x86_64-linux-gnu", "x86_64-windows-msvc"}, "compilerIdentity": map[string]any{"kind": "in-process CLI compiler in Go test executable", "executableSHA256": hash(compilerBytes), "goVersion": runtime.Version(), "buildInfo": buildInfo}, "images": images}
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

func TestCheckedI64DecimalResultRejections(t *testing.T) {
	for _, tc := range []struct{ name, source, feature string }{
		{"unhandled", "fn i32 main() { var u8[1] raw = [48]; num::parse_i64_decimal(ref raw); return 0; }", ""},
		{"nonexhaustive", "fn i32 main() { var u8[1] raw = [48]; return switch(num::parse_i64_decimal(ref raw)) { case .ok(_): { 0 } case .err(e): { switch(e) { case .Empty: { 1 } case .InvalidDecimal: { 2 } } } }; }", "non-exhaustive enum match"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checked, err := workspacecompiler.Check(checkedIntegerConversionWorkspace(t, "module main; import std::num as num; "+tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified {
				t.Fatal("excluded source accepted")
			}
			if tc.feature == "" {
				if len(checked.Semantic.Diagnostics) == 0 {
					t.Fatal("missing Result diagnostic")
				}
			} else {
				found := false
				for _, unsupported := range checked.Semantic.Unsupported {
					if unsupported.Feature == tc.feature {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing exact rejection: %+v", checked.Semantic.Unsupported)
				}
			}
		})
	}
}
