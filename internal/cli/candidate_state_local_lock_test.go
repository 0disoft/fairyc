//go:build windows

package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"golang.org/x/sys/windows"
)

const stateLocalLockProbe = `module main;
import windows_rooted_file as rooted;
import std::bytes as bytes;
import std::process as process;
import std::io as io;
@effects(filesystem) @link_name("fairyc_private_windows_handle_count_v1") unsafe extern C fn i32 handle_count(ptr mut u32 count);
fn bytes::Buffer argument(usize at){var u8[4096] data=[0;4096];let n=switch(process::arg(at,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] raw=ref data;return switch(bytes::copy(raw[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(console) fn void failure(rooted::Problem e){discard io::write_bytes(b"E");discard io::write_u32(e.stage);discard io::write_bytes(b":");discard io::write_u32(e.os_code);}
@effects(console) fn void wait(){var u8[1] data=[0;1];assert(io::read_stdin(ref mut data).is_ok);}
@effects(filesystem) fn u32 handles(){var u32 n=0;assert(unsafe{handle_count(ref mut n as ptr mut u32)}!=0);return n;}
@effects(filesystem) fn bool busy(ref rooted::Directory root,ref rooted::StateLock authority){return switch(rooted::state_lock_acquire(root,authority)){case .ok(_):{false}case .err(e):{e.stage==15 && e.os_code==33}};}
@effects(filesystem) fn bool early(ref rooted::Directory root,ref rooted::StateLock authority){let lease=switch(rooted::state_lock_acquire(root,authority)){case .ok(p):{p}case .err(_):{return false;}};return true;}
@effects(filesystem,console) fn i32 main(){
 let path=argument(1);let action=argument(2);let mode=bytes::view(ref action);
 let root=switch(rooted::open(bytes::view(ref path))){case .ok(p):{p}case .err(e):{failure(e);return 0;}};
 {
 let authority=switch(rooted::state_lock_open(ref root,bytes::equal(mode,b"create"))){case .ok(p):{p}case .err(e):{failure(e);return 0;}};
 if(bytes::equal(mode,b"create")){return 0;}
 if(bytes::equal(mode,b"stale")){discard io::write_bytes(b"R");wait();}
 if(bytes::equal(mode,b"other-root")){let other_path=argument(3);let other=switch(rooted::open(bytes::view(ref other_path))){case .ok(p):{p}case .err(_):{return 1;}};switch(rooted::state_lock_acquire(ref other,ref authority)){case .ok(_):{return 2;}case .err(e):{failure(e);return 0;}};}
 let baseline=handles();
 {
 let lease=switch(rooted::state_lock_acquire(ref root,ref authority)){case .ok(p):{p}case .err(e):{failure(e);return 0;}};
 if(bytes::equal(mode,b"hold")){discard io::write_bytes(b"L");wait();}
 if(bytes::equal(mode,b"cycles")){
  let held=handles();var usize i=0;while(i<64){assert(busy(ref root,ref authority));assert(switch(rooted::state_lock_open(ref root,true)){case .ok(_):{false}case .err(e):{rooted::collision(e)}});i+=1;}assert(handles()==held);
  let independent=switch(rooted::state_lock_open(ref root,false)){case .ok(p):{p}case .err(_):{return 3;}};assert(busy(ref root,ref independent));
 }
 }
 assert(handles()==baseline);
 // Both ordinary scope exit and an early return release the lease while the
 // retained authority is still alive. Repeat to expose failed-acquire leaks.
 var usize i=0;while(i<64){assert(early(ref root,ref authority));i+=1;}assert(handles()==baseline);
 }
 discard io::write_bytes(b"K");return 0;
}`

type stateLockChild struct {
	cmd    *exec.Cmd
	input  io.WriteCloser
	output io.ReadCloser
	stderr bytes.Buffer
	cancel context.CancelFunc
	waited bool
}

func stateLockStart(t *testing.T, image, root, mode string, ready byte) *stateLockChild {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	p := &stateLockChild{cancel: cancel}
	p.cmd = exec.CommandContext(ctx, image, filepath.ToSlash(root), mode)
	var err error
	if p.input, err = p.cmd.StdinPipe(); err != nil {
		cancel()
		t.Fatal(err)
	}
	if p.output, err = p.cmd.StdoutPipe(); err != nil {
		cancel()
		t.Fatal(err)
	}
	p.cmd.Stderr = &p.stderr
	if err = p.cmd.Start(); err != nil {
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
	var signal [1]byte
	if _, err = io.ReadFull(p.output, signal[:]); err != nil || signal[0] != ready {
		t.Fatalf("%s ready: %q %v", mode, signal, err)
	}
	return p
}

func (p *stateLockChild) finish(t *testing.T, want string) {
	t.Helper()
	if _, err := p.input.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	p.input.Close()
	out, err := io.ReadAll(p.output)
	if err != nil {
		t.Fatal(err)
	}
	err = p.cmd.Wait()
	p.waited = true
	p.cancel()
	if err != nil || string(out) != want || p.stderr.Len() != 0 {
		t.Fatalf("child exit: %v output=%q want=%q stderr=%s", err, out, want, &p.stderr)
	}
}

func stateLockOracle(t *testing.T, root string) windows.Handle {
	t.Helper()
	path, err := windows.UTF16PtrFromString(filepath.Join(root, "state-v1.lock"))
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(path, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { windows.CloseHandle(h) })
	return h
}

func stateLockTry(h windows.Handle) error {
	return windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{})
}

