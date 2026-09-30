package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPracticalReplaceFileNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			root := t.TempDir()
			old := filepath.Join(root, "한글 old")
			tmp := filepath.Join(root, "한글 new")
			if err := os.WriteFile(old, []byte("old"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(tmp, []byte("new-content"), 0600); err != nil {
				t.Fatal(err)
			}
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.replace-native\"\nname = \"replace-native\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			program := `module main; import std::fs as fs;
@effects(filesystem)
fn i32 main() {
 assert(switch (fs::rename_file(TMP, OLD)) { case .ok(_) : { false } case .err(_) : { true } });
 assert(switch (fs::replace_file(TMP, OLD)) { case .ok(_) : { true } case .err(_) : { false } });
 assert(switch (fs::replace_file(TMP, OLD)) { case .ok(_) : { false } case .err(error) : { fs::error_os_code(error) == 2 } });
 assert(switch (fs::replace_file(ROOT, OLD)) { case .ok(_) : { false } case .err(_) : { true } });
 assert(switch (fs::replace_file(c"", OLD)) { case .ok(_) : { false } case .err(_) : { true } });
 return 0;
}`
			q := func(p string) string { return "c" + strconv.Quote(filepath.ToSlash(p)) }
			writeTestSource(t, root, "src/main.fy", strings.NewReplacer("TMP", q(tmp), "OLD", q(old), "ROOT", q(root)).Replace(program))
			var diagnostics strings.Builder
			compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("compile: %d %s", code, diagnostics.String())
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(compiled.artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("replace: %+v %v", outcome, err)
			}
			practicalFileEquals(t, old, "new-content")
			if _, err := os.Stat(tmp); !os.IsNotExist(err) {
				t.Fatalf("replacement still exists: %v", err)
			}
		})
	}
}

// Test-only Win32 interception uses real files and the production runtime. It never fills
// a host volume or changes machine settings. No physical power-loss proof is claimed.
func TestPracticalSaveFaultsAndTermination(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	support, err := cstdlib.WindowsFileSource()
	if err != nil {
		t.Fatal(err)
	}
	support = strings.NewReplacer("WriteFile(", "practical_write(", "FlushFileBuffers(", "practical_sync(", "MoveFileExW(", "practical_move(").Replace(support)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, practicalSaveHooks+support+practicalSaveProbe)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			image := filepath.Join(root, "probe.exe")
			if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"ok", "first-write-full", "partial-full", "sync-failure", "replace-full", "sharing", "before", "after", "during-write"} {
				t.Run(mode, func(t *testing.T) {
					dir := t.TempDir()
					old, tmp := filepath.Join(dir, "old"), filepath.Join(dir, "new")
					if err := os.WriteFile(old, []byte("old"), 0600); err != nil {
						t.Fatal(err)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					cmd := exec.CommandContext(ctx, image, mode, old, tmp)
					cmd.WaitDelay = time.Second
					if mode == "before" || mode == "after" || mode == "during-write" {
						pipe, err := cmd.StdoutPipe()
						if err != nil {
							t.Fatal(err)
						}
						if err := cmd.Start(); err != nil {
							t.Fatal(err)
						}
						defer func() { _ = cmd.Process.Kill() }()
						var marker [1]byte
						if n, err := pipe.Read(marker[:]); err != nil || n != 1 || marker[0] != 'R' {
							_ = cmd.Process.Kill()
							_ = cmd.Wait()
							t.Fatalf("checkpoint: %d %v %q", n, err, marker)
						}
						if err := cmd.Process.Kill(); err != nil {
							t.Fatal(err)
						}
						if err := cmd.Wait(); err == nil {
							t.Fatal("process was not forcibly terminated")
						}
					} else if output, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("probe: %v %s", err, output)
					}
					want := "old"
					if mode == "ok" || mode == "after" {
						want = "new-content"
					}
					practicalFileEquals(t, old, want)
					switch mode {
					case "ok", "after":
						if _, err := os.Stat(tmp); !os.IsNotExist(err) {
							t.Fatalf("replacement not consumed: %v", err)
						}
					case "first-write-full":
						practicalFileEquals(t, tmp, "")
					case "partial-full", "during-write":
						practicalFileEquals(t, tmp, "ne")
					default:
						practicalFileEquals(t, tmp, "new-content")
					}
				})
			}
		})
	}
}

func practicalFileEquals(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("%s: got %q, want %q: %v", path, got, want, err)
	}
}

const practicalSaveHooks = `
#include <windows.h>
#include <stdio.h>
#include <string.h>
static const char *practical_mode;
static int practical_calls;
static void practical_checkpoint(void) { fputc('R',stdout); fflush(stdout); for (;;) Sleep(1000); }
static BOOL practical_write(HANDLE h, LPCVOID b, DWORD n, LPDWORD done, LPOVERLAPPED o) {
 if (!strcmp(practical_mode,"first-write-full")) { *done=0; SetLastError(ERROR_DISK_FULL); return FALSE; }
 if (!strcmp(practical_mode,"partial-full") || !strcmp(practical_mode,"during-write")) {
  if (practical_calls++ == 0) { return WriteFile(h,b,2,done,o); }
  if (!strcmp(practical_mode,"during-write")) practical_checkpoint();
  *done=0; SetLastError(ERROR_DISK_FULL); return FALSE;
 }
 return WriteFile(h,b,n,done,o);
}
static BOOL practical_sync(HANDLE h) {
 if (!strcmp(practical_mode,"sync-failure")) { SetLastError(ERROR_WRITE_FAULT); return FALSE; }
 return FlushFileBuffers(h);
}
static BOOL practical_move(LPCWSTR a, LPCWSTR b, DWORD flags) {
 if (!strcmp(practical_mode,"replace-full")) { SetLastError(ERROR_DISK_FULL); return FALSE; }
 return MoveFileExW(a,b,flags);
}
`

const practicalSaveProbe = `
int main(int argc, char **argv) {
 if (argc!=4) return 10;
 practical_mode=argv[1];
 struct fy_std_windows_file_v0 file = {0};
 struct fy_std_windows_open_options_v0 options = {0,1,1,0,1,0};
 struct fy_std_windows_file_result_v0 r=fy_std_windows_file_open_v0((const uint8_t *)argv[3],&options,&file);
 if (r.status) return 11;
 r=fy_std_windows_file_write_all_v0(&file,(const uint8_t *)"new-content",11);
 if (!strcmp(practical_mode,"first-write-full") || !strcmp(practical_mode,"partial-full")) {
  if (r.status!=1 || r.os_code!=ERROR_DISK_FULL || r.count!=(size_t)(!strcmp(practical_mode,"partial-full") ? 2 : 0)) return 12;
  return fy_std_windows_file_close_v0(&file).status ? 13 : 0;
 }
 if (r.status || r.count!=11) return 14;
 r=fy_std_windows_file_sync_v0(&file);
 if (!strcmp(practical_mode,"sync-failure")) {
  if (r.status!=1 || r.os_code!=ERROR_WRITE_FAULT) return 15;
  return fy_std_windows_file_close_v0(&file).status ? 16 : 0;
 }
 if (r.status || fy_std_windows_file_close_v0(&file).status) return 17;
 if (!strcmp(practical_mode,"before")) practical_checkpoint();
 HANDLE lock=INVALID_HANDLE_VALUE;
 DWORD sharing_error=0;
 if (!strcmp(practical_mode,"sharing")) {
  lock=CreateFileA(argv[2],GENERIC_READ,FILE_SHARE_READ,NULL,OPEN_EXISTING,0,NULL);
  if (lock==INVALID_HANDLE_VALUE) return 21;
  WCHAR wide_tmp[FY_STD_WINDOWS_PATH_CAPACITY_V0], wide_old[FY_STD_WINDOWS_PATH_CAPACITY_V0];
  if (fy_std_windows_path_v0((const uint8_t *)argv[3],wide_tmp) || fy_std_windows_path_v0((const uint8_t *)argv[2],wide_old)) return 22;
  if (MoveFileExW(wide_tmp,wide_old,MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH)) return 23;
  sharing_error=GetLastError();
  if (sharing_error!=ERROR_ACCESS_DENIED && sharing_error!=ERROR_SHARING_VIOLATION) return 24;
 }
 r=fy_std_windows_file_replace_v0((const uint8_t *)argv[3],(const uint8_t *)argv[2]);
 if (lock!=INVALID_HANDLE_VALUE) CloseHandle(lock);
 if (!strcmp(practical_mode,"replace-full")) return r.status==1 && r.os_code==ERROR_DISK_FULL ? 0 : 18;
 if (!strcmp(practical_mode,"sharing")) {
  return r.status==1 && r.os_code==sharing_error ? 0 : 19;
 }
 if (r.status) return 20;
 if (!strcmp(practical_mode,"after")) practical_checkpoint();
 return 0;
}
`
