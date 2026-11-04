package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

// Test-only native fault injection leaves the actual Fairy command in charge
// of locking, validation, ownership, publication and error reporting.
func candidateEditCommandProbe(t *testing.T, main string, verify func(*testing.T, string)) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows edit command; native Linux is batched")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_edit_cli")
	writeTestSource(t, root, "src/main.fy", main)
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 2<<20)
	for _, function := range []string{"open", "write", "sync", "delete", "rename", "close"} {
		result := "int32_t "
		if function == "open" {
			result = "size_t "
		}
		name := "fairyc_private_windows_root_file_" + function + "_v1"
		old := result + name + "("
		if strings.Count(support, old) != 1 {
			t.Fatal("ambiguous fault hook:", name)
		}
		support = strings.Replace(support, old, result+"fy_edit_real_"+function+"(", 1)
	}
	const directory = "size_t fairyc_private_windows_root_directory_create_v1("
	if strings.Count(support, directory) != 1 {
		t.Fatal("ambiguous directory fault hook")
	}
	support = strings.Replace(support, directory, "size_t fy_edit_real_directory_create(", 1) + editNativeFaults
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
			path := filepath.Join(root, "edit-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			verify(t, path)
		})
	}
}

const editNativeFaults = `
static size_t fy_edit_stage = 0, fy_edit_lock = 0;
static unsigned int fy_edit_renames = 0;
static int fy_edit_fault(const char *name) {
    char value[64]; DWORD count=GetEnvironmentVariableA("FAIRYC_TEST_EDIT_FAULT",value,sizeof(value));
    return count>0 && count<sizeof(value) && strcmp(value,name)==0;
}
size_t fairyc_private_windows_root_directory_create_v1(size_t root,const uint8_t *leaf,size_t bytes) {
    size_t result=fy_edit_real_directory_create(root,leaf,bytes);
    if(result!=0) { fy_edit_lock=result; } return result;
}
size_t fairyc_private_windows_root_file_open_v1(size_t root,const uint8_t *leaf,size_t bytes,uint32_t mode) {
    size_t result=fy_edit_real_open(root,leaf,bytes,mode);
    if(result!=0 && mode==2) { fy_edit_stage=result; } return result;
}
int32_t fairyc_private_windows_root_file_write_v1(size_t file,const uint8_t *data,size_t bytes,size_t *written) {
    if(file==fy_edit_stage && fy_edit_fault("write")) {
        int32_t result=fy_edit_real_write(file,data,bytes>0?1u:0u,written);
        (void)result; SetLastError(ERROR_WRITE_FAULT); return 0;
    }
    return fy_edit_real_write(file,data,bytes,written);
}
int32_t fairyc_private_windows_root_file_sync_v1(size_t file) {
    int32_t result=fy_edit_real_sync(file);
    if(file==fy_edit_stage && fy_edit_fault("sync")) { SetLastError(ERROR_WRITE_FAULT); return 0; }
    if(file==fy_edit_stage && fy_edit_fault("gate")) {
        DWORD count=0; char byte=0;
        if(!WriteFile(GetStdHandle(STD_OUTPUT_HANDLE),"staged\n",7,&count,NULL) || count!=7 ||
           !ReadFile(GetStdHandle(STD_INPUT_HANDLE),&byte,1,&count,NULL) || count!=1) { return 0; }
    }
    return result;
}
int32_t fairyc_private_windows_root_file_rename_v1(size_t file,size_t root,const uint8_t *leaf,size_t bytes) {
    fy_edit_renames++;
    if(fy_edit_renames==1 && (fy_edit_fault("rename") || fy_edit_fault("rename-lock-cleanup"))) { SetLastError(ERROR_ACCESS_DENIED); return 0; }
    return fy_edit_real_rename(file,root,leaf,bytes);
}
int32_t fairyc_private_windows_root_file_delete_v1(size_t file) {
    if(file==fy_edit_lock && (fy_edit_fault("lock-cleanup") || fy_edit_fault("rename-lock-cleanup"))) { SetLastError(ERROR_ACCESS_DENIED); return 0; }
    return fy_edit_real_delete(file);
}
int32_t fairyc_private_windows_root_file_close_v1(size_t file) {
    int injected=file==fy_edit_stage && fy_edit_fault("post-close");
    int32_t result=fy_edit_real_close(file);
    if(injected) { fy_edit_stage=0; SetLastError(ERROR_GEN_FAILURE); return 0; }
    return result;
}
`

