package cli

import "testing"

func TestLinuxPackageFilePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_package_file"}, "linux_package_file", `
@effects(filesystem) fn i32 main(){checks_package_metadata_without_weakening_generic_files();preserves_exact_conditional_removal_and_partial_errors();return 0;}
`, linuxRootedFilePolicyStub+linuxPackageFilePolicyStub)
}

const linuxPackageFilePolicyStub = `
static uint32_t package_case,package_infos,package_erases;
void package_file_probe_mode(uint32_t value){package_case=value;package_infos=package_erases=0;rooted_probe_mode(0);}
uint32_t package_file_probe_counts(void){return package_infos*100+package_erases;}
uint32_t fairyc_private_linux_package_file_info_v1(uint32_t file,void *output){
    assert(file==8);++package_infos;memset(output,0,sizeof(struct rooted_metadata));
    if(package_case==1){return 31;}if(package_case==2){return 5;}
    struct rooted_metadata value={71,123,3,91,42,32768};
    if(package_case==3){value.identity=0;}if(package_case==4){value.nanos=1000000000;}if(package_case==5){value.mode=16384;}
    memcpy(output,&value,sizeof value);return 0;
}
uint32_t fairyc_private_linux_root_file_erase_if_v1(uint32_t root,const uint8_t *leaf,size_t size,const void *expected,uint32_t *removed,uint32_t *closed){
    assert(root==1 && strcmp((const char *)leaf,"input")==0 && size==6);const struct rooted_metadata *info=(const struct rooted_metadata *)expected;
    assert(info->device==71 && info->identity==123 && info->size==3 && info->seconds==91 && info->nanos==42 && info->mode==32768);++package_erases;*closed=0;*removed=package_case==6?0:1;
    if(package_case==7){*closed=4;}if(package_case==8){*removed=2;}if(package_case==9){*closed=4;return 5;}return 0;
}
`
