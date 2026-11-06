package candidatehost

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxEditFileSelection(t *testing.T) {
	for _, name := range []string{"prepare", "publish"} {
		source, err := LinuxSource("fairyc_private_linux_edit_file_" + name + "_v1(")
		if err != nil || source != linuxRootedFile+"\n"+linuxEditFile {
			t.Fatalf("edit %s selection: %v", name, err)
		}
	}
	source, err := LinuxSource("fairyc_private_linux_edit_file_publish_v1( fairyc_private_linux_package_file_info_v1( fairyc_private_linux_root_file_erase_if_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxPackageFile+"\n"+linuxEditFile+"\n"+linuxRetention {
		t.Fatalf("combined edit source order: %v", err)
	}
	if _, err := LinuxSource("fairyc_private_linux_edit_file_unknown_v1("); err == nil {
		t.Fatal("unknown edit entry admitted")
	}
}

func TestLinuxEditFileAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC independent POSIX model; actual Linux execution is separate")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxEditPOSIXModel+linuxEditFile+linuxEditBoundaryProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "edit-file.exe")
		})
	}
}

func TestLinuxEditFileAdapterNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires GNU/Linux amd64, GCC and Clang; execution stays batched")
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, linuxRootedFile+linuxEditFile+linuxEditNativeProbe, "typedef int fy_edit_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "edit-file")
			})
		}
	}
}

// No Linux syscalls execute on Windows. This independent namespace model checks
// ordering, preservation and fault propagation in the actual C adapter.
const linuxEditPOSIXModel = `
#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <limits.h>
#define ESTALE 116
#define AT_SYMLINK_NOFOLLOW 256
#define S_ISREG(m) (((m)&61440u)==32768u)
#define S_ISDIR(m) (((m)&61440u)==16384u)
typedef uint32_t mode_t;
struct stat {uint64_t st_dev,st_ino,st_nlink;int64_t st_size;uint32_t st_mode;struct{int64_t tv_sec,tv_nsec;}st_mtim;};
struct fy_linux_root_metadata_v1{size_t device,identity,size;int64_t modified_seconds;uint32_t modified_nanos,mode;};
static int fault,stats,chmods,syncs,links,renames,unlinks,checks,body,target_new;
static uint32_t staged_mode;
static int fy_linux_root_fd(uint32_t h){return h!=0 && h<=(uint32_t)INT_MAX+1u?(int)(h-1u):-1;}
static int fy_linux_root_path(const uint8_t *p,size_t n,int leaf){assert(leaf);return p && n>=2 && n<=256 && p[n-1]==0 && !memchr(p,0,n-1) && !memchr(p,'/',n-1) && !(n==2 && p[0]=='.') && !(n==3 && p[0]=='.' && p[1]=='.');}
static int edit_fstat(int fd,struct stat *s){
 ++stats;if(fault==7 && stats==1){errno=EINTR;return -1;}
 if((fault==1 && fd==0)||(fault==21 && fd==1)||(fault==22 && fd==2)){errno=EIO;return -1;}
 assert(fd>=0 && fd<=2);memset(s,0,sizeof *s);s->st_dev=71;s->st_ino=(uint64_t)(111+fd);s->st_mode=fd==0?32768u|staged_mode:16384u;
 s->st_nlink=fd==0?(body || fault==3?1:0):1;s->st_size=3;
 if(fault==2 && fd==0){s->st_mode=4096;}
 if(fault==8 && fd==2){s->st_ino=112;}
 if(fault==9 && fd==2){s->st_dev=72;}
 return 0;
}
static int edit_fchmod(int fd,mode_t mode){assert(fd==0);++chmods;if(fault==4){errno=EPERM;return -1;}if(fault==5 && chmods==1){errno=EINTR;return -1;}staged_mode=mode;return 0;}
static uint32_t fairyc_private_linux_root_file_sync_v1(uint32_t h){assert(h==1);++syncs;return fault==6?EIO:0;}
static int edit_fstatat(int fd,const char *name,struct stat *s,int flags){
 assert(flags==AT_SYMLINK_NOFOLLOW);memset(s,0,sizeof *s);s->st_dev=71;s->st_ino=111;s->st_nlink=1;s->st_size=3;s->st_mode=32768u|0640u;s->st_mtim.tv_sec=91;s->st_mtim.tv_nsec=42;
 if(fd==1){assert(strcmp(name,"body")==0 && body);if(fault==18){s->st_ino=999;}if(fault==19){errno=EIO;return -1;}return 0;}
 assert(fd==2 && strcmp(name,"target")==0);++checks;
 if(fault==10){errno=ENOENT;return -1;}
 if(fault==13 && body){++s->st_mtim.tv_nsec;}
 if(fault==14 && checks==1){errno=EINTR;return -1;}
 if(fault==23){s->st_mode=40960;}
 if(fault==24){s->st_nlink=2;}
 if(fault==26){s->st_mtim.tv_nsec=INT64_C(4294967338);}
 if(fault==27){s->st_mtim.tv_nsec=1000000000;}
 return 0;
}
static uint32_t fy_linux_root_publish_new(uint32_t h,uint32_t root,const uint8_t *leaf,size_t size){assert(h==1 && root==2 && size==5 && strcmp((const char *)leaf,"body")==0);++links;if(fault==12){return EEXIST;}assert(!body);body=1;return 0;}
static int edit_renameat(int a,const char *x,int b,const char *y){assert(a==1 && b==2 && strcmp(x,"body")==0 && strcmp(y,"target")==0 && body);++renames;if(fault>=15 && fault<=19){errno=fault==16?EINTR:EIO;return -1;}body=0;target_new=1;return 0;}
static int edit_unlinkat(int fd,const char *name,int flags){assert(fd==1 && strcmp(name,"body")==0 && flags==0 && body);++unlinks;if(fault==17){errno=EPERM;return -1;}body=0;return 0;}
#define fstat edit_fstat
#define fchmod edit_fchmod
#define fstatat edit_fstatat
#define renameat edit_renameat
#define unlinkat edit_unlinkat
`

