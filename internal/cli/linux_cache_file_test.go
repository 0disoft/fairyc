package cli

import "testing"

func TestLinuxCacheFilePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_directory", "linux_retention", "linux_workspace_fs", "linux_host_paths", "linux_cache_file"}, "linux_cache_file", `
@effects(filesystem) fn i32 main(){anonymous_stage_publication_and_collision_preserve_ownership();inventory_ignores_named_stages_and_keeps_exact_removal_metadata();return 0;}
`, linuxCacheFileStub)
}

const linuxCacheFileStub = `
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <assert.h>
static uint32_t mode, published, closes, row;
static size_t written, offset;
static uint8_t content[32];
struct info {size_t device,identity,size;int64_t seconds;uint32_t nanos,mode;};
void cache_probe_reset(uint32_t value){mode=value;published=value!=0;closes=row=0;written=offset=0;}
uint32_t cache_probe_counts(void){return closes;}
static void name(uint8_t *out){memcpy(out,"fyproject0-",11);memset(out+11,'a',129);out[75]='-';memcpy(out+140,".fyc",5);}
uint32_t fairyc_private_linux_directory_open_v1(const uint8_t *path,size_t size,uint32_t *out){assert(size==2 && path[0]=='.');*out=1;return 0;}
uint32_t fairyc_private_linux_root_directory_open_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *out){(void)root;(void)leaf;(void)size;(void)out;assert(0);return 5;}
uint32_t fairyc_private_linux_root_file_close_v1(uint32_t file){if(file==9){++closes;}return 0;}
uint32_t fairyc_private_linux_root_file_create_v1(uint32_t root,uint32_t *out){assert(root==1);*out=9;written=0;return 0;}
uint32_t fairyc_private_linux_root_file_write_v1(uint32_t file,const uint8_t *data,size_t size,size_t *out){assert(file==9 && written+size<=32);size_t n=size>3 ? 3 : size;memcpy(content+written,data,n);written+=n;*out=n;return 0;}
uint32_t fairyc_private_linux_root_file_sync_v1(uint32_t file){assert(file==9);return 0;}
uint32_t fairyc_private_linux_root_file_publish_new_v1(uint32_t file,uint32_t root,const uint8_t *leaf,size_t size){uint8_t expected[145];name(expected);assert(file==9 && root==1 && size==145 && memcmp(leaf,expected,145)==0);if(published){return 17;}published=1;return 0;}
uint32_t fairyc_private_linux_root_file_open_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *out){assert(root==1 && size==145 && leaf[144]==0 && published);*out=8;offset=0;return 0;}
uint32_t fairyc_private_linux_root_file_info_v1(uint32_t file,void *out){assert(file==8);struct info value={71,123,11,-7,99,32768};memcpy(out,&value,sizeof value);return 0;}
uint32_t fairyc_private_linux_root_file_read_v1(uint32_t file,uint8_t *data,size_t size,size_t *out){assert(file==8);size_t n=11-offset;if(n>size){n=size;}memcpy(data,content+offset,n);offset+=n;*out=n;return 0;}
uint32_t fairyc_private_linux_directory_scan_open_v1(uint32_t root,size_t *out){assert(root==1);row=0;*out=77;return 0;}
uint32_t fairyc_private_linux_directory_scan_close_v1(size_t cursor){assert(cursor==0 || cursor==77);return 0;}
uint32_t fairyc_private_linux_directory_scan_next_v1(size_t cursor,uint8_t *data,size_t capacity,uint32_t *length,uint32_t *state,void *out){
    assert(cursor==77 && capacity==256);*state=*length=0;if(row==3){return 0;}
    if(row==0){name(data);*length=144;}else if(row==1){memcpy(data,".stage",6);name(data+6);memcpy(data+150,"-0",3);*length=152;}else{memcpy(data,"notes",6);*length=5;}++row;
    struct info value={71,123,11,-7,99,32768};memcpy(out,&value,sizeof value);*state=1;return 0;
}
uint32_t fairyc_private_linux_root_file_erase_if_v1(uint32_t root,const uint8_t *leaf,size_t size,const void *expected,uint32_t *removed,uint32_t *closed){
    assert(root==1 && size==145 && leaf[144]==0);const struct info *p=(const struct info *)expected;assert(p->device==71 && p->identity==123 && p->size==11 && p->seconds==-7 && p->nanos==99 && p->mode==32768);*removed=1;*closed=mode==2 ? 5 : 0;return 0;
}
`