func stateLocalLockNativeProbe(t *testing.T, main string, verify func(*testing.T, string), modules ...string) {
	t.Helper()
	if runtime.GOARCH != "amd64" {
		t.Skip("native rooted Windows probe requires amd64 MSVC")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, append([]string{"windows_rooted_file"}, modules...)...)
	writeTestSource(t, root, "src/main.fy", main)
	unit, support, tc := windowsCandidateProbe(t, root)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support})
			if err != nil {
				t.Fatal(err)
			}
			image := filepath.Join(root, "lock-"+profile.Name+".exe")
			if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			verify(t, image)
		})
	}
}

func TestCandidateStateLocalLockWindows(t *testing.T) {
	// Exercise the real closed selector; the existing rooted prefix selects the
	// same host source without admitting arbitrary private symbols.
	for _, symbol := range []string{"open", "same", "lock"} {
		name := "fairyc_private_windows_root_file_state_" + symbol + "_v1"
		if host, err := candidatehost.WindowsSource(name + "("); err != nil || !strings.Contains(host, name) {
			t.Fatalf("state lock host selection: %s %v", name, err)
		}
	}
	if _, err := candidatehost.WindowsSource("fairyc_private_windows_root_file_state_unknown_v1("); err == nil {
		t.Fatal("closed host selector accepted an unknown state lock symbol")
	}
	stateLocalLockNativeProbe(t, stateLocalLockProbe, func(t *testing.T, image string) {
		run := func(t *testing.T, root, mode, want string, args ...string) {
			t.Helper()
			code, out := candidateAtlasProcess(t, image, nil, append([]string{filepath.ToSlash(root), mode}, args...)...)
			if code != 0 || out != want {
				t.Fatalf("%s: exit=%d output=%q want=%q", mode, code, out, want)
			}
		}
		created := func(t *testing.T) string {
			t.Helper()
			root := t.TempDir()
			run(t, root, "create", "")
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 || entries[0].Name() != "state-v1.lock" {
				t.Fatalf("fixed flat authority inventory: %v %v", entries, err)
			}
			return root
		}
		t.Run("exclusive-create-and-reopen-preserve-bytes", func(t *testing.T) {
			root := created(t)
			path := filepath.Join(root, "state-v1.lock")
			payload := []byte("unchanged\x00authority")
			if err := os.WriteFile(path, payload, 0600); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			run(t, root, "create", "E4:183")
			run(t, root, "cycles", "K")
			after, err := os.Stat(path)
			if err != nil || !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
				t.Fatalf("authority identity/time changed: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, payload) {
				t.Fatalf("reopen initialized authority: %q %v", got, err)
			}
			// No native handles survive the process and no scratch names exist.
			if err := os.Rename(path, path+".moved"); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 {
				t.Fatalf("scratch cleanup: %v %v", entries, err)
			}
		})
		t.Run("go-owner-native-contender", func(t *testing.T) {
			root := created(t)
			h := stateLockOracle(t, root)
			if err := stateLockTry(h); err != nil {
				t.Fatal(err)
			}
			run(t, root, "try", "E15:33")
			if err := windows.UnlockFileEx(h, 0, 1, 0, &windows.Overlapped{}); err != nil {
				t.Fatal(err)
			}
			run(t, root, "try", "K")
		})
		t.Run("native-owner-cross-process-and-go-contenders", func(t *testing.T) {
			root := created(t)
			p := stateLockStart(t, image, root, "hold", 'L')
			h := stateLockOracle(t, root)
			if err := stateLockTry(h); !errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
				t.Fatalf("Go oracle must see native one-byte lock: %v", err)
			}
			// Only byte zero is covered; the adjacent byte remains available.
			adjacent := windows.Overlapped{Offset: 1}
			if err := windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &adjacent); err != nil {
				t.Fatalf("lease covered more than byte zero: %v", err)
			}
			if err := windows.UnlockFileEx(h, 0, 1, 0, &adjacent); err != nil {
				t.Fatal(err)
			}
			run(t, root, "try", "E15:33")
			// The lease prevents namespace replacement after validation.
			path := filepath.Join(root, "state-v1.lock")
			if err := os.Rename(path, path+".moved"); err == nil {
				t.Fatal("active lease allowed lock path replacement")
			}
			p.finish(t, "K")
			if err := stateLockTry(h); err != nil {
				t.Fatalf("normal lease cleanup did not release lock: %v", err)
			}
			if err := windows.UnlockFileEx(h, 0, 1, 0, &windows.Overlapped{}); err != nil {
				t.Fatal(err)
			}
		})
		t.Run("process-death-releases-lock", func(t *testing.T) {
			root := created(t)
			p := stateLockStart(t, image, root, "hold", 'L')
			h := stateLockOracle(t, root)
			if err := stateLockTry(h); !errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
				t.Fatal(err)
			}
			if err := p.cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			p.cmd.Wait()
			p.waited = true
			deadline := time.Now().Add(3 * time.Second)
			for {
				err := stateLockTry(h)
				if err == nil {
					break
				}
				if !errors.Is(err, windows.ERROR_LOCK_VIOLATION) || time.Now().After(deadline) {
					t.Fatalf("process death did not release lock: %v", err)
				}
				time.Sleep(10 * time.Millisecond)
			}
			if err := windows.UnlockFileEx(h, 0, 1, 0, &windows.Overlapped{}); err != nil {
				t.Fatal(err)
			}
			run(t, root, "try", "K")
		})
		t.Run("retained-identity-refuses-replacement-and-missing", func(t *testing.T) {
			root := created(t)
			p := stateLockStart(t, image, root, "stale", 'R')
			path := filepath.Join(root, "state-v1.lock")
			if err := os.Rename(path, path+".old"); err != nil {
				t.Fatal(err)
			}
			run(t, root, "create", "")
			p.finish(t, "E16:1006")
			run(t, root, "try", "K")
			p = stateLockStart(t, image, root, "stale", 'R')
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			p.finish(t, "E4:2")
			run(t, root, "try", "E4:2")
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("reopen recreated missing authority: %v", err)
			}
		})
		t.Run("different-root-refused", func(t *testing.T) {
			root, other := created(t), created(t)
			run(t, root, "other-root", "E16:1006", filepath.ToSlash(other))
		})
		t.Run("missing-and-nonregular-preconditions", func(t *testing.T) {
			root := t.TempDir()
			run(t, root, "try", "E4:2")
			path := filepath.Join(root, "state-v1.lock")
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			run(t, root, "try", "E4:5")
			run(t, root, "create", "E4:5")
		})
		for _, directory := range []bool{false, true} {
			t.Run(fmt.Sprintf("symlink-precondition-directory-%v", directory), func(t *testing.T) {
				root := t.TempDir()
				target := t.TempDir()
				if !directory {
					target = filepath.Join(target, "target")
					if err := os.WriteFile(target, []byte("preserve"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				path := filepath.Join(root, "state-v1.lock")
				if err := os.Symlink(target, path); err != nil {
					t.Skipf("symlink privilege unavailable: %v", err)
				}
				run(t, root, "try", "E4:5")
				collision := "E4:183"
				if directory {
					collision = "E4:5"
				}
				run(t, root, "create", collision)
				if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink == 0 {
					t.Fatalf("symlink collision was modified: %v", err)
				}
			})
		}
	})
}

