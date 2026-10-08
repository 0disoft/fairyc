module type_statement;
import scalar_views as views;
import reference_types as refs;
import resource_abilities as abilities;
import name_resolution as names;
import syntax as syntax;
import type_state as state;
import nominal_types as nominal;
import type_control as control;
import std::bytes as bytes;
import std::vec as v;
import type_check as checking;

pub fn state::Value statement(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 result_type, usize depth, usize loop_depth) {
    if (id == 0) { return checking::value(7,false); }
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Statement item = syntax::statement_at(parsed,id);
    if (depth >= 64) { return checking::error(problem,10,item.start,item.end,0,0); }
        if (item.kind == 2) {
            let usize symbol_id = state::entries(tables,item.entry - 1); let names::Symbol symbol = names::symbol_at(resolved,symbol_id);
            let u32 expected = nominal::resolve(parsed,symbol.type_root);
            let state::Value actual = checking::expression(resolved,tables,problem,item.expression,expected,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            let syntax::Expression source = syntax::expression_at(parsed,item.expression);
            if (actual.type == 7 || actual.type == 8) { return checking::error(problem,8,source.start,source.end,expected,actual.type); }
            if (expected != 0 && expected != actual.type) { return checking::error(problem,1,source.start,source.end,expected,actual.type); }
            state::set_symbols(tables,symbol_id - 1,actual.type); return checking::value(7,actual.terminates);
        } else if (item.kind == 3) {
            let state::Value actual = checking::expression(resolved,tables,problem,item.expression,result_type,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            if (actual.type != result_type) {
                let syntax::Expression source = if (item.expression == 0) { syntax::Expression { .kind = 0, .start = item.start, .end = item.end, .token = 0, .left = 0, .right = 0 } } else { syntax::expression_at(parsed,item.expression) };
                return checking::error(problem,1,source.start,source.end,result_type,actual.type);
            }
            return checking::value(7,true);
        } else if (item.kind == 4) {
            let state::Value target = checking::expression(resolved,tables,problem,item.target,0,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return target; }
            var usize place_id = item.target;
            while (syntax::expression_at(parsed,place_id).kind == 3) { place_id = syntax::expression_at(parsed,place_id).left; }
            let syntax::Expression checked_place = syntax::expression_at(parsed,place_id);
            if (checked_place.kind == 9 && views::valid(state::expressions(tables,checked_place.left - 1)) && !views::mutable(state::expressions(tables,checked_place.left - 1)) || checked_place.kind == 8 && state::flags(tables,place_id - 1) == 0) { return checking::error(problem,4,checked_place.start,checked_place.end,0,target.type); }
            var usize root = item.target; var usize fields = 0;
            while (syntax::expression_at(parsed,root).kind == 8 || syntax::expression_at(parsed,root).kind == 3 || syntax::expression_at(parsed,root).kind == 9 || syntax::expression_at(parsed,root).kind==4 && bytes::equal(syntax::expression_text(parsed,root),b"*")) {
                if (syntax::expression_at(parsed,root).kind == 8) { fields += 1; }
                root = syntax::expression_at(parsed,root).left;
            }
            if (fields > 1 && nominal::contains_tagged(parsed,state::expressions(tables,root - 1),0)) { return checking::error(problem,4,checked_place.start,checked_place.end,0,target.type); }
            let usize symbol_id = if (syntax::expression_at(parsed,root).kind == 2) { state::uses(tables,root - 1) } else { 0 };
            let bool indirect=refs::mutable(state::expressions(tables,root-1)) && item.target!=root;
            if(checked_place.kind==4 && (!indirect || !bytes::equal(syntax::assignment_operator(parsed,id),b"="))){return checking::error(problem,4,checked_place.start,checked_place.end,0,target.type);}
            let u32 expected = target.type;
            let state::Value actual = checking::expression(resolved,tables,problem,item.expression,expected,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            let syntax::Expression place = syntax::expression_at(parsed,item.target);
            if (symbol_id == 0 || (names::symbol_at(resolved,symbol_id).kind != 4 && names::symbol_at(resolved,symbol_id).kind != 10 && !indirect && !(checked_place.kind == 9 && views::mutable(state::expressions(tables,checked_place.left - 1))))) { return checking::error(problem,4,place.start,place.end,0,target.type); }
            if(names::symbol_at(resolved,symbol_id).kind==10 && fields>1){return checking::error(problem,4,place.start,place.end,0,target.type);}
            if(names::symbol_at(resolved,symbol_id).kind==10 && (fields==0 || nominal::contains_resource(parsed,expected,0))){return checking::error(problem,48,place.start,place.end,0,target.type);}
            if (expected != actual.type) { let syntax::Expression source = syntax::expression_at(parsed,item.expression); return checking::error(problem,1,source.start,source.end,expected,actual.type); }
            if (!bytes::equal(syntax::assignment_operator(parsed,id),b"=") && !state::integer(expected)) { return checking::error(problem,9,item.start,item.end,0,expected); }
            return checking::value(7,actual.terminates);
        } else if(item.kind==16){
            let actual=checking::expression(resolved,tables,problem,item.expression,0,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}
            let e=syntax::expression_at(parsed,item.expression);let usize local=state::uses(tables,item.expression-1);
            if(e.kind!=2 || local==0 || (names::symbol_at(resolved,local).kind!=2 && names::symbol_at(resolved,local).kind!=3 && names::symbol_at(resolved,local).kind!=4) || !abilities::burnable(parsed,actual.type)){return checking::error(problem,51,item.start,item.end,0,actual.type);}return checking::value(7,false);
        } else if (item.kind == 6 || item.kind == 12 || item.kind == 13) {
            let state::Value actual = checking::expression(resolved,tables,problem,item.expression,0,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            if (item.kind == 13 && (syntax::expression_at(parsed,item.expression).kind != 6 || actual.type != 7)) {
                return checking::error(problem,15,item.start,item.end,7,actual.type);
            }
            if (item.kind == 13) {
                var usize i = 0; while (i < syntax::call_argument_count(parsed,item.expression)) {
                    let usize argument = syntax::call_argument(parsed,item.expression,i);
                    if (state::expressions(tables,argument-1)<1 || state::expressions(tables,argument-1)>6) { return checking::error(problem,29,item.start,item.end,0,0); } i += 1;
                }
            }
            return checking::value(7,actual.terminates);
        }

    if (item.kind == 9 || item.kind == 10) {
        if (loop_depth == 0) { return checking::error(problem,13,item.start,item.end,0,0); }
        return checking::value(7,false);
    }
    if (item.kind == 11 || item.kind == 8) {
        let state::Value condition = checking::expression(resolved,tables,problem,item.expression,6,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return condition; }
        if (condition.type != 6) { let syntax::Expression source = syntax::expression_at(parsed,item.expression); return checking::error(problem,3,source.start,source.end,6,condition.type); }
        if (item.kind == 8) {
            let state::Value body = checking::block(resolved,tables,problem,item.first,7,result_type,depth + 1,loop_depth + 1);
            if (problem.code != 0) { return body; }
        }
        return checking::value(7,false);
    }
    if(item.kind==18){
        let u32 item_type=nominal::resolve(parsed,item.second);let view=checking::expression(resolved,tables,problem,item.expression,0,result_type,depth+1,loop_depth);
        if(problem.code!=0){return view;}if(item_type!=3 || view.type!=10){return checking::error(problem,11,item.start,item.end,3,item_type);}
        var usize root=item.expression;if(syntax::expression_at(parsed,root).kind==26){root=syntax::expression_at(parsed,root).left;}
        var usize steps=0;while(syntax::expression_at(parsed,root).kind==3 && steps<64){root=syntax::expression_at(parsed,root).left;steps+=1;}
        if(syntax::expression_at(parsed,root).kind!=2){return checking::error(problem,11,item.start,item.end,10,view.type);}
        let usize binding=state::uses(tables,root-1);if(binding==0 || (names::symbol_at(resolved,binding).kind!=2 && names::symbol_at(resolved,binding).kind!=3)){return checking::error(problem,11,item.start,item.end,10,view.type);}
        state::set_expressions(tables,item.target-1,3);
        let body=checking::block(resolved,tables,problem,item.first,7,result_type,depth+1,loop_depth+1);return if(problem.code!=0){body}else{checking::value(7,false)};
    }
    if (item.kind == 14) {
        let syntax::Statement header = syntax::statement_at(parsed,item.second);
        let state::Value initial = statement(resolved,tables,problem,header.first,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return initial; }
        if (header.first != 0) {
            let syntax::Statement clause = syntax::statement_at(parsed,header.first);
            if (!control::plain_header(parsed,clause.expression,0)) { return checking::error(problem,14,clause.start,syntax::expression_at(parsed,clause.expression).end,0,0); }
        }
        if (item.expression != 0) {
            let state::Value condition = checking::expression(resolved,tables,problem,item.expression,6,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return condition; }
            let syntax::Expression source = syntax::expression_at(parsed,item.expression);
            if (!control::plain_header(parsed,item.expression,0)) { return checking::error(problem,14,source.start,source.end,0,0); }
            if (condition.type != 6) { return checking::error(problem,3,source.start,source.end,6,condition.type); }
        }
        let state::Value body = checking::block(resolved,tables,problem,item.first,7,result_type,depth + 1,loop_depth + 1);
        if (problem.code != 0) { return body; }
        let state::Value post = statement(resolved,tables,problem,header.second,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return post; }
        if (header.second != 0) {
            let syntax::Statement clause = syntax::statement_at(parsed,header.second);
            if (!control::plain_header(parsed,clause.expression,0)) { return checking::error(problem,14,clause.start,syntax::expression_at(parsed,clause.expression).end,0,0); }
        }
        return checking::value(7,false);
    }
    return checking::error(problem,11,item.start,item.end,0,0);
}
