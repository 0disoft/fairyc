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

func TestLinuxFilePathsNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("actual Linux GNU amd64 required; model is a separate test")
	}
	support, err := SourceForSymbols([]string{"fy_std_windows_file_copy_bytes_v0", "fy_std_windows_file_hard_link_v0", "fy_std_windows_file_rename_v0"})
	if err != nil {
		t.Fatal(err)
	}
	source := support + `
#define CHECK(x) do{if(!(x))return __LINE__;}while(0)
int main(void){struct fy_std_file_result_v0 r;
r=fy_std_windows_file_copy_bytes_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==0&&r.count==3);
r=fy_std_windows_file_copy_bytes_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==1&&r.count==0);
char bytes[4]={0};FILE *f=fopen("destination","rb");CHECK(f!=NULL&&fread(bytes,1,4,f)==3&&!memcmp(bytes,"abc",3));CHECK(fclose(f)==0);
r=fy_std_windows_file_hard_link_v0((const uint8_t *)"source",(const uint8_t *)"alias");CHECK(r.status==0);struct stat a,b;CHECK(lstat("source",&a)==0&&lstat("alias",&b)==0&&a.st_dev==b.st_dev&&a.st_ino==b.st_ino);
r=fy_std_windows_file_rename_v0((const uint8_t *)"alias",(const uint8_t *)"destination");CHECK(r.status==1&&lstat("alias",&b)==0);
r=fy_std_windows_file_rename_v0((const uint8_t *)"alias",(const uint8_t *)"renamed");CHECK(r.status==0&&lstat("alias",&b)!=0&&lstat("renamed",&b)==0&&a.st_ino==b.st_ino);return 0;}
`
	for _, cc := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
				if err != nil {
					t.Fatal(err)
				}
				image, err := cdriver.Build(toolchain, profile, source, "typedef int fy_linux_paths_probe_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				dir := t.TempDir()
				exe := filepath.Join(dir, "paths-probe")
				if err := os.WriteFile(exe, image.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "source"), []byte("abc"), 0600); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				defer cancel()
				cmd := exec.CommandContext(ctx, exe)
				cmd.Dir = dir
				out, err := cmd.CombinedOutput()
				if err != nil || len(out) != 0 {
					t.Fatalf("Linux paths: %v %s", err, out)
				}
			})
		}
	}
}
