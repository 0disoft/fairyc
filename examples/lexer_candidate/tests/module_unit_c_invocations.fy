module tests::module_unit_c_invocations;
import module_unit_c_calls as calls;
import module_unit_body_ir_model as ir;
import module_unit_value_model as values;
import module_unit_region_model as regions;
import std::bytes as bytes;
import std::vec as v;

// These are lookup-only tables, not executable IR or admitted compiler owners.
fn ir::Body rows(usize count,u32 mode){var v::Vec<values::Invocation> entries=v::empty<values::Invocation>();var usize i=0;
    while(i<count){let usize expression=if(mode==1){(count-i)*3}else if(mode==2 && i==count-1){3}else{(i+1)*3};
        assert(v::push<values::Invocation>(ref mut entries,values::Invocation{.kind=1,.expression,.source_index=expression,.target_module=0,.target_declaration=1,.instance=0,.argument_start=0,.argument_count=0,.start=0,.end=0,.checked=true}).is_ok);i+=1;}
    return ir::assemble_body(0,0,0,0,v::empty<ir::Node>(),v::empty<ir::Statement>(),v::empty<values::Local>(),entries,v::empty<values::Argument>(),v::empty<regions::Scope>(),v::empty<regions::Exit>(),bytes::empty());
}
@test
fn void ordered_invocations_reduce_total_work_and_keep_exact_budget_failures(){
    let b=rows(128,0);var usize work=0;let index=switch(calls::invocation_index(ref b,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var u32 code=0;var usize i=0;while(i<128){assert(calls::invocation(ref index,ref b,(i+1)*3,ref mut work,ref mut code)==i+1 && code==0);i+=1;}
    assert(work<128*129/2 && calls::invocation(ref index,ref b,1,ref mut work,ref mut code)==0 && code==0);
    assert(calls::invocation(ref index,ref b,999,ref mut work,ref mut code)==0 && code==0);
    work=1048576-128;let exact=switch(calls::invocation_index(ref b,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(work==1048576);
    assert(calls::invocation(ref exact,ref b,3,ref mut work,ref mut code)==0 && code==1002 && work==1048576);
    work=1048576-127;assert(switch(calls::invocation_index(ref b,ref mut work)){case .ok(_):{false}case .err(e):{e==1002}} && work==1048576);
}
@test
fn void reordered_duplicate_and_short_tables_keep_first_match_semantics(){
    var u32 mode=0;while(mode<3){let b=rows(if(mode==0){3}else{12},mode);var usize work=0;let index=switch(calls::invocation_index(ref b,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var u32 code=0;
        assert(calls::invocation(ref index,ref b,3,ref mut work,ref mut code)==if(mode==1){12}else{1} && code==0);
        assert(calls::invocation(ref index,ref b,2,ref mut work,ref mut code)==0 && code==0);mode+=1;}
    let empty=rows(0,0);var usize work=0;let index=switch(calls::invocation_index(ref empty,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var u32 code=0;
    assert(calls::invocation(ref index,ref empty,3,ref mut work,ref mut code)==0 && code==0 && work==0);
    let other=rows(1,0);assert(calls::invocation(ref index,ref other,3,ref mut work,ref mut code)==0 && code==1001 && work==0);
}
