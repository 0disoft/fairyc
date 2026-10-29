package candidatehost

import (
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
	"runtime"
	"strings"
	"testing"
)

func TestPrivateLinuxStarterSelection(t *testing.T) {
	s, err := LinuxSource("fairyc_private_linux_starter_create_v1(")
	if err != nil || s != linuxRootedFile+"\n"+linuxStarter {
		t.Fatal("starter selection", err)
	}
	if _, err := LinuxSource("fairyc_private_linux_starter_unknown_v1("); err == nil {
		t.Fatal("unknown starter admitted")
	}
}
func TestLinuxStarterAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxRootedFile
	for _, h := range []string{"fcntl.h", "sys/stat.h", "sys/file.h", "unistd.h"} {
		source = strings.ReplaceAll(source, "#include <"+h+">\n", "")
	}
	for _, p := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		a, err := cdriver.BuildWindowsMSVCProbe(tc, p, linuxRootedPOSIXStub+source+linuxStarterPOSIXStub+linuxStarter+linuxStarterBoundaryProbe)
		if err != nil {
			t.Fatal(err)
		}
		runLinuxPathProbe(t, a.Bytes, "starter.exe")
	}
}
func TestLinuxStarterAdapterNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native rooted starter requires Linux amd64")
	}
	for _, cc := range []string{"gcc", "clang"} {
		tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			a, err := cdriver.Build(tc, p, linuxRootedFile+"\n"+linuxStarter+linuxStarterNativeProbe, "typedef int fy_runtime;\n")
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, a.Bytes, "starter")
		}
	}
}

