module type_expr_scalar;
import pointer_types as pointers;
import reference_types as refs;
import name_resolution as names;
import syntax as syntax;
import scalar_casts as casts;
import unsafe_scopes as unsafe_scopes;
import type_state as state;
import nominal_types as nominal;
import std::bytes as bytes;
import std::vec as v;
import type_check as checking;

pub fn state::Value expression(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected, u32 result_type, usize depth, usize loop_depth) {
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Expression node = syntax::expression_at(parsed,id);
    let ref u8[] op = syntax::expression_text(parsed,id);
    if (node.kind == 1) {
        if(bytes::equal(op,b"null")){if(!pointers::valid(expected)){return checking::error(problem,1,node.start,node.end,expected,0);}return checking::recorded(tables,id,checking::value(expected,false));}
        let u32 literal = syntax::literal_kind(parsed,id);
        if (literal == 7 || literal == 8 || literal == 9) { return checking::recorded(tables,id,checking::value(if (literal == 7) { 9 } else if(literal==9){12}else{10},false)); }
        let u32 type = if (syntax::literal_kind(parsed,id) == 5) { if (state::integer(expected)) { expected } else { 1 } } else { 6 };
        if (state::integer(type) && !state::fits(op,type,false)) { return checking::error(problem,2,node.start,node.end,type,0); }
        return checking::recorded(tables,id,checking::value(type,false));
    }
    if (node.kind == 2) {
        let usize symbol_id = state::uses(tables,id - 1); assert(symbol_id != 0);
        let names::Symbol symbol = names::symbol_at(resolved,symbol_id);
        if (symbol.kind == 6 || symbol.kind == 7 || symbol.kind==9) { return checking::error(problem,16,node.start,node.end,0,0); }
        let u32 type = if (symbol.kind == 1 || symbol.kind == 5 || symbol.kind==11) { 8 } else { state::symbols(tables,symbol_id - 1) };
        return checking::recorded(tables,id,checking::value(type,false));
    }
    if(node.kind==23){let actual=checking::expression(resolved,tables,problem,node.left,0,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}let u32 target=nominal::resolve(parsed,node.right);
        if(pointers::cast_allowed(actual.type,target)){
            if(pointers::inner(target)==7 || target==1048601){var usize child=node.left;while(syntax::expression_at(parsed,child).kind==3){child=syntax::expression_at(parsed,child).left;}let e=syntax::expression_at(parsed,child);if(e.kind!=2 && !((e.kind==21 || e.kind==22) && syntax::expression_at(parsed,e.left).kind==2)){return checking::error(problem,52,node.start,node.end,target,actual.type);}}
            if(!unsafe_scopes::allowed(parsed,node.start,node.end)){return checking::error(problem,5101,node.start,node.end,0,0);}
        }else if(!casts::allowed(actual.type,target)){return checking::error(problem,52,node.start,node.end,target,actual.type);}
        return checking::recorded(tables,id,checking::value(target,actual.terminates));}
    if (node.kind == 3) { let state::Value actual = checking::expression(resolved,tables,problem,node.left,expected,result_type,depth + 1,loop_depth); return checking::recorded(tables,id,actual); }
    if (node.kind == 4) {
        let bool minus = bytes::equal(op,b"-"); let bool plus = bytes::equal(op,b"+"); let bool complement = bytes::equal(op,b"~");
        let u32 signed_type = if (expected == 2) { 2 } else { 1 };
        let syntax::Expression child = syntax::expression_at(parsed,node.left);
        if (minus && child.kind == 1 && syntax::literal_kind(parsed,node.left) == 5 &&
            !state::fits(syntax::expression_text(parsed,node.left),signed_type,false) &&
            state::fits(syntax::expression_text(parsed,node.left),signed_type,true)) {
            state::set_expressions(tables,node.left - 1,signed_type); state::set_flags(tables,id - 1,1);
            return checking::recorded(tables,id,checking::value(signed_type,false));
        }
        let u32 child_context = if (((minus || plus || complement) && expected == 2) || (complement && state::integer(expected))) { expected } else { 0 };
        let state::Value actual = checking::expression(resolved,tables,problem,node.left,child_context,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return actual; }
        if(bytes::equal(op,b"*")){if(!refs::valid(actual.type)){return checking::error(problem,16,node.start,node.end,0,actual.type);}return checking::recorded(tables,id,checking::value(refs::inner(actual.type),actual.terminates));}
        if (bytes::equal(op,b"!")) {
            if (actual.type != 6) { return checking::error(problem,3,child.start,child.end,6,actual.type); }
            return checking::recorded(tables,id,checking::value(6,actual.terminates));
        }
        if ((minus || plus) && actual.type != 1 && actual.type != 2) { return checking::error(problem,1,child.start,child.end,1,actual.type); }
        if (complement && !state::integer(actual.type)) { return checking::error(problem,9,node.start,node.end,0,actual.type); }
        return checking::recorded(tables,id,actual);
    }
    if (node.kind == 5) {
        let bool arithmetic = state::operation(op); let bool compare = state::comparison(op);
        let u32 context = if (arithmetic && state::integer(expected)) { expected } else { 0 };
        var state::Value left = checking::value(0,false); var state::Value right = checking::value(0,false);
        if(compare && bytes::equal(syntax::expression_text(parsed,node.left),b"null")){
            right=checking::expression(resolved,tables,problem,node.right,0,result_type,depth+1,loop_depth);if(problem.code!=0){return right;}
            left=checking::expression(resolved,tables,problem,node.left,right.type,result_type,depth+1,loop_depth);
        } else if ((arithmetic || compare) && state::flexible(tables,node.left - 1) && !state::flexible(tables,node.right - 1)) {
            right = checking::expression(resolved,tables,problem,node.right,context,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return right; }
            left = checking::expression(resolved,tables,problem,node.left,if (state::integer(right.type)) { right.type } else { context },result_type,depth + 1,loop_depth);
        } else {
            left = checking::expression(resolved,tables,problem,node.left,context,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return left; }
            right = checking::expression(resolved,tables,problem,node.right,left.type,result_type,depth + 1,loop_depth);
        }
        if (problem.code != 0) { return checking::value(0,false); }
        let syntax::Expression lhs = syntax::expression_at(parsed,node.left); let syntax::Expression rhs = syntax::expression_at(parsed,node.right);
        if (bytes::equal(op,b"&&") || bytes::equal(op,b"||")) {
            if (left.type != 6) { return checking::error(problem,3,lhs.start,lhs.end,6,left.type); }
            if (right.type != 6) { return checking::error(problem,3,rhs.start,rhs.end,6,right.type); }
            return checking::recorded(tables,id,checking::value(6,left.terminates && right.terminates));
        }
        if (left.type != right.type) { return checking::error(problem,1,rhs.start,rhs.end,left.type,right.type); }
        if (!state::integer(left.type) && !(state::equal(op) && (left.type == 6 || pointers::valid(left.type)))) { return checking::error(problem,9,node.start,node.end,0,left.type); }
        return checking::recorded(tables,id,checking::value(if (compare) { 6 } else { left.type },left.terminates || right.terminates));
    }
    return checking::error(problem,11,node.start,node.end,0,0);
}
