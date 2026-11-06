package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxEditApplyPolicy(t *testing.T) {
	// Keep the real host/file/reader/application owners. Only the expensive
	// source-check/snapshot ports are controlled here; full command acceptance
	// must separately exercise those owners with current real source.
	optionSource, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "project_edit_options.fy"))
	if err != nil {
		t.Fatal(err)
	}
	support := "struct remote_meta;\nstatic struct remote_meta remote_metadata(int id);\n" + linuxRemoteFileSystemStub
	for _, name := range []string{"metadata", "close", "directory", "acquire", "erase", "read", "create", "write"} {
		declaration := map[string]string{
			"metadata":  "static struct remote_meta remote_metadata(",
			"close":     "uint32_t fairyc_private_linux_root_file_close_v1(",
			"directory": "uint32_t fairyc_private_linux_root_directory_open_v1(",
			"acquire":   "uint32_t fairyc_private_linux_package_directory_create_v1(",
			"erase":     "uint32_t fairyc_private_linux_package_directory_erase_v1(",
			"read":      "uint32_t fairyc_private_linux_root_file_read_v1(",
			"create":    "uint32_t fairyc_private_linux_root_file_create_v1(",
			"write":     "uint32_t fairyc_private_linux_root_file_write_v1(",
		}[name]
		// The forward metadata declaration has a different spelling (int id).
		if name == "metadata" {
			declaration += "int id){"
		}
		if strings.Count(support, declaration) != 1 {
			t.Fatal("ambiguous namespace-model boundary", name)
		}
		replacement := "uint32_t edit_real_" + name + "("
		if name == "metadata" {
			replacement = "static struct remote_meta edit_real_metadata(int id){"
		}
		support = strings.Replace(support, declaration, replacement, 1)
	}
	runLinuxCandidateFixtureSources(t, []string{"linux_command", "linux_rooted_file", "linux_directory", "linux_package_directory", "linux_edit_fs", "linux_edit_file", "linux_edit_apply", "project_manifest_values", "project_edit_options"}, "linux_edit_apply", `
@effects(filesystem,environment) fn i32 main(){
 publishes_once_with_private_staging_and_preserved_old_alias();
 refuses_lock_stage_and_anonymous_creation_failures();
 preserves_source_on_write_prepare_and_validation_failures();
 refuses_byte_metadata_drift_and_never_replays_publication();
 retains_known_publication_through_every_cleanup_error();return 0;
}
`, support+linuxEditApplyModel, map[string]string{
		"project_edit_preview":  linuxEditApplyPreviewPort,
		"project_edit_snapshot": linuxEditApplySnapshotPort,
		"project_edit_options": string(optionSource) + `
pub fn Options test_options(){return Options{.uid=bytes::empty(),.symbol=bytes::empty(),.body=bytes::empty(),.expected=bytes::empty(),.apply=true};}
`,
	})
}

const linuxEditApplyPreviewPort = `module project_edit_preview;
import std::bytes as bytes;
pub resource Preview{bool valid;cleanup{}}
pub struct Problem{u32 stage;u32 code;}
@link_name("edit_apply_recheck") unsafe extern C fn u32 native_recheck();
pub fn Preview test_proposal(){return Preview{.valid=true};}
pub fn ref u8[] path(ref Preview p){assert(p.valid);return b"src/input";}
pub fn ref u8[] before(ref Preview p){assert(p.valid);return b"a\0\xff";}
pub fn ref u8[] after(ref Preview p){assert(p.valid);return b"n\0\xfe";}
pub fn ref u8[] fingerprint(ref Preview p){assert(p.valid);return b"expected";}
@effects(filesystem,environment) pub fn Result<bool,Problem> recheck(ref u8[] root,ref u8[] uid,ref u8[] symbol,ref u8[] expected,ref Preview p){
 assert(bytes::equal(root,b"/work") && p.valid);discard uid;discard symbol;discard expected;
 return if(unsafe{native_recheck()}==0){.ok(true)}else{.err(Problem{.stage=7,.code=5})};
}
`

const linuxEditApplySnapshotPort = `module project_edit_snapshot;
pub resource Snapshot{bool current;cleanup{}}
@link_name("edit_apply_capture") unsafe extern C fn u32 native_capture();
@effects(filesystem) pub fn Result<Snapshot,u32> capture_path(ref u8[] path){discard path;let code=unsafe{native_capture()};if(code==1){return .err(5);}return .ok(Snapshot{.current=code==0});}
pub fn ref u8[] digest(ref Snapshot p){if(p.current){return b"expected";}return b"different";}
`