const linuxStarterPOSIXStub = `
#include <stdarg.h>
#define O_WRONLY 1
#define O_CREAT 64
#define O_EXCL 128
#define AT_SYMLINK_NOFOLLOW 256
#define AT_REMOVEDIR 512
#ifndef ESTALE
#define ESTALE 116
#endif
static int starter_created[8],starter_held[8],starter_removed[8],starter_op,starter_slot,starter_injected,starter_replace,starter_extra;
static size_t starter_written[8];
static int starter_index(int parent,const char *name){
 if(parent==20){assert(strcmp(name,"app")==0);return 0;}
 if(parent==0){if(strcmp(name,"src")==0){return 1;}if(strcmp(name,"tests")==0){return 2;}if(strcmp(name,".gitignore")==0){return 3;}if(strcmp(name,"README.md")==0){return 4;}assert(strcmp(name,"fairy.toml")==0);return 5;}
 assert((parent==1 || parent==2) && strcmp(name,"main.fy")==0);return parent==1 ? 6 : 7;
}
static int starter_fail(int op,int index){if(starter_op==op && starter_slot==index && !starter_injected){starter_injected=1;errno=op==1 && index==0 ? EEXIST : EIO;return 1;}return 0;}
static int starter_mkdir(int parent,const char *name,int mode){int i=starter_index(parent,name);assert(mode==0755 && !starter_created[i]);if(starter_fail(1,i)){return -1;}starter_created[i]=1;return 0;}
static int starter_open(int parent,const char *name,int flags,...){
 int i=starter_index(parent,name);
 if(starter_fail(2,i)){return -1;}
 if(i<3){assert(starter_created[i] && flags==(O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC));}
 else{assert(!starter_created[i] && flags==(O_WRONLY|O_CREAT|O_EXCL|O_NOFOLLOW|O_CLOEXEC));va_list args;va_start(args,flags);assert(va_arg(args,int)==0644);va_end(args);starter_created[i]=1;}
 starter_held[i]=1;return i;
}
static void starter_info(int i,struct stat *info){memset(info,0,sizeof *info);info->st_dev=42;info->st_ino=(uint64_t)(100+i);info->st_mode=i<3 ? 16384 : 32768;}
static int starter_stat(int fd,struct stat *info){assert(fd>=0 && fd<8 && starter_held[fd]);if(starter_fail(3,fd)){return -1;}starter_info(fd,info);return 0;}
static int starter_statat(int parent,const char *name,struct stat *info,int flags){int i=starter_index(parent,name);assert(flags==AT_SYMLINK_NOFOLLOW);if(!starter_created[i]){errno=ENOENT;return -1;}starter_info(i,info);if(starter_replace==i+1){info->st_ino+=1000;}if(starter_fail(7,i)){return -1;}return 0;}
static ssize_t starter_write(int fd,const void *data,size_t size){assert(fd>=3 && fd<8 && starter_held[fd] && data && size>0 && size<=65536);if(starter_fail(4,fd)){return -1;}size_t n=size>2 ? 2 : size;starter_written[fd]+=n;return (ssize_t)n;}
static int starter_sync(int fd){assert(fd==20 || (fd>=0 && fd<8 && starter_held[fd]));return starter_fail(5,fd) ? -1 : 0;}
static int starter_close(int fd){assert(fd>=0 && fd<8 && starter_held[fd]);starter_held[fd]=0;return starter_fail(6,fd) ? -1 : 0;}
static int starter_unlink(int parent,const char *name,int flags){
 int i=starter_index(parent,name);assert(starter_created[i] && flags==(i<3 ? AT_REMOVEDIR : 0));assert(starter_replace!=i+1);
 if(i<3){for(int j=1;j<8;++j){if(starter_created[j] && (i==0 || i==1 && j==6 || i==2 && j==7)){errno=ENOTEMPTY;return -1;}}if(starter_extra && i==0){errno=ENOTEMPTY;return -1;}}
 if(starter_fail(8,i)){return -1;}starter_created[i]=0;++starter_removed[i];return 0;
}
#undef openat
#undef fstat
#undef write
#undef close
#undef fsync
#define mkdirat starter_mkdir
#define openat starter_open
#define fstat starter_stat
#define fstatat starter_statat
#define write starter_write
#define close starter_close
#define fsync starter_sync
#define unlinkat starter_unlink
`
const linuxStarterBoundaryProbe = `
static void reset_starter(int op,int slot){memset(starter_created,0,sizeof starter_created);memset(starter_held,0,sizeof starter_held);memset(starter_removed,0,sizeof starter_removed);memset(starter_written,0,sizeof starter_written);starter_op=op;starter_slot=slot;starter_injected=starter_replace=starter_extra=0;}
int main(void){
 const uint8_t name[]="app",body[]="abcdefghijklmno";size_t lengths[5]={3,3,3,3,3};uint32_t slot=0,rollback=0;
 reset_starter(0,0);assert(fairyc_private_linux_starter_create_v1(21,name,sizeof name,body,15,lengths,&slot,&rollback)==0 && rollback==0);
 for(int i=0;i<8;++i){assert(starter_created[i] && !starter_held[i]);if(i>=3){assert(starter_written[i]==3);}}
 for(int op=1;op<=7;++op){for(int i=0;i<8;++i){
  if(op==1 && i>=3 || op==4 && i<3){continue;}
  reset_starter(op,i);uint32_t code=fairyc_private_linux_starter_create_v1(21,name,sizeof name,body,15,lengths,&slot,&rollback);
  if(!starter_injected){continue;}assert(code!=0);
  for(int j=0;j<8;++j){assert(!starter_held[j]);}
  if(op==1 && i==0){assert(code==EEXIST && rollback==0);for(int j=0;j<8;++j){assert(!starter_created[j] && !starter_removed[j]);}}
  if(op==4 || op==5 || op==6 && i>=3 || op==7){assert(rollback==0);for(int j=0;j<8;++j){assert(!starter_created[j]);}}
  if(op==2 && i<3 || op==3){assert(rollback==1);}
  if(op==6 && i<3){assert(rollback==1);for(int j=0;j<8;++j){assert(starter_created[j]);}}
 }}
 reset_starter(5,20);assert(fairyc_private_linux_starter_create_v1(21,name,sizeof name,body,15,lengths,&slot,&rollback)!=0 && rollback==0);for(int j=0;j<8;++j){assert(!starter_created[j]);}
 reset_starter(4,7);starter_replace=7;assert(fairyc_private_linux_starter_create_v1(21,name,sizeof name,body,15,lengths,&slot,&rollback)!=0 && rollback==1 && !starter_removed[6]);
 reset_starter(4,7);starter_extra=1;assert(fairyc_private_linux_starter_create_v1(21,name,sizeof name,body,15,lengths,&slot,&rollback)!=0 && rollback==1 && starter_created[0]);
 reset_starter(0,0);lengths[0]=65537;assert(fairyc_private_linux_starter_create_v1(21,name,sizeof name,body,15,lengths,&slot,&rollback)==EINVAL);
 lengths[0]=3;assert(fairyc_private_linux_starter_create_v1(21,(const uint8_t *)"../app",7,body,15,lengths,&slot,&rollback)==EINVAL);
 return 0;
}
`
const linuxStarterNativeProbe = `
#include <assert.h>
#include <stdlib.h>
int main(void){
 char root[]="/tmp/fairyc-new-XXXXXX";assert(mkdtemp(root)!=NULL);uint32_t handle=0,slot=0,rollback=0;
 assert(fairyc_private_linux_directory_open_v1((const uint8_t *)root,strlen(root)+1,&handle)==0);
 const size_t lengths[5]={1,1,1,1,1};const uint8_t data[]={0,255,2,3,4};
 assert(fairyc_private_linux_starter_create_v1(handle,(const uint8_t *)"app",4,data,5,lengths,&slot,&rollback)==0 && rollback==0);
 assert(fairyc_private_linux_starter_create_v1(handle,(const uint8_t *)"app",4,data,5,lengths,&slot,&rollback)==EEXIST && rollback==0);
 int parent=fy_linux_root_fd(handle);assert(symlinkat("app",parent,"link")==0);
 assert(fairyc_private_linux_starter_create_v1(handle,(const uint8_t *)"link",5,data,5,lengths,&slot,&rollback)==EEXIST && rollback==0);
 const char *paths[]={".gitignore","README.md","fairy.toml","src/main.fy","tests/main.fy"};
 int project=openat(parent,"app",O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC);assert(project>=0);
 for(int i=0;i<5;++i){int fd=openat(project,paths[i],O_RDONLY|O_NOFOLLOW|O_CLOEXEC);assert(fd>=0);uint8_t value=99;assert(read(fd,&value,1)==1 && value==data[i] && close(fd)==0);assert(unlinkat(project,paths[i],0)==0);}
 assert(unlinkat(project,"src",AT_REMOVEDIR)==0 && unlinkat(project,"tests",AT_REMOVEDIR)==0 && close(project)==0);
 assert(unlinkat(parent,"app",AT_REMOVEDIR)==0 && unlinkat(parent,"link",0)==0);
 assert(fairyc_private_linux_root_file_close_v1(handle)==0 && rmdir(root)==0);return 0;
}
`
