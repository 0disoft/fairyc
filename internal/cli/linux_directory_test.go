package cli

import "testing"

func TestLinuxDirectoryPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_directory"}, "linux_directory", `
@effects(filesystem) fn i32 main() {
    owns_names_metadata_and_restarts_each_scan();
    preserves_primary_failure_and_closes_cursor_once();
    enforces_exact_item_and_cumulative_work_limits(); return 0;
}
`, linuxRootedFilePolicyStub+linuxDirectoryPolicyStub)
}

const linuxDirectoryPolicyStub = `
#include <stdio.h>
static uint32_t scan_case,scan_opens,scan_reads,scan_closes,scan_row;
void directory_probe_mode(uint32_t value) {scan_case=value;scan_opens=scan_reads=scan_closes=scan_row=0;}
uint32_t directory_probe_counts(void) {return scan_opens*1000+scan_reads*10+scan_closes;}
uint32_t fairyc_private_linux_directory_scan_open_v1(uint32_t root,size_t *output) {
    assert(root==1);++scan_opens;scan_row=0;*output=0;if(scan_case==4) {return 12;}*output=100+scan_opens;return 0;
}
uint32_t fairyc_private_linux_directory_scan_next_v1(size_t cursor,uint8_t *data,size_t capacity,uint32_t *length,uint32_t *state,void *output) {
    assert(cursor==100+scan_opens && capacity==256);++scan_reads;*length=*state=0;
    memset(output,0,sizeof(struct rooted_metadata));
    if(scan_case==1 || (scan_case==3 && scan_row==3)) {return 5;}
    if(scan_case==15) {return 0;}
    if(scan_case==17) {*length=1;return 0;}
    const char *names[]={".","..","alpha","\xed\x95\x9c","link","gone"};
    if(scan_case==12) {memcpy(data,"gone",5);*state=2;*length=4;return 0;}
    if(scan_case==13 || scan_case==14) {
        if(scan_row==(scan_case==13 ? 4096u : 4097u)) {return 0;}
        int size=snprintf((char *)data,256,"entry%04u",scan_row);assert(size==9);*length=9;*state=1;
    } else {
        if(scan_row==6) {return 0;}
        const char *name=names[scan_row];size_t size=strlen(name);memcpy(data,name,size+1);*length=(uint32_t)size;*state=scan_row<2 || scan_row==5 ? 2 : 1;
    }
    struct rooted_metadata info={71,123+scan_row,3,-2,42,scan_row==4 ? 40960u : 32768u};memcpy(output,&info,sizeof info);++scan_row;
    if(scan_case==5) {*state=3;}
    if(scan_case==6) {*length=256;}
    if(scan_case==7 || scan_case==16) {data[0]=255;*length=1;*state=scan_case==7 ? 1 : 2;}
    if(scan_case==8) {memcpy(data,"a/b",4);*length=3;*state=1;}
    if(scan_case==9) {data[0]='x';*length=1;*state=1;((struct rooted_metadata *)output)->nanos=1000000000;}
    if(scan_case==10) {*length=0;*state=1;}
    if(scan_case==11) {*state=1;}
    if(scan_case==18) {data[0]='x';*length=1;*state=1;((struct rooted_metadata *)output)->mode=0;}
    return 0;
}
uint32_t fairyc_private_linux_directory_scan_close_v1(size_t cursor) {
    if(cursor==0) {return 0;}assert(cursor==100+scan_opens);++scan_closes;return scan_case==2 || scan_case==3 ? 4 : 0;
}
`
