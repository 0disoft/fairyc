module type_check;
import name_resolution as names;
import syntax as syntax;
import type_state as state;
import std::vec as v;
import type_expr_tagged as tagged_expr;
import type_expr_scalar as scalar_expr;
import type_expr_call as call_expr;
import type_expr_access as access_expr;
import type_expr_record as record_expr;
import type_statement as statements;
import type_collect as declarations;

pub resource Typed { names::Resolved source; v::Vec<state::Slot> tables; state::Problem problem; cleanup { } }
pub fn state::Value value(u32 type, bool terminates) { return state::Value { .type, .terminates }; }
pub fn state::Value error(ref mut state::Problem problem, u32 code, usize start, usize end, u32 expected, u32 actual) {
    if (problem.code == 0) { *problem = state::fail(code,start,end,expected,actual); }
    return value(0,false);
}
pub fn state::Value recorded(ref mut v::Vec<state::Slot> tables, usize id, state::Value result) {
    state::set_expressions(tables,id - 1,result.type); return result;
}
pub fn state::Value expression(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected, u32 result_type, usize depth, usize loop_depth) {
    if (id == 0) { return value(7,false); }
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (depth >= 64) { return error(problem,10,node.start,node.end,0,0); }
    if(node.kind==30 || node.kind==28 || node.kind==15 || node.kind==6 && syntax::expression_at(parsed,node.left).kind==28){return tagged_expr::expression(resolved,tables,problem,id,expected,result_type,depth,loop_depth);}
    if(node.kind==1 || node.kind==2 || node.kind==23 || node.kind==3 || node.kind==4 || node.kind==5){return scalar_expr::expression(resolved,tables,problem,id,expected,result_type,depth,loop_depth);}
    if(node.kind==6){return call_expr::expression(resolved,tables,problem,id,expected,result_type,depth,loop_depth);}
    if(node.kind==24 || node.kind==29 || node.kind==21 || node.kind==22 || node.kind==26 || node.kind==9 || node.kind==8){return access_expr::expression(resolved,tables,problem,id,expected,result_type,depth,loop_depth);}
    if(node.kind==10 || node.kind==13 || node.kind==12 || node.kind==31){return record_expr::expression(resolved,tables,problem,id,expected,result_type,depth,loop_depth);}
    return error(problem,11,node.start,node.end,0,0);
}
pub fn state::Value block(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected_tail, u32 result_type, usize depth, usize loop_depth) {
    let ref syntax::Parsed parsed = names::parsed(resolved); let syntax::Statement body = syntax::statement_at(parsed,id);
    if (depth >= 64) { return error(problem,10,body.start,body.end,0,0); }
    var bool terminates = false; var usize index = 0;
    while (index < syntax::block_statement_count(parsed,id)) {
        let state::Value checked = statements::statement(resolved,tables,problem,syntax::block_statement(parsed,id,index),result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return checked; }
        terminates = terminates || checked.terminates; index += 1;
    }
    let state::Value tail = expression(resolved,tables,problem,body.expression,expected_tail,result_type,depth + 1,loop_depth);
    return value(tail.type,terminates || tail.terminates);
}
pub fn Typed check(names::Resolved resolved) {
    assert(names::problem(ref resolved).code == 0);
    var v::Vec<state::Slot> tables = state::empty(); let state::Problem problem = declarations::collect(ref resolved,ref mut tables);
    return Typed { .source = resolved, .tables, .problem };
}
pub fn state::Problem problem(ref Typed typed) { return typed.problem; }
pub fn ref u8[] source_view(ref Typed typed) { return names::source_view(ref typed.source); }
pub fn ref names::Resolved resolved(ref Typed typed) { return ref typed.source; }
pub fn u32 symbol_type(ref Typed typed, usize id) { assert(typed.problem.code == 0 && id > 0 && id <= names::symbol_count(ref typed.source)); return state::symbols(ref typed.tables,id - 1); }
pub fn u32 expression_type(ref Typed typed, usize id) { assert(typed.problem.code == 0 && id > 0 && id <= syntax::expression_count(names::parsed(ref typed.source))); return state::expressions(ref typed.tables,id - 1); }
pub fn usize field_target(ref Typed typed, usize id) { assert(typed.problem.code == 0); let u32 flag=state::flags(ref typed.tables,id-1); return if(flag==2048){0}else{flag as usize}; }
pub fn bool result_is_ok(ref Typed typed, usize id) { assert(typed.problem.code==0); return state::flags(ref typed.tables,id-1)==2048; }
pub fn bool moving(ref Typed typed, usize id) { assert(typed.problem.code==0); return state::moving(ref typed.tables,id-1); }
pub fn bool signed_minimum(ref Typed typed, usize id) { assert(typed.problem.code == 0 && id > 0 && id <= syntax::expression_count(names::parsed(ref typed.source))); return syntax::expression_at(names::parsed(ref typed.source),id).kind == 4 && state::flags(ref typed.tables,id - 1) == 1; }

pub fn bool shared_argument(ref Typed p,usize id){return state::flags(ref p.tables,id-1)==4096;}
