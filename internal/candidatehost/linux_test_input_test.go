package candidatehost

import (
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
	"runtime"
	"strings"
	"testing"
)

func TestPrivateLinuxTestInputSelection(t *testing.T) {
	source, err := LinuxSource("fairyc_private_linux_test_capture_open_v1(")
	if err != nil || source != linuxProcess+"\n"+linuxCapture+"\n"+linuxTestInput {
		t.Fatal("input selection", err)
	}
}
func TestLinuxTestInputBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxProcess + "\n" + linuxCapture + "\n" + linuxTestInput
	for _, h := range []string{"spawn.h", "sys/wait.h", "unistd.h", "fcntl.h", "poll.h", "sys/stat.h"} {
		source = strings.ReplaceAll(source, "#include <"+h+">\n", "")
	}
	stub := strings.ReplaceAll(linuxProcessPOSIXStub, "low==3", "low==4")
	stub = strings.ReplaceAll(stub, "(to==1 || to==2)", "(to>=0 && to<=3)")
	captureStub := strings.ReplaceAll(linuxCapturePOSIXStub, "assert(duplications==2)", "assert(duplications==3 || duplications==4)")
	captureStub = strings.ReplaceAll(captureStub, "live_fd[from] && !(fd_flags[from]&O_NONBLOCK)", "live_fd[from] && (to==0 || !(fd_flags[from]&O_NONBLOCK))")
	captureStub = strings.ReplaceAll(captureStub, "for(int i=3;i<32;++i)", "for(int i=4;i<32;++i)")
	for _, p := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		a, err := cdriver.BuildWindowsMSVCProbe(tc, p, stub+captureStub+linuxTestInputPOSIXStub+source+linuxTestInputBoundaryProbe)
		if err != nil {
			t.Fatal(err)
		}
		runLinuxPathProbe(t, a.Bytes, "test-input.exe")
	}
}
func TestLinuxTestInputNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native stdin capture requires Linux amd64")
	}
	for _, cc := range []string{"gcc", "clang"} {
		tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			a, err := cdriver.Build(tc, p, linuxProcess+"\n"+linuxCapture+"\n"+linuxTestInput+linuxTestInputNativeProbe, "typedef int fy_runtime;\n")
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, a.Bytes, "test-input")
		}
	}
}

const linuxTestInputPOSIXStub = `
#define O_RDONLY 0
#define O_NOFOLLOW 131072
struct stat {int st_mode;int64_t st_size;};
#define S_ISREG(mode) ((mode)==1)
static int input_opens;
static int test_input_open(const char *path,int flags){
 assert(strcmp(path,"/tmp/replay")==0 && flags==(O_RDONLY|O_CLOEXEC|O_NOFOLLOW|O_NONBLOCK));++input_opens;
 if(scenario==200){errno=ENOENT;return -1;}return capture_fd();
}
static int test_input_stat(int fd,struct stat *info){
 assert(live_fd[fd]);if(scenario==201){errno=EIO;return -1;}
 info->st_mode=scenario==202 ? 2 : 1;info->st_size=scenario==203 ? 65537 : 3;return 0;
}
#define open test_input_open
#define fstat test_input_stat
`
const linuxTestInputBoundaryProbe = `
static void reset_input(int mode,int closed){scenario=mode;input_opens=0;pipes_created=duplications=0;collecting=0;memset(live_fd,0,sizeof live_fd);memset(fd_flags,0,sizeof fd_flags);if(!closed){live_fd[0]=live_fd[1]=live_fd[2]=1;}}
int main(void){
 const uint8_t data[]="x\0\0a b",cwd[]="/tmp",path[]="/tmp/replay";
 for(uint32_t mode=0;mode<3;++mode){for(int closed=0;closed<2;++closed){
  reset_input(0,closed);void *raw=NULL;
  assert(fairyc_private_linux_test_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,mode,mode==2 ? path : NULL,mode==2 ? sizeof path : 0,&raw)==0 && raw!=NULL);
  assert(input_opens==(mode==2) && duplications==(mode==0 ? 3 : 4));
  assert(fairyc_private_linux_capture_drop_v1(raw)==0);
  for(int i=closed ? 0 : 3;i<32;++i){assert(!live_fd[i]);}
 }}
 for(int mode=200;mode<204;++mode){reset_input(mode,0);void *raw=(void *)(uintptr_t)1;
  assert(fairyc_private_linux_test_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,2,path,sizeof path,&raw)!=0 && raw==NULL);
  for(int i=3;i<32;++i){assert(!live_fd[i]);}
 }
 reset_input(100,0);void *raw=NULL;assert(fairyc_private_linux_test_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,1,NULL,0,&raw)!=0 && raw==NULL);
 assert(fairyc_private_linux_test_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,3,NULL,0,&raw)==EINVAL);
 assert(fairyc_private_linux_test_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,0,path,sizeof path,&raw)==EINVAL);
 assert(fairyc_private_linux_test_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,2,(const uint8_t *)"relative",9,&raw)==EINVAL);
 return 0;
}
`
const linuxTestInputNativeProbe = `
#include <assert.h>
#include <stdio.h>
int main(int argc,char **argv){
 if(argc>1){uint8_t data[8];ssize_t n=read(0,data,sizeof data);assert(n>=0);if(n>0){assert(write(1,data,(size_t)n)==n);}return 0;}
 char image[4096];ssize_t n=readlink("/proc/self/exe",image,sizeof image-1);assert(n>0);image[n]=0;
 char packed[8192];size_t a=strlen(image)+1;memcpy(packed,image,a);memcpy(packed+a,"child",6);
 char replay[]="/tmp/fairyc-input-XXXXXX";int fd=mkstemp(replay);assert(fd>=0);const uint8_t data[]={0,255,7};assert(write(fd,data,3)==3 && close(fd)==0);
 for(uint32_t mode=0;mode<3;++mode){
  if(mode==0){int pair[2];assert(pipe(pair)==0);assert(write(pair[1],data,3)==3 && close(pair[1])==0 && dup2(pair[0],0)==0 && close(pair[0])==0);}
  void *raw=NULL;assert(fairyc_private_linux_test_capture_open_v1((uint8_t *)packed,a+6,2,(const uint8_t *)"/tmp",5,mode,mode==2 ? (const uint8_t *)replay : NULL,mode==2 ? strlen(replay)+1 : 0,&raw)==0);
  uint8_t out[4],err[1];size_t x=0,y=0;uint32_t kind=0,value=0,pid=0;
  assert(fairyc_private_linux_capture_collect_v1(raw,1000,out,4,&x,err,1,&y,&kind,&value,&pid)==0 && kind==1 && value==0);
  assert(x==(mode==1 ? 0u : 3u) && y==0);if(x){assert(memcmp(out,data,3)==0);}assert(fairyc_private_linux_capture_drop_v1(raw)==0);
 }
 char link[4096];snprintf(link,sizeof link,"%s-link",replay);assert(symlink(replay,link)==0);void *raw=NULL;
 assert(fairyc_private_linux_test_capture_open_v1((uint8_t *)packed,a+6,2,(const uint8_t *)"/tmp",5,2,(const uint8_t *)link,strlen(link)+1,&raw)!=0 && raw==NULL);
 assert(unlink(link)==0 && unlink(replay)==0);return 0;
}
`
