module tests::module_unit_external_calls;
import module_unit_call_closure as closure;
import module_unit_calls as calls;
import module_unit_bodies as bodies;
import module_unit_bind as binding;
import module_unit_values as values;
import module_unit_moves as moves;
import module_unit_body_ir_lower as lower;
import module_unit_c_emit as emit;
import module_signatures as signatures;
import module_unit_call_contracts as contracts;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] source){
    let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;
}
@test fn void units_type_both_sides_and_require_peer_bodies_for_standalone_c(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::run(40);}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn i32 run(i32 n){return n+2;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let unit=switch(closure::build_unit(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(bodies::count(closure::owned(ref unit))==0 && calls::count(closure::call_plan(ref unit))==1);
    let c=calls::at(closure::call_plan(ref unit),0);assert(!c.foreign && c.target_module==1 && c.target_declaration==2 && c.parameter_count==1);
    let typed=switch(values::build(ref sources,ref metadata,ref unit)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::count(ref typed)==1 && values::checked_calls(ref typed)==1 && values::checked_arguments(ref typed)==1);
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let ir=switch(lower::lower(ref sources,ref typed,closure::imported(ref unit),ref moved)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var bytes::Buffer unlinked=bytes::empty();assert(emit::render(ref ir,1,ref mut unlinked).code!=0);
    let peer=switch(binding::bind(ref sources,ref metadata,1,2)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let ready=switch(closure::build_unit(ref sources,ref metadata,ref peer)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let checked=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref checked)==1);
    let legacy=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bodies::count(closure::owned(ref legacy))==1);
    let complete=switch(values::build(ref sources,ref metadata,ref legacy)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref complete)==2);
}
fn void rejected(ref u8[] caller,ref u8[] provider,u32 expected){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(caller)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let unit=switch(closure::build_unit(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{assert(p.code==expected && p.module_index==0 && p.start<p.end);return;}};
    switch(values::build(ref sources,ref metadata,ref unit)){case .ok(_):{assert(false);}case .err(p):{assert(expected==0 && p.code!=0 && p.module_index==0 && p.start<p.end);}};
}
@test fn void unit_calls_preserve_argument_effect_and_must_use_checks(){
    rejected(b"module main;import provider as p;fn i32 main(){return p::run(true);}",b"module provider;pub fn i32 run(i32 n){return n;}",0);
    rejected(b"module main;import provider as p;fn i32 main(){return p::run(1);}",b"module provider;@effects(console) pub fn i32 run(i32 n){return n;}",7102);
    rejected(b"module main;import provider as p;fn i32 main(){p::run(1);return 0;}",b"module provider;@must_use pub fn i32 run(i32 n){return n;}",4202);
    rejected(b"module main;import provider as p;fn i32 main(){return p::run(1);}",b"module provider;pub unsafe fn i32 run(i32 n){return n;}",5101);
}
fn void contract(ref u8[] caller,ref u8[] provider,u32 expected){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(caller)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var usize work=0;let p=contracts::check(ref sources,ref metadata,0,3,false,ref mut work);
    assert(p.code==expected && work>0 && work<=1048576);if(expected!=0){assert(p.module_index==0 && p.start<p.end);}
}
@test fn void nested_calls_keep_authority_handling_and_unsafe_context(){
    contract(b"module main;import provider as p;fn i32 main(){let i32[1] xs=[1];for(let ref i32 x in xs){p::run(*x);}return 0;}",b"module provider;@must_use pub fn i32 run(i32 n){return n;}",4202);
    contract(b"module main;import provider as p;fn i32 main(){return if(true){p::run(1)}else{0};}",b"module provider;@effects(console) pub fn i32 run(i32 n){return n;}",7102);
    contract(b"module main;import provider as p;fn i32 main(){discard p::run(1);return p::run(2);}",b"module provider;@must_use pub fn i32 run(i32 n){return n;}",0);
    contract(b"module main;import provider as p;fn i32 main(){return unsafe{p::run(1)};}",b"module provider;pub unsafe fn i32 run(i32 n){return n;}",0);
    contract(b"module main;import provider as p;unsafe fn i32 main(){return p::run(1);}",b"module provider;pub unsafe fn i32 run(i32 n){return n;}",0);
    contract(b"module main;import provider as p;@effects(console) fn i32 main(){return p::run(1);}",b"module provider;@effects(console) pub fn i32 run(i32 n){return n;}",0);
}
