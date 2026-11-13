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

func TestLinuxDirectoryNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("actual Linux GNU amd64 required")
	}
	support, err := SourceForSymbols([]string{"fy_std_windows_read_dir_open_v0"})
	if err != nil {
		t.Fatal(err)
	}
	source := support + `
#define CHECK(x) do{if(!(x))return __LINE__;}while(0)
int main(void){CHECK(symlink("file","root/link")==0&&symlink("root","alias")==0);struct fy_std_directory_v0 directory={NULL,0};uint8_t name[16];memset(name,85,sizeof name);CHECK(fy_std_windows_read_dir_open_v0((const uint8_t *)"root",4,&directory).status==0);struct fy_std_directory_entry_result_v0 e=fy_std_windows_read_dir_next_v0(&directory,NULL,0);CHECK(e.status==6&&e.name_length==4);unsigned seen=0;
for(;;){e=fy_std_windows_read_dir_next_v0(&directory,name,sizeof name);CHECK(e.status==0);if(e.kind==0)break;CHECK(e.name_length==4&&name[4]==85);if(!memcmp(name,"file",4)){CHECK(e.kind==1&&e.size==3&&e.reparse==0);seen|=1;}else if(!memcmp(name,"link",4)){CHECK(e.kind==3&&e.reparse==1);seen|=2;}else{return 1;}}CHECK(seen==3);CHECK(fy_std_windows_read_dir_close_v0(&directory).status==0&&fy_std_windows_read_dir_drop_v0(&directory)==0);CHECK(fy_std_windows_read_dir_open_v0((const uint8_t *)"alias",5,&directory).status!=0&&directory.open==0);return 0;}
`
	for _, cc := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
				if err != nil {
					t.Fatal(err)
				}
				image, err := cdriver.Build(toolchain, profile, source, "typedef int fy_linux_directory_probe_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				dir := t.TempDir()
				exe := filepath.Join(dir, "directory-probe")
				if err := os.WriteFile(exe, image.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(dir, "root"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "root", "file"), []byte("abc"), 0600); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				defer cancel()
				cmd := exec.CommandContext(ctx, exe)
				cmd.Dir = dir
				out, err := cmd.CombinedOutput()
				if err != nil || len(out) != 0 {
					t.Fatalf("Linux directory: %v %s", err, out)
				}
			})
		}
	}
}
