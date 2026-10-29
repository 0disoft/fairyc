package candidatehost

import (
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
	"runtime"
	"strings"
	"testing"
)

func TestPrivateLinuxRunCaptureSelection(t *testing.T) {
	for _, name := range []string{"open", "receipt"} {
		source, err := LinuxSource("fairyc_private_linux_run_capture_" + name + "_v1(")
		if err != nil || source != linuxProcess+"\n"+linuxCapture {
			t.Fatalf("run capture %s: %v", name, err)
		}
	}
}

func TestLinuxRunCaptureAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxProcess + "\n" + linuxCapture
	for _, h := range []string{"spawn.h", "sys/wait.h", "unistd.h", "fcntl.h", "poll.h"} {
		source = strings.ReplaceAll(source, "#include <"+h+">\n", "")
	}
	stub := strings.ReplaceAll(linuxProcessPOSIXStub, "low==3", "low==4")
	stub = strings.ReplaceAll(stub, "(to==1 || to==2)", "(to>=1 && to<=3)")
	captureStub := strings.ReplaceAll(linuxCapturePOSIXStub, "wanted[2],copied[2]", "wanted[3],copied[3]")
	captureStub = strings.ReplaceAll(captureStub, "(fd==3 || fd==5)", "(fd==5 || fd==7 || fd==9)")
	captureStub = strings.ReplaceAll(captureStub, "fd==3 ? 0 : 1", "fd==5 ? 0 : fd==7 ? 1 : 2")
	captureStub = strings.ReplaceAll(captureStub, "count==2", "(count==2 || count==3)")
	captureStub = strings.ReplaceAll(captureStub, "assert(duplications==2)", "assert(duplications==3)")
	captureStub = strings.ReplaceAll(captureStub, "for(int i=3;i<32;++i)", "for(int i=4;i<32;++i)")
	for _, p := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		a, err := cdriver.BuildWindowsMSVCProbe(tc, p, stub+captureStub+source+linuxRunCaptureBoundaryProbe)
		if err != nil {
			t.Fatal(err)
		}
		runLinuxPathProbe(t, a.Bytes, "run-capture.exe")
	}
}

func TestLinuxRunCaptureAdapterNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native fd3 capture requires Linux amd64")
	}
	for _, compiler := range []string{"gcc", "clang"} {
		tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			a, err := cdriver.Build(tc, p, linuxProcess+"\n"+linuxCapture+linuxRunCaptureNativeProbe, "typedef int fy_runtime;\n")
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, a.Bytes, "run-capture")
		}
	}
}

const linuxRunCaptureBoundaryProbe = `
int main(void) {
 const uint8_t data[]="x\0\0a b",cwd[]="/tmp";
 for(int mode=0;mode<3;++mode) {
  scenario=0;live_fd[0]=live_fd[1]=live_fd[2]=1;
  pipes_created=duplications=0;void *raw=NULL;
  assert(fairyc_private_linux_run_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,&raw)==0);
  fy_linux_capture *c=raw;assert(c->readers[0]==5 && c->readers[1]==7 && c->readers[2]==9);
  uint8_t receipt[276];size_t length=99;
  assert(fairyc_private_linux_run_capture_receipt_v1(raw,receipt,sizeof receipt,&length)==EINVAL && length==0);
  collecting=1;capture_case=0;fake_now=1995000000;clock_calls=0;poll_calls=0;read_calls=0;
  wanted[0]=wanted[1]=0;wanted[2]=mode==0 ? 0 : mode==1 ? 276 : 277;
  copied[0]=copied[1]=copied[2]=0;
  size_t a=0,b=0;uint32_t kind=0,value=0,pid=0;
  uint32_t code=fairyc_private_linux_capture_collect_v1(raw,25,NULL,0,&a,NULL,0,&b,&kind,&value,&pid);
  if(mode==2) { assert(code==EOVERFLOW); }
  else {
   assert(code==0 && kind==1 && value==41);
   assert(fairyc_private_linux_run_capture_receipt_v1(raw,receipt,sizeof receipt,&length)==0 && length==wanted[2]);
   for(size_t i=0;i<length;++i) {assert(receipt[i]==(uint8_t)(i*37+2));}
  }
  collecting=0;assert(fairyc_private_linux_capture_drop_v1(raw)==0);
  for(int i=3;i<32;++i) {assert(!live_fd[i]);}
 }
 memset(live_fd,0,sizeof live_fd);memset(fd_flags,0,sizeof fd_flags);
 pipes_created=duplications=0;void *closed_raw=NULL;
 assert(fairyc_private_linux_run_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,&closed_raw)==0);
 fy_linux_capture *closed=closed_raw;
 assert(closed->readers[0]>=4 && closed->readers[1]>=4 && closed->readers[2]>=4);
 assert(fairyc_private_linux_capture_drop_v1(closed_raw)==0);
 for(int i=0;i<32;++i) {assert(!live_fd[i]);}
 return 0;
}
`

const linuxRunCaptureNativeProbe = `
#include <assert.h>
int main(int argc,char **argv) {
 if(argc>1) { if(strcmp(argv[1],"signal")==0){raise(SIGTERM);return 1;} if(strcmp(argv[1],"timeout")==0){for(;;){pause();}} if(strcmp(argv[1],"output")==0){assert(write(1,"x",1)==1);return 0;} uint8_t data[277];memset(data,5,sizeof data);size_t n=(size_t)atoi(argv[1]);assert(write(3,data,n)==(ssize_t)n);return 133; }
 char image[4096];ssize_t n=readlink("/proc/self/exe",image,sizeof image-1);assert(n>0);image[n]=0;
 for(int mode=0;mode<6;++mode) {
  const char *arg=mode==0 ? "0" : mode==1 ? "276" : mode==2 ? "277" : mode==3 ? "signal" : mode==4 ? "timeout" : "output";
  char packed[8192];size_t a=strlen(image)+1,b=strlen(arg)+1;memcpy(packed,image,a);memcpy(packed+a,arg,b);
  void *raw=NULL;assert(fairyc_private_linux_run_capture_open_v1((uint8_t *)packed,a+b,2,(const uint8_t *)"/tmp",5,&raw)==0);
  size_t out=0,err=0,length=0;uint32_t kind=0,value=0,pid=0;uint8_t receipt[276];
  uint32_t code=fairyc_private_linux_capture_collect_v1(raw,mode==4 ? 50 : 1000,NULL,0,&out,NULL,0,&err,&kind,&value,&pid);
  if(mode==2) {assert(code==EOVERFLOW);}
  else if(mode==3){assert(code==0 && kind==2 && value==SIGTERM);assert(fairyc_private_linux_run_capture_receipt_v1(raw,receipt,sizeof receipt,&length)==0 && length==0);}
  else if(mode==4){assert(code==0 && kind==3 && value==0);}
  else if(mode==5){assert(code==0 && kind==4 && value==1);}
  else {
   assert(code==0 && kind==1 && value==133);
   assert(fairyc_private_linux_run_capture_receipt_v1(raw,receipt,sizeof receipt,&length)==0 && length==(mode==0 ? 0u : 276u));
  }
  assert(fairyc_private_linux_capture_drop_v1(raw)==0);
 }
 return 0;
}
`
