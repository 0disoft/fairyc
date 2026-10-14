module tests::module_project_checked_bodies;
import module_project_compile as compile;
import module_unit_values as values;
import module_unit_moves as moves;
import module_unit_interfaces as interfaces;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let raw=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let tokens=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn v::Vec<syntax::Parsed> source(ref u8[] text){var v::Vec<syntax::Parsed> p=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut p,parsed(text)).is_ok);return p;}
fn compile::CheckedBodies detached(){
    var p=source(b"module main;import provider as p;resource Owner{i32 value;cleanup{}}fn i32 main(){let Owner owner=Owner{.value=40};return p::id<i32>(owner.value)+2;}");
    assert(v::push<syntax::Parsed>(ref mut p,parsed(b"module provider;pub fn T id<T>(T n){return n;}")).is_ok);
    let metadata=switch(signatures::build(ref p)){case .ok(p):{p}case .err(_):{assert(false);return detached();}};
    return switch(compile::check_range(ref p,ref metadata,0,2)){case .ok(p):{p}case .err(_):{assert(false);detached()}};
}
@test fn void checked_owner_retains_generic_closure_values_and_cleanup_facts(){
    // The source arenas and signature owner are already destroyed here.
    let p=detached();let range=compile::checked_range(ref p);assert(range.first==0 && range.stop==2);
    let typed=compile::checked_values(ref p);assert(values::count(typed)>=2 && values::type_count(typed)>0 && values::checked_calls(typed)>0 && values::work(typed)>0);
    let moved=compile::checked_moves(ref p);assert(moves::count(moved)==0 && moves::cleanup_count(moved)>0 && moves::work(moved)>0);
    assert(interfaces::callable_count(compile::checked_interfaces(ref p))>0);
}
fn void rejected(ref u8[] text,u32 stage,u32 code){
    let p=source(text);let metadata=switch(signatures::build(ref p)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let first=switch(compile::check_range(ref p,ref metadata,0,1)){case .ok(_):{assert(false);return;}case .err(e):{e}};
    let second=switch(compile::range(ref p,ref metadata,0,1)){case .ok(_):{assert(false);return;}case .err(e):{e}};
    assert(first.stage==stage && first.code==code && first.module_index==0 && first.start<first.end && first.first==0 && first.stop==1);
    assert(first.stage==second.stage && first.code==second.code && first.start==second.start && first.end==second.end && first.expected==second.expected && first.actual==second.actual);
    assert(first.declaration_start==second.declaration_start && first.declaration_end==second.declaration_end && first.move_start==second.move_start && first.move_end==second.move_end);
}
@test fn void checking_rejects_types_moves_and_borrow_escape_before_lowering(){
    rejected(b"module main;fn i32 main(){return true;}",4,1009);
    rejected(b"module main;resource Owner{i32 value;cleanup{}}fn void take(Owner p){}fn void main(){let Owner owner=Owner{.value=1};take(owner);take(owner);}",5,1101);
    rejected(b"module main;fn ref i32 bad(){let i32 local=1;return ref local;}",7,2102);
}
@test fn void ordinary_lowering_still_verifies_and_stale_metadata_rejects(){
    let p=source(b"module main;fn i32 main(){return 42;}");let metadata=switch(signatures::build(ref p)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let result=switch(compile::range(ref p,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(ir::count(ref result)==1 && proof::check(ref result).code==0);
    let changed=source(b"module main;fn i32 main(){return 43;}");
    switch(compile::check_range(ref changed,ref metadata,0,1)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage>=2 && e.stage<=3 && e.code==12);}};
}
