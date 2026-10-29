package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
)

// Exercise the connected shared emitters with real Windows file I/O. Linux
// target selection is retained; these executions do not prove Linux native I/O.
func TestLinuxCEmissionWindows(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("emitter substitute requires Windows amd64")
	}
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	selectLinuxCEmission(t, root)
	writeTestSource(t, root, "src/main.fy", `module main;
import project_emit_cli as workspace;
import file_source as files;
import file_frontend_cli as frontend;
import std::bytes as bytes;
import std::process as process;
@effects(filesystem,console) fn i32 main(){
    var u8[4096] raw=[0;4096];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return 2;}};
    let ref u8[] flag=ref raw;
    if(bytes::equal(flag[..n],b"--c-workspace")){return workspace::arguments();}
    let length=switch(process::arg(2,ref mut raw)){case .ok(p):{p}case .err(_):{return 2;}};
    let ref u8[] path=ref raw;
    let source=switch(files::load(path[..length])){case .ok(p):{p}case .err(_):{return 2;}};
    return frontend::run(source,path[..length],8);
}
`)
	unit, support, tc := windowsCandidateProbe(t, root)
	// This is a compiler-host probe: SDK projection needs more than MSVC's
	// default 1 MiB stack. Reserve 64 MiB without changing product defaults.
	unit.Source = "#pragma comment(linker, \"/STACK:67108864\")\n" + unit.Source
	// MSVC /O2 for this complete workspace emitter exceeds the one-minute
	// compiler budget. Keep full integration low; the file path runs both below.
	for _, profile := range []cdriver.Profile{cdriver.LowProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support})
			if err != nil {
				t.Fatal(err)
			}
			checkLinuxCEmission(t, tc, artifact)
		})
	}
}

func TestLinuxCFileWindows(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("emitter substitute requires Windows amd64")
	}
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	writeTestSource(t, root, "src/main.fy", `module main;
import file_source as files;
import file_frontend_cli as frontend;
import std::process as process;
@effects(filesystem,console) fn i32 main(){
    var u8[4096] raw=[0;4096];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return 2;}};
    let ref u8[] path=ref raw;
    let source=switch(files::load(path[..n])){case .ok(p):{p}case .err(_):{return 2;}};
    return frontend::run(source,path[..n],8);
}
`)
	unit, support, tc := windowsCandidateProbe(t, root)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support})
			if err != nil {
				t.Fatal(err)
			}
			client := t.TempDir()
			path := filepath.Join(client, "input.fy")
			writeTestSource(t, client, "input.fy", "module main; fn i32 main(){return 23;}\n")
			result, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: []string{filepath.ToSlash(path)}, Timeout: time.Minute})
			if err != nil || result.ExitCode != 0 || result.Stderr != "" || !strings.Contains(result.Stdout, "fy_candidate_entry") {
				t.Fatalf("single-file emission: %+v %v", result, err)
			}
			runtimeSource, err := nativesupport.Source(target.X8664WindowsMSVC, nil, nativesupport.Options{})
			if err != nil {
				t.Fatal(err)
			}
			image, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: result.Stdout, RuntimeSource: runtimeSource})
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := tc.Execute(image, cdriver.NativeExecutionOptions{Timeout: time.Minute})
			if err != nil || outcome.ExitCode != 23 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("single-file program: %+v %v", outcome, err)
			}
			writeTestSource(t, client, "input.fy", "module main; fn i32 main(){return true;}\n")
			result, err = tc.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: []string{filepath.ToSlash(path)}, Timeout: time.Minute})
			if err != nil || result.ExitCode == 0 || result.Stdout != "" {
				t.Fatalf("single-file rejection: %+v %v", result, err)
			}
		})
	}
}

