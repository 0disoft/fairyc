module tests::module_unit_c_casts;
import module_unit_c_emit as emit;
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


@test @effects(console) fn void cast_c_preserves_all_admitted_integer_ranges(){
    let p=retained(b"module main;import helper as h;fn i32 main(){let u8 byte=255;let u32 wide=4294967295;let i32 lo=-2147483648;let i32 hi=2147483647;assert((byte as i32)==255 && (byte as u32)==255 && (byte as usize)==255 && (byte as i64)==255);assert((wide as usize)==4294967295 && (wide as i64)==4294967295);assert(h::widen(lo)==-2147483648 && h::widen(hi)==2147483647);assert((hi as i64)+1==2147483648);let i64 widened=h::convert<i64>(hi);assert(widened==2147483647);return 0;}",b"module helper;pub fn i64 widen(i32 n){return n as i64;}pub fn T convert<T>(i32 n){return n as T;}");output(ref p,entry(ref p));
}
@test @effects(console) fn void cast_c_roundtrips_signed32_through_separate_c_functions(){
    let p=retained(b"module main;@link_name(\"produce_c_int\")unsafe extern C fn c::int produce(i32 n);@link_name(\"consume_c_int\")@effects(console)unsafe extern C fn c::int consume(c::int n);fn c::int outward(i32 n){return n as c::int;}fn i32 inward(c::int n){return n as i32;}@effects(console)fn i32 main(){let i32 lo=-2147483648;let i32 hi=2147483647;var c::int n=outward(lo);assert(inward(n)==lo);n=outward(hi);assert(inward(n)==hi);unsafe{let c::int minimum=produce(lo);let c::int maximum=produce(hi);let c::int answer=produce(42);assert(inward(consume(minimum))==lo);assert(inward(consume(maximum))==hi);assert(inward(consume(outward(inward(answer))))==42);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void cast_c_projects_operands_and_evaluates_calls_once(){
    let p=retained(b"module main;struct Box{u8 n;}@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn u8 next(ref mut i32 calls){*calls+=1;unsafe{trace(*calls);}return 255;}@effects(console)fn void done(i64 n){assert(n==1);unsafe{trace(9);}}@effects(console)fn i32 main(){var i32 calls=0;let i64 n=next(ref mut calls) as i64;assert(calls==1 && n==255);let Box b=Box{.n=254};assert((b.n as i32)==254);let u8[2] items=[253,252];let ref u8[] view=ref items;assert((view[0] as usize)==253);let ref u8 item=ref items[1];assert((*item as u32)==252);defer done(calls as i64);calls=7;unsafe{trace(calls);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void cast_c_widened_arithmetic_keeps_checked_overflow(){
    let p=retained(b"module main;fn i32 main(){let i32 n=1;let i64 wide=n as i64;let i64 maximum=9223372036854775807;discard maximum+wide;return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void cast_c_rejects_other_c_scalars_and_malformed_casts_transactionally(){
    let pending=retained(b"module main;fn c::uint identity(c::uint n){return n;}fn i32 main(){return 0;}",b"");var bytes::Buffer c=switch(bytes::copy(b"sentinel")){case .ok(b):{b}case .err(_):{assert(false);return;}};
    assert(emit::render(ref pending,entry(ref pending),ref mut c).code==1004);assert(bytes::equal(bytes::view(ref c),b"sentinel"));
    let p=retained(b"module main;fn i32 main(){let i32 n=1;discard n as i64;return 0;}",b"");let b=ir::at(ref p,entry(ref p));
    var usize mode=0;while(mode<4){var v::Vec<ir::Node> nodes=v::empty<ir::Node>();var usize id=0;var usize i=0;
        while(i<ir::node_count(b)){var n=ir::node_at(b,i);if(n.value.kind==23){id=i+1;if(mode==0){n.value.target=0;}else if(mode==1){n.value.left=0;}else if(mode==2){n.text_end=n.text_start;}else{let operand=ir::node_at(b,lookup::node(b,n.value.left)-1);n.value.type=operand.value.type;n.value.target=n.value.type;}}assert(v::push<ir::Node>(ref mut nodes,n).is_ok);i+=1;}
        let text=switch(bytes::copy(ir::text_view(b))){case .ok(t):{t}case .err(_):{assert(false);return;}};
        let changed=ir::assemble_body(1,ir::source_bytes(b),ir::start(b),ir::end(b),nodes,v::empty<ir::Statement>(),v::empty<m::Local>(),v::empty<m::Invocation>(),v::empty<m::Argument>(),v::empty<regions::Scope>(),v::empty<regions::Exit>(),text);
        var w::State state=w::State{.code=0};assert(id!=0 && !scalar::expression(ref p,ref changed,id,ref mut c,ref mut state));assert(state.code==if(mode==3){1004}else{1001});assert(bytes::equal(bytes::view(ref c),b"sentinel"));mode+=1;
    }
}
