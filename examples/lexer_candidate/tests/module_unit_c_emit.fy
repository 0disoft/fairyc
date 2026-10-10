module tests::module_unit_c_emit;
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
fn usize entry(ref ir::Program p){var usize i=1;while(i<=ir::count(p)){let f=clean_ir::frame_at(ir::cleanup_ir(p),i);if(f.module_index==0 && f.kind==3){return i;}i+=1;}return 0;}
@effects(console) fn void output(ref ir::Program p,usize frame){var bytes::Buffer c=bytes::empty();let e=emit::render(p,frame,ref mut c);if(e.code!=0){discard io::write_u32(e.code);discard io::write(" C at ");discard io::write_usize(e.start);discard io::write("\n");}assert(e.code==0);assert(bytes::view(ref c).len>100);assert(io::write_bytes(bytes::view(ref c)).is_ok);}
@test @effects(console) fn void module_c_executes_calls_loops_and_lazy_branch_values(){
    let p=retained(b"module main;import helper as h;fn i32 main(){var i32 sum=0;for(var i32 i=0;i<8;i+=1){if(i==2){continue;}if(i==6){break;}sum+=h::sub(i+20,20);}assert(sum==13);var i32 n=0;while(n<3){n+=1;}assert(n==3);assert(!(false && bad()));assert(true || bad());let i32 answer=if(n==3){h::sub(14,3)}else{0};assert(h::id<i32>(answer)==11);assert(factorial(5)==120);return 0;}fn bool bad(){assert(false);return false;}fn i32 factorial(i32 n){return if(n==0){1}else{n*factorial(n-1)};}",b"module helper;pub fn i32 sub(i32 left,i32 right){return left-right;}pub fn T id<T>(T n){return n;}");output(ref p,entry(ref p));
}
@test @effects(console) fn void module_c_freezes_defer_arguments_before_reassignment_and_return(){
    let p=retained(b"module main;fn i32 main(){var i32 n=1;defer done(n);n=9;{defer other(2);n+=1;}return if(n==10){0}else{1};}fn void done(i32 n){assert(n==1);}fn void other(i32 n){assert(n==2);}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void module_c_emits_closed_integer_widths_and_normalized_literals(){
    let p=retained(b"module main;fn i32 main(){let i64 lo=-9223372036854775808;let i64 hi=9223372036854775807;assert(lo<hi);assert((hi-1)+1==hi);assert((lo+1)-1==lo);let u8 small=0xff;assert(small/3==85);let u32 top=4_294_967_295;assert((top & 0xff)==255);let usize huge=18446744073709551615;assert(huge>>63==1);let i32 bits=-8;assert(bits>>2==-2);assert(~bits==7);return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void module_c_emits_overflow_trap(){let p=retained(b"module main;fn i32 main(){var i32 n=2147483647;return n+1;}",b"");output(ref p,entry(ref p));}
@test @effects(console) fn void module_c_emits_divide_by_zero_trap(){let p=retained(b"module main;fn i32 main(){var i32 n=0;return 1/n;}",b"");output(ref p,entry(ref p));}
@test @effects(console) fn void module_c_rejects_foreign_text_abi_and_bad_entry_without_partial_output(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;@link_name(\"pending_descriptor_abi\")unsafe extern C fn void use(ref str n);")).is_ok);let rejected=switch(signatures::build(ref sources)){case .ok(_):{false}case .err(e):{e.code==5102}};assert(rejected);let p=retained(b"module main;fn i32 main(){let text=\"owned\";assert(text.len==5);return 0;}",b"");var bytes::Buffer c=switch(bytes::copy(b"sentinel")){case .ok(b):{b}case .err(_):{assert(false);return;}};let invalid=emit::render(ref p,0,ref mut c);assert(invalid.code==1001);assert(bytes::equal(bytes::view(ref c),b"sentinel"));assert(emit::render(ref p,0,ref mut c).code==1001);assert(bytes::equal(bytes::view(ref c),b"sentinel"));
}
