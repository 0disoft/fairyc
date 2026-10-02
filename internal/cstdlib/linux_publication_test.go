package cstdlib

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxPreparedPublication(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux GNU amd64")
	}
	support, err := SourceForSymbols([]string{"fy_std_file_replace_v0", "fy_std_file_remove_v0", "fy_std_file_read_exact_v0", "fy_std_process_arg_v0"})
	if err != nil {
		t.Fatal(err)
	}
	// Probe production Linux functions; no replacement implementation in the fixture.
	source := "#define _POSIX_C_SOURCE 200809L\n" + support + `
#include <assert.h>
static void assert_file(const char *path, const char *expected) {
    char data[16] = {0};
    FILE *f = fopen(path, "rb"); assert(f != NULL);
    size_t length = fread(data, 1, sizeof data, f);
    assert(length == strlen(expected) && memcmp(data, expected, length) == 0);
    assert(fclose(f) == 0);
}
int main(int argc, char **argv) {
    fy_std_process_set_args_v0(argc,argv);
    uint8_t sentinel[8] = {85,85,85,85,85,85,85,85};
    struct fy_std_process_args_result_v0 a=fy_std_process_arg_v0(1,sentinel,0);
    assert(a.status==3 && a.needed==4 && sentinel[0]==85);
    a=fy_std_process_arg_v0(1,sentinel,8);
    assert(a.status==0 && a.needed==4 && memcmp(sentinel,"한x",4)==0 && sentinel[4]==85);
    a=fy_std_process_arg_v0(2,sentinel,8); assert(a.status==2 && sentinel[4]==85);
    a=fy_std_process_arg_v0(3,sentinel,8); assert(a.status==0 && a.needed==0 && sentinel[4]==85);
    a=fy_std_process_arg_v0(4,sentinel,8); assert(a.status==1);
    struct fy_std_file_result_v0 r=fy_std_file_replace_v0((const uint8_t *)"missing",(const uint8_t *)"live"); assert(r.status==1);
    r=fy_std_file_replace_v0((const uint8_t *)"",(const uint8_t *)"live"); assert(r.status==2);
    r=fy_std_file_replace_v0((const uint8_t *)"directory",(const uint8_t *)"live"); assert(r.status==2);
    r=fy_std_file_replace_v0((const uint8_t *)"prepared",(const uint8_t *)"directory"); assert(r.status==1);
    assert_file("live","old"); assert_file("prepared","new");
    assert(symlink("prepared","alias") == 0);
    r=fy_std_file_replace_v0((const uint8_t *)"alias",(const uint8_t *)"live"); assert(r.status==2);
    assert_file("live","old"); assert_file("prepared","new");
    FILE *old = fopen("live","rb"); assert(old != NULL);
    r=fy_std_file_replace_v0((const uint8_t *)"prepared",(const uint8_t *)"live"); assert(r.status==0);
    char old_bytes[3]; assert(fread(old_bytes,1,3,old)==3 && memcmp(old_bytes,"old",3)==0); assert(fclose(old)==0);
    struct fy_std_file_storage_v0 f={0,0};
    struct fy_std_file_open_options_v0 options={1,0,0,0,0,0};
    r=fy_std_file_open_v0((const uint8_t *)"live",&options,&f); assert(r.status==0);
    r=fy_std_file_read_exact_v0(&f,sentinel,5); assert(r.status==7 && r.count==3 && memcmp(sentinel,"new",3)==0);
    assert(sentinel[3]=='x' && sentinel[4]==85);
    r=fy_std_file_close_v0(&f); assert(r.status==0);
    r=fy_std_file_read_exact_v0(&f,NULL,0); assert(r.status==3);
    r=fy_std_file_replace_v0((const uint8_t *)"first",(const uint8_t *)"created"); assert(r.status==0);
    r=fy_std_file_remove_v0((const uint8_t *)"created"); assert(r.status==0);
    r=fy_std_file_remove_v0((const uint8_t *)"directory"); assert(r.status==1);
    r=fy_std_file_remove_v0((const uint8_t *)"missing"); assert(r.status==1);
    r=fy_std_file_remove_v0((const uint8_t *)"alias"); assert(r.status==0);
    return 0;
}
`
	for _, cc := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, source, "typedef int fy_publication_probe_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				root := t.TempDir()
				exe := filepath.Join(root, "probe")
				for name, data := range map[string]string{"live": "old", "prepared": "new", "first": "first"} {
					if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Mkdir(filepath.Join(root, "directory"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(exe, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, exe, "한x", string([]byte{0xff}), "")
				cmd.Dir = root
				if out, err := cmd.CombinedOutput(); err != nil || len(out) != 0 {
					t.Fatalf("probe: %v %q", err, out)
				}
				data, err := os.ReadFile(filepath.Join(root, "live"))
				if err != nil || string(data) != "new" {
					t.Fatalf("live=%q err=%v", data, err)
				}
				if _, err := os.Stat(filepath.Join(root, "prepared")); !os.IsNotExist(err) {
					t.Fatal("prepared name survived successful publication")
				}
			})
		}
	}
}
