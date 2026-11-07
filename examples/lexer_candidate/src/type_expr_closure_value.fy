module type_expr_closure_value;
import syntax as syntax;
import name_resolution as names;
import nominal_types as nominal;
import type_state as state;
import type_check as tc;
import std::vec as v;

// Only preparation can publish kind 36 and its source-owned helper receipt.
// The written source still owns signature, captures, body and byte intervals.
pub fn state::Value construct(ref names::Resolved resolved,ref mut v::Vec<state::Slot> tables,ref mut state::Problem problem,usize id,u32 result_type,usize depth,usize loop_depth){
    let p=names::parsed(resolved);let n=syntax::expression_at(p,id);
    if(n.kind!=36 || n.left==0 || syntax::expression_at(p,n.left).kind!=2){return tc::error(problem,4502,n.start,n.end,0,0);}
    let target=state::uses(tables,n.left-1);
    if(target==0 || target>names::symbol_count(resolved)){return tc::error(problem,4502,n.start,n.end,0,0);}
    let symbol=names::symbol_at(resolved,target);
    if(symbol.kind!=1 || symbol.declaration==0 || symbol.declaration>syntax::declaration_count(p)){return tc::error(problem,4502,n.start,n.end,0,0);}
    let count=syntax::closure_value_captures(p,symbol.declaration);let captures=syntax::call_argument_count(p,id);
    let function=syntax::declaration_at(p,symbol.declaration-1);
    if(count==0 || count!=captures+1 || captures>4 || function.unsafe_end!=0 || function.generic_parameters!=0 || nominal::resolve(p,function.result_type)!=1){return tc::error(problem,4502,n.start,n.end,0,0);}
    var usize parameters=0;var usize i=0;
    while(i<function.entry_count){let e=syntax::entry_at(p,function.entry_start+i);if(e.kind==1){if(nominal::resolve(p,e.type_root)!=1){return tc::error(problem,4502,e.start,e.end,1,0);}parameters+=1;}i+=1;}
    if(parameters!=captures+1){return tc::error(problem,4502,n.start,n.end,0,0);}
    state::set_expressions(tables,n.left-1,8);var bool terminates=false;i=0;
    while(i<captures){let child=syntax::call_argument(p,id,i);let actual=tc::expression(resolved,tables,problem,child,1,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}if(actual.type!=1){let e=syntax::expression_at(p,child);return tc::error(problem,1,e.start,e.end,1,actual.type);}terminates=terminates || actual.terminates;i+=1;}
    return tc::recorded(tables,id,tc::value(13,terminates));
}

pub fn state::Value invoke(ref names::Resolved resolved,ref mut v::Vec<state::Slot> tables,ref mut state::Problem problem,usize id,u32 result_type,usize depth,usize loop_depth){
    let p=names::parsed(resolved);let n=syntax::expression_at(p,id);
    if(n.kind!=6 || n.left==0 || syntax::expression_at(p,n.left).kind!=2 || syntax::call_argument_count(p,id)!=1){return tc::error(problem,4502,n.start,n.end,0,0);}
    let target=state::uses(tables,n.left-1);
    if(target==0 || target>names::symbol_count(resolved)){return tc::error(problem,4502,n.start,n.end,0,0);}
    let symbol=names::symbol_at(resolved,target);
    if((symbol.kind!=2 && symbol.kind!=3) || state::symbols(tables,target-1)!=13){return tc::error(problem,4502,n.start,n.end,0,0);}
    let callable=tc::expression(resolved,tables,problem,n.left,13,result_type,depth+1,loop_depth);if(problem.code!=0){return callable;}
    let child=syntax::call_argument(p,id,0);let actual=tc::expression(resolved,tables,problem,child,1,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}
    if(actual.type!=1){let e=syntax::expression_at(p,child);return tc::error(problem,1,e.start,e.end,1,actual.type);}
    return tc::recorded(tables,id,tc::value(1,callable.terminates || actual.terminates));
}
