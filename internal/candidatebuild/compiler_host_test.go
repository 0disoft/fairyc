package candidatebuild

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func compilerHostPacket(t *testing.T, field string, size int) []byte {
	t.Helper()
	p, err := Decode(packetFixture("valid"))
	if err != nil {
		t.Fatal(err)
	}
	switch field {
	case "header":
		p.Header = strings.TrimSuffix(p.Header, "#endif\n") + strings.Repeat(" ", size-len(p.Header)) + "#endif\n"
	case "literals":
		p.Literals += strings.Repeat(" ", size-len(p.Literals))
	case "unit":
		p.Units[0] += strings.Repeat(" ", size-len(p.Units[0]))
	}
	var data bytes.Buffer
	fmt.Fprintf(&data, "FYCPARTS1\n%d\n%s%d\n%s1\n%d\n%s", len(p.Header), p.Header, len(p.Literals), p.Literals, len(p.Units[0]), p.Units[0])
	return data.Bytes()
}

func TestCompilerHostPacketHeaderLimits(t *testing.T) {
	for _, field := range []string{"header", "literals", "unit"} {
		for _, size := range []int{maxPacketPart - 1, maxPacketPart, maxPacketPart + 1, maxCompilerHostHeader, maxCompilerHostHeader + 1} {
			t.Run(fmt.Sprintf("%s/%d", field, size), func(t *testing.T) {
				data := compilerHostPacket(t, field, size)
				for _, host := range []bool{false, true} {
					decoder, limit := Decode, maxPacketPart
					if host {
						decoder = DecodeCompilerHost
						if field == "header" {
							limit = maxCompilerHostHeader
						}
					}
					_, err := decoder(data)
					if (err == nil) != (size <= limit) {
						t.Fatalf("compilerHost=%t size=%d limit=%d: %v", host, size, limit, err)
					}
				}
			})
		}
	}
}

func TestCompilerHostInputRequiresExplicitOptIn(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native compiler-host bridge requires Windows amd64 MSVC")
	}
	root := t.TempDir()
	input, output := filepath.Join(root, "host.bin"), filepath.Join(root, "host.exe")
	if err := os.WriteFile(input, compilerHostPacket(t, "header", maxPacketPart+1), 0600); err != nil {
		t.Fatal(err)
	}
	previous := []byte("previous image")
	if err := os.WriteFile(output, previous, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildInput(Options{Output: output, StackReserveBytes: 16 << 20}, input); err == nil || !strings.Contains(err.Error(), "oversized packet part") {
		t.Fatalf("stack reservation implicitly admitted compiler-host header: %v", err)
	}
	if got, err := os.ReadFile(output); err != nil || !bytes.Equal(got, previous) {
		t.Fatalf("rejected input changed previous image: %v", err)
	}
	for _, release := range []bool{false, true} {
		result, err := BuildInput(Options{Output: output, CompilerHost: true, Release: release}, input)
		if err != nil || result.ImageBytes == 0 || result.Units != 1 {
			t.Fatalf("explicit compiler-host build release=%t: %+v %v", release, result, err)
		}
		if streams, err := exec.CommandContext(t.Context(), output).CombinedOutput(); err != nil || len(streams) != 0 {
			t.Fatalf("compiler-host image release=%t: %v %s", release, err, streams)
		}
	}
}

func TestLinuxRejectsWindowsCompilerHostBeforeIO(t *testing.T) {
	_, err := BuildLinuxInput(Options{CompilerHost: true}, "must-not-read", "", "")
	if err == nil || !strings.Contains(err.Error(), "Windows stack options") {
		t.Fatalf("Windows compiler-host option reached Linux I/O: %v", err)
	}
}
