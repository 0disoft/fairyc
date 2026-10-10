module tests::module_unit_c_owned;
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
@test @effects(console) fn void owned_c_replaces_owners_after_rhs_and_mutates_cleanup_receiver(){
    let p=retained(b"module main;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)resource R{i32 id;cleanup{this.id+=100;unsafe{trace(this.id);}}}@effects(console)fn i32 main(){var R r=R{.id=1};r.id=2;r=R{.id=3};return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void owned_c_drops_active_tagged_payloads_and_nested_fields_in_reverse_order(){
    let p=retained(b"module main;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)resource R{i32 id;cleanup{unsafe{trace(this.id);}}}@effects(console)resource Outer{R left;R right;cleanup{unsafe{trace(100);}}}@effects(console)fn i32 main(){let Outer outer=Outer{.left=R{.id=1},.right=R{.id=2}};let Option<R> some=.some(R{.id=3});let Option<R> none=.none;let Result<R,R> ok=.ok(R{.id=4});let Result<R,R> err=.err(R{.id=5});return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void owned_c_moves_returns_and_unwinds_completed_partial_values_on_propagation(){
    let p=retained(b"module main;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)resource R{i32 id;cleanup{unsafe{trace(this.id);}}}@effects(console)resource Outer{R first;i32 number;cleanup{unsafe{trace(100);}}}fn R make(i32 n){return R{.id=n};}@effects(console)fn void consume(R r){unsafe{trace(20);}}fn Result<i32,i32> fail(){return .err(7);}@effects(console)fn Result<Outer,i32> partial(){let R local=make(3);return .ok(Outer{.first=make(2),.number=fail()?});}fn Result<R,i32> wrapped(){return .ok(make(4));}@effects(console)fn i32 main(){let R moved=make(1);consume(moved);assert(!partial().is_ok);unsafe{trace(9);}assert(wrapped().is_ok);return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void owned_c_invokes_loop_defer_suffix_before_drops_and_keeps_outer_owner(){
    let p=retained(b"module main;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)resource R{i32 id;cleanup{unsafe{trace(this.id);}}}@effects(console)fn void done(i32 n){unsafe{trace(n);}}@effects(console)fn i32 main(){let R outer=R{.id=100};var i32 n=0;while(n<3){let R inner=R{.id=n};defer done(n+10);defer done(n+20);n+=1;if(n==1){continue;}break;}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void owned_c_keeps_provider_nominal_identity_and_shared_foreign_linkage(){
    let p=retained(b"module main;import helper as h;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)resource R{i32 value;cleanup{unsafe{trace(this.value);}}}@effects(console)fn i32 main(){let R local=R{.value=10};let h::R imported=h::make();return 0;}",b"module helper;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)pub resource R{i32 value;cleanup{unsafe{trace(this.value);}}}pub fn R make(){return R{.value=30};}");output(ref p,entry(ref p));
}
@test @effects(console) fn void owned_c_rejects_foreign_text_abi_without_partial_output(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;@link_name(\"pending_descriptor_abi\")unsafe extern C fn void use(ref str n);")).is_ok);let rejected=switch(signatures::build(ref sources)){case .ok(_):{false}case .err(e):{e.code==5102}};assert(rejected);let p=retained(b"module main;fn i32 main(){let text=\"owned\";assert(text.len==5);return 0;}",b"");var bytes::Buffer c=switch(bytes::copy(b"sentinel")){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(emit::render(ref p,0,ref mut c).code==1001);assert(bytes::equal(bytes::view(ref c),b"sentinel"));
}