const linuxEditBoundaryProbe = `
static void reset(int f){fault=f;stats=chmods=syncs=links=renames=unlinks=checks=body=target_new=0;staged_mode=0640;}
static struct fy_linux_root_metadata_v1 expected(void){struct fy_linux_root_metadata_v1 m={71,111,3,91,42,32768u|0640u};return m;}
static uint32_t publish(struct fy_linux_root_metadata_v1 *m,uint32_t *cleanup){return fairyc_private_linux_edit_file_publish_v1(1,2,3,(const uint8_t *)"target",7,m,cleanup);}
int main(void){
 for(uint32_t mode=0;mode<=0777u;++mode){reset(0);assert(fairyc_private_linux_edit_file_prepare_v1(1,mode)==0 && chmods==1 && syncs==1 && staged_mode==mode);}
 reset(0);assert(fairyc_private_linux_edit_file_prepare_v1(1,01000)==EINVAL && stats==0);assert(fairyc_private_linux_edit_file_prepare_v1(0,0644)==EINVAL && stats==0);
 assert(fairyc_private_linux_edit_file_prepare_v1(UINT32_MAX,0644)==EINVAL && stats==0);
 for(int f=1;f<=7;++f){reset(f);uint32_t code=fairyc_private_linux_edit_file_prepare_v1(1,0644);assert(code==(uint32_t)(f==1 || f==6?EIO:f==2 || f==3?EINVAL:f==4?EPERM:0));if(f==4){assert(syncs==0);}if(f==5){assert(chmods==2);}if(f==7){assert(stats==2);}}
 struct fy_linux_root_metadata_v1 m=expected(),saved=m;uint32_t cleanup=99;
 reset(0);assert(publish(&m,&cleanup)==0 && cleanup==0 && links==1 && renames==1 && unlinks==0 && target_new && !body && memcmp(&m,&saved,sizeof m)==0);
 for(int field=0;field<6;++field){reset(0);m=expected();if(field==0){++m.device;}if(field==1){++m.identity;}if(field==2){++m.size;}if(field==3){++m.modified_seconds;}if(field==4){++m.modified_nanos;}if(field==5){m.mode^=01000;}assert(publish(&m,&cleanup)==ESTALE && cleanup==0 && links==0 && renames==0 && !target_new);}
 const int failures[]={1,2,3,8,9,10,12,13,15,16,17,18,19,21,22,23,26,27};
 const uint32_t codes[]={EIO,EINVAL,EINVAL,EINVAL,EXDEV,ENOENT,EEXIST,ESTALE,EIO,EINTR,EIO,EIO,EIO,EIO,EIO,ESTALE,ESTALE,ESTALE};
 for(size_t i=0;i<sizeof failures/sizeof failures[0];++i){reset(failures[i]);m=expected();cleanup=99;assert(publish(&m,&cleanup)==codes[i] && !target_new && renames<=1);assert(cleanup==(uint32_t)(fault==17?EPERM:fault==18?ESTALE:fault==19?EIO:0));if(fault==13){assert(renames==0 && unlinks==1 && !body);}if(fault==16){assert(renames==1 && unlinks==1);}if(fault==18){assert(unlinks==0 && body);}}
 reset(14);m=expected();assert(publish(&m,&cleanup)==0 && checks==3 && renames==1);
 reset(24);m=expected();assert(publish(&m,&cleanup)==0 && cleanup==0 && renames==1 && target_new);
 reset(0);m=expected();m.mode=32768|0400;assert(publish(&m,&cleanup)==EACCES && stats==0 && links==0);
 m=expected();m.modified_nanos=1000000000;assert(publish(&m,&cleanup)==EINVAL && stats==0);
 m=expected();m.mode=16384|0700;assert(publish(&m,&cleanup)==EINVAL && stats==0);
 m=expected();staged_mode=0600;assert(publish(&m,&cleanup)==EINVAL && links==0);
 reset(0);m=expected();assert(fairyc_private_linux_edit_file_publish_v1(1,2,3,(const uint8_t *)"a/b",4,&m,&cleanup)==EINVAL && stats==0);
 assert(fairyc_private_linux_edit_file_publish_v1(0,2,3,(const uint8_t *)"target",7,&m,&cleanup)==EINVAL && stats==0);
 assert(fairyc_private_linux_edit_file_publish_v1(1,2,3,(const uint8_t *)"target",7,NULL,&cleanup)==EINVAL);
 assert(fairyc_private_linux_edit_file_publish_v1(1,2,3,(const uint8_t *)"target",7,&m,NULL)==EINVAL);
 assert(fairyc_private_linux_edit_file_publish_v1(1,2,3,(const uint8_t *)"target",7,&m,(uint32_t *)&m)==EINVAL && memcmp(&m,&saved,sizeof m)==0);
 return 0;
}
`

