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
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestDigitsOnlyExampleExecution(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "digits_only_quickstart")
	mainBytes, err := os.ReadFile(filepath.Join(root, "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	// Compare the committed fixed results to the existing arbitrary-precision
	// oracle. Digits-only rejects a minus as syntax before signed magnitude work.
	var cases []decimalOracleCase
	pattern := regexp.MustCompile(`check\(b("(?:[^"\\]|\\.)*"), ([0-3]), ([0-9]+)\);`)
	for _, match := range pattern.FindAllStringSubmatch(string(mainBytes), -1) {
		input, err := strconv.Unquote(match[1])
		if err != nil {
			t.Fatal(err)
		}
		want := decimalOracle([]byte(input))
		if strings.HasPrefix(input, "-") {
			want.Error, want.Value = "InvalidDecimal", "0"
		}
		code := map[string]string{"": "0", "Empty": "1", "InvalidDecimal": "2", "OutOfRange": "3"}[want.Error]
		if match[2] != code || (want.Error == "" && match[3] != want.Value) {
			t.Fatalf("fixed expectation %q differs from oracle: %+v", input, want)
		}
		cases = append(cases, want)
	}
	if len(cases) != 27 {
		t.Fatalf("fixed string case count: %d", len(cases))
	}
	rawPattern := regexp.MustCompile(`let u8\[([0-9]+)\] ([a-z_][a-z_0-9]*) = \[([0-9, ]+)\]; check\(ref ([a-z_][a-z_0-9]*), 2, 0\);`)
	rawCases := rawPattern.FindAllStringSubmatch(string(mainBytes), -1)
	if len(rawCases) != 4 {
		t.Fatalf("fixed raw case count: %d", len(rawCases))
	}
	for _, match := range rawCases {
		if match[2] != match[4] {
			t.Fatal("raw fixture owner mismatch")
		}
		var input []byte
		for _, part := range strings.Split(match[3], ",") {
			value, err := strconv.ParseUint(strings.TrimSpace(part), 10, 8)
			if err != nil {
				t.Fatal(err)
			}
			input = append(input, byte(value))
		}
		if strconv.Itoa(len(input)) != match[1] {
			t.Fatal("raw fixture length mismatch")
		}
		want := decimalOracle(input)
		if want.Error != "InvalidDecimal" {
			t.Fatalf("raw input oracle: %+v", want)
		}
		cases = append(cases, want)
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
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
	t.Logf("%d fixed inputs: full/subviews, direct/propagated Results and unchanged padded owners passed after MIR round trip", len(cases))
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 native evidence; interpreter passed")
	}
	evidenceRoot, err := filepath.Abs(filepath.Join("..", "..", ".cache", "digits-only-example"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	hash := func(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
	definition, err := sdkstdlib.Load()
	if err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{}
	for _, p := range []string{"fairy.toml", "src/main.fy", "src/numbers.fy", "README.md"} {
		data, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			t.Fatal(err)
		}
		sources[p] = hash(data)
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
		command := exec.CommandContext(ctx, exe)
		command.Dir, command.Stdin = evidenceRoot, strings.NewReader("")
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		runErr := command.Run()
		cancel()
		// Decode the raw byte pipes explicitly as UTF-8, with no codepage fallback.
		if !utf8.Valid(stdout.Bytes()) || !utf8.Valid(stderr.Bytes()) {
			t.Fatal("invalid UTF-8 native output")
		}
		if runErr != nil || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("native run: %v stdout=%q stderr=%q", runErr, stdout.String(), stderr.String())
		}
		images = append(images, map[string]any{"profile": profile.Name, "exeSHA256": hash(artifact.Bytes), "exeBytes": len(artifact.Bytes), "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version, "buildDiagnostics": diagnostics.String(), "argv": command.Args, "cwd": command.Dir, "stdin": "closed", "outputEncoding": "UTF-8", "stdout": stdout.String(), "stderr": stderr.String(), "stdoutHex": hex.EncodeToString(stdout.Bytes()), "stderrHex": hex.EncodeToString(stderr.Bytes()), "exitCode": command.ProcessState.ExitCode()})
	}
	data, err := json.MarshalIndent(map[string]any{"schema": "fairyc.digits-only-example/v1", "sdk": sdkstdlib.Version, "sdkSourceIdentity": definition.SourceIdentity, "language": sdkstdlib.LanguageRevision, "runtime": sdkstdlib.RuntimeABI, "target": target.X8664WindowsMSVC, "sourceSHA256": sources, "cases": cases, "checksPerInput": 4, "checkVerified": true, "mirRoundTrip": true, "interpreterExit": outcome.Value, "images": images, "compilerIdentity": map[string]any{"kind": "in-process CLI compiler in Go test executable", "executableSHA256": hash(compilerBytes), "goVersion": runtime.Version(), "buildInfo": buildInfo}}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("digits-only receipt: %s\n", filepath.Join(evidenceRoot, "receipt.json"))
}
