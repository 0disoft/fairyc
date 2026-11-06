package cli

import "testing"

func TestLinuxPackageDirectoryPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_package_directory"}, "linux_package_directory", `
@effects(filesystem) fn i32 main(){
    consuming_close_preserves_both_errors_and_never_retries();
    validates_input_and_transfers_owned_handles();
    preserves_native_errors_and_reclaims_partial_or_invalid_outputs();
    publication_and_removal_freeze_the_old_name();
    failed_publication_retains_cleanup_authority_without_retry();
    duplicates_owned_directory_without_losing_or_closing_borrowed_handles();return 0;
}
`, linuxRootedFilePolicyStub+linuxPackageDirectoryPolicyStub)
}

const linuxPackageDirectoryPolicyStub = `
#include <stdlib.h>
static uint32_t package_mode,package_acquires,package_publishes,package_erases;
void package_directory_probe_mode(uint32_t value){const char *path=getenv("PATH");assert(path==NULL || path[0]==0);package_mode=value;package_acquires=package_publishes=package_erases=0;rooted_probe_mode(value>=19 && value<=21?value-2:0);}
uint32_t package_directory_probe_calls(void){return package_acquires*100+package_publishes*10+package_erases;}
static uint32_t package_acquire(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *child,uint32_t *parent,uint32_t *cleanup){
    assert(root==1 && strcmp((const char *)leaf,"stage")==0 && size==6);++package_acquires;*child=2;*parent=8;*cleanup=0;
    if(package_mode==1){*child=*parent=0;return 17;}
    if(package_mode==2){return 5;}
    if(package_mode==3){*child=0;}
    if(package_mode==4){*parent=0;}
    if(package_mode==5){*parent=2;}
    if(package_mode==6){*child=UINT32_MAX;}
    if(package_mode==7){*cleanup=5;}
    if(package_mode==8){*cleanup=4;scenario=1;return 5;}
    if(package_mode==11){*child=1;}
    if(package_mode==12){*parent=1;}
    if(package_mode==13){*child=*parent=1;}
    return 0;
}
uint32_t fairyc_private_linux_package_directory_create_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *child,uint32_t *parent,uint32_t *cleanup){return package_acquire(root,leaf,size,child,parent,cleanup);}
uint32_t fairyc_private_linux_package_directory_edit_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *child,uint32_t *parent,uint32_t *cleanup){return package_acquire(root,leaf,size,child,parent,cleanup);}
uint32_t fairyc_private_linux_package_directory_clone_v1(uint32_t root,uint32_t *output,uint32_t *cleanup){
    assert(root==2);*output=9;*cleanup=0;if(package_mode==14){*output=2;}if(package_mode==15){*output=8;}if(package_mode==16){*output=0;}if(package_mode==17){*output=UINT32_MAX;}if(package_mode==18){*cleanup=4;scenario=1;return 5;}return 0;
}
uint32_t fairyc_private_linux_package_directory_publish_v1(uint32_t child,uint32_t parent,const uint8_t *from,size_t from_size,uint32_t root,const uint8_t *leaf,size_t size){
    assert(child==2 && parent==8 && strcmp((const char *)from,"stage")==0 && from_size==6 && root==1 && strcmp((const char *)leaf,"package")==0 && size==8);++package_publishes;return package_mode==9?4:0;
}
uint32_t fairyc_private_linux_package_directory_erase_v1(uint32_t child,uint32_t parent,const uint8_t *leaf,size_t size){
    assert(child==2 && parent==8 && strcmp((const char *)leaf,"stage")==0 && size==6);++package_erases;return package_mode==10?39:0;
}
`
