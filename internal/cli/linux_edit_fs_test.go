package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/sourceedit"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestLinuxEditSourcePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_directory", "project_manifest_values", "linux_edit_fs"}, "linux_edit_fs", `
@effects(filesystem) fn i32 main(){
 preserves_binary_metadata_hard_links_and_exact_spelling();
 rejects_paths_before_io_and_bounds_exact_bytes();
 detects_open_and_read_metadata_drift_and_native_failures();
 preserves_close_causes_and_releases_every_owned_handle_once();
 shares_entry_and_work_limits_without_allocating_a_listing();return 0;
}
`, linuxEditSourcePolicyStub)
}

func TestLinuxEditSourceGoReference(t *testing.T) {
	directory := t.TempDir()
	want := []byte{'a', 0, 255}
	if err := os.WriteFile(filepath.Join(directory, "input"), want, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(directory, "input"), filepath.Join(directory, "alias")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, name := range []string{"input", "alias"} {
		for _, maximum := range []int64{2, 3, 4} {
			got, err := sourceedit.ReadFile(root, name, maximum)
			if maximum < 3 {
				if err == nil {
					t.Fatalf("oversized %s admitted", name)
				}
			} else if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("hard-linked %s: %x, %v", name, got, err)
			}
		}
	}
	for _, name := range []string{"Input", "../input", "input/", "a//input", "con.txt", "file?", "file.", "file "} {
		if _, err := sourceedit.ReadFile(root, name, 3); err == nil {
			t.Fatalf("noncanonical %q admitted", name)
		}
	}
}

func TestLinuxEditSourceNative(t *testing.T) {
	project := t.TempDir()
	writeTestSource(t, project, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.edit-source-native\"\nname = \"client\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n")
	for _, name := range []string{"linux_command", "linux_rooted_file", "linux_directory", "project_manifest_values", "linux_edit_fs"} {
		source, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, project, "src/"+name+".fy", string(source))
	}
	writeTestSource(t, project, "src/main.fy", linuxEditSourceNativeMain)
	workspace, err := workspaceloader.LoadVectorCandidate(project)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "Linux edit source", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower: %d %s", code, &diagnostics)
	}
	unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0Profile())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile()); err != nil {
		t.Fatal(err)
	}
	support, err := nativesupport.Source(cbackend.Stage0Profile().Target, unit.ForeignSymbols, nativesupport.Options{SDKStorage: true, CandidateHost: true})
	if err != nil || !strings.Contains(support, "fairyc_private_linux_root_file_info_v1") {
		t.Fatalf("closed native runtime selection: %v", err)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("both-target edit-source C emission and actual Linux runtime selection passed; native Linux execution is batched")
	}
	root := filepath.Join(t.TempDir(), "root with space")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte{'a', 0, 255}, 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(root, "input"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "tree"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"empty": {}, "tree/item": {'a', 0, 255}, "oversize": make([]byte, 8<<20+1)} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range map[string]string{"linked": "input", "shortcut": "tree"} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, compiler := range []string{"gcc", "clang"} {
		t.Run(compiler, func(t *testing.T) {
			tc, err := cdriver.ResolveNativeToolchain(cbackend.Stage0Profile().Target, compiler)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support})
					if err != nil {
						t.Fatal(err)
					}
					t.Setenv("PATH", "")
					outcome, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: []string{root}, Timeout: 20 * time.Second})
					if err != nil || outcome.ExitCode != 0 || outcome.TrapCategory != "" || outcome.Stdout != "" || outcome.Stderr != "" {
						t.Fatalf("native edit read: %+v %v", outcome, err)
					}
				})
			}
		})
	}
}

const linuxEditSourceNativeMain = `
module main;
import linux_edit_fs as fs;
import linux_rooted_file as rooted;
import std::bytes as bytes;
import std::process as process;
@effects(filesystem) fn u32 problem(ref rooted::Directory root,ref u8[] path,usize maximum){return switch(fs::read(root,path,maximum)){case .ok(_):{0}case .err(p):{p}};}
@effects(filesystem,environment) fn i32 main(){
 var u8[4096] raw=[0;4096];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return 1;}};let ref u8[] path=ref raw;
 let root=switch(rooted::open(path[..n])){case .ok(p):{p}case .err(_):{return 2;}};
 var info=fs::empty_metadata();let input=switch(fs::read_observed(ref root,b"input",3,ref mut info)){case .ok(p):{p}case .err(_):{return 3;}};
 assert(bytes::equal(bytes::view(ref input),b"a\0\xff") && info.size==3 && (info.mode&128)==0);
 var alias_info=fs::empty_metadata();let alias=switch(fs::read_observed(ref root,b"alias",3,ref mut alias_info)){case .ok(p):{p}case .err(_):{return 4;}};
 assert(bytes::equal(bytes::view(ref alias),b"a\0\xff") && fs::same(info,alias_info));
 let nested=switch(fs::read(ref root,b"tree/item",3)){case .ok(p):{p}case .err(_):{return 5;}};assert(bytes::equal(bytes::view(ref nested),b"a\0\xff"));
 let empty=switch(fs::read(ref root,b"empty",0)){case .ok(p):{p}case .err(_):{return 6;}};assert(bytes::len(ref empty)==0);
 assert(problem(ref root,b"Input",3)==5 && problem(ref root,b"linked",3)==4 && problem(ref root,b"shortcut/item",3)==4);
 assert(problem(ref root,b"oversize",8388608)==3 && problem(ref root,b"tree",3)==4 && problem(ref root,b"input",2)==3);
 assert(rooted::close_directory(root).is_ok);return 0;
}
`

