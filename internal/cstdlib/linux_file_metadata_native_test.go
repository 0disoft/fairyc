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

func TestLinuxFileMetadataNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("actual Linux GNU amd64 required")
	}
	support, err := SourceForSymbols([]string{"fy_std_windows_file_kind_v0"})
	if err != nil {
		t.Fatal(err)
	}
	source := support + `
#define CHECK(x) do{if(!(x))return __LINE__;}while(0)
int main(void){struct fy_std_file_storage_v0 file={0,0};struct fy_std_file_open_options_v0 options={1,1,0,0,0,0};CHECK(fy_std_file_open_v0((const uint8_t *)"source",&options,&file).status==0);
CHECK(fy_std_windows_file_kind_v0(&file).kind==1);CHECK(fy_std_windows_file_path_kind_v0((const uint8_t *)"source").kind==1);
struct stat expected;CHECK(fstat((int)(file.handle-1),&expected)==0);struct fy_std_file_id_v0 id;CHECK(fy_std_windows_file_id_v0(&file,&id).status==0);CHECK(id.volume_low==(uint32_t)expected.st_dev&&id.volume_high==(uint32_t)((uint64_t)expected.st_dev>>32)&&id.file_low_0==(uint32_t)expected.st_ino&&id.file_low_1==(uint32_t)((uint64_t)expected.st_ino>>32));
CHECK(link("source","alias")==0);CHECK(fy_std_windows_file_link_count_v0(&file).size>=2);CHECK(fy_std_windows_file_alloc_size_v0(&file).size==(size_t)expected.st_blocks*512);
CHECK(fy_std_windows_file_set_readonly_v0(&file,1).status==0&&fy_std_windows_file_read_only_v0(&file).enabled==1);CHECK(fstat((int)(file.handle-1),&expected)==0&&(expected.st_mode&0222)==0);CHECK(fy_std_windows_file_set_readonly_v0(&file,0).status==0&&fy_std_windows_file_read_only_v0(&file).enabled==0);
CHECK(fy_std_file_seek_start_v0(&file,1).status==0);CHECK(fy_std_windows_file_set_len_v0(&file,32).size==32);CHECK(fy_std_file_size_v0(&file).size==32&&fy_std_file_position_v0(&file).position==1);CHECK(fy_std_windows_file_set_len_v0(&file,UINT64_MAX).status==2);CHECK(fy_std_file_close_v0(&file).status==0);CHECK(fy_std_windows_file_kind_v0(&file).status==3);return 0;}
`
	for _, cc := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
				if err != nil {
					t.Fatal(err)
				}
				image, err := cdriver.Build(toolchain, profile, source, "typedef int fy_linux_metadata_probe_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				dir := t.TempDir()
				exe := filepath.Join(dir, "metadata-probe")
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
					t.Fatalf("Linux metadata: %v %s", err, out)
				}
			})
		}
	}
}
