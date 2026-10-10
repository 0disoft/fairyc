module tests::module_unit_c_pointers;
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


@test @effects(console) fn void pointer_c_transports_null_foreign_results_and_void_output_slots(){
    let p=retained(b"module main;@link_name(\"provide_i32\")unsafe extern C fn ptr i32 provide();@link_name(\"read_i32\")unsafe extern C fn i32 read(ptr i32 n);@link_name(\"fill_void_slot\")unsafe extern C fn void fill(ptr mut ptr mut void out);@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);fn ptr i32 pass(ptr i32 n){return n;}@effects(console)fn i32 main(){var ptr i32 n=null;assert(n==null && null==n);unsafe{n=provide();assert(n!=null && pass(n)==n);trace(read(n));var ptr mut void out=null;fill(ref mut out as ptr mut ptr mut void);assert(out!=null);let ptr mut void copy=out;assert(copy==out);trace(77);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void pointer_c_forwards_mutable_and_shared_scalar_storage(){
    let p=retained(b"module main;import helper as h;@link_name(\"write_i32\")unsafe extern C fn void write(ptr mut i32 n,i32 value);@link_name(\"read_i32\")unsafe extern C fn i32 read(ptr i32 n);@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 main(){var i32 n=1;let ref mut i32 parent=ref mut n;unsafe{write(parent as ptr mut i32,9);trace(read(parent as ptr i32));write(h::address<i32>(parent),6);trace(read(parent as ptr i32));}assert(*parent==6);return 0;}",b"module helper;pub unsafe fn ptr mut T address<T>(ref mut T n){return n as ptr mut T;}");output(ref p,entry(ref p));
}
@test @effects(console) fn void pointer_c_uses_owner_and_view_data_for_opaque_crossings(){
    let p=retained(b"module main;@link_name(\"write_opaque_i32\")unsafe extern C fn void write(ptr mut void n,i32 value);@link_name(\"write_u8\")unsafe extern C fn void byte(ptr mut u8 n,u8 value);@link_name(\"read_u8\")unsafe extern C fn i32 read(ptr u8 n);@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)resource R{i32 n;cleanup{unsafe{trace(this.n);}}}@effects(console)fn i32 main(){var R r=R{.n=3};var u8[2] bytes=[1,2];var i32[2] cells=[4,5];{let ref mut u8[] data=ref mut bytes;unsafe{byte(data as ptr mut u8,8);}}unsafe{write(ref mut r as ptr mut void,9);write(ref mut cells as ptr mut void,5);let ref u8[] data=ref bytes;trace(read(data as ptr u8));trace(cells[0]);trace(r.n);}assert(bytes[0]==8 && cells[0]==5 && r.n==9);return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void pointer_c_preserves_nominal_identity_across_internal_calls(){
    let p=retained(b"module main;struct S{i32 n;}fn ptr S address(ref S n){return unsafe{n as ptr S};}fn ptr S pass(ptr S n){return n;}fn i32 main(){let S first=S{.n=1};let S second=S{.n=1};let ptr S a=address(ref first);assert(a!=null && a==pass(address(ref first)) && a!=address(ref second));let ptr S empty=null;assert(empty==null && empty!=a);return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void pointer_c_rejects_unverified_nominal_abi_and_forged_unsafe_casts(){
    let pending=retained(b"module main;@repr(C)struct S{i32 n;}@link_name(\"nominal_unverified\")unsafe extern C fn void use(ptr S n);fn i32 main(){let S n=S{.n=1};unsafe{use(ref n as ptr S);}return 0;}",b"");var bytes::Buffer c=switch(bytes::copy(b"sentinel")){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(emit::render(ref pending,entry(ref pending),ref mut c).code==1004);assert(bytes::equal(bytes::view(ref c),b"sentinel"));
    let p=retained(b"module main;fn ptr u32 spare(ptr u32 n){return n;}fn i32 main(){var i32 n=1;discard unsafe{ref mut n as ptr mut i32};return 0;}",b"");let usize frame=entry(ref p);let b=ir::at(ref p,frame);var usize wrong=0;var usize t=1;while(t<=clean_ir::type_count(ir::cleanup_ir(ref p))){let node=clean_ir::type_at(ir::cleanup_ir(ref p),t);if(node.kind==12 && node.first==0 && node.scalar==1048584){wrong=t;}t+=1;}assert(wrong!=0);
    var usize mode=0;while(mode<3){var v::Vec<ir::Node> nodes=v::empty<ir::Node>();var usize i=0;while(i<ir::node_count(b)){var n=ir::node_at(b,i);if(n.value.kind==23){if(mode==0){n.value.builtin.unsafe_context=false;}if(mode==2){n.value.type=wrong;n.value.target=wrong;}}if(mode==1 && n.value.kind==31){n.value.kind=12;n.syntax.kind=12;}assert(v::push<ir::Node>(ref mut nodes,n).is_ok);i+=1;}
        let text=switch(bytes::copy(ir::text_view(b))){case .ok(t):{t}case .err(_):{assert(false);return;}};let changed=ir::assemble_body(ir::body(b),ir::source_bytes(b),ir::start(b),ir::end(b),nodes,v::empty<ir::Statement>(),v::empty<m::Local>(),v::empty<m::Invocation>(),v::empty<m::Argument>(),v::empty<regions::Scope>(),v::empty<regions::Exit>(),text);var usize work=0;assert(proof::check_casts(ref p,ref changed,frame,ref mut work).code==1001);mode+=1;
    }
}
