package cli

import "testing"

func TestLinuxRemoteCachePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_directory", "linux_package_directory", "linux_package_file", "linux_remote_fs", "sha256", "sha256_stream", "linux_remote_cache"}, "linux_remote_cache", `
@effects(filesystem) fn i32 main(){
    validates_identity_and_size_before_creating_cache_directories();
    stores_and_rehashes_hits_without_replacing_cache_bytes();
    rejects_corrupt_linked_aliased_and_leftover_cache_entries();
    preserves_collision_and_close_results_without_replaying_publication();return 0;
}
`, linuxRemoteFileSystemStub+linuxRemoteCacheStub)
}

const linuxRemoteCacheStub = `
void remote_cache_probe_case(uint32_t value){remote_case=value;remote_infos=0;}
uint32_t remote_cache_probe_attempts(void){return remote_anonymous*100+remote_publications;}
void remote_cache_probe_alter(uint32_t value){
    for(size_t i=1;i<remote_node_count;++i){struct remote_node *node=&remote_nodes[i];size_t size=strlen(node->name);
        if(node->active && size==68 && strcmp(node->name+64,".zip")==0){
            if(value==1){assert(node->size>0);node->data[0]^=1;}else if(value==2){node->links=2;}else if(value==3){node->kind=40960;}else if(value==4){assert(node->name[0]=='b');node->name[0]='B';}return;
        }
    }assert(0);
}
`
