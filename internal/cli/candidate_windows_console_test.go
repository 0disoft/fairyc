//go:build windows

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"golang.org/x/sys/windows"
)

const windowsConsoleProbe = `module main;
import windows_cancel as cancel;import std::process as process;import std::io as io;
@effects(filesystem) @link_name("fairyc_test_console_enable") unsafe extern C fn i32 enable();
@effects(filesystem) @link_name("GenerateConsoleCtrlEvent") unsafe extern C fn i32 control(u32 kind,u32 group);
@effects(filesystem) @link_name("FreeConsole") unsafe extern C fn i32 detach();
@effects(clock) @link_name("Sleep") unsafe extern C fn void pause(u32 ms);
@effects(filesystem) @link_name("GetProcessHandleCount") unsafe extern C fn i32 handles(ptr mut void process,ptr mut u32 count);
@link_name("GetCurrentProcess") unsafe extern C fn ptr mut void current();
@effects(filesystem) @link_name("SetLastError") unsafe extern C fn void set_error(u32 code);
@effects(filesystem) @link_name("GetLastError") unsafe extern C fn u32 last_error();
@effects(filesystem)fn u32 count(){var u32 n=0;assert(unsafe{handles(current(),ref mut n as ptr mut u32)}!=0);return n;}
@effects(filesystem,clock)fn void cycle(u32 kind,bool headless){
 let token=switch(cancel::local()){case .ok(p):{p}case .err(_):{assert(false);cancel::none()}};
 assert(!switch(cancel::requested(ref token)){case .ok(p):{p}case .err(_):{true}});
 let binding=switch(cancel::console(ref token)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(cancel::active(ref binding)==!headless);
 if(headless){discard cancel::signal(ref token);}else{
  let other=switch(cancel::local()){case .ok(p):{p}case .err(_):{assert(false);cancel::none()}};
  switch(cancel::console(ref other)){case .ok(_):{assert(false);}case .err(e):{assert(e.kind==8 && e.os_code==170);}};
  assert(!switch(cancel::requested(ref other)){case .ok(p):{p}case .err(_):{true}});
  assert(unsafe{control(kind,0)}!=0);
 }
 var usize waited=0;while(!switch(cancel::requested(ref token)){case .ok(p):{p}case .err(_):{assert(false);false}}){assert(waited<1000);unsafe{pause(5);};waited+=1;}
 assert(switch(cancel::close_console(binding)){case .ok(p):{p}case .err(_):{false}});
 assert(switch(cancel::requested(ref token)){case .ok(p):{p}case .err(_):{false}});
}
@effects(filesystem,console,clock)fn i32 main(){
 let empty=cancel::none();unsafe{set_error(12345);};switch(cancel::console(ref empty)){case .ok(_):{return 1;}case .err(e):{assert(e.kind==1 && e.os_code==0);}};assert(unsafe{last_error()}==12345);
 let headless=process::arg_count()==2;if(headless){assert(unsafe{detach()}!=0);}else{assert(unsafe{enable()}!=0);}
 cycle(0,headless);let before=count();var usize i=0;while(i<32){cycle(if(i%2==0){0}else{1},headless);i+=1;}let after=count();assert(after==before);
 assert(io::write_bytes(if(headless){b"headless-ok\n"}else{b"control-ok\n"}).is_ok);return 0;
}`

func windowsConsoleBuild(t *testing.T, main string, observe func(*testing.T, string) string, verify func(*testing.T, string)) {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "windows_cancel", "module_state_command_windows")
	writeTestSource(t, root, "src/main.fy", main)
	partBytes := 512 << 10
	var imports map[string]bool
	if main == windowsConsoleProbe {
		partBytes = 16 << 10
		imports = map[string]bool{"FreeConsole": true, "GenerateConsoleCtrlEvent": true,
			"fairyc_test_console_enable": true, "Sleep": true, "GetProcessHandleCount": true,
			"GetCurrentProcess": true, "GetLastError": true, "SetLastError": true}
	}
	build, support := candidateParityBuildPartsWithLimit(t, root, imports, partBytes)
	if observe != nil {
		support = observe(t, support)
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			t.Logf("actual entry sha256=%x; image sha256=%x; support sha256=%x", sha256.Sum256([]byte(main)), sha256.Sum256(image.Bytes), sha256.Sum256([]byte(support)))
			verify(t, path)
		})
	}
}

func windowsIsolatedConsoleProcess(t *testing.T, image string, args ...string) (int, []byte, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.WaitDelay = 2 * time.Second
	// Never join or broadcast to the test runner's/user's console.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE, HideWindow: true}
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, out.Bytes(), stderr.Bytes()
}

