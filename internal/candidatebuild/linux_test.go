package candidatebuild

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLinuxPacketRuntimeSelection(t *testing.T) {
	for _, header := range []string{"", LinuxPacketMarker + "\nfy_std_windows_file_kind_v0(", LinuxPacketMarker + "\nfy_std_unknown_v0(", LinuxPacketMarker + "\nfy_private_vector_unknown_v0(", LinuxPacketMarker + "\nfairyc_private_windows_host_path_v1("} {
		if _, err := linuxRuntimeSource(header); err == nil {
			t.Fatalf("admitted incompatible runtime: %q", header)
		}
	}
	source, err := linuxRuntimeSource(LinuxPacketMarker + "\nfy_std_file_open_v0( fy_private_vector_reserve_v0(")
	if err != nil || !strings.Contains(source, "struct fy_std_file_result_v0") || strings.Contains(source, "windows.h") {
		t.Fatalf("Linux runtime closure: %v", err)
	}
	runSource, err := linuxRuntimeSourceForRun(LinuxPacketMarker, true)
	if err != nil || !strings.Contains(runSource, "write(3,") || strings.Contains(source, "write(3,") {
		t.Fatalf("private Linux run receipt selection: %v", err)
	}

	for _, options := range []Options{{Candidate: "no"}, {RunTrapControl: true, Output: "build/program", OutputRoot: "elsewhere"}, {StackReserveBytes: 65536}, {CompileTimeout: -time.Second}} {
		if _, err := BuildLinuxInput(options, "must-not-read", "", ""); err == nil {
			t.Fatal("invalid Linux build options admitted")
		}
	}
}

func TestLinuxPacketBuildAndPublication(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native GNU packet build requires Linux amd64")
	}
	root := t.TempDir()
	output := filepath.Join(root, "build", "program")
	for _, release := range []bool{false, true} {
		for _, mode := range []string{"valid", "compile-error", "link-error"} {
			packet, err := Decode(packetFixture(mode))
			if err != nil {
				t.Fatal(err)
			}
			packet.Header = strings.Replace(packet.Header, "\n", "\n"+LinuxPacketMarker+"\n", 1)
			var data bytes.Buffer
			data.WriteString("FYCPARTS1\n")
			fmt.Fprintf(&data, "%d\n%s%d\n%s%d\n", len(packet.Header), packet.Header, len(packet.Literals), packet.Literals, len(packet.Units))
			for _, unit := range packet.Units {
				fmt.Fprintf(&data, "%d\n%s", len(unit), unit)
			}
			stage := t.TempDir()
			input := filepath.Join(stage, "packet.bin")
			if err := os.WriteFile(input, data.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			previous, _ := os.ReadFile(output)
			result, err := BuildLinuxInput(Options{OutputRoot: root, Output: "build/program", Release: release}, input, stage, "")
			if mode == "valid" {
				if err != nil || result.ImageBytes == 0 {
					t.Fatalf("GNU build: %+v %v", result, err)
				}
				info, err := os.Stat(output)
				if err != nil || info.Mode().Perm()&0100 == 0 {
					t.Fatal("published image is not executable", err)
				}
				private := t.TempDir()
				packetPath := filepath.Join(private, "packet.bin")
				if err := os.WriteFile(packetPath, data.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
				_, err = BuildLinuxInput(Options{OutputRoot: private, Output: "program", RunTrapControl: true, Release: release}, packetPath, private, "")
				if err != nil {
					t.Fatalf("private run image: %v", err)
				}
				if info, err := os.Stat(filepath.Join(private, "program")); err != nil || info.Mode().Perm()&0100 == 0 {
					t.Fatal("private run image missing", err)
				}

			} else {
				actual, _ := os.ReadFile(output)
				if err == nil || !bytes.Equal(previous, actual) {
					t.Fatalf("%s damaged previous output: %v", mode, err)
				}
			}
		}
	}
}
