//go:build windows

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/0disoft/fairyc/internal/workspaceloader"
	"golang.org/x/sys/windows"
)

// Only the actual authored windows_cancel import closure is compiled. Go opens,
// polls and signals the real named objects independently of the Fairy probe.
const windowsCancelProbe = `module main;
import windows_cancel as cancel;
import std::bytes as bytes;
import std::process as process;
import std::io as io;
@link_name("SetLastError") unsafe extern C fn void set_error(u32 code);
@link_name("GetLastError") unsafe extern C fn u32 get_error();
fn bytes::Buffer argument(usize at){var u8[512] data=[0;512];let n=switch(process::arg(at,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] raw=ref data;return switch(bytes::copy(raw[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(console) fn void failure(cancel::Problem e){discard io::write_bytes(b"E");discard io::write_u32(e.kind);discard io::write_bytes(b":");discard io::write_u32(e.os_code);}
@effects(console) fn u8 instruction(){var u8[1] data=[0;1];assert(switch(io::read_stdin(ref mut data)){case .ok(n):{n==1}case .err(_):{false}});return data[0];}
@effects(filesystem) fn bool state(ref cancel::Token token){let value=switch(cancel::requested(token)){case .ok(p):{p}case .err(_):{assert(false);false}};assert(switch(cancel::requested(token)){case .ok(p):{p==value}case .err(_):{false}});return value;}
@effects(filesystem,console) fn void report(ref cancel::Token token){assert(io::write_bytes(if(state(token)){b"1"}else{b"0"}).is_ok);}
@effects(filesystem) fn bool reopen(ref u8[] name){let token=switch(cancel::open(name)){case .ok(p):{p}case .err(_):{return false;}};discard state(ref token);return true;}
@effects(filesystem) fn bool collision(ref u8[] name){return switch(cancel::create(name)){case .ok(_):{false}case .err(e):{e.kind==3 && e.os_code==183}};}
@effects(filesystem) fn bool invalid(ref u8[] name){
 unsafe{set_error(12345);};let created=cancel::create(name);let create_error=unsafe{get_error()};
 assert(switch(created){case .ok(_):{false}case .err(e):{e.kind==1 && e.os_code==0}});assert(create_error==12345);
 unsafe{set_error(12345);};let opened=cancel::open(name);let open_error=unsafe{get_error()};
 assert(switch(opened){case .ok(_):{false}case .err(e):{e.kind==1 && e.os_code==0}});assert(open_error==12345);return true;
}
@effects(filesystem,console) fn i32 main(){
 let action=argument(1);let mode=bytes::view(ref action);
 if(bytes::equal(mode,b"none")){
  let token=cancel::none();unsafe{set_error(12345);};let result=cancel::requested(ref token);let polled=unsafe{get_error()};
  assert(switch(result){case .ok(p):{!p}case .err(_):{false}});assert(polled==12345);
  unsafe{set_error(12345);};let rejected=cancel::signal(ref token);let signaled=unsafe{get_error()};
  assert(switch(rejected){case .ok(_):{false}case .err(e):{e.kind==1 && e.os_code==0}});assert(signaled==12345);
  discard io::write_bytes(b"K");return 0;
 }
 if(bytes::equal(mode,b"invalid")){
  var bytes::Buffer name=bytes::empty();var u8[512] data=[0;512];
  while(true){let n=switch(io::read_stdin(ref mut data)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] raw=ref data;assert(bytes::append(ref mut name,raw[..n]).is_ok);assert(bytes::len(ref name)<=4096);}
  assert(invalid(bytes::view(ref name)));discard io::write_bytes(b"K");return 0;
 }
 let name=argument(2);let raw=bytes::view(ref name);
 // A stale ERROR_ALREADY_EXISTS must not reject a fresh creation.
 unsafe{set_error(183);};
 let token=switch(if(bytes::equal(mode,b"create") || bytes::equal(mode,b"hold-create")){cancel::create(raw)}else{cancel::open(raw)}){case .ok(p):{p}case .err(e):{failure(e);return 0;}};
 if(bytes::equal(mode,b"signal")){assert(switch(cancel::signal(ref token)){case .ok(p):{p}case .err(_):{false}});}
 report(ref token);
 if(bytes::equal(mode,b"hold-create") || bytes::equal(mode,b"hold-open")){
  while(true){let op=instruction();if(op==113){return 0;}
   if(op==99){var usize i=0;while(i<64){assert(collision(raw));assert(reopen(raw));i+=1;}}
   else if(op==115){assert(switch(cancel::signal(ref token)){case .ok(p):{p}case .err(_):{false}});}
   else{assert(op==112);}
   report(ref token);
  }
 }
 return 0;
}`

type windowsCancelChild struct {
	cmd    *exec.Cmd
	input  io.WriteCloser
	output io.ReadCloser
	stderr bytes.Buffer
	cancel context.CancelFunc
	waited bool
}