const linuxEditApplyModel = `
#include <stdio.h>
static uint32_t edit_mode,edit_rechecks,edit_captures,edit_prepares,edit_renames,edit_creates,edit_writes,edit_drift;
static uint32_t edit_parent_handle;
static int edit_source,edit_alias,edit_stage,edit_file;
static uint32_t edit_lease_role[256];
static int edit_stage_name(const char *name){const char *prefix=".fairyc-edit-stage-";return strncmp(name,prefix,strlen(prefix))==0;}
void edit_apply_probe(uint32_t value){
 remote_fs_probe_mode(0);edit_mode=value;edit_rechecks=edit_captures=edit_prepares=edit_renames=edit_creates=edit_writes=edit_drift=0;
 edit_parent_handle=0;edit_stage=edit_file=-1;memset(edit_lease_role,0,sizeof edit_lease_role);
 int src=remote_add(0,"src",16384);edit_source=remote_add(src,"input",32768);edit_alias=remote_add(src,"alias",32768);
 memcpy(remote_nodes[edit_source].data,"a\0\xff",3);remote_nodes[edit_source].size=3;remote_nodes[edit_source].links=2;
 memcpy(remote_nodes[edit_alias].data,"a\0\xff",3);remote_nodes[edit_alias].size=3;remote_nodes[edit_alias].links=2;
 if(value==1){(void)remote_add(0,".fairyc-edit.lock",16384);}
 if(value==2 || value==3){for(size_t i=0;i<(value==2?15u:16u);++i){char name[64];int n=snprintf(name,sizeof name,".fairyc-edit-stage-%zu.tmp",i);assert(n>0 && (size_t)n<sizeof name);int directory=remote_add(src,name,16384);(void)remote_add(directory,"foreign",32768);}}
 if(value==26){remote_nodes[edit_source].data[0]=remote_nodes[edit_alias].data[0]='x';}
}
static struct remote_meta remote_metadata(int id){
 struct remote_meta p=edit_real_metadata(id);if(remote_nodes[id].kind==32768){p.mode|=0640;}
 if(id==edit_source || id==edit_alias){p.identity=100+(size_t)edit_source;p.seconds+=(int64_t)edit_drift;if(edit_mode==13){p.mode=0100444;}}
 return p;
}
uint32_t fairyc_private_linux_root_file_info_v1(uint32_t handle,void *output){struct remote_meta p=remote_metadata(remote_held(handle));memcpy(output,&p,sizeof p);return 0;}
uint32_t fairyc_private_linux_root_directory_open_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *output){
 uint32_t code=edit_real_directory(root,leaf,size,output);if(code==0 && strcmp((const char *)leaf,"src")==0){edit_parent_handle=*output;}return code;
}
uint32_t fairyc_private_linux_package_directory_create_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *child,uint32_t *parent,uint32_t *cleanup){
 ++edit_creates;*child=*parent=*cleanup=0;int stage=edit_stage_name((const char *)leaf);
 if((stage && edit_mode==24) || (!stage && edit_mode==25)){*cleanup=66;return 17;}
 if(stage && edit_mode==4){return 4;}
 uint32_t code=edit_real_acquire(root,leaf,size,child,parent,cleanup);
 if(code==0){edit_lease_role[*parent]=stage?1u:3u;edit_lease_role[*child]=stage?2u:4u;if(stage){edit_stage=remote_held(*child);}}
 return code;
}
uint32_t fairyc_private_linux_root_file_create_v1(uint32_t root,uint32_t *output){
 assert(edit_stage>=0 && remote_held(root)==edit_stage);*output=0;if(edit_mode==5){return 95;}
 uint32_t code=edit_real_create(root,output);if(code==0){edit_file=remote_held(*output);}return code;
}
uint32_t fairyc_private_linux_root_file_write_v1(uint32_t handle,const uint8_t *data,size_t size,size_t *copied){
 ++edit_writes;if(edit_mode==6 && edit_writes==2){*copied=0;return 5;}return edit_real_write(handle,data,size,copied);
}
uint32_t fairyc_private_linux_root_file_read_v1(uint32_t handle,uint8_t *data,size_t size,size_t *copied){
 if(edit_mode==14 || edit_mode==27){*copied=0;return 5;}return edit_real_read(handle,data,size,copied);
}
uint32_t edit_apply_recheck(void){assert(remote_find(0,".fairyc-edit.lock")>=0 && edit_prepares==0);++edit_rechecks;return edit_mode==8?1u:0u;}
uint32_t fairyc_private_linux_edit_file_prepare_v1(uint32_t handle,uint32_t mode){
 assert(edit_rechecks==1 && remote_held(handle)==edit_file && remote_nodes[edit_file].size==3 && mode==0640);++edit_prepares;
 if(edit_mode==7){return 5;}remote_nodes[edit_file].synced=1;return 0;
}
uint32_t edit_apply_capture(void){
 assert(edit_rechecks==1 && edit_prepares==1 && edit_file>=0 && remote_nodes[edit_file].synced);++edit_captures;
 /* Inject a writer after the simulated closure observation, before final read. */
 if(edit_mode==11){remote_nodes[edit_source].data[0]=remote_nodes[edit_alias].data[0]='x';}if(edit_mode==12){++edit_drift;}
 return edit_mode==9?1u:edit_mode==10?2u:0u;
}
uint32_t fairyc_private_linux_edit_file_publish_v1(uint32_t handle,uint32_t stage,uint32_t root,const uint8_t *leaf,size_t size,const void *metadata,uint32_t *cleanup){
 assert(edit_rechecks==1 && edit_captures==1 && edit_prepares==1 && edit_renames==0);++edit_renames;*cleanup=0;
 int file=remote_held(handle),parent=remote_held(root);assert(file==edit_file && remote_held(stage)==edit_stage && remote_nodes[file].synced && remote_nodes[edit_stage].parent==parent);
 assert(size==6 && strcmp((const char *)leaf,"input")==0 && remote_find(parent,"input")==edit_source);
 struct remote_meta actual=remote_metadata(edit_source);const struct remote_meta *expected=(const struct remote_meta *)metadata;
 assert(actual.device==expected->device && actual.identity==expected->identity && actual.size==expected->size && actual.seconds==expected->seconds && actual.nanos==expected->nanos && actual.mode==expected->mode);
 if(edit_mode==15){return 4;}if(edit_mode==22){remote_nodes[file].parent=edit_stage;remote_nodes[file].links=1;memcpy(remote_nodes[file].name,"body",sizeof "body");*cleanup=13;++remote_mutations;return 5;}
 remote_nodes[edit_source].active=0;remote_nodes[edit_source].links=remote_nodes[edit_alias].links=1;remote_nodes[file].parent=parent;memcpy(remote_nodes[file].name,"input",sizeof "input");remote_nodes[file].links=1;++remote_mutations;
 if(edit_mode==21){*cleanup=13;}return 0;
}
uint32_t fairyc_private_linux_package_directory_erase_v1(uint32_t child,uint32_t parent,const uint8_t *leaf,size_t size){
 int stage=edit_stage_name((const char *)leaf);if(stage && edit_mode==17){return 39;}if(!stage && (edit_mode==22 || edit_mode==23)){return 43;}
 return edit_real_erase(child,parent,leaf,size);
}
uint32_t fairyc_private_linux_root_file_close_v1(uint32_t handle){
 if(handle==0){return 0;}int node=remote_held(handle);uint32_t role=edit_lease_role[handle];uint32_t code=edit_real_close(handle);assert(code==0);
 if(node==edit_file && edit_mode==16){return 55;}if(node==edit_source && edit_mode==27){return 54;}
 if(edit_mode==18 && handle==edit_parent_handle){return 62;}
 if(edit_mode==19 && (role==3 || role==4)){return role==3?63u:64u;}
 if(edit_mode==20 && (role==1 || role==2)){return role==1?61u:62u;}return 0;
}
void edit_apply_verify(uint32_t published){
 assert(remote_live==1 && edit_renames<=1);int src=remote_find(0,"src"),file=remote_find(src,"input");assert(file>0);
 assert(published?file==edit_file && file!=edit_source:file==edit_source);
 assert(memcmp(remote_nodes[edit_alias].data,edit_mode==11 || edit_mode==26?"x\0\xff":"a\0\xff",3)==0 && remote_nodes[edit_alias].active && remote_nodes[edit_alias].links==(published?1u:2u));
 assert(memcmp(remote_nodes[file].data,published?"n\0\xfe":edit_mode==11 || edit_mode==26?"x\0\xff":"a\0\xff",3)==0);
 assert(remote_find(0,".fairyc-edit.lock")>=0?edit_mode==1 || edit_mode==22 || edit_mode==23:edit_mode!=1 && edit_mode!=22 && edit_mode!=23);
 for(size_t i=1;i<remote_node_count;++i){if(!remote_nodes[i].active){continue;}assert(remote_nodes[i].parent>=0);if(edit_stage_name(remote_nodes[i].name)){assert(edit_mode==2 || edit_mode==3 || edit_mode==17 || edit_mode==22);}}
 if(edit_mode==0 || edit_mode==2 || edit_mode==15 || (edit_mode>=16 && edit_mode<=23)){assert(edit_renames==1);}else{assert(edit_renames==0);}
 if(edit_mode==1 || edit_mode==25){assert(edit_creates==1 && edit_rechecks==0);}else{assert(edit_rechecks==1);}
 if(edit_mode==2){assert(edit_creates==17);}if(edit_mode==3){assert(edit_creates==17 && remote_anonymous==0);}
 if(edit_mode==2 || edit_mode==3){for(size_t i=0;i<(edit_mode==2?15u:16u);++i){char name[64];int n=snprintf(name,sizeof name,".fairyc-edit-stage-%zu.tmp",i);assert(n>0 && (size_t)n<sizeof name);int original=remote_find(src,name);assert(original>0 && remote_nodes[original].kind==16384 && remote_find(original,"foreign")>0);}}
 if(edit_mode==4 || edit_mode==24){assert(edit_creates==2);}if(edit_mode==27){assert(remote_anonymous==0);}
}
`