func TestCandidateStateLocalLockRootedCompatibility(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "windows_rooted_file.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.NewReplacer("module tests::windows_rooted_file;", "module main;", "@test ", "").Replace(string(fixture)) + `
@effects(filesystem,console) fn i32 main(){
 opened_directory_survives_root_path_replacement_and_binary_publication();
 invalid_leaves_collisions_and_failed_publication_preserve_existing_bytes();
 unpublished_files_and_failed_reads_release_all_handles();
 retained_inventory_handles_multiple_batches_unicode_and_exact_bounds();
 stale_inventory_preserves_replacements_and_current_metadata_deletes();
 native_batch_decoder_rejects_malformed_suffixes_and_exhaustion();return 0;
}`
	stateLocalLockNativeProbe(t, main, func(t *testing.T, image string) {
		data := t.TempDir()
		manifest := filepath.Join(data, "root.txt")
		if err := os.WriteFile(manifest, []byte(filepath.ToSlash(data)), 0600); err != nil {
			t.Fatal(err)
		}
		if code, out := candidateAtlasProcess(t, image, nil, manifest); code != 0 || out != "" {
			t.Fatalf("existing rooted adapter fixtures: exit=%d output=%q", code, out)
		}
	}, "file_source")
}

func TestCandidateStateLocalLockSealed(t *testing.T) {
	for _, owner := range []string{"StateLock", "StateLease"} {
		for _, trial := range []struct{ name, body string }{
			{"constructor", `fn rooted::OWNER forge(rooted::File file){return rooted::OWNER{.file};}`},
			{"field", `fn void forge(ref rooted::OWNER owner){let file=ref owner.file;}`},
		} {
			t.Run(owner+"/"+trial.name, func(t *testing.T) {
				root := t.TempDir()
				copyLexerExample(t, root)
				copyCandidateParityModules(t, root, "windows_rooted_file")
				writeTestSource(t, root, "src/main.fy", "module main;import windows_rooted_file as rooted;"+strings.ReplaceAll(trial.body, "OWNER", owner)+"fn i32 main(){return 0;}")
				workspace, err := workspaceloader.LoadVectorCandidate(root)
				if err != nil {
					t.Fatal(err)
				}
				var diagnostic bytes.Buffer
				_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed state lock", &diagnostic)
				if code == ExitSuccess || !strings.Contains(diagnostic.String(), "E0103") {
					t.Fatalf("sealed owner: code=%d diagnostics=%s", code, &diagnostic)
				}
			})
		}
	}
}
