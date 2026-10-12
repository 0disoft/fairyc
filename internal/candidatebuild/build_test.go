package candidatebuild

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func packetFixture(mode string) []byte { return packetFixtureFrame(mode, 1) }

func TestFrontendDeadlineAdmissionAndPacket(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second, 10*time.Minute + time.Nanosecond} {
		if _, err := Emit("must-not-run", "ignored", "", timeout); err == nil || !strings.Contains(err.Error(), "timeout in (0,10m]") {
			t.Fatalf("invalid frontend deadline %s: %v", timeout, err)
		}
	}
	t.Setenv("FAIRYC_CANDIDATE_TEST_FRONTEND", "valid")
	for _, timeout := range []time.Duration{time.Minute, 2 * time.Minute, 10 * time.Minute} {
		data, err := Emit(os.Args[0], "ignored", "", timeout)
		if err != nil || !bytes.Equal(data, packetFixture("valid")) {
			t.Fatalf("frontend deadline %s changed packet: %v", timeout, err)
		}
	}
}

func TestWorkspaceFrontendArgumentsAndAdmission(t *testing.T) {
	t.Setenv("FAIRYC_CANDIDATE_TEST_FRONTEND", "valid")
	for _, locked := range []bool{false, true} {
		want := []string{"--c-workspace", "workspace with spaces"}
		if locked {
			want = append(want, "--locked")
		}
		encoded, _ := json.Marshal(want)
		t.Setenv("FAIRYC_CANDIDATE_TEST_ARGUMENTS", string(encoded))
		data, err := EmitWorkspace(os.Args[0], "workspace with spaces", "", locked, time.Minute)
		if err != nil || !bytes.Equal(data, packetFixture("valid")) {
			t.Fatalf("workspace transport locked=%v: %v", locked, err)
		}
	}
	for _, options := range []Options{{}, {Manifest: "m", Workspace: "w"}, {Manifest: "m", Locked: true}} {
		if _, err := Build(options); err == nil || !strings.Contains(err.Error(), "exactly one") {
			t.Fatalf("mixed frontend inputs admitted: %+v %v", options, err)
		}
	}
	for _, options := range []Options{{Workspace: "w"}, {Locked: true}} {
		if _, err := BuildCached(options, "must-not-read", ""); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
			t.Fatalf("mixed packet workspace admitted: %v", err)
		}
	}
	for _, timeout := range []time.Duration{0, -time.Second, 10*time.Minute + 1} {
		if _, err := EmitWorkspace("must-not-run", "w", "", false, timeout); err == nil {
			t.Fatal("invalid workspace deadline admitted")
		}
	}
}

func TestCompileTimeoutRejectedBeforePacketOrCompiler(t *testing.T) {
	for _, timeout := range []time.Duration{-time.Second, 10*time.Minute + time.Nanosecond} {
		if _, err := buildPacket(Options{CompileTimeout: timeout}, nil); err == nil || !strings.Contains(err.Error(), "compile timeout") {
			t.Fatalf("invalid compile timeout %s: %v", timeout, err)
		}
	}
}

func TestStackReserveRejectedBeforePacketOrCompiler(t *testing.T) {
	for _, reserve := range []uint64{1, 65535, 65537, 16*1024*1024 + 1, ^uint64(0)} {
		if _, err := buildPacket(Options{StackReserveBytes: reserve}, nil); err == nil || !strings.Contains(err.Error(), "stack reserve") {
			t.Fatalf("invalid stack reserve %d: %v", reserve, err)
		}
	}
}

func TestCachedPacketRejectsChangedBytesWithoutFrontendFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "candidate.bin")
	data := packetFixture("")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	got, err := cachedPacket(path, digest)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("cached packet changed: %v", err)
	}
	if err := os.WriteFile(path, append(data, 'x'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cachedPacket(path, digest); err == nil || !strings.Contains(err.Error(), "SHA256 mismatch") {
		t.Fatalf("accepted changed packet: %v", err)
	}
	if _, err := BuildCached(Options{Candidate: "must-not-run"}, path, digest); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("accepted mixed frontend/cache mode: %v", err)
	}
}

func packetFixtureFrame(mode string, frame int) []byte {
	header := "#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n#define FY_CANDIDATE_PROJECT_HEADER_V1\n#include <stdint.h>\nextern int32_t missing(void);\n#endif\n"
	literals := "/* fairyc.candidate-project-literals-c/v1 */\n#include \"fy_project.h\"\n"
	value := "0"
	if mode == "link-error" {
		value = "missing()"
	}
	if mode == "compile-error" {
		header = strings.Replace(header, "#endif\n", "#error compile_failure\n#endif\n", 1)
	}
	if mode == "runtime-error" {
		header = strings.Replace(header, "#endif\n", "extern uint32_t fy_std_unknown_v0(void);\n#endif\n", 1)
	}
	unit := "/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\nextern int32_t fy_fn_1(void){return " + value + ";}\nint32_t fy_candidate_entry(void){return fy_fn_1();}\nint main(void){return (int)fy_candidate_entry();}\n"
	unit = strings.ReplaceAll(unit, "fy_fn_1", fmt.Sprintf("fy_fn_%d", frame))
	var out bytes.Buffer
	out.WriteString("FYCPARTS1\n")
	fmt.Fprintf(&out, "%d\n%s%d\n%s1\n%d\n%s", len(header), header, len(literals), literals, len(unit), unit)
	return out.Bytes()
}

