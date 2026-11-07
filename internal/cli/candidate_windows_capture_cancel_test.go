//go:build windows

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"golang.org/x/sys/windows"
)

const candidateCaptureCancelMain = `module main;
import windows_cancel as cancel;import windows_run_capture as capture;import std::process as process;import std::bytes as bytes;import std::vec as v;import std::io as console;
@link_name("GetCurrentProcess") unsafe extern C fn usize current();
@link_name("GetProcessHandleCount") unsafe extern C fn i32 handles(usize process,ptr mut u32 count);
fn u32 count(){var u32 value=0;assert(unsafe{handles(current(),ref mut value as ptr mut u32)}!=0);return value;}
@effects(console)fn void number(usize value){var out=bytes::empty();assert(bytes::append_usize(ref mut out,value).is_ok && bytes::append_byte(ref mut out,10).is_ok);discard console::write_bytes(bytes::view(ref out));}
@effects(filesystem,console,environment,clock)fn i32 main(){
 var u8[4096] a=[0;4096];var u8[4096] b=[0;4096];var u8[4096] c=[0;4096];var u8[4096] d=[0;4096];
 let na=switch(process::arg(1,ref mut a)){case .ok(p):{p}case .err(_):{return 1;}};let nb=switch(process::arg(2,ref mut b)){case .ok(p):{p}case .err(_):{return 2;}};let nc=switch(process::arg(3,ref mut c)){case .ok(p):{p}case .err(_):{return 3;}};let nd=switch(process::arg(4,ref mut d)){case .ok(p):{p}case .err(_):{return 4;}};
 let ref u8[] program=ref a;let ref u8[] root=ref b;let ref u8[] name=ref c;let ref u8[] mode=ref d;
 let token=switch(cancel::open(name[..nc])){case .ok(p):{p}case .err(_):{return 5;}};
 var args=v::empty<bytes::Buffer>();assert(v::push<bytes::Buffer>(ref mut args,switch(bytes::copy(b"parent")){case .ok(p):{p}case .err(_):{return 6;}}).is_ok);assert(v::push<bytes::Buffer>(ref mut args,switch(bytes::copy(root[..nb])){case .ok(p):{p}case .err(_):{return 7;}}).is_ok);
 // First CreateProcessW may initialize process-wide Windows facilities. Two
 // ordinary completed calls establish a stable baseline before cancellation.
 if(bytes::equal(mode[..nd],b"running")){
  var warm=v::empty<bytes::Buffer>();assert(v::push<bytes::Buffer>(ref mut warm,switch(bytes::copy(b"warm")){case .ok(p):{p}case .err(_):{return 9;}}).is_ok);assert(v::push<bytes::Buffer>(ref mut warm,switch(bytes::copy(root[..nb])){case .ok(p):{p}case .err(_):{return 10;}}).is_ok);
  var usize i=0;var u32 warmed=0;while(i<2){{let result=switch(capture::run_input_limit(program[..na],ref warm,root[..nb],60000,1,b"",65536)){case .ok(p):{p}case .err(_):{return 11;}};assert(capture::status(ref result)==0);}
   if(i==0){warmed=count();}else{assert(count()==warmed);}i+=1;}
 }
 let before=count();var u32 after=0;let u32 timeout=if(bytes::equal(mode[..nd],b"invalid")){0}else{60000};
 switch(capture::run_input_cancel(program[..na],ref args,root[..nb],timeout,1,b"",65536,ref token)){case .ok(_):{return 8;}case .err(e):{after=count();number(e.kind as usize);number(e.os_code as usize);number(e.pid as usize);number(before as usize);number(after as usize);}};
 assert(after==before);return 0;
}`

// The leaf reports both PIDs before sleeping. Cancellation is signaled only
// after Go observes both live processes, without relying on elapsed-time races.
const candidateCaptureCancelChild = `
#include <windows.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wchar.h>
int main(int argc,char **argv){
    if(argc<3)return 2;
    if(strcmp(argv[1],"warm")==0)return 0;
    if(strcmp(argv[1],"leaf")==0){
        if(argc!=4)return 3;
        DWORD pids[2]={ (DWORD)strtoul(argv[3],NULL,10),GetCurrentProcessId() };
        HANDLE file=CreateFileA(argv[2],GENERIC_WRITE,FILE_SHARE_READ,NULL,CREATE_ALWAYS,FILE_ATTRIBUTE_NORMAL,NULL);
        if(file==INVALID_HANDLE_VALUE)return 4;
        DWORD written=0;
        if(!WriteFile(file,pids,sizeof pids,&written,NULL)||written!=sizeof pids||!FlushFileBuffers(file)){CloseHandle(file);return 5;}
        CloseHandle(file);Sleep(INFINITE);return 6;
    }
    wchar_t image[32768],root[32768],line[65536];
    DWORD size=GetModuleFileNameW(NULL,image,32768);
    if(!size||size>=32768||!MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,argv[2],-1,root,32768))return 7;
    if(swprintf_s(line,65536,L"\"%ls\" leaf \"%ls\\ready.bin\" %lu",image,root,GetCurrentProcessId())<0)return 8;
    STARTUPINFOW startup;PROCESS_INFORMATION process;ZeroMemory(&startup,sizeof startup);startup.cb=sizeof startup;ZeroMemory(&process,sizeof process);
    if(!CreateProcessW(image,line,NULL,NULL,FALSE,0,NULL,NULL,&startup,&process))return 9;
    CloseHandle(process.hThread);CloseHandle(process.hProcess);Sleep(INFINITE);return 10;
}
`

