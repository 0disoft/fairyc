module type_expr_record;
import enum_types as enums;
import name_resolution as names;
import syntax as syntax;
import type_state as state;
import nominal_types as nominal;
import resource_identity as resources;
import std::vec as v;
import type_check as tc;

pub fn state::Value expression(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected, u32 result_type, usize depth, usize loop_depth) {
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Expression node = syntax::expression_at(parsed,id);
    let ref u8[] op = syntax::expression_text(parsed,id);
    if (node.kind == 10 && syntax::expression_at(parsed,node.left).kind==28) {
        if(!enums::valid(expected)){return tc::error(problem,24,node.start,node.end,0,expected);}
        let usize member=enums::member(parsed,expected,syntax::expression_text(parsed,node.left));if(member==0){return tc::error(problem,25,node.start,node.end,expected,0);}
        let m=syntax::field_at(parsed,member-1);let usize count=syntax::record_field_count(parsed,id);
        if(m.payload_count==0 || count!=m.payload_count){return tc::error(problem,31,node.start,node.end,expected,0);}
        var u8[1024] seen=[0;1024];var usize i=0;var bool terminates=false;
        while(i<count){let usize entry=syntax::record_field(parsed,id,i);let f=syntax::expression_at(parsed,entry);
            let usize target=enums::payload_field(parsed,member,syntax::expression_text(parsed,entry));if(target==0){return tc::error(problem,18,f.start,f.end,expected,0);}
            if(seen[target-1]!=0){return tc::error(problem,17,f.start,f.end,0,0);}seen[target-1]=1;
            let u32 type=state::primitive(parsed,syntax::field_at(parsed,target-1).type_root);let state::Value actual=tc::expression(resolved,tables,problem,f.left,type,result_type,depth+1,loop_depth);
            if(problem.code!=0){return actual;}if(actual.type!=type){return tc::error(problem,1,f.start,f.end,type,actual.type);}
            state::set_flags(tables,entry-1,state::small_id(target));state::set_expressions(tables,entry-1,type);terminates=terminates || actual.terminates;i+=1;
        }
        state::set_flags(tables,id-1,state::small_id(member));return tc::recorded(tables,id,tc::value(expected,terminates));
    }
    if (node.kind == 10) {
        let syntax::Expression target = syntax::expression_at(parsed,node.left);
        if (target.kind != 2) { return tc::error(problem,16,target.start,target.end,0,0); }
        let usize symbol_id = state::uses(tables,node.left - 1);
        if (symbol_id == 0 || (names::symbol_at(resolved,symbol_id).kind != 6 && names::symbol_at(resolved,symbol_id).kind != 9)) { return tc::error(problem,16,target.start,target.end,0,0); }
        let u32 type = state::symbols(tables,symbol_id - 1);
        let syntax::Declaration declaration = syntax::declaration_at(parsed,(type - if(resources::valid(type)){98304}else{256}) as usize);
        if(resources::valid(type) && syntax::module_index(parsed,declaration.start)!=syntax::module_index(parsed,node.start)){return tc::error(problem,6201,node.start,node.end,0,0);}
        var u8[1024] seen = [0;1024]; var bool terminates = false; var usize i = 0;
        while (i < syntax::record_field_count(parsed,id)) {
            let usize entry = syntax::record_field(parsed,id,i); let syntax::Expression field = syntax::expression_at(parsed,entry);
            let ref u8[] name = syntax::expression_text(parsed,entry); let usize at = nominal::field(parsed,type,name);
            if (at == 0) { return tc::error(problem,18,field.start,field.end,type,0); }
            if (seen[at - 1] != 0) { return tc::error(problem,17,field.start,field.end,0,0); } seen[at - 1] = 1;
            let u32 expected_field = nominal::resolve(parsed,syntax::field_at(parsed,at - 1).type_root);
            let state::Value actual = tc::expression(resolved,tables,problem,field.left,expected_field,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            if (actual.type != expected_field) { let syntax::Expression source = syntax::expression_at(parsed,field.left); return tc::error(problem,1,source.start,source.end,expected_field,actual.type); }
            state::set_flags(tables,entry - 1,state::small_id(at)); state::set_expressions(tables,entry - 1,actual.type);
            terminates = terminates || actual.terminates; i += 1;
        }
        if (i != declaration.field_count) { return tc::error(problem,19,node.start,node.end,type,0); }
        state::set_expressions(tables,node.left - 1,type);
        return tc::recorded(tables,id,tc::value(type,terminates));
    }
    if (node.kind == 13) {
        let state::Value condition = tc::expression(resolved,tables,problem,node.left,6,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return condition; }
        if (condition.type != 6) { let syntax::Expression source = syntax::expression_at(parsed,node.left); return tc::error(problem,3,source.start,source.end,6,condition.type); }
        let syntax::Expression branches = syntax::expression_at(parsed,node.right);
        let state::Value consequent = tc::block(resolved,tables,problem,branches.left,expected,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return consequent; }
        if (branches.right == 0) { return tc::recorded(tables,id,tc::value(7,false)); }
        let state::Value alternative = tc::expression(resolved,tables,problem,branches.right,expected,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return alternative; }
        if (consequent.type != alternative.type) {
            let syntax::Expression source = syntax::expression_at(parsed,branches.right); return tc::error(problem,1,source.start,source.end,consequent.type,alternative.type);
        }
        return tc::recorded(tables,id,tc::value(consequent.type,consequent.terminates && alternative.terminates));
    }
    if ((node.kind==12 || node.kind==31)) { let state::Value actual = tc::block(resolved,tables,problem,node.left,expected,result_type,depth + 1,loop_depth); return tc::recorded(tables,id,actual); }
    return tc::error(problem,11,node.start,node.end,0,0);
}