func init() {
	mode := os.Getenv("FAIRYC_CANDIDATE_TEST_FRONTEND")
	if mode == "" {
		return
	}
	if encoded := os.Getenv("FAIRYC_CANDIDATE_TEST_ARGUMENTS"); encoded != "" {
		var want []string
		if err := json.Unmarshal([]byte(encoded), &want); err != nil || !reflect.DeepEqual(os.Args[1:], want) {
			fmt.Fprintf(os.Stderr, "wrong frontend arguments: %q", os.Args[1:])
			os.Exit(2)
		}
	}
	switch mode {
	case "source-error":
		fmt.Fprint(os.Stdout, "FYCPARTS1\npartial")
		fmt.Fprint(os.Stderr, "source failure\n")
		os.Exit(2)
	case "framing-error":
		fmt.Fprint(os.Stdout, "FYCPARTS1\n")
		os.Exit(0)
	case "timeout":
		time.Sleep(5 * time.Second)
		os.Exit(0)
	case "output-limit":
		fmt.Fprint(os.Stderr, strings.Repeat("x", 65537))
		os.Exit(0)
	case "diagnostics":
		fmt.Fprint(os.Stderr, "unexpected diagnostics\n")
	}
	_, _ = os.Stdout.Write(packetFixture(mode))
	os.Exit(0)
}

func TestPacketRejectsTruncationAndExtraBytes(t *testing.T) {
	valid := packetFixture("")
	if _, err := Decode(valid); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(valid); i++ {
		if _, err := Decode(valid[:i]); err == nil {
			t.Fatalf("accepted incomplete prefix %d", i)
		}
	}
	for index, bad := range [][]byte{append(append([]byte(nil), valid...), 'x'), bytes.Replace(valid, []byte("fy_fn_1(void){"), []byte("fy_fn_0(void){"), 1), append([]byte("FYCPARTS1\n0"), valid[len("FYCPARTS1\n"):]...)} {
		if _, err := Decode(bad); err == nil {
			t.Fatalf("accepted malformed packet %d", index)
		}
	}
}

func TestPacketFrameBounds(t *testing.T) {
	if _, err := Decode(packetFixtureFrame("", maxProjectFrames)); err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(packetFixtureFrame("", maxProjectFrames+1)); err == nil {
		t.Fatal("accepted frame beyond the project cap")
	}
}

func TestBuildFailuresKeepPreviousImage(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires MSVC")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"source-error", "framing-error", "diagnostics", "output-limit", "timeout", "runtime-error", "compile-error", "link-error"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("FAIRYC_CANDIDATE_TEST_FRONTEND", mode)
			directory := t.TempDir()
			output := filepath.Join(directory, "previous.exe")
			old := []byte("previous-image")
			if err := os.WriteFile(output, old, 0700); err != nil {
				t.Fatal(err)
			}
			timeout := time.Minute
			if mode == "timeout" {
				timeout = 50 * time.Millisecond
			}
			if _, err := Build(Options{Candidate: executable, Manifest: "ignored", Directory: directory, Output: output, Timeout: timeout}); err == nil {
				t.Fatal("expected build failure")
			}
			actual, err := os.ReadFile(output)
			if err != nil || !bytes.Equal(actual, old) {
				t.Fatalf("previous image changed: %v", err)
			}
		})
	}
}

func TestSuccessfulBuildReplacesCompleteImage(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires MSVC")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAIRYC_CANDIDATE_TEST_FRONTEND", "valid")
	for _, release := range []bool{false, true} {
		output := filepath.Join(t.TempDir(), "program.exe")
		if err := os.WriteFile(output, []byte("old"), 0700); err != nil {
			t.Fatal(err)
		}
		result, err := Build(Options{Candidate: executable, Manifest: "ignored", Output: output, Timeout: time.Minute, Release: release})
		if err != nil {
			t.Fatal(err)
		}
		image, err := os.ReadFile(output)
		if err != nil || len(image) != result.ImageBytes || !bytes.HasPrefix(image, []byte("MZ")) {
			t.Fatalf("invalid published image: %v", err)
		}
		command := exec.CommandContext(t.Context(), output)
		streams, err := command.CombinedOutput()
		if err != nil || len(streams) != 0 {
			t.Fatalf("published program failed: %v, %s", err, streams)
		}
	}
}

func TestPublicationFailuresKeepPreviousImage(t *testing.T) {
	injected := errors.New("injected publication failure")
	for _, phase := range []string{"write", "sync", "close", "rename"} {
		t.Run(phase, func(t *testing.T) {
			directory := t.TempDir()
			output := filepath.Join(directory, "program.exe")
			old := []byte("old-image")
			if err := os.WriteFile(output, old, 0700); err != nil {
				t.Fatal(err)
			}
			hooks := publishHooks{}
			switch phase {
			case "write":
				hooks.write = func(f *os.File, data []byte) error { _, _ = f.Write(data[:1]); return injected }
			case "sync":
				hooks.sync = func(*os.File) error { return injected }
			case "close":
				hooks.close = func(*os.File) error { return injected }
			case "rename":
				hooks.rename = func(*os.Root, string, string) error { return injected }
			}
			if err := publish(output, []byte("new-image"), hooks); !errors.Is(err, injected) {
				t.Fatalf("wrong failure: %v", err)
			}
			actual, err := os.ReadFile(output)
			if err != nil || !bytes.Equal(actual, old) {
				t.Fatalf("old image changed: %v", err)
			}
			files, err := os.ReadDir(directory)
			if err != nil || len(files) != 1 {
				t.Fatalf("staging file leaked: %v", err)
			}
		})
	}
}
