package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// The oracle is Go's mathematical integer values, not the Fairy accumulation loop.
// Bit boundaries catch intermediate-power overflow and missing high bits in both directions.
func checkedIntegerConversionSource() (string, []uint64) {
	unique := map[uint64]bool{0: true, 1: true, math.MaxInt32: true, uint64(math.MaxInt32) + 1: true,
		math.MaxUint32: true, uint64(math.MaxUint32) + 1: true, math.MaxInt64: true}
	for bit := uint(0); bit < 63; bit++ {
		value := uint64(1) << bit
		for _, candidate := range []uint64{value - 1, value, value + 1} {
			unique[candidate] = true
		}
	}
	values := make([]uint64, 0, len(unique))
	for value := range unique {
		values = append(values, value)
	}
	slices.Sort(values)
	var source strings.Builder
	source.WriteString(`module main;
import std::num as num;
fn Result<usize, num::ConversionError> round_trip(usize value) {
    let i64 intermediate = num::i64_from_usize(value)?;
    return num::usize_from_i64(intermediate);
}
fn void pair(usize unsigned_value, i64 signed_value) {
    let bool signed_ok = switch (num::i64_from_usize(unsigned_value)) {
        case .ok(v): { v == signed_value } case .err(_): { false }
    };
    let bool unsigned_ok = switch (num::usize_from_i64(signed_value)) {
        case .ok(v): { v == unsigned_value } case .err(_): { false }
    };
    let bool round_trip_ok = switch (round_trip(unsigned_value)) {
        case .ok(v): { v == unsigned_value } case .err(_): { false }
    };
    assert(signed_ok && unsigned_ok && round_trip_ok);
}
fn void negative(i64 value) {
    let bool rejected = switch (num::usize_from_i64(value)) {
        case .ok(_): { false }
        case .err(e): { switch (e) {
            case .Negative: { true } case .OutOfRange: { false }
        } }
    };
    assert(rejected);
}
fn void outside(usize value) {
    let bool rejected = switch (num::i64_from_usize(value)) {
        case .ok(_): { false }
        case .err(e): { switch (e) {
            case .Negative: { false } case .OutOfRange: { true }
        } }
    };
    let bool propagated = switch (round_trip(value)) {
        case .ok(_): { false }
        case .err(e): { switch (e) {
            case .Negative: { false } case .OutOfRange: { true }
        } }
    };
    assert(rejected && propagated);
}
fn i32 main() {
`)
	for _, value := range values {
		fmt.Fprintf(&source, "    pair(%d, %d);\n", value, value)
	}
	source.WriteString(`    negative(-1);
    negative(-9223372036854775808);
    outside(9223372036854775808);
    outside(18446744073709551615);
    return 0;
}
`)
	return source.String(), values
}

func checkedIntegerConversionWorkspace(t *testing.T, text string) workspaceloader.Workspace {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.checked-integer\"\nname = \"checked-integer\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	writeTestSource(t, root, "src/main.fy", text)
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return workspace
}

func TestCheckedIntegerConversionExecution(t *testing.T) {
	source, values := checkedIntegerConversionSource()
	checked, err := workspacecompiler.Check(checkedIntegerConversionWorkspace(t, source))
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
		if _, err := cbackend.Emit(checked.MIR, entry, profile); err != nil {
			t.Fatalf("C emission: %v", err)
		}
	}
	t.Logf("interpreter/artifact/C emission: %d success vectors and 4 exact error vectors", len(values))
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native evidence requires Windows AMD64 MSVC; interpreter already checked")
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", "checked-integer-conversion")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	definition, err := sdkstdlib.Load()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(source))
	var images []map[string]any
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			var diagnostics bytes.Buffer
			artifact, code := compileWindowsProgram(checked.MIR, entry, target.X8664WindowsMSVC, profile, &diagnostics)
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
			imageHash := sha256.Sum256(artifact.Bytes)
			if err := os.WriteFile(filepath.Join(evidenceRoot, profile.Name+".exe"), artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			images = append(images, map[string]any{"profile": profile.Name, "exeSHA256": hex.EncodeToString(imageHash[:]), "exeBytes": len(artifact.Bytes), "exitCode": result.ExitCode, "stdout": result.Stdout, "stderr": result.Stderr, "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version})
		})
	}
	if t.Failed() {
		return
	}
	// Decimal strings preserve values above 2^53 in JSON consumers such as Node.
	decimalValues := make([]string, len(values))
	for index, value := range values {
		decimalValues[index] = strconv.FormatUint(value, 10)
	}
	receipt := map[string]any{"schema": "fairyc.checked-integer-conversion/v1", "sdk": sdkstdlib.Version, "sdkSourceIdentity": definition.SourceIdentity, "language": sdkstdlib.LanguageRevision, "runtime": sdkstdlib.RuntimeABI, "target": target.X8664WindowsMSVC, "sourceSHA256": hex.EncodeToString(sum[:]), "successValues": decimalValues, "errorValues": []string{"-1", "-9223372036854775808", "9223372036854775808", "18446744073709551615"}, "interpreter": "passed after MIR codec round trip", "cEmission": []string{"x86_64-linux-gnu", "x86_64-windows-msvc"}, "images": images}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "main.fy"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCheckedIntegerConversionRetainsRejections(t *testing.T) {
	for _, tc := range []struct{ name, source, feature string }{
		{"usize-as-i64", "fn i32 main() { let usize v = 1; let i64 n = v as i64; return 0; }", "cast from usize to i64"},
		{"i64-as-usize-unsafe", "fn i32 main() { let i64 v = 1; let usize n = unsafe { v as usize }; return 0; }", "cast from i64 to usize"},
		{"unhandled-result", "import std::num as num; fn i32 main() { num::i64_from_usize(1); return 0; }", ""},
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
				for _, u := range checked.Semantic.Unsupported {
					if u.Feature == tc.feature {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing excluded cast: %+v", checked.Semantic.Unsupported)
				}
			} else if len(checked.Semantic.Diagnostics) == 0 {
				t.Fatalf("missing Result diagnostic: %+v", checked.Semantic.Unsupported)
			}
		})
	}
}
