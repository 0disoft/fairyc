package cli

import "testing"

func TestLinuxRetentionPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_directory", "linux_retention"}, "linux_retention", `
@effects(filesystem) fn i32 main() {
    evicts_oldest_and_oversize_but_preserves_selected_and_unrelated_files();
    reports_stale_busy_and_partial_mutation_without_claiming_success();
    validates_before_mutation_and_enforces_shared_work_budget();
    sorts_full_inventory_and_refuses_unprovable_protected_bounds(); return 0;
}
`, linuxRootedFilePolicyStub+linuxRetentionPolicyStub)
}

const linuxRetentionPolicyStub = `
static uint32_t retention_case,retention_row,retention_mask,retention_opens,retention_erases;
void retention_probe_mode(uint32_t value) {retention_case=value;retention_row=retention_mask=retention_opens=retention_erases=0;}
uint32_t retention_probe_mask(void) {return retention_mask;}
uint32_t retention_probe_counts(void) {return retention_opens*100+retention_erases;}
void retention_probe_name(uint8_t *data,uint32_t index) {
    memcpy(data,"fyproject0-",11);memset(data+11,'0',64);data[75]='-';memset(data+76,(int)('a'+index),64);memcpy(data+140,".fyc",5);
}
uint32_t fairyc_private_linux_directory_scan_open_v1(uint32_t root,size_t *output) {assert(root==1);++retention_opens;retention_row=0;*output=77;return 0;}
uint32_t fairyc_private_linux_directory_scan_next_v1(size_t cursor,uint8_t *data,size_t capacity,uint32_t *length,uint32_t *state,void *output) {
    assert(cursor==77 && capacity==256);*length=*state=0;
    if(retention_case==11) {
        if(retention_row==4096) {return 0;}
        uint32_t index=retention_row++;retention_probe_name(data,0);const char *digits="0123456789abcdef";
        data[137]=(uint8_t)digits[index/256];data[138]=(uint8_t)digits[index/16%16];data[139]=(uint8_t)digits[index%16];
        struct rooted_metadata info={71,100+index,8,4096-index,0,32768};memcpy(output,&info,sizeof info);*length=144;*state=1;return 0;
    }
    if(retention_case==12) {
        if(retention_row==2) {return 0;}++retention_row;retention_probe_name(data,1);
        struct rooted_metadata info={71,101,8,0,0,32768};memcpy(output,&info,sizeof info);*length=144;*state=1;return 0;
    }
    if(retention_row==6) {return 0;}
    uint32_t index=retention_row++;struct rooted_metadata info={71,100+index,8,30,0,32768};
    if(index==1) {info.size=12;info.seconds=10;}if(index==2) {info.size=10;info.seconds=20;}
    if((retention_case==6 || retention_case==4) && index==1) {info.size=1000;}
    if(retention_case==4 && index==2) {info.size=8;}
    retention_probe_name(data,index);*length=144;*state=1;
    if(index==3) {memcpy(data,"notes",6);*length=5;info.size=999;}
    if(index==4) {info.mode=40960;}if(index==5) {memcpy(data,".stage-orphan",14);*length=13;}
    if(retention_case==8) {info.mode=40960;}
    if(retention_case==9) {info.seconds=5;}
    if(retention_case==10) {info.seconds=5;info.nanos=2-index%3;}
    memcpy(output,&info,sizeof info);return 0;
}
uint32_t fairyc_private_linux_directory_scan_close_v1(size_t cursor) {assert(cursor==0 || cursor==77);return 0;}
uint32_t fairyc_private_linux_root_file_erase_if_v1(uint32_t root,const uint8_t *leaf,size_t size,const void *expected,uint32_t *removed,uint32_t *closed) {
    assert(root==1 && size==145 && leaf[144]==0);++retention_erases;*removed=*closed=0;
    if(retention_case==11) {
        const struct rooted_metadata *info=(const struct rooted_metadata *)expected;assert(info->identity==100+4096-retention_erases);*removed=1;return 0;
    }
    uint32_t index=leaf[76]-'a';assert(index<3);const struct rooted_metadata *info=(const struct rooted_metadata *)expected;assert(info->identity==100+index);
    if(retention_case==1) {return 11;}if(retention_case==2) {return 0;}if(retention_case==5) {*removed=2;return 0;}
    if(retention_case==4 && retention_erases==2) {return 5;}
    retention_mask|=1u<<index;*removed=1;if(retention_case==3) {*closed=4;}return 0;
}
`
