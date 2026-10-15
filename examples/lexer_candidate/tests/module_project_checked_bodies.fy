module tests::module_project_checked_bodies;
import module_project_compile as compile;
import module_unit_values as values;
import module_unit_moves as moves;
import module_unit_interfaces as interfaces;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_checked_bodies as checking;
import module_unit_cleanup_policy as policy;
import module_unit_body_ir_lower as lower;
import module_unit_cleanup_mir_lower as cleaning;
import module_unit_cleanup_mir_model as clean_ir;
import module_program_artifact_encode as encoding;
import candidate_artifact_wire as wire;
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
    assert(policy::count(compile::checked_cleanup(ref p))==0 && policy::work(compile::checked_cleanup(ref p))>=moves::work(moved));
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
    rejected(b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void f(){let R owner=make();}",6,1108);
    rejected(b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn void f(){let R owner=make();}",6,7102);
}
@test fn void source_seals_reject_same_size_edits_missing_inputs_and_work_exhaustion(){
    let p=source(b"module main;fn i32 main(){return 42;}");let metadata=switch(signatures::build(ref p)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let checked=switch(compile::check_range(ref p,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};let state=compile::checked_state(ref checked);var usize required=0;
    assert(checking::validate_sources(state,ref p,ref mut required).code==0 && required==syntax::source_view(v::at<syntax::Parsed>(ref p,0)).len+1);
    let clone=source(b"module main;fn i32 main(){return 42;}");var usize work=16777216-required;assert(checking::validate_sources(state,ref clone,ref mut work).code==0 && work==16777216);
    work=16777217-required;assert(checking::validate_sources(state,ref p,ref mut work).code==10);work=16777217;assert(checking::validate_sources(state,ref p,ref mut work).code==10 && work==16777217);
    let changed=source(b"module main;fn i32 main(){return 43;}");work=0;let stale=checking::validate_sources(state,ref changed,ref mut work);assert(stale.stage==6 && stale.code==12 && stale.module_index==0 && work==required);
    switch(lower::lower_checked(ref changed,state)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==12 && e.module_index==0);}};
    switch(cleaning::lower_checked(ref changed,state)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==12 && e.module_index==0);}};
    let missing=v::empty<syntax::Parsed>();work=0;assert(checking::validate_sources(state,ref missing,ref mut work).code==12 && work==0);
}
@test fn void checked_cleanup_reuse_preserves_whole_ir_bytes_and_lowering_history(){
    let p=source(b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=40};}@effects(console)fn i32 main(){let R owner=make();return owner.n+2;}");
    let metadata=switch(signatures::build(ref p)){case .ok(p):{p}case .err(_):{assert(false);return;}};let checked=switch(compile::check_range(ref p,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let raw=switch(lower::lower(ref p,compile::checked_values(ref checked),compile::checked_interfaces(ref checked),compile::checked_moves(ref checked))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let reused=switch(lower::lower_checked(ref p,compile::checked_state(ref checked))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(proof::check(ref raw).code==0 && proof::check(ref reused).code==0 && ir::count(ref raw)==ir::count(ref reused) && ir::work(ref raw)==ir::work(ref reused));
    assert(clean_ir::work(ir::cleanup_ir(ref raw))==clean_ir::work(ir::cleanup_ir(ref reused)));
    var usize first=0;let a=switch(encoding::encode(ref raw,ref mut first)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize second=0;let b=switch(encoding::encode(ref reused,ref mut second)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(first==second && bytes::equal(wire::content(ref a),wire::content(ref b)));
}
@test fn void ordinary_lowering_still_verifies_and_stale_metadata_rejects(){
    let p=source(b"module main;fn i32 main(){return 42;}");let metadata=switch(signatures::build(ref p)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let result=switch(compile::range(ref p,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(ir::count(ref result)==1 && proof::check(ref result).code==0);
    let changed=source(b"module main;fn i32 main(){return 43;}");
    switch(compile::check_range(ref changed,ref metadata,0,1)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage>=2 && e.stage<=3 && e.code==12);}};
}

fn bytes::Buffer branch_chain(bool moved){
    var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"module main;resource Owner{i32 n;cleanup{}}fn void take(Owner p){}fn i32 f(i32 n){let Owner owner=Owner{.n=42};return ").is_ok);var usize i=0;
    while(i<40){assert(bytes::append(ref mut out,b"if(n==").is_ok && bytes::append_usize(ref mut out,i).is_ok && bytes::append(ref mut out,b"){0}else ").is_ok);i+=1;}
    assert(bytes::append(ref mut out,if(moved){b"{take(owner);owner.n};}fn i32 main(){return f(99);}"}else{b"{owner.n};}fn i32 main(){return f(99);}"}).is_ok);return out;
}
@test fn void branch_dispatch_counts_ast_edges_and_retains_late_move_errors(){
    var usize mode=0;while(mode<2){let text=branch_chain(mode!=0);let sources=source(bytes::view(ref text));let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        switch(compile::check_range(ref sources,ref metadata,0,1)){case .ok(_):{assert(mode==0);}case .err(e):{assert(mode==1 && e.stage==5 && e.code==1101);assert(bytes::equal(syntax::source_view(v::at<syntax::Parsed>(ref sources,0))[e.start..e.end],b"owner"));}};mode+=1;
    }
}