// Independent descriptor/stream model: no package-file observer, mutation API,
// shell helper or filesystem implementation from linux_edit_fs is substituted.
const linuxEditSourcePolicyStub = `
#include <stdint.h>
#include <stddef.h>
#include <assert.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
struct edit_source_meta {size_t device,identity,size;int64_t seconds;uint32_t nanos,mode;};
struct edit_source_cursor {int active;uint32_t parent;size_t at;};
static struct edit_source_cursor edit_source_cursors[16];
static uint32_t edit_source_mode,edit_source_flags,edit_source_live,edit_source_calls,edit_source_reads,edit_source_infos;
static size_t edit_source_size,edit_source_actual,edit_source_offset,edit_source_siblings;
static int edit_source_handles[9];
void edit_source_probe(uint32_t mode,size_t size,size_t actual,uint32_t flags){
 assert(edit_source_live==0);const char *path=getenv("PATH");assert(path==NULL || path[0]==0);
 memset(edit_source_cursors,0,sizeof edit_source_cursors);memset(edit_source_handles,0,sizeof edit_source_handles);
 edit_source_mode=mode;edit_source_size=size;edit_source_actual=actual;edit_source_flags=flags;
 edit_source_calls=edit_source_reads=edit_source_infos=0;edit_source_offset=0;edit_source_siblings=1;
}
uint32_t edit_source_probe_live(void){return edit_source_live;}
uint32_t edit_source_probe_calls(void){return edit_source_calls;}
uint32_t edit_source_probe_reads(void){return edit_source_reads;}
void edit_source_probe_siblings(size_t count){assert(count>=1 && edit_source_live==0);edit_source_siblings=count;}
static struct edit_source_meta edit_source_metadata(void){struct edit_source_meta p={71,123,edit_source_size,91,42,0100640};if(edit_source_mode==40){p.mode=0100444;}return p;}
static void edit_source_drift(struct edit_source_meta *p,uint32_t field){
 if(field==0){++p->device;}else if(field==1){++p->identity;}else if(field==2){++p->size;}else if(field==3){++p->seconds;}else if(field==4){++p->nanos;}else{p->mode^=0200;}
}
uint32_t fairyc_private_linux_directory_open_v1(const uint8_t *path,size_t size,uint32_t *output){
 ++edit_source_calls;assert(size==6 && strcmp((const char *)path,"/work")==0 && !edit_source_handles[1]);edit_source_handles[1]=1;++edit_source_live;*output=1;return 0;
}
uint32_t fairyc_private_linux_root_directory_open_v1(uint32_t parent,const uint8_t *leaf,size_t size,uint32_t *output){
 ++edit_source_calls;assert(size==strlen((const char *)leaf)+1 && parent<4 && edit_source_handles[parent]);*output=0;
 uint32_t next=parent==1 && strcmp((const char *)leaf,"tree")==0?2u:parent==2 && strcmp((const char *)leaf,"branch")==0?3u:0u;
 if(next==0){return 2;}assert(!edit_source_handles[next]);edit_source_handles[next]=1;++edit_source_live;*output=next;return 0;
}
uint32_t fairyc_private_linux_root_file_open_v1(uint32_t parent,const uint8_t *leaf,size_t size,uint32_t *output){
 ++edit_source_calls;assert(parent>=1 && parent<=3 && edit_source_handles[parent] && size==strlen((const char *)leaf)+1);*output=0;
 assert(strcmp((const char *)leaf,parent==1?"input":"item")==0);if((edit_source_flags&128)!=0){return 13;}
 assert(!edit_source_handles[8]);edit_source_handles[8]=1;++edit_source_live;edit_source_offset=0;*output=8;return 0;
}
uint32_t fairyc_private_linux_root_file_info_v1(uint32_t file,void *output){
 ++edit_source_calls;++edit_source_infos;assert(file==8 && edit_source_handles[file]);
 if((edit_source_flags&16)!=0 || ((edit_source_flags&32)!=0 && edit_source_infos==2)){return 5;}
 struct edit_source_meta p=edit_source_metadata();
 if(edit_source_mode>=10 && edit_source_mode<=15 && edit_source_infos==1){edit_source_drift(&p,edit_source_mode-10);}
 if(edit_source_mode>=20 && edit_source_mode<=25 && edit_source_infos==2){edit_source_drift(&p,edit_source_mode-20);}
 memcpy(output,&p,sizeof p);return 0;
}
uint32_t fairyc_private_linux_root_file_read_v1(uint32_t file,uint8_t *data,size_t size,size_t *copied){
 ++edit_source_calls;++edit_source_reads;assert(file==8 && edit_source_handles[file] && size>0 && size<=65536);*copied=0;
 if((edit_source_flags&8)!=0 && edit_source_reads==2){return 5;}
 if(edit_source_mode==45){*copied=size+1;return 0;}
 assert(edit_source_offset<=edit_source_actual);size_t count=edit_source_actual-edit_source_offset;if(count>size){count=size;}
 if(edit_source_reads==1 && count>0){count=edit_source_mode==30 && count>2?2:1;}
 static const uint8_t pattern[3]={'a',0,255};for(size_t i=0;i<count;++i){data[i]=pattern[(edit_source_offset+i)%3];}
 edit_source_offset+=count;*copied=count;return 0;
}
uint32_t fairyc_private_linux_root_file_close_v1(uint32_t file){
 if(file==0){return 0;}++edit_source_calls;assert(file<9 && edit_source_handles[file]);edit_source_handles[file]=0;assert(edit_source_live>0);--edit_source_live;
 return file==8 && (edit_source_flags&1)!=0?61u:(file==2 || file==3) && (edit_source_flags&2)!=0?62u:0u;
}
uint32_t fairyc_private_linux_directory_scan_open_v1(uint32_t parent,size_t *output){
 ++edit_source_calls;assert(parent>=1 && parent<=3 && edit_source_handles[parent]);
 for(size_t i=0;i<16;++i){if(!edit_source_cursors[i].active){edit_source_cursors[i].active=1;edit_source_cursors[i].parent=parent;edit_source_cursors[i].at=0;++edit_source_live;*output=100+i;return 0;}}
 assert(0);return 5;
}
uint32_t fairyc_private_linux_directory_scan_next_v1(size_t handle,uint8_t *name,size_t capacity,uint32_t *length,uint32_t *state,void *metadata){
 ++edit_source_calls;assert(handle>=100 && handle<116 && edit_source_cursors[handle-100].active && capacity==256);
 struct edit_source_cursor *cursor=&edit_source_cursors[handle-100];size_t at=cursor->at++;*length=*state=0;
 if((edit_source_flags&64)!=0){return 5;}if(edit_source_mode==46){*state=3;*length=1;name[0]='x';name[1]=0;return 0;}
 if(edit_source_mode==47){*state=1;*length=256;return 0;}if(edit_source_mode==48){*state=1;*length=5;memset(name,'x',256);return 0;}
 const char *leaf=NULL;uint32_t kind=0100000;
 if(at<2 || edit_source_mode==50){leaf=at==0?".":"..";*state=2;}
 else if(edit_source_mode==49){leaf="vanished";*state=2;}
 else if(cursor->parent==1 && edit_source_siblings>1){size_t row=at-2;if(row>=edit_source_siblings){return 0;}if(row+1==edit_source_siblings){leaf="input";}else{int n=snprintf((char *)name,capacity,"extra%zu",row);assert(n>0 && (size_t)n<capacity);*length=(uint32_t)n;*state=1;struct edit_source_meta p=edit_source_metadata();memcpy(metadata,&p,sizeof p);return 0;}}
 else if(cursor->parent==1){
  if(at==2){leaf=edit_source_mode==51?"\xff":"tree";kind=edit_source_mode==44?0120000u:0040000u;}
  else if(at==3){leaf=edit_source_mode==41 || edit_source_mode==42?"INPUT":"input";if(edit_source_mode==43){kind=0120000;}}
  else if(at==4 && edit_source_mode==42){leaf="input";}else{return 0;}
 }else if(cursor->parent==2){if(at==2){leaf="branch";kind=0040000;}else if(at==3){leaf="item";}else{return 0;}}
 else{if(at==2){leaf="item";}else{return 0;}}
 size_t size=strlen(leaf);memcpy(name,leaf,size+1);*length=(uint32_t)size;if(*state==0){*state=1;}
 struct edit_source_meta p=edit_source_metadata();p.mode=(p.mode&07777)|kind;memcpy(metadata,&p,sizeof p);return 0;
}
uint32_t fairyc_private_linux_directory_scan_close_v1(size_t handle){
 if(handle==0){return 0;}++edit_source_calls;assert(handle>=100 && handle<116 && edit_source_cursors[handle-100].active);
 edit_source_cursors[handle-100].active=0;assert(edit_source_live>0);--edit_source_live;return (edit_source_flags&4)!=0?73u:0u;
}
`
