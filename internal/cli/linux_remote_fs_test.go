package cli

import "testing"

func TestLinuxRemoteFileSystemPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_directory", "linux_package_directory", "linux_package_file", "linux_remote_fs"}, "linux_remote_fs", `
@effects(filesystem) fn i32 main(){
    reads_exact_binary_files_and_refuses_aliases_links_and_changes();
    writes_syncs_and_publishes_complete_private_trees();
    rejects_paths_before_mutation_and_bounds_stage_cleanup();
    failures_release_anonymous_files_without_publishing_names();return 0;
}
`, linuxRemoteFileSystemStub)
}

// Independent bounded filesystem: handles own nodes, scans own cursors, and
// publication/removal update a namespace. Uncertain closes consume their handle.
const linuxRemoteFileSystemStub = `
#include <assert.h>
#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
struct remote_node {int active,parent,synced;uint32_t kind,links;char name[256];uint8_t data[256];size_t size;};
struct remote_handle {int active,node;size_t offset;};
struct remote_cursor {int active,node;size_t next;};
struct remote_meta {size_t device,identity,size;int64_t seconds;uint32_t nanos,mode;};
static struct remote_node remote_nodes[128];
static struct remote_handle remote_handles[256];
static struct remote_cursor remote_cursors[64];
static uint32_t remote_case,remote_live,remote_mutations,remote_reads,remote_infos,remote_anonymous,remote_publications;
static size_t remote_node_count,remote_handle_count,remote_cursor_count;
static int remote_find(int parent,const char *name){for(size_t i=1;i<remote_node_count;++i){if(remote_nodes[i].active && remote_nodes[i].parent==parent && strcmp(remote_nodes[i].name,name)==0){return (int)i;}}return -1;}
static int remote_add(int parent,const char *name,uint32_t kind){
    assert(remote_node_count<128 && strlen(name)<256);size_t id=remote_node_count++;struct remote_node *node=&remote_nodes[id];
    memset(node,0,sizeof *node);node->active=1;node->parent=parent;node->kind=kind;node->links=parent<0?0:1;memcpy(node->name,name,strlen(name)+1);return (int)id;
}
static uint32_t remote_handle(int node){assert(remote_handle_count<256);size_t id=remote_handle_count++;remote_handles[id].active=1;remote_handles[id].node=node;remote_handles[id].offset=0;++remote_live;return (uint32_t)id;}
static int remote_held(uint32_t handle){assert(handle>0 && handle<remote_handle_count && remote_handles[handle].active);return remote_handles[handle].node;}
static struct remote_meta remote_metadata(int id){
    struct remote_meta info={71,100+(size_t)id,remote_nodes[id].size,91,42,remote_nodes[id].kind};if(remote_case==3 && strcmp(remote_nodes[id].name,"input")==0){info.size=3;}return info;
}
void remote_fs_probe_mode(uint32_t value){
    assert(remote_live==0);const char *path=getenv("PATH");assert(path==NULL || path[0]==0);
    memset(remote_nodes,0,sizeof remote_nodes);memset(remote_handles,0,sizeof remote_handles);memset(remote_cursors,0,sizeof remote_cursors);
    remote_case=value;remote_mutations=remote_reads=remote_infos=remote_anonymous=remote_publications=0;remote_node_count=1;remote_handle_count=1;remote_cursor_count=0;
    remote_nodes[0].active=1;remote_nodes[0].parent=-1;remote_nodes[0].kind=16384;
    if((value>=1 && value<=6) || value==11 || value==12){int file=remote_add(0,"input",value==12?40960u:32768u);memcpy(remote_nodes[file].data,"a\0\xffx",4);remote_nodes[file].size=value==3?4:3;
        if(value==4){remote_nodes[file].links=2;}if(value==11){(void)remote_add(0,"INPUT",32768);}
        int folder=remote_add(0,"tree",16384);file=remote_add(folder,"item",32768);memcpy(remote_nodes[file].data,"a\0\xff",3);remote_nodes[file].size=3;
    }
}
uint32_t remote_fs_probe_live(void){return remote_live;}
uint32_t remote_fs_probe_mutations(void){return remote_mutations;}
uint32_t remote_fs_probe_reads(void){return remote_reads;}
uint32_t fairyc_private_linux_directory_open_v1(const uint8_t *path,size_t size,uint32_t *output){assert(size==6 && strcmp((const char *)path,"/work")==0);*output=remote_handle(0);return 0;}
uint32_t fairyc_private_linux_root_file_close_v1(uint32_t handle){
    if(handle==0){return 0;}int node=remote_held(handle);remote_handles[handle].active=0;assert(remote_live>0);--remote_live;
    if(remote_nodes[node].parent<0 && node!=0){remote_nodes[node].active=0;}return remote_case==6 && remote_nodes[node].kind==32768?4:0;
}
uint32_t fairyc_private_linux_root_directory_open_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *output){
    assert(size==strlen((const char *)leaf)+1);int node=remote_find(remote_held(root),(const char *)leaf);*output=0;if(node<0){return 2;}if(remote_nodes[node].kind!=16384){return 20;}*output=remote_handle(node);return 0;
}
uint32_t fairyc_private_linux_package_directory_create_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *child,uint32_t *parent,uint32_t *closed){
    assert(size==strlen((const char *)leaf)+1);int held=remote_held(root);*child=*parent=*closed=0;if(remote_find(held,(const char *)leaf)>=0){return 17;}
    int node=remote_add(held,(const char *)leaf,16384);*child=remote_handle(node);*parent=remote_handle(held);++remote_mutations;return 0;
}
uint32_t fairyc_private_linux_package_directory_edit_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *child,uint32_t *parent,uint32_t *closed){
    assert(size==strlen((const char *)leaf)+1);int held=remote_held(root),node=remote_find(held,(const char *)leaf);*child=*parent=*closed=0;
    if(node<0){return 2;}if(remote_nodes[node].kind!=16384){return 20;}*child=remote_handle(node);*parent=remote_handle(held);return 0;
}
uint32_t fairyc_private_linux_package_directory_clone_v1(uint32_t root,uint32_t *output,uint32_t *closed){*closed=0;int node=remote_held(root);assert(remote_nodes[node].kind==16384);*output=remote_handle(node);return 0;}
uint32_t fairyc_private_linux_package_directory_publish_v1(uint32_t child,uint32_t parent,const uint8_t *from,size_t from_size,uint32_t root,const uint8_t *leaf,size_t size){
    assert(from_size==strlen((const char *)from)+1 && size==strlen((const char *)leaf)+1);int node=remote_held(child),destination=remote_held(root);
    assert(remote_find(remote_held(parent),(const char *)from)==node);if(remote_find(destination,(const char *)leaf)>=0){return 17;}
    remote_nodes[node].parent=destination;memcpy(remote_nodes[node].name,leaf,size);++remote_mutations;return 0;
}
uint32_t fairyc_private_linux_package_directory_erase_v1(uint32_t child,uint32_t parent,const uint8_t *leaf,size_t size){
    assert(size==strlen((const char *)leaf)+1);int node=remote_held(child);if(remote_find(remote_held(parent),(const char *)leaf)!=node){return 116;}
    for(size_t i=1;i<remote_node_count;++i){if(remote_nodes[i].active && remote_nodes[i].parent==node){return 39;}}
    remote_nodes[node].active=0;++remote_mutations;return 0;
}
uint32_t fairyc_private_linux_directory_scan_open_v1(uint32_t root,size_t *output){assert(remote_cursor_count<64);size_t id=remote_cursor_count++;remote_cursors[id].active=1;remote_cursors[id].node=remote_held(root);remote_cursors[id].next=1;++remote_live;*output=1000+id;return 0;}
uint32_t fairyc_private_linux_directory_scan_next_v1(size_t raw,uint8_t *name,size_t capacity,uint32_t *length,uint32_t *state,void *metadata){
    assert(raw>=1000 && raw-1000<remote_cursor_count && capacity==256);struct remote_cursor *cursor=&remote_cursors[raw-1000];assert(cursor->active);*length=*state=0;
    while(cursor->next<remote_node_count){size_t id=cursor->next++;if(remote_nodes[id].active && remote_nodes[id].parent==cursor->node){size_t size=strlen(remote_nodes[id].name);memcpy(name,remote_nodes[id].name,size+1);*length=(uint32_t)size;*state=1;struct remote_meta info=remote_metadata((int)id);memcpy(metadata,&info,sizeof info);break;}}return 0;
}
uint32_t fairyc_private_linux_directory_scan_close_v1(size_t raw){if(raw==0){return 0;}assert(raw>=1000 && raw-1000<remote_cursor_count && remote_cursors[raw-1000].active);remote_cursors[raw-1000].active=0;--remote_live;return 0;}
uint32_t fairyc_private_linux_root_file_open_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *output){
    assert(size==strlen((const char *)leaf)+1);int node=remote_find(remote_held(root),(const char *)leaf);*output=0;if(node<0){return 2;}if(remote_nodes[node].kind!=32768){return 22;}*output=remote_handle(node);return 0;
}
uint32_t fairyc_private_linux_package_file_info_v1(uint32_t handle,void *output){int node=remote_held(handle);++remote_infos;memset(output,0,sizeof(struct remote_meta));if(remote_nodes[node].links!=1){return 31;}struct remote_meta info=remote_metadata(node);if(remote_case==5 && remote_infos>1){++info.seconds;}memcpy(output,&info,sizeof info);return 0;}
uint32_t fairyc_private_linux_root_file_read_v1(uint32_t handle,uint8_t *data,size_t size,size_t *copied){
    int node=remote_held(handle);++remote_reads;size_t offset=remote_handles[handle].offset;size_t available=remote_nodes[node].size;
    if(remote_case==2 && strcmp(remote_nodes[node].name,"input")==0){available=2;}assert(offset<=available);size_t count=available-offset;if(count>size){count=size;}memcpy(data,remote_nodes[node].data+offset,count);remote_handles[handle].offset+=count;*copied=count;return 0;
}
uint32_t fairyc_private_linux_root_file_create_v1(uint32_t root,uint32_t *output){(void)remote_held(root);++remote_anonymous;int node=remote_add(-1,"",32768);*output=remote_handle(node);return 0;}
uint32_t fairyc_private_linux_root_file_write_v1(uint32_t handle,const uint8_t *data,size_t size,size_t *copied){
    int node=remote_held(handle);assert(remote_nodes[node].parent<0 && size>0);*copied=0;if(remote_case==9){return 0;}size_t count=size>2?2:size;assert(remote_nodes[node].size+count<=256);memcpy(remote_nodes[node].data+remote_nodes[node].size,data,count);remote_nodes[node].size+=count;*copied=count;return 0;
}
uint32_t fairyc_private_linux_root_file_sync_v1(uint32_t handle){int node=remote_held(handle);if(remote_case==7){return 5;}remote_nodes[node].synced=1;return 0;}
uint32_t fairyc_private_linux_root_file_publish_new_v1(uint32_t handle,uint32_t root,const uint8_t *leaf,size_t size){
    assert(size==strlen((const char *)leaf)+1);int node=remote_held(handle),parent=remote_held(root);assert(remote_nodes[node].parent<0 && remote_nodes[node].links==0 && remote_nodes[node].synced);
    ++remote_publications;
    if((remote_case==14 || remote_case==15) && remote_find(parent,(const char *)leaf)<0){int competing=remote_add(parent,(const char *)leaf,32768);remote_nodes[competing].size=remote_nodes[node].size;memcpy(remote_nodes[competing].data,remote_nodes[node].data,remote_nodes[node].size);if(remote_case==15){assert(remote_nodes[competing].size>0);remote_nodes[competing].data[0]^=1;}++remote_mutations;return 17;}
    if(remote_case==8 || remote_find(parent,(const char *)leaf)>=0){return 17;}remote_nodes[node].parent=parent;remote_nodes[node].links=1;memcpy(remote_nodes[node].name,leaf,size);++remote_mutations;return 0;
}
uint32_t fairyc_private_linux_root_file_erase_if_v1(uint32_t root,const uint8_t *leaf,size_t size,const void *expected,uint32_t *removed,uint32_t *closed){
    assert(size==strlen((const char *)leaf)+1);*removed=*closed=0;int node=remote_find(remote_held(root),(const char *)leaf);if(node<0){return 0;}struct remote_meta actual=remote_metadata(node);const struct remote_meta *wanted=(const struct remote_meta *)expected;
    if(remote_case==13 || remote_nodes[node].links!=1 || actual.device!=wanted->device || actual.identity!=wanted->identity || actual.size!=wanted->size || actual.seconds!=wanted->seconds || actual.nanos!=wanted->nanos || actual.mode!=wanted->mode){return 0;}
    remote_nodes[node].active=0;*removed=1;++remote_mutations;return 0;
}
`
