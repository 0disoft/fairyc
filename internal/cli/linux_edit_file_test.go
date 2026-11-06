package cli

import "testing"

func TestLinuxEditFilePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_edit_file"}, "linux_edit_file", `
@effects(filesystem) fn i32 main(){
 preserves_preparation_publication_and_consuming_close();
 bounds_writes_and_invalidates_preparation();
 rejects_metadata_and_preserves_native_failure_phases();
 preserves_partial_writes_and_never_retries_close();
 retains_publication_on_consuming_close_error();return 0;
}
`, linuxRootedFilePolicyStub+linuxEditFilePolicyStub)
}

const linuxEditFilePolicyStub = `
static uint32_t edit_mode,prepares,publications;
void edit_probe_mode(uint32_t value,uint32_t root_mode){edit_mode=value;prepares=publications=0;rooted_probe_mode(root_mode);}
uint32_t edit_probe_calls(void){return prepares*100+publications;}
uint32_t fairyc_private_linux_edit_file_prepare_v1(uint32_t file,uint32_t mode){assert(file==9 && mode==0640);++prepares;if(edit_mode==1){return 5;}return fairyc_private_linux_root_file_sync_v1(file);}
uint32_t fairyc_private_linux_edit_file_publish_v1(uint32_t file,uint32_t stage,uint32_t root,const uint8_t *leaf,size_t size,const void *metadata,uint32_t *cleanup){
 assert(file==9 && stage==2 && root==1 && size==7 && strcmp((const char *)leaf,"target")==0 && prepares>0 && syncs>0);
 const struct rooted_metadata *m=(const struct rooted_metadata *)metadata;assert(m->device==71 && m->identity==123 && m->size==3 && m->seconds==91 && m->nanos==42 && m->mode==(32768|0640));
 ++publications;*cleanup=edit_mode>=3?13u:0u;return edit_mode==2?116u:edit_mode==3?5u:0u;
}
`
