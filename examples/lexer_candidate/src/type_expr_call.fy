module type_expr_call;
import reference_types as refs;
import name_resolution as names;
import syntax as syntax;
import effect_declarations as effects;
import unsafe_scopes as unsafe_scopes;
import raw_view_intrinsics as raw;
import storage_intrinsics as storage;
import type_state as state;
import nominal_types as nominal;
import std::vec as v;
import type_check as tc;

pub fn state::Value expression(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected, u32 result_type, usize depth, usize loop_depth) {
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Expression node = syntax::expression_at(parsed,id);
    let ref u8[] op = syntax::expression_text(parsed,id);
    if (node.kind == 6) {
        let u32 storage_code=storage::call(parsed,id);
        if(storage_code!=0){
            if(!storage::allowed(parsed,id)){return tc::error(problem,5102,node.start,node.end,0,0);}
            if(!unsafe_scopes::allowed(parsed,node.start,node.end)){return tc::error(problem,5101,node.start,node.end,0,0);}
            let u32 element=storage::element(parsed,id);let u32 type=storage::result(storage_code,element);
            if(type==0){return tc::error(problem,4502,node.start,node.end,0,0);}
            let usize count=storage::arguments(storage_code);if(syntax::call_argument_count(parsed,id)!=count){return tc::error(problem,5,node.start,node.end,0,0);}
            var bool terminates=false;var usize arg=0;while(arg<count){let usize child=syntax::call_argument(parsed,id,arg);let u32 expected_argument=if(storage_code>=6 && arg==0){0}else{storage::argument_type(storage_code,element,arg)};
                let actual=tc::expression(resolved,tables,problem,child,expected_argument,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}
                if(!(if(storage_code>=6 && arg==0){storage::owner(storage_code,actual.type)}else{actual.type==expected_argument})){let a=syntax::expression_at(parsed,child);return tc::error(problem,1,a.start,a.end,expected_argument,actual.type);}terminates=terminates || actual.terminates;arg+=1;
            }return tc::recorded(tables,id,tc::value(type,terminates));
        }
        let u32 raw_type=raw::call(parsed,id);
        if(raw_type!=0){
            if(!unsafe_scopes::allowed(parsed,node.start,node.end)){return tc::error(problem,5101,node.start,node.end,0,0);}
            if(syntax::call_argument_count(parsed,id)!=3){return tc::error(problem,5,node.start,node.end,0,0);}
            var bool terminates=false;var usize arg=0;
            while(arg<3){let usize child=syntax::call_argument(parsed,id,arg);
                let state::Value actual=tc::expression(resolved,tables,problem,child,if(arg==2){5}else{0},result_type,depth+1,loop_depth);
                if(problem.code!=0){return actual;}
                if(!(if(arg==0){raw::owner(actual.type,raw_type)}else if(arg==1){raw::data(actual.type,raw_type)}else{actual.type==5})){let a=syntax::expression_at(parsed,child);return tc::error(problem,16,a.start,a.end,if(arg==2){5}else{0},actual.type);}
                terminates=terminates || actual.terminates;arg+=1;
            }
            return tc::recorded(tables,id,tc::value(raw_type,terminates));
        }
        let usize callee_id = state::uses(tables,node.left - 1); let names::Symbol callee = names::symbol_at(resolved,callee_id);
        if (callee.kind != 1 && callee.kind != 5 && callee.kind!=11) { let syntax::Expression source = syntax::expression_at(parsed,node.left); return tc::error(problem,6,source.start,source.end,8,state::symbols(tables,callee_id - 1)); }
        state::set_expressions(tables,node.left - 1,8);
        if (callee.kind == 5) {
            let signature=names::external_signature(resolved,callee_id);if(signature.unsafe_call && !unsafe_scopes::allowed(parsed,node.start,node.end)){return tc::error(problem,5101,node.start,node.end,0,0);}if(!effects::includes(effects::caller(parsed,node.start,node.end),signature.effects)){return tc::error(problem,7102,node.start,node.end,signature.effects,0);}
            let usize count = names::external_signature(resolved,callee_id).parameter_count;
            let usize start = names::external_signature(resolved,callee_id).parameter_start;
            if (count != syntax::call_argument_count(parsed,id)) { return tc::error(problem,5,node.start,node.end,0,0); }
            var usize argument = 0; var bool terminates = false;
            while (argument < count) {
                let usize argument_id = syntax::call_argument(parsed,id,argument);
                let u32 type = names::external_parameter(resolved,start + argument);
                let state::Value actual = tc::expression(resolved,tables,problem,argument_id,type,result_type,depth + 1,loop_depth);
                if (problem.code != 0) { return actual; }
                if(actual.type!=type){if(refs::valid(type) && !refs::mutable(type) && refs::mutable(actual.type) && refs::inner(type)==refs::inner(actual.type)){state::set_flags(tables,argument_id-1,4096);}else{let source=syntax::expression_at(parsed,argument_id);return tc::error(problem,1,source.start,source.end,type,actual.type);}}
                terminates = terminates || actual.terminates; argument += 1;
            }
            let u32 type = state::symbols(tables,callee_id - 1);
            return tc::recorded(tables,id,tc::value(type,terminates));
        }
        let required=effects::read(parsed,callee.declaration-1);
        if(!effects::includes(effects::caller(parsed,node.start,node.end),required.bits)){return tc::error(problem,7102,node.start,node.end,required.bits,0);}
        let syntax::Declaration function = syntax::declaration_at(parsed,callee.declaration - 1);
        if(function.unsafe_end!=0 && !unsafe_scopes::allowed(parsed,node.start,node.end)){return tc::error(problem,5101,node.start,node.end,0,0);}
        var usize parameters = 0; var usize entry_index = function.entry_start;
        while (entry_index < function.entry_start + function.entry_count) {
            if (syntax::entry_at(parsed,entry_index).kind == 1) { parameters += 1; } entry_index += 1;
        }
        if (parameters != syntax::call_argument_count(parsed,id)) { return tc::error(problem,5,node.start,node.end,0,0); }
        entry_index = function.entry_start; var usize argument = 0; var bool terminates = false;
        while (entry_index < function.entry_start + function.entry_count) {
            let syntax::Entry entry = syntax::entry_at(parsed,entry_index);
            if (entry.kind == 1) {
                let usize argument_id = syntax::call_argument(parsed,id,argument); let u32 type = nominal::resolve(parsed,entry.type_root);
                let state::Value actual = tc::expression(resolved,tables,problem,argument_id,type,result_type,depth + 1,loop_depth);
                if (problem.code != 0) { return actual; }
                if(actual.type!=type){if(refs::valid(type) && !refs::mutable(type) && refs::mutable(actual.type) && refs::inner(type)==refs::inner(actual.type)){state::set_flags(tables,argument_id-1,4096);}else{let source=syntax::expression_at(parsed,argument_id);return tc::error(problem,1,source.start,source.end,type,actual.type);}}
                terminates = terminates || actual.terminates; argument += 1;
            }
            entry_index += 1;
        }
        let u32 type = state::symbols(tables,callee_id - 1); return tc::recorded(tables,id,tc::value(type,terminates));
    }
    return tc::error(problem,11,node.start,node.end,0,0);
}
