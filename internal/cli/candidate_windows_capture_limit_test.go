package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// Run the five original authored integration tests against the current module
// closure. Keep the fixture bodies intact, including their process/handle oracles.
func TestCandidateWindowsCaptureAuthoredRegression(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows capture authored regression requires windows/amd64")
	}
	fixture, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "windows_run_capture.fy"))
	if err != nil {
		t.Fatal(err)
	}
	const declaration = "module tests::windows_run_capture;"
	if !strings.HasPrefix(string(fixture), declaration) {
		t.Fatal("unexpected authored fixture module")
	}
	names := []string{
		"retains_extended_trap_locations_from_the_private_pipe",
		"retains_the_exact_shared_output_limit_and_drains_overflow",
		"separates_traps_and_drains_an_oversized_control_pipe",
		"deadline_terminates_the_observed_live_parent_and_child",
		"captures_binary_stdin_and_streams_before_forwarding",
	}
	main := strings.Replace(string(fixture), declaration, "module main;", 1)
	main = strings.ReplaceAll(main, "@test ", "")
	main += "\n@effects(filesystem,console,environment,clock) fn i32 main(){"
	for _, name := range names {
		if !strings.Contains(string(fixture), "fn void "+name+"(") {
			t.Fatalf("missing authored test %s", name)
		}
		main += name + "();"
	}
	main += "return 0;}\n"

	directory := t.TempDir()
	fixturePath := filepath.Join(directory, "fixture.txt")
	if err := os.WriteFile(fixturePath, []byte(filepath.ToSlash(directory)), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAIRYC_CONTROL_PRESERVE", "한글 environment 🧚")
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	receiver := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-o", filepath.Join(directory, "receiver.exe"), "./tools/candidate/fairyc-candidate-process-reference")
	receiver.Dir = filepath.Join("..", "..")
	receiver.WaitDelay = 2 * time.Second
	if output, err := receiver.CombinedOutput(); err != nil {
		t.Fatalf("build current process reference: %v\n%s", err, output)
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range []struct {
		name string
		trap int
	}{{"trap.exe", 1}, {"normal.exe", 0}} {
		// A tiny protocol fixture supplies a category-5 receipt with exit 133;
		// normal.exe supplies exit 134 without a receipt. No cached images are used.
		image, err := cdriver.BuildWindowsMSVCProbe(tc, cdriver.LowProfile, fmt.Sprintf(candidateWindowsCaptureReceiptChild, child.trap))
		if err != nil {
			t.Fatalf("build %s: %v", child.name, err)
		}
		if err := os.WriteFile(filepath.Join(directory, child.name), image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
	}
	candidateAtlasProbeSize(t, "windows_run_capture", main, 128<<10, func(t *testing.T, image string) {
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, image, fixturePath)
		command.Stdin = bytes.NewReader([]byte("in\x00put\n"))
		command.WaitDelay = 2 * time.Second
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		if err := command.Run(); err != nil {
			t.Fatalf("authored capture fixture: %v (context: %v); stdout=%q stderr=%q", err, ctx.Err(), &stdout, &stderr)
		}
		if stdout.String() != "child stdout\n" || stderr.String() != "child stderr\n" {
			t.Fatalf("authored forwarded streams: stdout=%q stderr=%q", &stdout, &stderr)
		}
		t.Logf("passed %d authored tests: %s", len(names), strings.Join(names, ", "))
	}, "file_source")
}

const candidateWindowsCaptureReceiptChild = `
#include <windows.h>
#include <stdlib.h>
int main(void) {
    if (%d) {
        char value[32];
        DWORD size = GetEnvironmentVariableA("FAIRYC_TRAP_CONTROL_HANDLE_V1", value, sizeof value);
        unsigned char receipt = 5;
        DWORD written = 0;
        if (!size || size >= sizeof value ||
            !WriteFile((HANDLE)(ULONG_PTR)strtoull(value, NULL, 10), &receipt, 1, &written, NULL) || written != 1) {
            return 2;
        }
        ExitProcess(133);
    }
    return 134;
}
`

// The same real Fairy image is both capture caller and binary-output child.
// Runtime writes keep the 64MiB boundary out of compiler literals and adapters.
func TestCandidateWindowsCaptureOutputLimit(t *testing.T) {
	candidateAtlasProbeSize(t, "windows_run_capture", candidateWindowsCaptureLimitMain, 128<<10, func(t *testing.T, image string) {
		type captureCase struct {
			name                  string
			stdout, stderr, limit int
			kind                  int
		}
		explicit := []captureCase{
			{"minimum-exact", 1, 0, 1, 0},
			{"minimum-over", 1, 1, 1, 11},
			{"binary-exact", 131072, 0, 131072, 0},
			{"binary-one-over", 131073, 0, 131072, 11},
			{"stderr-exact", 0, 131072, 131072, 0},
			{"combined-exact", 65536, 65536, 131072, 0},
			{"combined-one-over", 65536, 65537, 131072, 11},
			{"zero-before-launch", 1, 0, 0, 1},
			{"too-large-before-launch", 1, 0, 67108865, 1},
		}
		for _, api := range []string{"input-limit", "environment-limit"} {
			for _, test := range explicit {
				t.Run(api+"/"+test.name, func(t *testing.T) {
					verifyCandidateWindowsCaptureLimit(t, image, api, test.stdout, test.stderr, test.limit, test.kind)
				})
			}
		}
		for _, api := range []string{"run", "input", "environment"} {
			for _, test := range []captureCase{
				{"default-exact", 32768, 32768, 65536, 0},
				{"default-one-over", 32768, 32769, 65536, 11},
			} {
				t.Run(api+"/"+test.name, func(t *testing.T) {
					verifyCandidateWindowsCaptureLimit(t, image, api, test.stdout, test.stderr, test.limit, test.kind)
				})
			}
		}
		// Exercise the inclusive upper bound using small repeated binary writes.
		t.Run("maximum-exact", func(t *testing.T) {
			verifyCandidateWindowsCaptureLimit(t, image, "environment-limit", 32<<20, 32<<20, 64<<20, 0)
		})
	})
}

func verifyCandidateWindowsCaptureLimit(t *testing.T, image, api string, stdout, stderr, limit, kind int) {
	t.Helper()
	directory := t.TempDir()
	marker := filepath.Join(directory, "child.done")
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, image, api, fmt.Sprint(stdout), fmt.Sprint(stderr), fmt.Sprint(limit), marker, directory)
	output, err := command.CombinedOutput()
	if err != nil || string(output) != fmt.Sprint(kind) {
		t.Fatalf("capture %s (%d+%d, limit %d): %v; output=%q, want kind %d", api, stdout, stderr, limit, err, output, kind)
	}
	data, err := os.ReadFile(marker)
	if kind == 1 {
		if !os.IsNotExist(err) {
			t.Fatalf("invalid limit launched child: marker=%q, error=%v", data, err)
		}
	} else if err != nil || string(data) != "done" {
		t.Fatalf("child did not finish draining its writes: marker=%q, error=%v", data, err)
	}
}

