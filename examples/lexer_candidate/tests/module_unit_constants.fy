module tests::module_unit_constants;
import module_unit_constants as constants;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_c_emit as emit;
import module_unit_cleanup_mir_model as clean;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn void rejected(ref u8[] source,u32 expected){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(source)).is_ok);switch(constants::build(ref sources)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==expected && p.module_index==0 && p.start<p.end);}};}
@test fn void private_literals_keep_declared_type_value_and_reject_unused_invalid_items(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;const i32 LOW=-2147483648;const u8 BYTE=(0xff);const u32 HIGH=0xffff_ffff;const bool YES=!false;const i32 ZERO=-0;static_assert(true);")).is_ok);let p=switch(constants::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;
    let low=switch(constants::get(ref p,0,2,ref mut work)){case .ok(v):{v}case .err(_):{assert(false);return;}};assert(low.scalar==1 && low.bits==2147483648);
    let byte=switch(constants::get(ref p,0,3,ref mut work)){case .ok(v):{v}case .err(_):{assert(false);return;}};assert(byte.scalar==3 && byte.bits==255);
    let high=switch(constants::get(ref p,0,4,ref mut work)){case .ok(v):{v}case .err(_):{assert(false);return;}};assert(high.scalar==4 && high.bits==4294967295);
    let yes=switch(constants::get(ref p,0,5,ref mut work)){case .ok(v):{v}case .err(_):{assert(false);return;}};assert(yes.scalar==6 && yes.bits==1);
    rejected(b"module main;pub const i32 X=1;",4301);rejected(b"module main;const usize X=1;",4301);rejected(b"module main;const u8[1] X=[1];",4301);
    rejected(b"module main;const u8 X=256;",1013);rejected(b"module main;const i32 X=2147483648;",1013);rejected(b"module main;const i32 X=-2147483649;",1013);rejected(b"module main;const u32 X=4294967296;",1013);rejected(b"module main;const bool X=1;",1006);
    rejected(b"module main;const i32 X=-(-2147483648);",4303);rejected(b"module main;@const_eval fn i32 value(){return 1;}",4302);rejected(b"module main;static_assert(false);",4305);rejected(b"module main;static_assert(1);",1007);
}
@test fn void forward_dependencies_short_circuit_and_i64_values_are_folded(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;const i32 FIRST=SECOND+6;const i32 SECOND=6*6;const u32 WIDE=40+2;const bool READY=42==WIDE;const bool SAFE=false && (1/0==0);const i64 LOW=-9223372036854775808;const i64 STEP=LOW+1;static_assert(READY && !SAFE);static_assert(STEP<0);static_assert(true);")).is_ok);let p=switch(constants::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;
    let first=switch(constants::get(ref p,0,2,ref mut work)){case .ok(v):{v}case .err(_):{assert(false);return;}};let ready=switch(constants::get(ref p,0,5,ref mut work)){case .ok(v):{v}case .err(_):{assert(false);return;}};let low=switch(constants::get(ref p,0,7,ref mut work)){case .ok(v):{v}case .err(_):{assert(false);return;}};let step=switch(constants::get(ref p,0,8,ref mut work)){case .ok(v):{v}case .err(_):{assert(false);return;}};
    assert(first.scalar==1 && first.bits==42 && ready.scalar==6 && ready.bits==1 && low.scalar==2 && low.bits==9223372036854775808 && step.scalar==2 && step.bits==9223372036854775809);
}
fn void rejected_at(ref u8[] text,u32 code,ref u8[] expected){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);switch(constants::build(ref sources)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && bytes::equal(text[p.start..p.end],expected));}};}
@test fn void cycles_dead_branch_types_and_traps_preserve_the_failing_source(){
    rejected_at(b"module main;const i32 FIRST=SECOND;const i32 SECOND=FIRST;",4304,b"FIRST");rejected_at(b"module main;const bool SELF=false && SELF;",4304,b"SELF");
    rejected_at(b"module main;const bool BAD=false && 1;",1007,b"1");rejected_at(b"module main;const i32 BAD=1/0;",4303,b"1/0");rejected_at(b"module main;const i64 BAD=-(-9223372036854775808);",4303,b"-(-9223372036854775808)");
}
@effects(console) fn void failure(ref str phase,u32 code){discard io::write(phase);discard io::write(" ");discard io::write_u32(code);discard io::write("\n");assert(false);}
@test @effects(console) fn void folded_literals_link_and_execute_as_scalar_operands(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;const u32 PAYLOAD_BODY=0;const i32 LOW=-2147483648;const u8 BYTE=255;const bool YES=true;fn i32 main(){assert(p::take(PAYLOAD_BODY)==0);assert(LOW==-2147483648);assert(BYTE==255);assert(YES);assert(p::high()==4294967295);return 0;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;const u32 HIGH=0xffff_ffff;pub fn u32 high(){return HIGH;}pub fn u32 take(u32 body){return body;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{failure("signatures",e.code);return;}};let units=switch(compile::build(ref sources,ref metadata)){case .ok(p):{p}case .err(e):{failure("compile",e.code);return;}};
    var p=switch(link::build(ref metadata,compile::programs(ref units))){case .ok(p):{p}case .err(e):{failure("link",e.code);return;}};assert(proof::check(ref p).code==0);var usize entry=0;var usize i=1;var usize folded=0;var usize row=0;
    while(i<=ir::count(ref p)){let f=clean::frame_at(ir::cleanup_ir(ref p),i);if(f.module_index==0){entry=i;}var usize n=0;while(n<ir::node_count(ir::at(ref p,i))){let e=ir::node_at(ir::at(ref p,i),n).value;assert(e.constant==0 && e.constant_bits==0);if(e.kind==1 && e.binding==0 && e.type!=0){folded+=1;if(i==entry){row=n;}}n+=1;}i+=1;}assert(entry!=0 && folded>=7);
    var bytes::Buffer text=bytes::empty();let rendered=emit::render(ref p,entry,ref mut text);if(rendered.code!=0){failure("emit",rendered.code);return;}assert(io::write_bytes(bytes::view(ref text)).is_ok);
}
@test fn void constant_argument_type_mismatch_is_rejected(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;const u32 X=0;fn i32 take(i32 n){return n;}fn i32 main(){return take(X);}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};switch(compile::build(ref sources,ref metadata)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1006 && e.start<e.end);}};
}
