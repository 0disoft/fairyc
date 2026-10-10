module tests::module_unit_c_literals;
import module_unit_c_emit as emit;
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
@effects(console) fn ir::Program retained(ref u8[] root,ref u8[] provider){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,v::len<syntax::Parsed>(ref sources))){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" at ");discard io::write_usize(e.start);discard io::write(".. ");discard io::write_usize(e.end);discard io::write(" lower\n");assert(false);retained(root,provider)}};
}
fn usize entry(ref ir::Program p){var usize found=0;var usize i=1;while(i<=ir::count(p)){let f=clean_ir::frame_at(ir::cleanup_ir(p),i);let type=clean_ir::type_at(ir::cleanup_ir(p),f.result);if(f.module_index==0 && f.kind==3 && type.kind==1 && type.scalar==1){var usize j=1;var usize params=0;let b=ir::at(p,i);while(j<=ir::local_count(b)){if(ir::local_at(b,j).kind==1){params+=1;}j+=1;}if(params==0){assert(found==0);found=i;}}i+=1;}return found;}
@effects(console) fn void output(ref ir::Program p,usize frame){var bytes::Buffer c=bytes::empty();let e=emit::render(p,frame,ref mut c);if(e.code!=0){discard io::write_u32(e.code);discard io::write(" C at ");discard io::write_usize(e.start);discard io::write("\n");}assert(e.code==0);assert(bytes::view(ref c).len>100);assert(io::write_bytes(bytes::view(ref c)).is_ok);}



@test @effects(console) fn void literal_c_preserves_exact_bytes_empty_views_and_cstr_storage(){
    let p=retained(b"module main;@link_name(\"check_literals\")unsafe extern C fn i32 check(ptr u8 text,usize text_len,ptr u8 data,usize data_len,ptr u8 c,ptr u8 empty_text,ptr u8 empty_bytes,ptr u8 empty_c);fn i32 main(){let ref str text=\"A\\x00\\u{03bb}\\n\\\"\\\\\";let ref u8[] data=b\"\\x00\\xff9\";let ref CStr c=c\"A\\xff9\";let ref str empty=\"\";let ref u8[] zero=b\"\";let ref CStr ec=c\"\";assert(text.len==7 && !text.is_empty && data.len==3 && data[0]==0 && data[1]==255 && data[2]==57 && empty.len==0 && empty.is_empty && zero.len==0 && zero.is_empty);unsafe{assert(check(text as ptr u8,text.len,data as ptr u8,data.len,c as ptr u8,empty as ptr u8,zero as ptr u8,ec as ptr u8)==42);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void literal_c_keeps_static_returns_and_provider_internal_transport(){
    let p=retained(b"module main;import helper as h;@link_name(\"read_u8\")unsafe extern C fn i32 read(ptr u8 n);fn ref str same(ref str n){return n;}fn ref CStr pass(ref CStr n){return n;}fn ref u8[] bytes(ref u8[] n){return n;}fn i32 main(){let ref str a=h::text();let ref str b=same(a);let ref CStr c=pass(h::cstr());let ref u8[] data=bytes(h::data());assert(a.len==5 && b.len==5 && data.len==2 && data[0]==9 && data[1]==255);unsafe{assert(read(b as ptr u8)==102 && read(c as ptr u8)==67);assert(a as ptr u8==b as ptr u8 && h::text() as ptr u8==a as ptr u8);}return 0;}",b"module helper;pub fn ref str text(){return \"fairy\";}pub fn ref CStr cstr(){return c\"C\";}pub fn ref u8[] data(){return b\"\\x09\\xff\";}");output(ref p,entry(ref p));
}
@test @effects(console) fn void literal_c_deduplicates_numeric_storage_across_types_and_functions(){
    let p=retained(b"module main;fn ref u8[] data(){return b\"fairy\";}fn i32 main(){let ref str text=\"fairy\";let ref u8[] bytes=b\"fairy\";let ref CStr c=c\"fairy\";unsafe{assert(text as ptr u8==bytes as ptr u8 && bytes as ptr u8==data() as ptr u8);}assert(text.len==5 && bytes.len==5);discard c;return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void literal_c_traps_byte_index_at_literal_length(){let p=retained(b"module main;fn i32 main(){let ref u8[] bytes=b\"\\x01\";return bytes[1] as i32;}",b"");output(ref p,entry(ref p));}
@test @effects(console) fn void literal_c_rejects_foreign_descriptor_abi_transactionally(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;@link_name(\"pending_descriptor_abi\")unsafe extern C fn void use(ref CStr n);")).is_ok);let rejected=switch(signatures::build(ref sources)){case .ok(_):{false}case .err(e):{e.code==5102}};assert(rejected);let p=retained(b"module main;fn i32 main(){let text=\"owned\";assert(text.len==5);return 0;}",b"");var bytes::Buffer c=switch(bytes::copy(b"sentinel")){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(emit::render(ref p,0,ref mut c).code==1001);assert(bytes::equal(bytes::view(ref c),b"sentinel"));
}
