//go:build windows

package candidatehost

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"golang.org/x/sys/windows"
)

// Compiled after the EXACT embedded product C, without substitutions. Only this
// isolated child generates console controls. The Go host never attaches to or
// generates signals on any existing console.
const windowsConsoleCancelProbe = `
#include <stdio.h>
#include <string.h>
#define REQUIRE(x) do { if (!(x)) { fprintf(stderr,"line=%d error=%lu\n",__LINE__,(unsigned long)GetLastError()); return __LINE__; } } while(0)
static DWORD probe_handles(void) {
    DWORD count=0;
    if (!GetProcessHandleCount(GetCurrentProcess(),&count)) { ExitProcess(200); }
    return count;
}
static BOOL WINAPI probe_fallback(DWORD control) {
    return control == CTRL_C_EVENT || control == CTRL_BREAK_EVENT;
}
static volatile LONG probe_stop=0;
static volatile LONG probe_iterations=0;
static HANDLE probe_started;
static DWORD WINAPI probe_callbacks(void *unused) {
    (void)unused;
    (void)SetEvent(probe_started);
    while (InterlockedCompareExchange(&probe_stop,0,0)==0) {
        (void)fy_windows_console_cancel_handler(CTRL_C_EVENT);
        (void)fy_windows_console_cancel_handler(CTRL_BREAK_EVENT);
        InterlockedIncrement(&probe_iterations);
    }
    return 0;
}
int main(int argc,char **argv) {
    REQUIRE(argc==2);
    DWORD clients[4];
    // A NEW console must contain only this child. Group zero cannot escape it.
    REQUIRE(GetConsoleProcessList(clients,4)==1 && clients[0]==GetCurrentProcessId());
    if (strcmp(argv[1],"headless")==0) {
        REQUIRE(FreeConsole());
        REQUIRE(GetConsoleProcessList(clients,4)==0 && GetLastError()==ERROR_INVALID_HANDLE);
        DWORD before=probe_handles();
        HANDLE token=fairyc_private_windows_cancel_local_v1();REQUIRE(token!=NULL);
        for (unsigned i=0;i<32;i++) {
            SetLastError(12345);
            REQUIRE(fairyc_private_windows_cancel_console_bind_v1(token)==NULL && GetLastError()==ERROR_INVALID_HANDLE);
            REQUIRE(probe_handles()==before+1);
        }
        REQUIRE(CloseHandle(token) && probe_handles()==before);
        puts("headless error=6 no duplicate/registration leaks");return 0;
    }
    REQUIRE(SetConsoleCtrlHandler(NULL,FALSE));
    REQUIRE(SetConsoleCtrlHandler(probe_fallback,TRUE));
    REQUIRE(!fy_windows_console_cancel_handler(CTRL_CLOSE_EVENT));
    REQUIRE(!fy_windows_console_cancel_handler(CTRL_LOGOFF_EVENT));
    REQUIRE(!fy_windows_console_cancel_handler(CTRL_SHUTDOWN_EVENT));
    DWORD before=probe_handles();
    REQUIRE(fairyc_private_windows_cancel_console_bind_v1(NULL)==NULL && GetLastError()==ERROR_INVALID_HANDLE);
    REQUIRE(probe_handles()==before);
    if (strcmp(argv[1],"poison")==0) {
        HANDLE token=fairyc_private_windows_cancel_local_v1();REQUIRE(token!=NULL);
        void *registration=fairyc_private_windows_cancel_console_bind_v1(token);REQUIRE(registration!=NULL);
        REQUIRE(CloseHandle(token));
        // Actual OS handler removal by another component, not a fake API.
        REQUIRE(SetConsoleCtrlHandler(fy_windows_console_cancel_handler,FALSE));
        REQUIRE(!SetConsoleCtrlHandler(fy_windows_console_cancel_handler,FALSE));
        DWORD original_error=GetLastError();REQUIRE(original_error!=0);
        REQUIRE(!fairyc_private_windows_cancel_console_close_v1(registration) && GetLastError()==original_error);
        REQUIRE(probe_handles()==before);
        for (unsigned i=0;i<32;i++) {
            REQUIRE(!fairyc_private_windows_cancel_console_close_v1(registration) && GetLastError()==ERROR_INVALID_HANDLE);
            HANDLE next=fairyc_private_windows_cancel_local_v1();REQUIRE(next!=NULL);
            REQUIRE(fairyc_private_windows_cancel_console_bind_v1(next)==NULL && GetLastError()==ERROR_BUSY);
            REQUIRE(CloseHandle(next) && probe_handles()==before);
        }
        puts("unregister error preserved; duplicate freed; future binds poisoned=170");return 0;
    }
    if (strcmp(argv[1],"race")==0) {
        probe_started=CreateEventW(NULL,TRUE,FALSE,NULL);REQUIRE(probe_started!=NULL);
        HANDLE thread=CreateThread(NULL,0,probe_callbacks,NULL,0,NULL);REQUIRE(thread!=NULL);
        REQUIRE(WaitForSingleObject(probe_started,5000)==WAIT_OBJECT_0);
        for (unsigned i=0;i<512;i++) {
            HANDLE token=fairyc_private_windows_cancel_local_v1();REQUIRE(token!=NULL);
            void *registration=fairyc_private_windows_cancel_console_bind_v1(token);REQUIRE(registration!=NULL);
            REQUIRE(CloseHandle(token));
            REQUIRE(fairyc_private_windows_cancel_console_close_v1(registration));
        }
        InterlockedExchange(&probe_stop,1);
        REQUIRE(WaitForSingleObject(thread,5000)==WAIT_OBJECT_0 && probe_iterations>0);
        REQUIRE(CloseHandle(thread) && CloseHandle(probe_started));
        REQUIRE(probe_handles()==before);
        printf("512 bind/close races; actual handler iterations=%ld; stable handles\n",(long)probe_iterations);
        return 0;
    }
    DWORD control=strcmp(argv[1],"ctrl-c")==0 ? CTRL_C_EVENT : CTRL_BREAK_EVENT;
    for (unsigned i=0;i<16;i++) {
        HANDLE token=fairyc_private_windows_cancel_local_v1();REQUIRE(token!=NULL);
        DWORD flags=HANDLE_FLAG_INHERIT;
        REQUIRE(GetHandleInformation(token,&flags) && (flags&HANDLE_FLAG_INHERIT)==0);
        REQUIRE(WaitForSingleObject(token,0)==WAIT_TIMEOUT);
        HANDLE observer=NULL;
        REQUIRE(DuplicateHandle(GetCurrentProcess(),token,GetCurrentProcess(),&observer,SYNCHRONIZE,FALSE,0));
        void *registration=fairyc_private_windows_cancel_console_bind_v1(token);REQUIRE(registration!=NULL);
        DWORD bound=probe_handles();
        REQUIRE(fairyc_private_windows_cancel_console_bind_v1(token)==NULL && GetLastError()==ERROR_BUSY);
        REQUIRE(probe_handles()==bound);
        // Registration owns a duplicate: discard the originating Token first.
        REQUIRE(CloseHandle(token));
        REQUIRE(!fy_windows_console_cancel_handler(CTRL_CLOSE_EVENT));
        REQUIRE(WaitForSingleObject(observer,0)==WAIT_TIMEOUT);
        REQUIRE(GenerateConsoleCtrlEvent(control,0));
        REQUIRE(WaitForSingleObject(observer,5000)==WAIT_OBJECT_0);
        REQUIRE(WaitForSingleObject(observer,0)==WAIT_OBJECT_0);
        REQUIRE(fairyc_private_windows_cancel_console_close_v1(registration));
        REQUIRE(!fairyc_private_windows_cancel_console_close_v1(registration) && GetLastError()==ERROR_INVALID_HANDLE);
        REQUIRE(CloseHandle(observer));
        REQUIRE(probe_handles()==before);
    }
    printf("actual control=%lu x16; token-close-first/collision/rebind/latch/handles passed\n",(unsigned long)control);
    return 0;
}
`

func TestWindowsConsoleCancellationNative(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("requires Windows amd64 MSVC")
	}
	t.Logf("authoritative windows_process.c sha256=%x; probe sha256=%x", sha256.Sum256([]byte(windowsProcess)), sha256.Sum256([]byte(windowsConsoleCancelProbe)))
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, "#define STRICT 1\n"+windowsProcess+windowsConsoleCancelProbe)
			if err != nil {
				t.Fatal(err)
			}
			image := filepath.Join(t.TempDir(), "console-cancel.exe")
			if err = os.WriteFile(image, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"headless", "ctrl-c", "ctrl-break", "race", "poison"} {
				t.Run(mode, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
					defer cancel()
					command := exec.CommandContext(ctx, image, mode)
					command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE, HideWindow: true}
					command.WaitDelay = 2 * time.Second
					var stdout, stderr bytes.Buffer
					command.Stdout, command.Stderr = &stdout, &stderr
					if err := command.Run(); err != nil || stderr.Len() != 0 {
						t.Fatalf("isolated console mode=%s err=%v stdout=%q stderr=%q", mode, err, &stdout, &stderr)
					}
					t.Logf("%s", &stdout)
				})
			}
		})
	}
}