// The candidate packet and Go reference C have different private encodings.
// Compare successful native behavior, and compare frontend failures exactly.
func checkLinuxCEmission(t *testing.T, tc cdriver.NativeToolchain, artifact cdriver.NativeArtifact) {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.linux-c-emission\"\nname = \"client\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n")
	source := "module main; fn i32 main(){return 23;}\n"
	writeTestSource(t, root, "src/main.fy", source)
	run := func(args []string, code int32) cdriver.NativeOutcome {
		t.Helper()
		result, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: args, WorkingDirectory: root, Timeout: time.Minute})
		if err != nil || result.ExitCode != code || result.TrapCategory != "" {
			t.Fatalf("C emission %v: %+v %v", args, result, err)
		}
		return result
	}
	args := []string{"--c-workspace", filepath.ToSlash(root)}
	// Locked load must fail without emitting a partial packet.
	if result := run(append(append([]string{}, args...), "--locked"), 7); result.Stdout != "" {
		t.Fatal("failed locked emission published C")
	}
	first := run(args, 0)
	packet, err := candidatebuild.Decode([]byte(first.Stdout))
	if err != nil || !strings.Contains(packet.Header, candidatebuild.LinuxPacketMarker) || first.Stderr != "" {
		t.Fatalf("Linux packet: %v %q", err, first.Stderr)
	}
	lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	locked := run(append(append([]string{}, args...), "--locked"), 0)
	after, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil || first.Stdout != locked.Stdout || locked.Stderr != "" || !bytes.Equal(lock, after) {
		t.Fatal("locked emission changed packet or lock", err)
	}
	for _, bad := range [][]string{{"--c-workspace"}, {"--c-workspace", "", "--locked"}, {"--c-workspace", ".", "--release"}, {"--c-workspace", ".", "--locked", "--locked"}} {
		if result := run(bad, 2); result.Stdout != "" {
			t.Fatal("invalid invocation published C", bad)
		}
	}
	file := run([]string{"--c-file", "src/main.fy"}, 0)
	if !strings.Contains(file.Stdout, "fy_candidate_entry") || file.Stderr != "" {
		t.Fatal("missing single-file C", file.Stderr)
	}
	if runtime.GOOS == "linux" {
		// Both independently emitted units must compile and return the same value.
		support, err := nativesupport.Source(target.X8664LinuxGNU, nil, nativesupport.Options{})
		if err != nil {
			t.Fatal(err)
		}
		stage := t.TempDir()
		if err := os.Chmod(stage, 0700); err != nil {
			t.Fatal(err)
		}
		built, err := cdriver.BuildLinuxTargetParts(stage, "gcc", cdriver.LowProfile, packet.Header, packet.Literals, packet.Units, support, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := tc.Execute(cdriver.NativeArtifact{Artifact: built}, cdriver.NativeExecutionOptions{Timeout: time.Minute})
		if err != nil || outcome.ExitCode != 23 || outcome.Stdout != "" || outcome.Stderr != "" {
			t.Fatalf("packet program: %+v %v", outcome, err)
		}
		code, reference, stderr := runForTest([]string{"experimental", "emit-workspace-c", "--target", "x86_64-linux-gnu", "--locked"}, root)
		if code != 0 || stderr != "" {
			t.Fatalf("Go reference emission: %d %s", code, stderr)
		}
		for _, emitted := range []string{file.Stdout, reference} {
			image, err := tc.Build(cdriver.NativeBuildOptions{Profile: cdriver.LowProfile, UnitSource: emitted, RuntimeSource: support})
			if err != nil {
				t.Fatal(err)
			}
			result, err := tc.Execute(image, cdriver.NativeExecutionOptions{Timeout: time.Minute})
			if err != nil || result.ExitCode != 23 || result.Stdout != "" || result.Stderr != "" {
				t.Fatalf("emitted program: %+v %v", result, err)
			}
		}
	}
	writeTestSource(t, root, "src/main.fy", `module main;
import std::fs as fs;
@effects(filesystem) fn i32 main(){
    let file=switch(fs::open_read_utf8("input.txt")){case .ok(p):{p}case .err(_):{return 1;}};
    return if(fs::close(file).is_ok){0}else{2};
}
`)
	sdk := run(args, 0)
	sdkPacket, err := candidatebuild.Decode([]byte(sdk.Stdout))
	// The SDK owns some Windows-only record declarations even in this packet;
	// selecting a Linux File result must not rewrite that authenticated source.
	fileWire := strings.Contains(sdkPacket.Header, "struct fy_std_file_result_v0{")
	signedError := strings.Contains(sdkPacket.Header, "int32_t error_number;")
	if err != nil || !fileWire || !signedError || !strings.Contains(sdkPacket.Header, candidatebuild.LinuxPacketMarker) {
		t.Fatalf("Linux SDK packet: %v, File wire=%t, signed errno=%t", err, fileWire, signedError)
	}
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return true;}\n")
	failed := run(args, 3)
	if failed.Stdout != "" || !strings.Contains(failed.Stderr, "E1009") {
		t.Fatal("semantic failure published C", failed.Stderr)
	}
	if runtime.GOOS == "linux" {
		code, stdout, stderr := runForTest([]string{"experimental", "emit-workspace-c", "--target", "x86_64-linux-gnu"}, root)
		if code != int(failed.ExitCode) || stdout != failed.Stdout || stderr != failed.Stderr {
			t.Fatalf("diagnostic differs from reference: %d %q %q versus %+v", code, stdout, stderr, failed)
		}
	}
}