func TestCandidateWindowsConsoleCancellation(t *testing.T) {
	windowsConsoleBuild(t, windowsConsoleProbe, func(t *testing.T, support string) string {
		return support + "\nint32_t fairyc_test_console_enable(void) { return SetConsoleCtrlHandler(NULL, FALSE); }\n"
	}, func(t *testing.T, image string) {
		for _, headless := range []bool{false, true} {
			t.Run(fmt.Sprintf("headless-%v", headless), func(t *testing.T) {
				var args []string
				want := []byte("control-ok\n")
				if headless {
					args = []string{"headless"}
					want = []byte("headless-ok\n")
				}
				code, out, stderr := windowsIsolatedConsoleProcess(t, image, args...)
				if code != 0 || !bytes.Equal(out, want) || len(stderr) != 0 {
					t.Fatalf("console lifecycle code=%d stdout=%q stderr=%q", code, out, stderr)
				}
				t.Log("33 binding lifetimes; stable handles; latched token after consuming close; exact unsupported-none refusal")
			})
		}
	})
}

func windowsConsoleEntryObserver(t *testing.T, support string, kind uint32) string {
	t.Helper()
	local := "void *fairyc_private_windows_cancel_local_v1(void) {"
	image := "uint32_t fairyc_private_windows_image_path_v1(uint8_t *data, uint32_t capacity) {"
	for _, def := range []string{local, image} {
		if strings.Count(support, def) != 1 {
			t.Fatalf("observer requires exactly one native definition: %s", def)
		}
	}
	renamedLocal := strings.Replace(local, "fairyc_private_windows_cancel_local_v1", "fairyc_test_original_local", 1)
	renamedImage := strings.Replace(image, "fairyc_private_windows_image_path_v1", "fairyc_test_original_image", 1)
	changed := strings.Replace(strings.Replace(support, local, renamedLocal, 1), image, renamedImage, 1)
	observer := fmt.Sprintf(`
static HANDLE fairyc_test_observed_token = NULL;
void *fairyc_private_windows_cancel_local_v1(void) {
    void *result = fairyc_test_original_local(); DWORD saved = GetLastError();
    fairyc_test_observed_token = (HANDLE)result; SetLastError(saved); return result;
}
uint32_t fairyc_private_windows_image_path_v1(uint8_t *data, uint32_t capacity) {
    uint32_t result = fairyc_test_original_image(data, capacity); DWORD saved = GetLastError();
    /* Only this hidden isolated test console adopts ordinary Ctrl-C processing. */
    if (fairyc_test_observed_token == NULL || !SetConsoleCtrlHandler(NULL, FALSE) ||
        !GenerateConsoleCtrlEvent(%du, 0u) ||
        WaitForSingleObject(fairyc_test_observed_token, 5000u) != WAIT_OBJECT_0) { ExitProcess(91u); }
    SetLastError(saved); return result;
}
`, kind)
	modified := changed + observer
	recovered := strings.Replace(strings.Replace(strings.TrimSuffix(modified, observer), renamedLocal, local, 1), renamedImage, image, 1)
	if recovered != support {
		t.Fatal("observer changed native support beyond wrappers")
	}
	t.Logf("actual support original/recovered sha256=%x; scheduling observer=%x", sha256.Sum256([]byte(support)), sha256.Sum256([]byte(observer)))
	return modified
}

func TestCandidateStateCommandConsoleCancellation(t *testing.T) {
	main, err := os.ReadFile(filepath.Join("..", "..", "tools", "dev", "fairyc-state", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []uint32{0, 1} {
		t.Run(fmt.Sprintf("control-%d", kind), func(t *testing.T) {
			windowsConsoleBuild(t, string(main), func(t *testing.T, support string) string { return windowsConsoleEntryObserver(t, support, kind) }, func(t *testing.T, image string) {
				store := t.TempDir()
				missing := filepath.Join(t.TempDir(), "ledger.fy")
				code, out, stderr := windowsIsolatedConsoleProcess(t, image, "execute", store, missing, "--engine", "native-graph", "--profile", "low")
				if code != 4 || len(out) != 0 || !bytes.Contains(stderr, []byte("canceled")) {
					t.Fatalf("actual main console cancellation code=%d stdout=%q stderr=%q", code, out, stderr)
				}
				entries, err := os.ReadDir(store)
				if err != nil || len(entries) != 0 {
					t.Fatalf("cancellation touched store: %v %v", entries, err)
				}
				t.Log("real OS control delivered before file I/O; actual main token observed; no source/packs/store required")
			})
		})
	}
}
