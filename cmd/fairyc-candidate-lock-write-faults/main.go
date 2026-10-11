// Test-only Win32 interception of the Fairy-owned lock writer. No save algorithm
// or compiler frontend is implemented here. Close injection closes the real
// handle before reporting an application-visible error; it is not a handle leak.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func hash(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func main() {
	if len(os.Args) != 4 {
		panic("usage: lock-write-faults PACKET CANONICAL_LOCK OUTPUT")
	}
	out, e := filepath.Abs(os.Args[3])
	must(e)
	must(os.MkdirAll(out, 0700))
	input, e := os.ReadFile(os.Args[2])
	must(e)
	raw, e := os.ReadFile(os.Args[1])
	must(e)
	must(os.WriteFile(filepath.Join(out, "packet.bin"), raw, 0600))
	packet, e := candidatebuild.Decode(raw)
	must(e)
	set := map[string]bool{}
	for _, m := range regexp.MustCompile(`\b(fy_std_[A-Za-z0-9_]+)\s*\(`).FindAllStringSubmatch(packet.Header, -1) {
		set[m[1]] = true
	}
	var symbols []string
	for s := range set {
		symbols = append(symbols, s)
	}
	sort.Strings(symbols)
	storage, e := cstdlib.WindowsMSVCCandidateSourceForSymbols(symbols, strings.Contains(packet.Header, "fy_private_vector_"))
	must(e)
	support, e := cruntime.ScalarSource(100)
	must(e)
	support += "\n" + hooks + "\n" + strings.NewReplacer("WriteFile(", "lock_write(", "FlushFileBuffers(", "lock_sync(", "CloseHandle(", "lock_close(", "MoveFileExW(", "lock_move(", "DeleteFileW(", "lock_delete(").Replace(storage)
	tool, e := cdriver.ResolveWindowsHostMSVC()
	must(e)
	var observations []map[string]any
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, e := cdriver.BuildWindowsMSVCTargetPartsWithTimeout(tool, profile, packet.Header, packet.Literals, packet.Units, support, 2*time.Minute)
		must(e)
		image := filepath.Join(out, profile.Name+".exe")
		must(os.WriteFile(image, artifact.Bytes, 0700))
		for _, mode := range []string{"ok", "first-write", "partial-write", "sync", "close", "replace", "secondary", "cleanup", "during-write", "before", "after"} {
			// Abrupt termination and cleanup-failure intentionally leave a sibling.
			// Fresh roots make repeated runs independent without removing evidence.
			parent := filepath.Join(out, profile.Name)
			must(os.MkdirAll(parent, 0700))
			root, e := os.MkdirTemp(parent, mode+"-")
			must(e)
			live := filepath.Join(root, "fairy.lock")
			tmp := filepath.Join(root, ".fairy-lock-0.tmp")
			must(os.WriteFile(live, []byte("OLD"), 0600))
			must(os.WriteFile(filepath.Join(root, "input.json"), input, 0600))
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			cmd := exec.CommandContext(ctx, image, root, mode)
			cmd.Env = append(os.Environ(), "FY_LOCK_FAULT="+mode)
			cmd.WaitDelay = time.Second
			abrupt := mode == "during-write" || mode == "before" || mode == "after"
			if abrupt {
				pipe, e := cmd.StdoutPipe()
				must(e)
				must(cmd.Start())
				var marker [1]byte
				n, e := pipe.Read(marker[:])
				if e != nil || n != 1 || marker[0] != 'R' {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
					panic(fmt.Sprintf("missing checkpoint: %s %d %v", mode, n, e))
				}
				must(cmd.Process.Kill())
				if cmd.Wait() == nil {
					panic("termination failed")
				}
			} else {
				b, e := cmd.CombinedOutput()
				if e != nil || len(b) != 0 {
					panic(fmt.Sprintf("%s/%s: %v %s", profile.Name, mode, e, b))
				}
			}
			cancel()
			want := []byte("OLD")
			if mode == "ok" || mode == "after" {
				want = input
			}
			actual, e := os.ReadFile(live)
			must(e)
			if string(actual) != string(want) {
				panic("live bytes changed: " + mode)
			}
			temporary, e := os.ReadFile(tmp)
			retained := mode == "cleanup" || mode == "during-write" || mode == "before"
			if retained {
				must(e)
				expected := input
				if mode == "during-write" {
					expected = input[:2]
				}
				if string(temporary) != string(expected) {
					panic("retained temporary mismatch: " + mode)
				}
			} else if !os.IsNotExist(e) {
				panic("owned temporary not cleaned: " + mode)
			}
			observations = append(observations, map[string]any{"profile": profile.Name, "mode": mode, "terminated": abrupt, "liveSHA256": hash(actual), "temporaryRetained": retained, "imageSHA256": hash(artifact.Bytes)})
			fmt.Println(profile.Name, mode, "passed")
		}
	}
	receipt := map[string]any{"schema": "fairyc.candidate-lock-write-faults/v1", "passed": true, "packetSHA256": hash(raw), "runtimeSHA256": hash([]byte(support)), "inputSHA256": hash(input), "observations": observations, "powerLossClaim": false, "closeInjection": "real handle closed before injected error", "productionRuntimeChanged": false}
	b, e := json.MarshalIndent(receipt, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(out, "receipt.json"), append(b, '\n'), 0600))
}

