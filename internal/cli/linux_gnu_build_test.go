package cli

import "testing"

func TestLinuxGNUCompanionBuildPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_gnu_build", "linux_gnu_host", "linux_gnu_policy", "linux_command", "linux_elf", "linux_runtime", "linux_runtime_contract", "native_packet_decode", "sha256"}, "linux_gnu_build", `
@effects(filesystem,console,environment,clock) fn i32 main(){
    compiles_links_and_publishes_both_profiles();
    initializes_arguments_and_resolves_clang_without_go_fallback();
    preserves_old_image_and_reports_failure_phase();return 0;
}
`, linuxGNUBuildPolicyStub)
}

const linuxGNUBuildPolicyStub = `
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <assert.h>
static uint32_t scenario,compiled,linked,published,closed,expected_release,expected_arguments;
void gnu_probe_mode(uint32_t value,uint32_t release,uint32_t arguments){scenario=value;compiled=linked=published=closed=0;expected_release=release;expected_arguments=arguments;}
uint32_t gnu_probe_counts(void){return compiled*100+linked*10+published;}
uint32_t fairyc_private_linux_gnu_stage_v1(const uint8_t *path,size_t size,uint32_t *out){assert(size==7 && strcmp((const char *)path,"/stage")==0);*out=9;return 0;}
uint32_t fairyc_private_linux_root_file_close_v1(uint32_t fd){if(fd==0){return 0;}assert(fd==9);++closed;return scenario==12?5:0;}
uint32_t fairyc_private_linux_gnu_find_v1(const uint8_t *name,size_t size,uint8_t *data,size_t capacity,size_t *length){assert(name && size && capacity==4096);const char *path=strcmp((const char *)name,"getconf")==0?"/tools/getconf":strcmp((const char *)name,"clang")==0?"/tools/clang":"/tools/gcc";*length=strlen(path);memcpy(data,path,*length+1);return 0;}
uint32_t fairyc_private_linux_gnu_write_v1(uint32_t root,const uint8_t *leaf,size_t size,const uint8_t *data,size_t length){assert(root==9 && leaf && size && data && length);return scenario==5?5:0;}
static void gnu_quad(uint8_t *data,size_t at,uint64_t n){for(size_t i=0;i<8;++i){data[at+i]=(uint8_t)n;n>>=8;}}
static void gnu_elf(uint8_t *data,size_t n,int image){memset(data,0,n);memcpy(data,"\177ELF",4);data[4]=2;data[5]=1;data[6]=1;gnu_quad(data,16,image?2:1);gnu_quad(data,18,62);gnu_quad(data,20,1);gnu_quad(data,52,64);
    if(image){gnu_quad(data,24,4194424);gnu_quad(data,32,64);gnu_quad(data,54,56);gnu_quad(data,56,1);gnu_quad(data,64,1);gnu_quad(data,68,5);gnu_quad(data,72,120);gnu_quad(data,80,4194424);gnu_quad(data,96,8);gnu_quad(data,104,8);gnu_quad(data,112,1);}
    else{gnu_quad(data,40,64);gnu_quad(data,58,64);gnu_quad(data,60,2);gnu_quad(data,132,1);gnu_quad(data,152,192);gnu_quad(data,160,16);gnu_quad(data,176,1);}}
uint32_t fairyc_private_linux_gnu_read_v1(uint32_t root,const uint8_t *leaf,size_t size,uint8_t *data,size_t capacity,size_t *length){assert(root==9 && leaf && size);int image=strcmp((const char *)leaf,"fy_program")==0;*length=image?128:208;if(capacity==0){return 0;}assert(capacity==*length);gnu_elf(data,capacity,image);if((scenario==3 && !image) || (scenario==4 && image)){data[0]=0;}if(scenario==13){++*length;}return 0;}
uint32_t fairyc_private_linux_gnu_publish_v1(const uint8_t *root,size_t root_size,const uint8_t *path,size_t size,const uint8_t *data,size_t length){assert(strcmp((const char *)root,"/root")==0 && root_size==6 && strcmp((const char *)path,"build/program")==0 && size==14 && data && length==128 && closed==1);if(scenario==8){return 5;}++published;return 0;}
uint32_t fairyc_private_linux_gnu_run_v1(const uint8_t *data,size_t size,size_t count,const uint8_t *cwd,size_t cwd_size,uint32_t timeout,uint8_t *out,size_t *out_length,uint8_t *err,size_t *err_length,uint32_t *kind,uint32_t *value){
    assert(data && size && count>=2 && count<=10 && strcmp((const char *)cwd,"/stage")==0 && cwd_size==7 && timeout==60000);const char *argv[11];size_t at=0;for(size_t i=0;i<count;++i){argv[i]=(const char *)data+at;at+=strlen(argv[i])+1;}assert(at==size);
    *out_length=*err_length=0;*kind=1;*value=0;const char *text="";
    if(scenario==6){return 5;}if(scenario==14){*value=1000;return 0;}
    if(strcmp(argv[0],"/tools/getconf")==0){assert(count==2 && strcmp(argv[1],"GNU_LIBC_VERSION")==0);text=scenario==7?"glibc 2.38\n":"glibc 2.41\n";}
    else if(strcmp(argv[1],"--version")==0){text=strstr(argv[0],"clang")?"clang version 18.1.8\n":"gcc 14.2.0\n";}
    else if(strcmp(argv[1],"-dumpmachine")==0){text="x86_64-linux-gnu\n";}
    else if(strcmp(argv[1],"-dumpfullversion")==0){assert(count==3 && strcmp(argv[2],"-dumpversion")==0);text=scenario==9?"12.0.0\n":"14.2.0\n";}
    else if(strcmp(argv[1],"@fy_objects.rsp")==0){assert(count==4 && strcmp(argv[2],"-o")==0 && strcmp(argv[3],"fy_program")==0);++linked;if(scenario==2){*value=1;text="link error\n";}}
    else{assert(count==(expected_arguments?10u:9u) && strcmp(argv[1],"-std=c11")==0 && strcmp(argv[2],"-fno-common")==0 && strcmp(argv[3],expected_release?"-O2":"-O0")==0 && strcmp(argv[4],"-pipe")==0 && strcmp(argv[5],"-c")==0 && strcmp(argv[7],"-o")==0);if(expected_arguments){assert(strcmp(argv[9],"-DFY_CANDIDATE_NO_MAIN")==0);}++compiled;
        if(scenario==1){*value=1;text="compile error\n";}else if(scenario==10){*kind=3;}else if(scenario==11){*kind=4;}
    }
    if(*value!=0){*err_length=strlen(text);memcpy(err,text,*err_length);}else{*out_length=strlen(text);memcpy(out,text,*out_length);}return 0;
}
`
