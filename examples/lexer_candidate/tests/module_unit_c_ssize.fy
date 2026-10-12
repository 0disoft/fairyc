module tests::module_unit_c_ssize;
import module_unit_c_emit as emit;
import module_project_link as link;
import module_unit_c_scalar as scalar;
import c_text as w;
import module_unit_body_ir_model as ir;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_moves as moves;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import module_types as types;
import module_unit_value_model as m;
import module_unit_region_model as regions;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn ir::Program retained(ref u8[] root,ref u8[] provider,bool project){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider,project);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,v::len<syntax::Parsed>(ref sources))){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider,project);}};
    let ready=switch(if(project){closure::build_unit(ref sources,ref metadata,ref bound)}else{closure::build(ref sources,ref metadata,ref bound)}){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider,project);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider,project);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider,project);}};
    let p=switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" at ");discard io::write_usize(e.start);discard io::write(".. ");discard io::write_usize(e.end);discard io::write(" lower\n");assert(false);retained(root,provider,project)}};
    if(!project){return p;}var v::Vec<ir::Program> units=v::empty<ir::Program>();assert(v::push<ir::Program>(ref mut units,p).is_ok);return switch(link::build(ref metadata,ref units)){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider,project);}};
}
fn usize entry(ref ir::Program p){var usize found=0;var usize i=1;while(i<=ir::count(p)){let f=clean_ir::frame_at(ir::cleanup_ir(p),i);let type=clean_ir::type_at(ir::cleanup_ir(p),f.result);if(f.module_index==0 && f.kind==3 && type.kind==1 && type.scalar==1){var usize j=1;var usize params=0;let b=ir::at(p,i);while(j<=ir::local_count(b)){if(ir::local_at(b,j).kind==1){params+=1;}j+=1;}if(params==0){assert(found==0);found=i;}}i+=1;}return found;}
@effects(console) fn void output(ref ir::Program p,usize frame){var bytes::Buffer c=bytes::empty();let e=emit::render(p,frame,ref mut c);if(e.code!=0){discard io::write_u32(e.code);discard io::write(" C at ");discard io::write_usize(e.start);discard io::write("\n");}assert(e.code==0);assert(bytes::view(ref c).len>100);assert(io::write_bytes(bytes::view(ref c)).is_ok);}


fn ref u8[] source(){return b"module main;@link_name(\"fairyc_fixture_make_c_ssize_t\")unsafe extern C fn c::ssize_t produce();@link_name(\"fairyc_fixture_consume_c_ssize_t\")unsafe extern C fn i32 consume(c::ssize_t n);fn i32 main(){let c::ssize_t n=unsafe{produce()};return unsafe{consume(n)};}";}
fn usize position(ref u8[] text,ref u8[] part){var usize i=0;while(i<=text.len){if(part.len<=text.len-i && bytes::equal(text[i..i+part.len],part)){return i;}i+=1;}return text.len;}
fn void posix(ref u8[] text){assert(position(text,b"#define _POSIX_C_SOURCE 200809L")<position(text,b"#include <stdint.h>"));assert(position(text,b"#include <sys/types.h>")<position(text,b"#include <stdint.h>"));assert(position(text,b"#error c.ssize_t is unavailable for candidate target")<text.len);assert(position(text,b"sizeof(ssize_t)==8 && _Alignof(ssize_t)==8 && ((ssize_t)-1)<0")<text.len);assert(position(text,b"extern ssize_t fairyc_fixture_make_c_ssize_t(void)")<text.len);assert(position(text,b"fairyc_fixture_consume_c_ssize_t(ssize_t)")<text.len);}
@test
@effects(console)
fn void owned_ssize_c_keeps_foreign_local_identity_and_target_proof(){let p=retained(source(),b"",false);assert(proof::check(ref p).code==0);var bytes::Buffer c=bytes::empty();assert(emit::render(ref p,entry(ref p),ref mut c).code==0);posix(bytes::view(ref c));assert(io::write_bytes(bytes::view(ref c)).is_ok);}
@test
@effects(console)
fn void partitioned_ssize_c_proves_posix_in_the_shared_header(){let p=retained(source(),b"",true);let frame=entry(ref p);let bundle=switch(emit::partition(p,frame,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};posix(emit::project_header(ref bundle));assert(emit::project_unit_count(ref bundle)>0);var usize i=0;while(i<emit::project_unit_count(ref bundle)){let text=emit::project_source(ref bundle,i);assert(position(text,b"#include \"fy_project.h\"")<text.len && position(text,b"#include <stdint.h>")==text.len);i+=1;}}
@test
@effects(console)
fn void iso_only_owned_c_avoids_posix_feature_selection(){let p=retained(b"module main;fn i32 main(){return 0;}",b"",false);var bytes::Buffer c=bytes::empty();assert(emit::render(ref p,entry(ref p),ref mut c).code==0);assert(position(bytes::view(ref c),b"_POSIX_C_SOURCE")==bytes::len(ref c));}