const hooks = `
#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <wchar.h>
static int lock_calls;
static int lock_mode(const char *want){char value[64];DWORD n=GetEnvironmentVariableA("FY_LOCK_FAULT",value,64);return n>0 && n<64 && !strcmp(value,want);}
static int lock_handle(HANDLE h){WCHAR path[4096];DWORD n=GetFinalPathNameByHandleW(h,path,4096,0);return n>0 && n<4096 && wcsstr(path,L".fairy-lock-")!=NULL;}
static void lock_checkpoint(void){fputc('R',stdout);fflush(stdout);for(;;)Sleep(1000);}
static BOOL lock_write(HANDLE h,LPCVOID b,DWORD n,LPDWORD done,LPOVERLAPPED o){
 if(lock_handle(h)){
  if(lock_mode("first-write") || lock_mode("secondary")){*done=0;SetLastError(ERROR_DISK_FULL);return FALSE;}
  if(lock_mode("partial-write") || lock_mode("during-write")){if(lock_calls++==0)return WriteFile(h,b,2,done,o);if(lock_mode("during-write"))lock_checkpoint();*done=0;SetLastError(ERROR_DISK_FULL);return FALSE;}
 }return WriteFile(h,b,n,done,o);
}
static BOOL lock_sync(HANDLE h){if(lock_handle(h) && (lock_mode("sync") || lock_mode("cleanup"))){SetLastError(ERROR_WRITE_FAULT);return FALSE;}return FlushFileBuffers(h);}
static BOOL lock_close(HANDLE h){int inject=lock_handle(h) && (lock_mode("close") || lock_mode("secondary"));BOOL ok=CloseHandle(h);if(inject){SetLastError(ERROR_INVALID_HANDLE);return FALSE;}return ok;}
static BOOL lock_move(LPCWSTR a,LPCWSTR b,DWORD flags){if(wcsstr(a,L".fairy-lock-")){if(lock_mode("replace")){SetLastError(ERROR_DISK_FULL);return FALSE;}if(lock_mode("before"))lock_checkpoint();BOOL ok=MoveFileExW(a,b,flags);if(ok && lock_mode("after"))lock_checkpoint();return ok;}return MoveFileExW(a,b,flags);}
static BOOL lock_delete(LPCWSTR path){if(wcsstr(path,L".fairy-lock-") && lock_mode("cleanup")){SetLastError(ERROR_ACCESS_DENIED);return FALSE;}return DeleteFileW(path);}
`
