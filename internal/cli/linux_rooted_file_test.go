package cli

import "testing"

func TestLinuxRootedFilePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file"}, "linux_rooted_file", `
@effects(filesystem) fn i32 main() {
    validates_leaves_before_native_calls();
    preserves_metadata_binary_reads_and_consuming_close();
    separates_missing_io_and_malformed_native_results();
    writes_partial_chunks_syncs_before_exclusive_publication_and_freezes();
    reports_partial_progress_and_cleans_up_failed_stages(); return 0;
}
`, linuxRootedFilePolicyStub)
}

const linuxRootedFilePolicyStub = `
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <assert.h>
static uint32_t scenario,opens,closes,writes,syncs,links;
static size_t written;
void rooted_probe_mode(uint32_t value) {scenario=value;opens=closes=writes=syncs=links=0;written=0;}
uint32_t rooted_probe_counts(void) {return opens*100+closes;}
size_t rooted_probe_written(void) {return written;}
uint32_t rooted_probe_steps(void) {return writes*100+syncs*10+links;}
uint32_t fairyc_private_linux_directory_open_v1(const uint8_t *path,size_t size,uint32_t *out) {
    assert(size==6 && strcmp((const char *)path,"/work")==0);++opens;*out=1;return 0;
}
uint32_t fairyc_private_linux_root_directory_open_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *out) {
    assert(root==1 && size==6 && strcmp((const char *)leaf,"child")==0);++opens;*out=2;return 0;
}
uint32_t fairyc_private_linux_root_file_open_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *out) {
    assert(root==1 && size==6 && strcmp((const char *)leaf,"input")==0);++opens;*out=0;
    if(scenario==2) {return 2;}*out=8;return 0;
}
struct rooted_metadata {size_t device,identity,size;int64_t seconds;uint32_t nanos,mode;};
uint32_t fairyc_private_linux_root_file_info_v1(uint32_t file,void *output) {
    assert(file==8);struct rooted_metadata value={71,123,3,91,42,scenario==5 ? 16384 : 32768};memcpy(output,&value,sizeof value);return 0;
}
uint32_t fairyc_private_linux_root_file_read_v1(uint32_t file,uint8_t *data,size_t size,size_t *copied) {
    assert(file==8 && size==4);*copied=0;if(scenario==4) {return 5;}
    memcpy(data,"a\0\xff",3);*copied=scenario==3 ? 5 : 3;return 0;
}
uint32_t fairyc_private_linux_root_file_close_v1(uint32_t handle) {
    if(handle==0) {return 0;}assert(handle==1 || handle==2 || handle==8 || handle==9);++closes;return scenario==1 ? 4 : 0;
}
uint32_t fairyc_private_linux_root_file_create_v1(uint32_t root,uint32_t *out) {
    assert(root==1);++opens;*out=0;if(scenario==10) {return 95;}*out=9;return 0;
}
uint32_t fairyc_private_linux_root_file_write_v1(uint32_t file,const uint8_t *data,size_t size,size_t *copied) {
    assert(file==9 && data && size>0 && size<=65536);++writes;*copied=0;
    if(scenario==11) {return 0;}if(scenario==12 && writes==2) {return 5;}
    if(scenario==15) {*copied=size+1;return 0;}
    size_t count=writes==1 && size>7 ? 7 : size;written+=count;*copied=count;return 0;
}
uint32_t fairyc_private_linux_root_file_sync_v1(uint32_t file) {
    assert(file==9);++syncs;return scenario==13 ? 5 : 0;
}
uint32_t fairyc_private_linux_root_file_publish_new_v1(uint32_t file,uint32_t root,const uint8_t *leaf,size_t size) {
    assert(file==9 && root==1 && size==4 && strcmp((const char *)leaf,"new")==0 && syncs==1);++links;
    return scenario==14 ? 17 : scenario==16 ? 5 : 0;
}
`