// These small host modules need no partition-count assumption. Compile their
// actual import closure with the same native Windows driver and strict flags.
func candidateWindowsCancellationProbe(t *testing.T, module, main string, verify func(*testing.T, string)) {
	t.Helper()
	if runtime.GOARCH != "amd64" {
		t.Skip("Windows cancellation requires amd64 MSVC")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, module)
	writeTestSource(t, root, "src/main.fy", main)
	// Only the independent test oracles import Kernel32 directly. Product
	// cancellation uses the same closed private host selector as capture.
	unit, support, tc := windowsCandidateProbe(t, root, "GetLastError", "SetLastError", "GetCurrentProcess", "GetProcessHandleCount")
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, module+"-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			verify(t, path)
		})
	}
}

func TestCandidateWindowsCaptureCancellation(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := cdriver.BuildWindowsMSVCProbe(tc, cdriver.LowProfile, candidateCaptureCancelChild)
	if err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(t.TempDir(), "tree.exe")
	if err = os.WriteFile(child, artifact.Bytes, 0700); err != nil {
		t.Fatal(err)
	}
	candidateWindowsCancellationProbe(t, "windows_run_capture", candidateCaptureCancelMain, func(t *testing.T, image string) {
		for _, mode := range []string{"before", "invalid", "running"} {
			t.Run(mode, func(t *testing.T) {
				root := t.TempDir()
				name, err := windows.UTF16PtrFromString(fmt.Sprintf("Local\\fairyc-cancel-%x", sha256.Sum256([]byte(root))))
				if err != nil {
					t.Fatal(err)
				}
				event, err := windows.CreateEvent(nil, 1, 0, name)
				if err != nil {
					t.Fatal(err)
				}
				defer windows.CloseHandle(event)
				if mode != "running" {
					if err = windows.SetEvent(event); err != nil {
						t.Fatal(err)
					}
					code, output := candidateAtlasProcess(t, image, nil, child, filepath.ToSlash(root), windows.UTF16PtrToString(name), mode)
					kind := 12
					if mode == "invalid" {
						kind = 1
					}
					var gotKind, native, pid, before, after uint32
					n, scan := fmt.Sscanf(output, "%d\n%d\n%d\n%d\n%d", &gotKind, &native, &pid, &before, &after)
					if code != 0 || scan != nil || n != 5 || gotKind != uint32(kind) || native != 0 || pid != 0 || before != after {
						t.Fatalf("prelaunch outcome=%d %q", code, output)
					}
					if _, err = os.Stat(filepath.Join(root, "ready.bin")); !os.IsNotExist(err) {
						t.Fatal("prelaunch refusal created a child marker", err)
					}
					return
				}
				ctx, stop := context.WithTimeout(t.Context(), 20*time.Second)
				defer stop()
				command := exec.CommandContext(ctx, image, child, filepath.ToSlash(root), windows.UTF16PtrToString(name), mode)
				command.WaitDelay = 2 * time.Second
				var stdout, stderr bytes.Buffer
				command.Stdout, command.Stderr = &stdout, &stderr
				if err = command.Start(); err != nil {
					t.Fatal(err)
				}
				done := make(chan error, 1)
				go func() { done <- command.Wait() }()
				t.Cleanup(func() { _ = command.Process.Kill() })
				var pids []byte
				for len(pids) != 8 {
					select {
					case err = <-done:
						t.Fatalf("caller ended before live tree: %v %q %q", err, &stdout, &stderr)
					case <-ctx.Done():
						t.Fatal("waiting for live parent/leaf", ctx.Err())
					case <-time.After(5 * time.Millisecond):
						pids, _ = os.ReadFile(filepath.Join(root, "ready.bin"))
					}
				}
				var processes []windows.Handle
				for _, pid := range []uint32{binary.LittleEndian.Uint32(pids[:4]), binary.LittleEndian.Uint32(pids[4:])} {
					handle, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
					if err != nil {
						t.Fatal(err)
					}
					defer windows.CloseHandle(handle)
					if status, err := windows.WaitForSingleObject(handle, 0); err != nil || status != uint32(windows.WAIT_TIMEOUT) {
						t.Fatal("tree was not live before cancellation", status, err)
					}
					processes = append(processes, handle)
				}
				if err = windows.SetEvent(event); err != nil {
					t.Fatal(err)
				}
				if err = <-done; err != nil || stderr.Len() != 0 {
					t.Fatalf("caller=%v stdout=%q stderr=%q", err, &stdout, &stderr)
				}
				var gotKind, native, pid, before, after uint32
				if n, scan := fmt.Sscanf(strings.TrimSpace(stdout.String()), "%d\n%d\n%d\n%d\n%d", &gotKind, &native, &pid, &before, &after); scan != nil || n != 5 || gotKind != 12 || native != 0 || pid != binary.LittleEndian.Uint32(pids[:4]) || before != after {
					t.Fatalf("cancellation cause/PID/handle counts=%q", &stdout)
				}
				for _, process := range processes {
					if status, err := windows.WaitForSingleObject(process, 2000); err != nil || status != windows.WAIT_OBJECT_0 {
						t.Fatal("canceled tree survived", status, err)
					}
				}
			})
		}
	})
}