func windowsCancelStart(t *testing.T, image, mode, name string, want byte) *windowsCancelChild {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	p := &windowsCancelChild{cancel: cancel}
	p.cmd = exec.CommandContext(ctx, image, mode, name)
	p.cmd.WaitDelay = time.Second
	var err error
	if p.input, err = p.cmd.StdinPipe(); err != nil {
		cancel()
		t.Fatal(err)
	}
	if p.output, err = p.cmd.StdoutPipe(); err != nil {
		p.input.Close()
		cancel()
		t.Fatal(err)
	}
	p.cmd.Stderr = &p.stderr
	if err = p.cmd.Start(); err != nil {
		p.input.Close()
		p.output.Close()
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		p.input.Close()
		p.output.Close()
		if !p.waited {
			p.cmd.Wait()
		}
	})
	p.read(t, want)
	return p
}

func (p *windowsCancelChild) read(t *testing.T, want byte) {
	t.Helper()
	var data [1]byte
	if _, err := io.ReadFull(p.output, data[:]); err != nil || data[0] != want {
		t.Fatalf("token process reply=%q want=%q error=%v", data, want, err)
	}
}

func (p *windowsCancelChild) command(t *testing.T, command, want byte) {
	t.Helper()
	if _, err := p.input.Write([]byte{command}); err != nil {
		t.Fatal(err)
	}
	p.read(t, want)
}

func (p *windowsCancelChild) finish(t *testing.T) {
	t.Helper()
	if _, err := p.input.Write([]byte{'q'}); err != nil {
		t.Fatal(err)
	}
	p.input.Close()
	out, readErr := io.ReadAll(p.output)
	err := p.cmd.Wait()
	p.waited = true
	p.cancel()
	if err != nil || readErr != nil || len(out) != 0 || p.stderr.Len() != 0 {
		t.Fatalf("token process exit=%v read=%v output=%q stderr=%s", err, readErr, out, &p.stderr)
	}
}

func windowsCancelRun(t *testing.T, image, mode, name, input, want string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, image, mode, name)
	command.Stdin = strings.NewReader(input)
	command.WaitDelay = time.Second
	out, err := command.CombinedOutput()
	if err != nil || string(out) != want {
		t.Fatalf("token %s %q: exit=%v output=%q want=%q", mode, name, err, out, want)
	}
}

func windowsCancelName(t *testing.T, suffix string) string {
	t.Helper()
	return fmt.Sprintf(`Local\fairyc-cancel-%d-%d-%s`, os.Getpid(), time.Now().UnixNano(), suffix)
}

func windowsCancelWide(t *testing.T, name string) *uint16 {
	t.Helper()
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func windowsCancelPoll(t *testing.T, handle windows.Handle, want uint32) {
	t.Helper()
	state, err := windows.WaitForSingleObject(handle, 0)
	if err != nil || state != want {
		t.Fatalf("independent event poll=%d want=%d error=%v", state, want, err)
	}
}

func windowsCancelMissing(t *testing.T, name string) {
	t.Helper()
	h, err := windows.OpenEvent(windows.SYNCHRONIZE, false, windowsCancelWide(t, name))
	if h != 0 {
		windows.CloseHandle(h)
	}
	if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		t.Fatalf("event survived its last owner: handle=%d error=%v", h, err)
	}
}

func windowsCancelHandleCount(t *testing.T, pid int) uint32 {
	t.Helper()
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	var count uint32
	getCount := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetProcessHandleCount")
	ok, _, err := getCount.Call(uintptr(h), uintptr(unsafe.Pointer(&count)))
	if ok == 0 {
		t.Fatal(err)
	}
	return count
}

