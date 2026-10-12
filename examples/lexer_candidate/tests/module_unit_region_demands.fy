module tests::module_unit_region_demands;
import module_unit_interfaces as interfaces;
import module_unit_bind as bind;
import module_signatures as signatures;
import module_ast_join as join;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import file_source as files;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(value):{value}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(value):{value}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(value):{assert(syntax::problem_count(ref value)==0);value}case .err(_):{assert(false);parsed(raw)}};}
@effects(filesystem)
fn syntax::Parsed allocator(){var u8[4096] path=[0;4096];let size=switch(process::arg(1,ref mut path)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] raw=ref path;let data=switch(files::load(raw[..size])){case .ok(value):{value}case .err(_):{assert(false);return allocator();}};return parsed(bytes::view(ref data));}
fn Result<interfaces::Interfaces,join::Problem> projected(ref v::Vec<syntax::Parsed> sources){let metadata=switch(signatures::build(sources)){case .ok(value):{value}case .err(p):{return .err(join::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};let bound=switch(bind::bind(sources,ref metadata,0,1)){case .ok(value):{value}case .err(p):{return .err(p);}};return interfaces::build(sources,ref metadata,ref bound);}
fn v::Vec<syntax::Parsed> single(u32 mode){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(source(mode))).is_ok);return sources;}
@test
fn void unused_generic_regions_do_not_require_allocator(){let sources=single(0);let graph=switch(projected(ref sources)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(interfaces::region_arena(ref graph)==0 && interfaces::region_error(ref graph)==0);}
@test
fn void selected_ordinary_regions_keep_allocator_admission(){let sources=single(1);assert(switch(projected(ref sources)){case .ok(_):{false}case .err(p):{p.code==2 && p.module_index==0 && bytes::equal(source(1)[p.start..p.end],b"missing")}});}
@test
@effects(filesystem)
fn void retained_generic_and_imported_regions_keep_allocator_types(){var u32 mode=2;while(mode<=4){var sources=single(mode);if(mode>=3){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider(mode))).is_ok);}assert(v::push<syntax::Parsed>(ref mut sources,allocator()).is_ok);let graph=switch(projected(ref sources)){case .ok(value):{value}case .err(_):{assert(false);return;}};let arena=interfaces::region_arena(ref graph);let error=interfaces::region_error(ref graph);assert(arena!=0 && error!=0 && interfaces::type_at(ref graph,arena).kind==3 && interfaces::type_at(ref graph,error).kind==2);mode+=1;}}
fn ref u8[] source(u32 mode){
if(mode==0){return b"module main;fn T unused<T>(T value){region r in missing{discard r.place(value);}return value;}fn i32 main(){return 0;}";}
if(mode==1){return b"module main;fn void f(){region r in missing{}}";}
if(mode==2){return b"module main;import std::alloc as alloc;fn void used<T>(ref mut alloc::Arena a,T value){region r in a{discard r.place(value);}}fn void f(ref mut alloc::Arena a){used<i32>(a,3);}";}
if(mode==3){return b"module main;import provider as p;import std::alloc as alloc;fn void f(ref mut alloc::Arena a){p::used<i32>(a,3);}";}
if(mode==4){return b"module main;import provider as p;import std::alloc as alloc;fn void f(ref mut alloc::Arena a){p::ordinary(a);}";}
return b"";}
fn ref u8[] provider(u32 mode){
if(mode==3){return b"module provider;import std::alloc as alloc;pub fn void used<T>(ref mut alloc::Arena a,T value){region r in a{discard r.place(value);}}";}
if(mode==4){return b"module provider;import std::alloc as alloc;pub fn void ordinary(ref mut alloc::Arena a){region r in a{discard r.place(3);}}";}
return b"";}
