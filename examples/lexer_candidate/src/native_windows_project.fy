module native_windows_project;
import project_json_tree as json;
import project_source_file as files;
import project_manifest_values as rules;
import std::bytes as bytes;
import std::fs as fs;
import std::text as text;

// Native build configuration, never language/cache authority. C/RC text is
// snapshotted; project includes/assets remain trusted compiler-tool inputs.
struct Metadata{bool gui;usize libraries;usize sources;usize files;usize[56] label_start;usize[56] label_end;usize[24] start;usize[24] end;}
fn Metadata metadata(){return Metadata{.gui=false,.libraries=0,.sources=0,.files=0,.label_start=[0;56],.label_end=[0;56],.start=[0;24],.end=[0;24]};}
pub resource Project{bool enabled;Metadata meta;bytes::Buffer root;bytes::Buffer labels;bytes::Buffer contents;cleanup{}}
pub fn Project empty(){return Project{.enabled=false,.meta=metadata(),.root=bytes::empty(),.labels=bytes::empty(),.contents=bytes::empty()};}
pub fn bool enabled(ref Project p){return p.enabled;}
pub fn bool gui(ref Project p){return p.meta.gui;}
pub fn ref u8[] root(ref Project p){return bytes::view(ref p.root);}
pub fn usize library_count(ref Project p){return p.meta.libraries;}
pub fn ref u8[] library(ref Project p,usize i){assert(i<p.meta.libraries);return bytes::view(ref p.labels)[p.meta.label_start[i]..p.meta.label_end[i]];}
pub fn usize source_count(ref Project p){return p.meta.sources;}
pub fn ref u8[] source(ref Project p,usize i){assert(i<p.meta.sources);return bytes::view(ref p.contents)[p.meta.start[i]..p.meta.end[i]];}
pub fn usize resource_count(ref Project p){return p.meta.files-p.meta.sources;}
pub fn ref u8[] resource_data(ref Project p,usize i){let at=p.meta.sources+i;assert(at<p.meta.files);return bytes::view(ref p.contents)[p.meta.start[at]..p.meta.end[at]];}
fn bool letter(u8 c){return c>=97 && c<=122;}
fn bool digit(u8 c){return c>=48 && c<=57;}
fn bool library_name(ref u8[] name){if(name.len<5 || name.len>68 || !letter(name[0]) || !bytes::equal(name[name.len-4..],b".lib")){return false;}var usize i=1;while(i<name.len-4){if(!letter(name[i]) && !digit(name[i]) && name[i]!=95){return false;}i+=1;}return true;}
fn bool path(ref u8[] name,bool is_resource){let suffix=if(is_resource){b".rc"}else{b".c"};return name.len<=240 && name.len>suffix.len && rules::path(name) && bytes::equal(name[name.len-suffix.len..],suffix);}
fn u8 lower(u8 c){return if(c>=65 && c<=90){c+32}else{c};}
fn bool same(ref u8[] a,ref u8[] b){if(a.len!=b.len){return false;}var usize i=0;while(i<a.len){if(lower(a[i])!=lower(b[i])){return false;}i+=1;}return true;}
fn ref u8[] label(ref bytes::Buffer labels,ref Metadata m,usize i){return bytes::view(labels)[m.label_start[i]..m.label_end[i]];}
fn Result<usize,u32> array(ref json::Tree tree,usize at,usize maximum,bool libraries,bool resources,usize base,ref mut bytes::Buffer labels,ref mut Metadata m){
 if(json::token(tree,at).kind!=3){return .err(1);}let end=json::next(tree,at)-1;var usize i=at+1;var usize count=0;
 while(i<end){if(json::token(tree,i).kind!=7 || count>=maximum){return .err(1);}let name=json::value(tree,i);if(!(if(libraries){library_name(name)}else{path(name,resources)})){return .err(1);}
  var usize j=0;while(j<count){if(same(label(labels,m,base+j),name)){return .err(1);}j+=1;}
  m.label_start[base+count]=bytes::len(labels);if(!bytes::append(labels,name).is_ok){return .err(2);}m.label_end[base+count]=bytes::len(labels);count+=1;
  i=json::next(tree,i);if(i<end){i+=1;}
 }return .ok(count);
}
@effects(filesystem) fn Result<bytes::Buffer,u32> read(ref u8[] root,ref u8[] name,usize bound){var usize work=0;let data=switch(files::read(root,name,bound,ref mut work)){case .ok(p):{p}case .err(_):{return .err(1);}};let raw=bytes::view(ref data);if(raw.len==0 || raw.len>bound || !text::validate_utf8(raw).is_ok){return .err(1);}var usize i=0;while(i<raw.len){if(raw[i]==0){return .err(1);}i+=1;}return .ok(data);}
// Codes 1 malformed/native file input, 2 allocation. Missing config is ordinary.
@effects(filesystem) pub fn Result<Project,u32> load(ref u8[] root){
 if(root.len==0){return .ok(empty());}var usize work=0;
 var location=switch(bytes::copy(root)){case .ok(p):{p}case .err(_):{return .err(2);}};if(!bytes::append(ref mut location,b"/fairy.windows.json").is_ok){return .err(2);}let name=switch(text::validate_utf8(bytes::view(ref location))){case .ok(p):{p}case .err(_):{return .err(1);}};
 let probe=switch(fs::open_read_utf8(name)){case .ok(p):{p}case .err(e):{if(e.os_code==2){return .ok(empty());}return .err(1);}};if(!fs::close(probe).is_ok){return .err(1);}

 let raw=switch(files::read(root,b"fairy.windows.json",65536,ref mut work)){case .ok(p):{p}case .err(_):{return .err(1);}};
 let tree=switch(json::parse(raw,ref mut work)){case .ok(p):{p}case .err(_):{return .err(1);}};
 if(json::token(ref tree,0).kind!=1){return .err(1);}var u32 seen=0;var usize i=1;
 var usize resources=0;var Metadata m=metadata();var bytes::Buffer labels=bytes::empty();
 while(i<json::count(ref tree)-1){let key=json::value(ref tree,i);let at=i+2;var u32 bit=0;
  if(bytes::equal(key,b"schema")){bit=1;if(json::token(ref tree,at).kind!=7 || !bytes::equal(json::value(ref tree,at),b"fairyc.windows-build/v1")){return .err(1);}}
  else if(bytes::equal(key,b"subsystem")){bit=2;let value=json::value(ref tree,at);if(json::token(ref tree,at).kind!=7 || !bytes::equal(value,b"console") && !bytes::equal(value,b"windows")){return .err(1);}m.gui=bytes::equal(value,b"windows");}
  else if(bytes::equal(key,b"libraries")){bit=4;let count=array(ref tree,at,32,true,false,0,ref mut labels,ref mut m)?;m.libraries=count;}
  else if(bytes::equal(key,b"sources")){bit=8;let count=array(ref tree,at,16,false,false,32,ref mut labels,ref mut m)?;m.sources=count;}
  else if(bytes::equal(key,b"resources")){bit=16;resources=array(ref tree,at,8,false,true,48,ref mut labels,ref mut m)?;}
  else{return .err(1);}if((seen&bit)!=0){return .err(1);}seen|=bit;i=json::next(ref tree,at);if(i<json::count(ref tree)-1){i+=1;}
 }
 if((seen&3)!=3){return .err(1);}m.files=m.sources+resources;var usize total=0;i=0;var bytes::Buffer contents=bytes::empty();
 while(i<m.files){
  let name=label(ref labels,ref m,if(i<m.sources){32+i}else{48+i-m.sources});let data=read(root,name,4194304)?;total+=bytes::len(ref data);if(total>33554432){return .err(1);}
  m.start[i]=bytes::len(ref contents);if(!bytes::append(ref mut contents,bytes::view(ref data)).is_ok){return .err(2);}m.end[i]=bytes::len(ref contents);i+=1;
 }
 let owned=switch(bytes::copy(root)){case .ok(p):{p}case .err(_):{return .err(2);}};return .ok(Project{.enabled=true,.meta=m,.root=owned,.labels,.contents});
}
