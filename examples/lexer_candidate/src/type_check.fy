module type_check;
import name_resolution as names;
import syntax as syntax;
import type_state as state;
import std::bytes as bytes;
import std::vec as v;

pub resource Typed { names::Resolved source; v::Vec<state::Slot> tables; state::Problem problem; cleanup { } }
fn state::Value value(u32 type, bool terminates) { return state::Value { .type, .terminates }; }
fn state::Value error(ref mut state::Problem problem, u32 code, usize start, usize end, u32 expected, u32 actual) {
    if (problem.code == 0) { *problem = state::fail(code,start,end,expected,actual); }
    return value(0,false);
}
fn state::Value recorded(ref mut v::Vec<state::Slot> tables, usize id, state::Value result) {
    state::set_expressions(tables,id - 1,result.type); return result;
}
fn state::Value expression(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected, u32 result_type, usize depth) {
    if (id == 0) { return value(7,false); }
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (depth >= 64) { return error(problem,10,node.start,node.end,0,0); }
    let ref u8[] op = syntax::expression_text(parsed,id);
    if (node.kind == 1) {
        let u32 type = if (syntax::literal_kind(parsed,id) == 5) { if (state::integer(expected)) { expected } else { 1 } } else { 6 };
        if (state::integer(type) && !state::fits(op,type,false)) { return error(problem,2,node.start,node.end,type,0); }
        return recorded(tables,id,value(type,false));
    }
    if (node.kind == 2) {
        let usize symbol_id = state::uses(tables,id - 1); assert(symbol_id != 0);
        let names::Symbol symbol = names::symbol_at(resolved,symbol_id);
        let u32 type = if (symbol.kind == 1) { 8 } else { state::symbols(tables,symbol_id - 1) };
        return recorded(tables,id,value(type,false));
    }
    if (node.kind == 3) { let state::Value actual = expression(resolved,tables,problem,node.left,expected,result_type,depth + 1); return recorded(tables,id,actual); }
    if (node.kind == 4) {
        let bool minus = bytes::equal(op,b"-"); let bool plus = bytes::equal(op,b"+"); let bool complement = bytes::equal(op,b"~");
        let u32 signed_type = if (expected == 2) { 2 } else { 1 };
        let syntax::Expression child = syntax::expression_at(parsed,node.left);
        if (minus && child.kind == 1 && syntax::literal_kind(parsed,node.left) == 5 &&
            !state::fits(syntax::expression_text(parsed,node.left),signed_type,false) &&
            state::fits(syntax::expression_text(parsed,node.left),signed_type,true)) {
            state::set_expressions(tables,node.left - 1,signed_type); state::set_flags(tables,id - 1,1);
            return recorded(tables,id,value(signed_type,false));
        }
        let u32 child_context = if (((minus || plus || complement) && expected == 2) || (complement && state::integer(expected))) { expected } else { 0 };
        let state::Value actual = expression(resolved,tables,problem,node.left,child_context,result_type,depth + 1);
        if (problem.code != 0) { return actual; }
        if (bytes::equal(op,b"!")) {
            if (actual.type != 6) { return error(problem,3,child.start,child.end,6,actual.type); }
            return recorded(tables,id,value(6,actual.terminates));
        }
        if ((minus || plus) && actual.type != 1 && actual.type != 2) { return error(problem,1,child.start,child.end,1,actual.type); }
        if (complement && !state::integer(actual.type)) { return error(problem,9,node.start,node.end,0,actual.type); }
        return recorded(tables,id,actual);
    }
    if (node.kind == 5) {
        let bool arithmetic = state::operation(op); let bool compare = state::comparison(op);
        let u32 context = if (arithmetic && state::integer(expected)) { expected } else { 0 };
        var state::Value left = value(0,false); var state::Value right = value(0,false);
        if ((arithmetic || compare) && state::flexible(tables,node.left - 1) && !state::flexible(tables,node.right - 1)) {
            right = expression(resolved,tables,problem,node.right,context,result_type,depth + 1);
            if (problem.code != 0) { return right; }
            left = expression(resolved,tables,problem,node.left,if (state::integer(right.type)) { right.type } else { context },result_type,depth + 1);
        } else {
            left = expression(resolved,tables,problem,node.left,context,result_type,depth + 1);
            if (problem.code != 0) { return left; }
            right = expression(resolved,tables,problem,node.right,left.type,result_type,depth + 1);
        }
        if (problem.code != 0) { return value(0,false); }
        let syntax::Expression lhs = syntax::expression_at(parsed,node.left); let syntax::Expression rhs = syntax::expression_at(parsed,node.right);
        if (bytes::equal(op,b"&&") || bytes::equal(op,b"||")) {
            if (left.type != 6) { return error(problem,3,lhs.start,lhs.end,6,left.type); }
            if (right.type != 6) { return error(problem,3,rhs.start,rhs.end,6,right.type); }
            return recorded(tables,id,value(6,left.terminates && right.terminates));
        }
        if (left.type != right.type) { return error(problem,1,rhs.start,rhs.end,left.type,right.type); }
        if (!state::integer(left.type) && !(state::equal(op) && left.type == 6)) { return error(problem,9,node.start,node.end,0,left.type); }
        return recorded(tables,id,value(if (compare) { 6 } else { left.type },left.terminates || right.terminates));
    }
    if (node.kind == 6) {
        let usize callee_id = state::uses(tables,node.left - 1); let names::Symbol callee = names::symbol_at(resolved,callee_id);
        if (callee.kind != 1) { let syntax::Expression source = syntax::expression_at(parsed,node.left); return error(problem,6,source.start,source.end,8,state::symbols(tables,callee_id - 1)); }
        state::set_expressions(tables,node.left - 1,8);
        let syntax::Declaration function = syntax::declaration_at(parsed,callee.declaration - 1);
        var usize parameters = 0; var usize entry_index = function.entry_start;
        while (entry_index < function.entry_start + function.entry_count) {
            if (syntax::entry_at(parsed,entry_index).kind == 1) { parameters += 1; } entry_index += 1;
        }
        if (parameters != syntax::call_argument_count(parsed,id)) { return error(problem,5,node.start,node.end,0,0); }
        entry_index = function.entry_start; var usize argument = 0; var bool terminates = false;
        while (entry_index < function.entry_start + function.entry_count) {
            let syntax::Entry entry = syntax::entry_at(parsed,entry_index);
            if (entry.kind == 1) {
                let usize argument_id = syntax::call_argument(parsed,id,argument); let u32 type = state::primitive(parsed,entry.type_root);
                let state::Value actual = expression(resolved,tables,problem,argument_id,type,result_type,depth + 1);
                if (problem.code != 0) { return actual; }
                if (actual.type != type) { let syntax::Expression source = syntax::expression_at(parsed,argument_id); return error(problem,1,source.start,source.end,type,actual.type); }
                terminates = terminates || actual.terminates; argument += 1;
            }
            entry_index += 1;
        }
        let u32 type = state::symbols(tables,callee_id - 1); return recorded(tables,id,value(type,terminates));
    }
    if (node.kind == 12) { let state::Value actual = block(resolved,tables,problem,node.left,expected,result_type,depth + 1); return recorded(tables,id,actual); }
    return error(problem,11,node.start,node.end,0,0);
}
fn state::Value block(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected_tail, u32 result_type, usize depth) {
    let ref syntax::Parsed parsed = names::parsed(resolved); let syntax::Statement body = syntax::statement_at(parsed,id);
    if (depth >= 64) { return error(problem,10,body.start,body.end,0,0); }
    var bool terminates = false; var usize index = 0;
    while (index < syntax::block_statement_count(parsed,id)) {
        let syntax::Statement item = syntax::statement_at(parsed,syntax::block_statement(parsed,id,index));
        if (item.kind == 2) {
            let usize symbol_id = state::entries(tables,item.entry - 1); let names::Symbol symbol = names::symbol_at(resolved,symbol_id);
            let u32 expected = state::primitive(parsed,symbol.type_root);
            let state::Value actual = expression(resolved,tables,problem,item.expression,expected,result_type,depth + 1);
            if (problem.code != 0) { return actual; }
            let syntax::Expression source = syntax::expression_at(parsed,item.expression);
            if (actual.type == 7 || actual.type == 8) { return error(problem,8,source.start,source.end,expected,actual.type); }
            if (expected != 0 && expected != actual.type) { return error(problem,1,source.start,source.end,expected,actual.type); }
            state::set_symbols(tables,symbol_id - 1,actual.type); terminates = terminates || actual.terminates;
        } else if (item.kind == 3) {
            let state::Value actual = expression(resolved,tables,problem,item.expression,result_type,result_type,depth + 1);
            if (problem.code != 0) { return actual; }
            if (actual.type != result_type) {
                let syntax::Expression source = if (item.expression == 0) { syntax::Expression { .kind = 0, .start = item.start, .end = item.end, .token = 0, .left = 0, .right = 0 } } else { syntax::expression_at(parsed,item.expression) };
                return error(problem,1,source.start,source.end,result_type,actual.type);
            }
            terminates = true;
        } else if (item.kind == 4) {
            let usize symbol_id = state::uses(tables,item.target - 1); let names::Symbol symbol = names::symbol_at(resolved,symbol_id);
            let u32 expected = state::symbols(tables,symbol_id - 1); let state::Value target = expression(resolved,tables,problem,item.target,0,result_type,depth + 1);
            let state::Value actual = expression(resolved,tables,problem,item.expression,expected,result_type,depth + 1);
            if (problem.code != 0) { return actual; }
            let syntax::Expression place = syntax::expression_at(parsed,item.target);
            if (symbol.kind != 4) { return error(problem,4,place.start,place.end,0,target.type); }
            if (expected != actual.type) { let syntax::Expression source = syntax::expression_at(parsed,item.expression); return error(problem,1,source.start,source.end,expected,actual.type); }
            if (!bytes::equal(syntax::assignment_operator(parsed,syntax::block_statement(parsed,id,index)),b"=") && !state::integer(expected)) { return error(problem,9,item.start,item.end,0,expected); }
            terminates = terminates || actual.terminates;
        } else if (item.kind == 6) {
            let state::Value actual = expression(resolved,tables,problem,item.expression,0,result_type,depth + 1);
            if (problem.code != 0) { return actual; } terminates = terminates || actual.terminates;
        } else { return error(problem,11,item.start,item.end,0,0); }
        index += 1;
    }
    let state::Value tail = expression(resolved,tables,problem,body.expression,expected_tail,result_type,depth + 1);
    return value(tail.type,terminates || tail.terminates);
}
fn state::Problem collect(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables) {
    let ref syntax::Parsed parsed = names::parsed(resolved);
    if (names::symbol_count(resolved) > 1152 || syntax::expression_count(parsed) > 4096 || syntax::entry_count(parsed) > 1024 || syntax::declaration_count(parsed) > 512) { return state::fail(10,0,syntax::source_view(parsed).len,0,0); }
    if (!state::initialize(tables,names::symbol_count(resolved),syntax::expression_count(parsed),syntax::entry_count(parsed))) { return state::fail(12,0,syntax::source_view(parsed).len,0,0); }
    var usize id = 1;
    while (id <= names::symbol_count(resolved)) {
        let names::Symbol symbol = names::symbol_at(resolved,id); let u32 type = state::primitive(parsed,symbol.type_root);
        state::set_symbols(tables,id - 1,type);
        if (symbol.entry != 0) { state::set_entries(tables,symbol.entry - 1,id); }
        id += 1;
    }
    var usize index = 0;
    while (index < names::reference_count(resolved)) { let names::Reference use = names::reference_at(resolved,index); state::set_uses(tables,use.expression - 1,use.symbol); index += 1; }
    id = 1;
    while (id <= syntax::expression_count(parsed)) {
        let syntax::Expression node = syntax::expression_at(parsed,id); let ref u8[] op = syntax::expression_text(parsed,id);
        let bool flexible = if (node.kind == 1) { syntax::literal_kind(parsed,id) == 5 }
            else if (node.kind == 3) { state::flexible(tables,node.left - 1) }
            else if (node.kind == 4) { (bytes::equal(op,b"+") || bytes::equal(op,b"-") || bytes::equal(op,b"~")) && state::flexible(tables,node.left - 1) }
            else if (node.kind == 5) { state::operation(op) && state::flexible(tables,node.left - 1) && state::flexible(tables,node.right - 1) } else { false };
        state::set_flexible(tables,id - 1,flexible); id += 1;
    }
    var state::Problem problem = state::ok(); index = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration function = syntax::declaration_at(parsed,index);
        if (function.kind == 3) {
            let u32 result_type = state::primitive(parsed,function.result_type);
            let state::Value actual = block(resolved,tables,ref mut problem,function.body,result_type,result_type,0);
            if (problem.code != 0) { return problem; }
            let syntax::Statement body = syntax::statement_at(parsed,function.body);
            if (body.expression != 0) {
                if (actual.type != result_type) { let syntax::Expression tail = syntax::expression_at(parsed,body.expression); return state::fail(1,tail.start,tail.end,result_type,actual.type); }
            } else if (result_type != 7 && !direct_return(parsed,function.body)) { return state::fail(7,body.start,body.end,result_type,7); }
        }
        index += 1;
    }
    return problem;
}
// Match the reference's admitted straight-line function return gate. Branch/loop
// termination and ownership Flow belong to the next component, not these facts.
fn bool direct_return(ref syntax::Parsed parsed, usize body) {
    var usize index = 0;
    while (index < syntax::block_statement_count(parsed,body)) {
        if (syntax::statement_at(parsed,syntax::block_statement(parsed,body,index)).kind == 3) { return true; }
        index += 1;
    }
    return false;
}
pub fn Typed check(names::Resolved resolved) {
    assert(names::problem(ref resolved).code == 0);
    var v::Vec<state::Slot> tables = state::empty(); let state::Problem problem = collect(ref resolved,ref mut tables);
    return Typed { .source = resolved, .tables, .problem };
}
pub fn state::Problem problem(ref Typed typed) { return typed.problem; }
pub fn ref u8[] source_view(ref Typed typed) { return names::source_view(ref typed.source); }
pub fn ref names::Resolved resolved(ref Typed typed) { return ref typed.source; }
pub fn u32 symbol_type(ref Typed typed, usize id) { assert(typed.problem.code == 0 && id > 0 && id <= names::symbol_count(ref typed.source)); return state::symbols(ref typed.tables,id - 1); }
pub fn u32 expression_type(ref Typed typed, usize id) { assert(typed.problem.code == 0 && id > 0 && id <= syntax::expression_count(names::parsed(ref typed.source))); return state::expressions(ref typed.tables,id - 1); }
pub fn bool signed_minimum(ref Typed typed, usize id) { assert(typed.problem.code == 0 && id > 0 && id <= syntax::expression_count(names::parsed(ref typed.source))); return state::flags(ref typed.tables,id - 1) == 1; }
