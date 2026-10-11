package candidatebuild

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func packetFixture(mode string) []byte {
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
