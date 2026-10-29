package cli

import "testing"

func TestLinuxStarterPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_starter", "project_starter", "project_manifest_values", "linux_rooted_file", "linux_command"}, "linux_starter", `
@effects(filesystem) fn i32 main(){validates_abi_errors_ownership_and_close();return 0;}
`, linuxStarterPolicyStub)
}

const linuxStarterPolicyStub = `
#include <stdint.h>
#include <stddef.h>
#include <assert.h>
#include <string.h>
static uint32_t mode,opens,creates,closes;
void linux_starter_mode(uint32_t value){mode=value;opens=creates=closes=0;}
uint32_t linux_starter_counts(void){return opens*100+creates*10+closes;}
uint32_t fairyc_private_linux_directory_open_v1(const uint8_t *path,size_t size,uint32_t *output){assert(size==2 && memcmp(path,".",2)==0);++opens;*output=mode==1 ? 0 : 1;return mode==1 ? 2 : 0;}
uint32_t fairyc_private_linux_root_file_close_v1(uint32_t raw){if(raw!=0){assert(raw==1);++closes;}return mode==7 && raw!=0 ? 5 : 0;}
uint32_t fairyc_private_linux_starter_create_v1(uint32_t parent,const uint8_t *name,size_t name_size,const uint8_t *data,size_t size,const void *raw_sizes,uint32_t *slot,uint32_t *rollback){
 assert(parent==1 && name_size==12 && memcmp(name,"hello-fairy",12)==0);++creates;
 const size_t *sizes=raw_sizes;size_t total=0;for(int i=0;i<5;++i){assert(sizes[i]>0 && sizes[i]<=65536-total);total+=sizes[i];}assert(total==size && size<=65536 && sizes[0]==8 && memcmp(data,"/build/\n",8)==0);
 assert(memcmp(data+sizes[0],"# hello-fairy\n",sizeof("# hello-fairy\n")-1)==0);
 size_t at=sizes[0]+sizes[1];assert(memcmp(data+at,"schema = ",sizeof("schema = ")-1)==0);
 at+=sizes[2];assert(memcmp(data+at,"module main;",sizeof("module main;")-1)==0);
 at+=sizes[3];assert(memcmp(data+at,"module tests::main;",sizeof("module tests::main;")-1)==0);

 *slot=mode==4 ? 6 : mode==6 ? 8 : 0;*rollback=mode==5 || mode==8 ? 1 : 0;
 return mode==2 ? 17 : mode==3 ? 13 : mode==4 || mode==5 ? 5 : 0;
}
`