const candidateWindowsCaptureLimitMain = `module main;
import windows_run_capture as capture;
import windows_run_control as control;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;
import std::fs as fs;
import std::text as text;

fn bytes::Buffer argument(usize index){
    var u8[4096] data=[0;4096];
    let size=switch(process::arg(index,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};
    let ref u8[] view=ref data;
    return switch(bytes::copy(view[..size])){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};
}
fn usize number(ref u8[] data){
    var usize value=0;var usize i=0;
    while(i<data.len){assert(data[i]>=48 && data[i]<=57);value=value*10+(data[i] as usize)-48;i+=1;}
    return value;
}
fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){
    let copy=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(v::push<bytes::Buffer>(args,copy).is_ok);
}
@effects(console) fn void emit(usize count,bool error){
    var u8[4096] chunk=[0;4096];var usize i=0;
    let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";let u8 top=255;
    while(i<4096){let value=(digits[(i/16)%16]<<4)|digits[i%16];chunk[i]=if(error){top-value}else{value};i+=1;}
    let ref u8[] view=ref chunk;var usize remaining=count;
    while(remaining!=0){let usize size=if(remaining>4096){4096}else{remaining};
        let written=switch(if(error){io::write_stderr_bytes(view[..size])}else{io::write_bytes(view[..size])}){case .ok(n):{n}case .err(_):{assert(false);return;}};
        assert(written==size);remaining-=size;
    }
}
fn void pattern(ref u8[] data,usize expected,bool error){
    assert(data.len==expected);var usize i=0;
    let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";let u8 top=255;
    while(i<data.len){let value=(digits[(i/16)%16]<<4)|digits[i%16];assert(data[i]==if(error){top-value}else{value});i+=1;}
}
@effects(filesystem) fn void mark(ref u8[] path){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var file=switch(fs::create_truncate_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(switch(fs::write(ref mut file,b"done")){case .ok(n):{n==4}case .err(_):{false}});
    assert(fs::close(file).is_ok);
}
@effects(filesystem,console,environment,clock) fn i32 main(){
    let api=argument(1);let output=argument(2);let error=argument(3);
    let out_size=number(bytes::view(ref output));let err_size=number(bytes::view(ref error));
    if(bytes::equal(bytes::view(ref api),b"child")){
        emit(out_size,false);emit(err_size,true);let marker=argument(4);mark(bytes::view(ref marker));return 42;
    }
    let raw_limit=argument(4);let limit=number(bytes::view(ref raw_limit));
    let marker=argument(5);let directory=argument(6);let program=argument(0);
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();
    add(ref mut args,b"child");add(ref mut args,bytes::view(ref output));add(ref mut args,bytes::view(ref error));add(ref mut args,bytes::view(ref marker));
    let inherited=switch(control::snapshot()){case .ok(p):{p}case .err(_):{assert(false);return 2;}};
    let result=if(bytes::equal(bytes::view(ref api),b"run")){
        capture::run(bytes::view(ref program),ref args,bytes::view(ref directory),120000)
    }else if(bytes::equal(bytes::view(ref api),b"input")){
        capture::run_input(bytes::view(ref program),ref args,bytes::view(ref directory),120000,1,b"")
    }else if(bytes::equal(bytes::view(ref api),b"environment")){
        capture::run_environment(bytes::view(ref program),ref args,bytes::view(ref directory),120000,1,b"",bytes::view(ref inherited))
    }else if(bytes::equal(bytes::view(ref api),b"input-limit")){
        capture::run_input_limit(bytes::view(ref program),ref args,bytes::view(ref directory),120000,1,b"",limit)
    }else{
        assert(bytes::equal(bytes::view(ref api),b"environment-limit"));
        // An invalid environment must not mask invalid-limit classification.
        capture::run_environment_limit(bytes::view(ref program),ref args,bytes::view(ref directory),120000,1,b"",if(limit==0 || limit>67108864){b""}else{bytes::view(ref inherited)},limit)
    };
    switch(result){
        case .ok(value):{
            assert(capture::status(ref value)==42 && capture::trap(ref value)==0 && capture::pid(ref value)!=0);
            pattern(capture::stdout(ref value),out_size,false);pattern(capture::stderr(ref value),err_size,true);
            assert(io::write_u32(0).is_ok);
        }
        case .err(problem):{
            assert(io::write_u32(problem.kind).is_ok);
            if(limit==0 || limit>67108864){assert(problem.kind==1 && problem.os_code==0 && problem.pid==0);}
            else{assert(problem.kind==11 && problem.os_code==0 && problem.pid!=0);}
        }
    };
    return 0;
}
`