func TestCandidateWindowsCancelNative(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "windows_cancel.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("windows_cancel.fy sha256=%x; probe sha256=%x", sha256.Sum256(source), sha256.Sum256([]byte(windowsCancelProbe)))
	candidateWindowsCancellationProbe(t, "windows_cancel", windowsCancelProbe, func(t *testing.T, image string) {
		windowsCancelRun(t, image, "none", "", "", "K")
		t.Run("invalid-before-os", func(t *testing.T) {
			prefix := `Local\fairyc-cancel-`
			for _, name := range []string{"", prefix, "x", `Global\fairyc-cancel-a`, `local\fairyc-cancel-a`, `Local\other-a`, prefix + strings.Repeat("a", 129), prefix + "a/b", prefix + `a\b`, prefix + "a:b", prefix + "a b", prefix + "a\tb", prefix + "a\nb", prefix + "한글", prefix + "🧚", prefix + "a\x00b", prefix + string([]byte{255})} {
				windowsCancelRun(t, image, "invalid", "", name, "K")
			}
		})
		t.Run("missing-does-not-create", func(t *testing.T) {
			name := windowsCancelName(t, "missing")
			windowsCancelRun(t, image, "open", name, "", "E4:2")
			windowsCancelMissing(t, name)
		})
		t.Run("fairy-owner-cross-process-and-cleanup", func(t *testing.T) {
			name := windowsCancelName(t, "owner")
			owner := windowsCancelStart(t, image, "hold-create", name, '0')
			observer := windowsCancelStart(t, image, "hold-open", name, '0')
			oracle, err := windows.OpenEvent(windows.SYNCHRONIZE|windows.EVENT_MODIFY_STATE, false, windowsCancelWide(t, name))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { windows.CloseHandle(oracle) })
			windowsCancelPoll(t, oracle, uint32(windows.WAIT_TIMEOUT))
			// Warm the command path before taking an external process-handle count.
			owner.command(t, 'c', '0')
			baseline := windowsCancelHandleCount(t, owner.cmd.Process.Pid)
			for range 4 {
				owner.command(t, 'c', '0')
			}
			if count := windowsCancelHandleCount(t, owner.cmd.Process.Pid); count != baseline {
				t.Fatalf("collision/open/early-return handle leak: before=%d after=%d", baseline, count)
			}
			windowsCancelRun(t, image, "create", name, "", "E3:183")
			windowsCancelPoll(t, oracle, uint32(windows.WAIT_TIMEOUT))
			if err := windows.SetEvent(oracle); err != nil {
				t.Fatal(err)
			}
			owner.command(t, 'p', '1')
			observer.command(t, 'p', '1')
			windowsCancelRun(t, image, "open", name, "", "1")
			// Collisions preserve the signaled state too; repeated polls never reset.
			owner.command(t, 'c', '1')
			windowsCancelRun(t, image, "create", name, "", "E3:183")
			windowsCancelPoll(t, oracle, windows.WAIT_OBJECT_0)
			windowsCancelPoll(t, oracle, windows.WAIT_OBJECT_0)
			owner.finish(t)
			observer.command(t, 'p', '1')
			observer.finish(t)
			windowsCancelPoll(t, oracle, windows.WAIT_OBJECT_0)
			windows.CloseHandle(oracle)
			oracle = 0
			windowsCancelMissing(t, name)
			// Every Fairy owner dropped; a fresh owner of the same name starts clear.
			windowsCancelRun(t, image, "create", name, "", "0")
			windowsCancelMissing(t, name)
		})
		t.Run("go-owner-fairy-signal-is-latched", func(t *testing.T) {
			name := windowsCancelName(t, "signal")
			h, err := windows.CreateEvent(nil, 1, 0, windowsCancelWide(t, name))
			if err != nil {
				t.Fatal(err)
			}
			defer windows.CloseHandle(h)
			windowsCancelRun(t, image, "create", name, "", "E3:183")
			windowsCancelPoll(t, h, uint32(windows.WAIT_TIMEOUT))
			windowsCancelRun(t, image, "signal", name, "", "1")
			windowsCancelPoll(t, h, windows.WAIT_OBJECT_0)
			windowsCancelPoll(t, h, windows.WAIT_OBJECT_0)
			windowsCancelRun(t, image, "open", name, "", "1")
			windowsCancelRun(t, image, "signal", name, "", "1")
			windowsCancelRun(t, image, "create", name, "", "E3:183")
			windowsCancelPoll(t, h, windows.WAIT_OBJECT_0)
		})
		t.Run("strict-name-boundaries", func(t *testing.T) {
			for _, suffix := range []string{"a", strings.Repeat("a", 128), "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._-"} {
				name := `Local\fairyc-cancel-` + suffix
				// Exact suffix lengths run sequentially; refuse an occupied name.
				windowsCancelMissing(t, name)
				windowsCancelRun(t, image, "create", name, "", "0")
				windowsCancelMissing(t, name)
			}
		})
		t.Run("native-cause-retained", func(t *testing.T) {
			name := windowsCancelName(t, "mutex")
			h, err := windows.CreateMutexEx(nil, windowsCancelWide(t, name), 0, windows.SYNCHRONIZE)
			if err != nil {
				t.Fatal(err)
			}
			defer windows.CloseHandle(h)
			windowsCancelRun(t, image, "create", name, "", "E3:6")
			windowsCancelRun(t, image, "open", name, "", "E4:6")
		})
	})
}

func TestCandidateWindowsCancelSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, body string }{
		{"constructor", `fn cancel::Token forge(){return cancel::Token{.handle=0};}`},
		{"read", `fn usize forge(ref cancel::Token token){return token.handle;}`},
		{"write", `fn void forge(ref mut cancel::Token token){token.handle=0;}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "windows_cancel")
			writeTestSource(t, root, "src/main.fy", "module main;import windows_cancel as cancel;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostic bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed cancellation token", &diagnostic)
			if code == ExitSuccess || !strings.Contains(diagnostic.String(), "E0103") {
				t.Fatalf("sealed Token: code=%d diagnostics=%s", code, &diagnostic)
			}
		})
	}
}
