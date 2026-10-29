package cli

import "testing"

func TestLinuxWorkspaceFilePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_workspace_fs", "linux_directory", "linux_command", "linux_host_paths", "linux_rooted_file"}, "linux_workspace_fs", `
@effects(filesystem) fn i32 main(){owns_snapshots_and_read_handles();rejects_link_prefixes_and_preserves_close_failure();return 0;}
`, linuxWorkspacePolicyStub)
}

const linuxWorkspacePolicyStub = `
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <assert.h>
static uint32_t scenario, file_closes;
static size_t row;
void workspace_probe_reset(uint32_t mode){scenario=mode;file_closes=0;}
uint32_t workspace_probe_closes(void){return file_closes;}
uint32_t fairyc_private_linux_directory_open_v1(const uint8_t *path,size_t size,uint32_t *out){assert(size==2 && path[0]=='.');*out=10;return 0;}
uint32_t fairyc_private_linux_root_directory_open_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *out){assert(root==10 && size==4 && strcmp((const char *)leaf,"src")==0);*out=0;if(scenario==1){return 40;}*out=11;return 0;}
uint32_t fairyc_private_linux_root_file_open_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *out){assert(root==11 && size==8 && strcmp((const char *)leaf,"main.fy")==0);*out=20;return 0;}
uint32_t fairyc_private_linux_root_file_close_v1(uint32_t file){if(file==20){++file_closes;return scenario==2 ? 5u : 0u;}return 0;}
struct info{size_t device,identity,size;int64_t seconds;uint32_t nanos,mode;};
uint32_t fairyc_private_linux_root_file_info_v1(uint32_t file,void *out){assert(file==20);struct info value={1,2,3,4,5,32768};memcpy(out,&value,sizeof value);return 0;}
uint32_t fairyc_private_linux_root_file_read_v1(uint32_t file,uint8_t *out,size_t capacity,size_t *count){assert(file==20 && capacity>=3);memcpy(out,"abc",3);*count=3;return 0;}
uint32_t fairyc_private_linux_directory_scan_open_v1(uint32_t root,size_t *out){assert(root==11);row=0;*out=123;return 0;}
uint32_t fairyc_private_linux_directory_scan_close_v1(size_t cursor){assert(cursor==0 || cursor==123);return 0;}
uint32_t fairyc_private_linux_directory_scan_next_v1(size_t cursor,uint8_t *name,size_t capacity,uint32_t *length,uint32_t *state,void *out){assert(cursor==123 && capacity>=7);*length=0;*state=0;if(row++==0){memcpy(name,"main.fy",7);*length=7;*state=1;struct info value={1,2,3,4,5,32768};memcpy(out,&value,sizeof value);}return 0;}
`