const linuxEditNativeProbe = `
#include <assert.h>
#include <stdlib.h>
int main(void){
 char path[]="/tmp/fairyc-edit-XXXXXX";assert(mkdtemp(path));uint32_t root=0,stage=0,old=0,file=0;
 assert(fairyc_private_linux_directory_open_v1((const uint8_t *)path,strlen(path)+1,&root)==0);int parent=fy_linux_root_fd(root);
 int original=openat(parent,"target",O_CREAT|O_EXCL|O_RDWR|O_CLOEXEC,0640);assert(original>=0);assert(write(original,"old",3)==3 && fchmod(original,0640)==0 && close(original)==0);
 assert(mkdirat(parent,"stage",0700)==0 && fairyc_private_linux_root_directory_open_v1(root,(const uint8_t *)"stage",6,&stage)==0);
 assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"target",7,&old)==0);struct fy_linux_root_metadata_v1 expected,changed;
 assert(fairyc_private_linux_root_file_info_v1(old,&expected)==0 && (expected.mode&0777u)==0640u);
 assert(fairyc_private_linux_root_file_create_v1(stage,&file)==0);size_t copied=0;
 assert(fairyc_private_linux_root_file_write_v1(file,(const uint8_t *)"n\0\xff",3,&copied)==0 && copied==3);
 assert(fairyc_private_linux_edit_file_prepare_v1(file,0640)==0);uint32_t cleanup=99;changed=expected;changed.modified_seconds^=1;
 assert(fairyc_private_linux_edit_file_publish_v1(file,stage,root,(const uint8_t *)"target",7,&changed,&cleanup)==ESTALE && cleanup==0);
 assert(fairyc_private_linux_edit_file_publish_v1(file,stage,root,(const uint8_t *)"target",7,&expected,&cleanup)==0 && cleanup==0);
 uint8_t data[3];assert(read(fy_linux_root_fd(old),data,3)==3 && memcmp(data,"old",3)==0);
 uint32_t current=0;assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"target",7,&current)==0);
 assert(fairyc_private_linux_root_file_info_v1(current,&changed)==0 && (changed.mode&0777u)==0640u && changed.identity!=expected.identity);
 assert(read(fy_linux_root_fd(current),data,3)==3 && memcmp(data,"n\0\xff",3)==0);
 assert(fairyc_private_linux_root_file_close_v1(old)==0 && fairyc_private_linux_root_file_close_v1(file)==0);
 assert(fchmod(fy_linux_root_fd(current),0400)==0 && fairyc_private_linux_root_file_info_v1(current,&expected)==0);
 assert(fairyc_private_linux_root_file_create_v1(stage,&file)==0 && fairyc_private_linux_edit_file_prepare_v1(file,0400)==0);
 assert(fairyc_private_linux_edit_file_publish_v1(file,stage,root,(const uint8_t *)"target",7,&expected,&cleanup)==EACCES && cleanup==0);
 assert(fairyc_private_linux_root_file_close_v1(file)==0 && fchmod(fy_linux_root_fd(current),0640)==0);
 assert(linkat(parent,"target",parent,"alias",0)==0 && fairyc_private_linux_root_file_info_v1(current,&expected)==0);
 assert(fairyc_private_linux_root_file_create_v1(stage,&file)==0);
 assert(fairyc_private_linux_root_file_write_v1(file,(const uint8_t *)"next",4,&copied)==0 && copied==4 && fairyc_private_linux_edit_file_prepare_v1(file,0640)==0);
 assert(fairyc_private_linux_edit_file_publish_v1(file,stage,root,(const uint8_t *)"target",7,&expected,&cleanup)==0 && cleanup==0);
 uint32_t alias=0;assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"alias",6,&alias)==0);
 assert(read(fy_linux_root_fd(alias),data,3)==3 && memcmp(data,"n\0\xff",3)==0 && fairyc_private_linux_root_file_close_v1(alias)==0);
 uint32_t replaced=0;assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"target",7,&replaced)==0);uint8_t next[4];
 assert(read(fy_linux_root_fd(replaced),next,4)==4 && memcmp(next,"next",4)==0 && fairyc_private_linux_root_file_close_v1(replaced)==0);
 assert(fairyc_private_linux_root_file_close_v1(file)==0 && fairyc_private_linux_root_file_close_v1(current)==0 && unlinkat(parent,"alias",0)==0);
 assert(unlinkat(parent,"target",0)==0 && fairyc_private_linux_root_file_close_v1(stage)==0 && unlinkat(parent,"stage",AT_REMOVEDIR)==0);
 assert(fairyc_private_linux_root_file_close_v1(root)==0 && rmdir(path)==0);return 0;
}
`