func verifyCandidateEditApply(t *testing.T, image string) {
	t.Helper()
	reference, referenceArgs := editFixture(t)
	code, report, diagnostic := runForTest(append(referenceArgs, "--apply"), reference)
	var want sourceedit.Report
	if code != 0 || diagnostic != "" || json.Unmarshal([]byte(report), &want) != nil || !want.Applied {
		t.Fatal(code, report, diagnostic)
	}
	updated, err := os.ReadFile(filepath.Join(reference, "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	for _, scenario := range []string{"success", "collision", "exhausted", "locked", "readonly", "write", "sync", "rename", "post-close", "lock-cleanup", "rename-lock-cleanup", "stdout"} {
		t.Run("apply-"+scenario, func(t *testing.T) {
			root, args := editFixture(t)
			t.Setenv("FAIRYC_TEST_EDIT_FAULT", scenario)
			args = append(args, "--apply")
			if scenario == "locked" {
				if err := os.Mkdir(filepath.Join(root, ".fairyc-edit.lock"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "readonly" {
				path := filepath.Join(root, "src", "main.fy")
				if err := os.Chmod(path, 0400); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(path, 0600) })
			}
			if scenario == "collision" || scenario == "exhausted" {
				count := 1
				if scenario == "exhausted" {
					count = 16
				}
				for i := 0; i < count; i++ {
					writeWorkspaceFile(t, root, fmt.Sprintf("src/.fairyc-edit-stage-%d.tmp", i), "unrelated crash leftover")
				}
			}
			before := remoteTree(t, root)
			broken := ""
			if scenario == "stdout" {
				broken = "stdout"
			}
			code, out, stderr := fetchCLIProcess(t, image, root, args, broken)
			published := scenario == "success" || scenario == "collision" || scenario == "post-close" || scenario == "lock-cleanup" || scenario == "stdout"
			wantCode := 7
			if scenario == "success" || scenario == "collision" {
				wantCode = 0
			}
			if scenario == "stdout" {
				wantCode = 4
			}
			if code != wantCode {
				t.Fatalf("apply: %d %s %s; want %d", code, out, stderr, wantCode)
			}
			if wantCode == 0 {
				var got sourceedit.Report
				if stderr != "" || json.Unmarshal([]byte(out), &got) != nil || got != want {
					t.Fatalf("applied report: %s %s; want %+v", out, stderr, want)
				}
			} else if out != "" || stderr == "" {
				t.Fatalf("failure output: %q %q", out, stderr)
			}
			if published {
				before["src/main.fy"] = sourceedit.Digest(updated)
				if wantCode != 0 && !strings.Contains(stderr, "edit was applied") {
					t.Fatal("publication ambiguity was hidden:", stderr)
				}
			}
			if scenario == "lock-cleanup" || scenario == "rename-lock-cleanup" {
				before[".fairyc-edit.lock/"] = "directory"
			}
			if scenario == "rename-lock-cleanup" && !strings.Contains(stderr, "cleanup also failed") {
				t.Fatal("publication error was hidden:", stderr)
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatalf("unexpected changed bytes or leftovers: want %v; got %v", before, remoteTree(t, root))
			}
			if scenario == "success" {
				code, out, stderr = fetchCLIProcess(t, image, root, args, "")
				if code != 7 || out != "" || stderr == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
					t.Fatalf("old-lock retry was not refused: %d %s %s", code, out, stderr)
				}
			}
		})
	}
	for _, scenario := range []string{"source", "dependency", "lock"} {
		t.Run("staged-"+scenario, func(t *testing.T) {
			root, args := editFixture(t)
			t.Setenv("FAIRYC_TEST_EDIT_FAULT", "gate")
			candidateEditStageBarrier(t, image, root, append(args, "--apply"), func() string {
				path := map[string]string{"source": "src/main.fy", "dependency": "libs/answer/src/answer.fy", "lock": "fairy.lock"}[scenario]
				absolute := filepath.Join(root, filepath.FromSlash(path))
				data, err := os.ReadFile(absolute)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(absolute, append(data, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
				return path
			})
		})
	}
}

func candidateEditStageBarrier(t *testing.T, image, root string, args []string, mutate func() string) {
	t.Helper()
	before := remoteTree(t, root)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.Dir = root
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		cancel()
		if !waited {
			_ = cmd.Wait()
		}
	}()
	reader := bufio.NewReader(out)
	ready, err := reader.ReadString('\n')
	if err != nil || ready != "staged\n" {
		t.Fatalf("staging barrier: %q %v", ready, err)
	}
	// Do not reopen the live deletion-capable lock or staged file through Go.
	// Directory listings and the separately edited production input suffice.
	if info, err := os.Lstat(filepath.Join(root, ".fairyc-edit.lock")); err != nil || !info.IsDir() {
		t.Fatalf("missing live edit lock: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "src"))
	if err != nil {
		t.Fatal(err)
	}
	stages := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".fairyc-edit-stage-") {
			stages++
		}
	}
	if stages != 1 {
		t.Fatalf("staged objects: %d", stages)
	}
	changed := mutate()
	if _, exists := before[changed]; !exists {
		t.Fatal("mutation must name an existing input", changed)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(changed)))
	if err != nil {
		t.Fatal(err)
	}
	before[changed] = sourceedit.Digest(data)
	if _, err := in.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	_ = in.Close()
	rest, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	waited = true
	failure, ok := err.(*exec.ExitError)
	if !ok || failure.ExitCode() != 7 || len(rest) != 0 || !strings.Contains(diagnostic.String(), "workspace changed") {
		t.Fatalf("staged drift: %v %q %q", err, rest, &diagnostic)
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("staged drift overwrote source or leaked owned files")
	}
}
