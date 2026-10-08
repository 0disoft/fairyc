module type_collect;
import candidate_limits as limits;
import resource_abilities as abilities;
import tagged_types as tagged;
import enum_types as enums;
import name_resolution as names;
import syntax as syntax;
import must_use_flow as must_use;
import type_state as state;
import nominal_types as nominal;
import type_control as control;
import result_flow as result_flow;
import resource_use as resource_use;
import resource_flow as resource_flow;
import std::bytes as bytes;
import std::vec as v;
import type_check as checking;

pub fn state::Problem collect(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables) {
    let ref syntax::Parsed parsed = names::parsed(resolved);
    if (names::symbol_count(resolved) > 1152 || syntax::expression_count(parsed) > limits::program_expressions() || syntax::entry_count(parsed) > 1024 || syntax::declaration_count(parsed) > 512) { return state::fail(10,0,syntax::source_view(parsed).len,0,0); }
    if(syntax::field_count(parsed)>1024 || tagged::field_slots(parsed)>1024-syntax::field_count(parsed)){return state::fail(10,0,syntax::source_view(parsed).len,0,0);}
    if (!state::initialize(tables,names::symbol_count(resolved),syntax::expression_count(parsed),syntax::entry_count(parsed))) { return state::fail(12,0,syntax::source_view(parsed).len,0,0); }
    let state::Problem ability=abilities::validate(parsed);if(ability.code!=0){return ability;}
    let state::Problem enumeration = enums::validate(parsed); if (enumeration.code != 0) { return enumeration; }
    let state::Problem declarations = nominal::validate(parsed); if (declarations.code != 0) { return declarations; }
    var usize id = 1;
    while (id <= names::symbol_count(resolved)) {
        let names::Symbol symbol = names::symbol_at(resolved,id); let u32 type = if (symbol.kind == 5) { names::external_signature(resolved,id).result } else if (symbol.kind == 6) { 255 + state::small_id(symbol.declaration) } else if(symbol.kind==9 || symbol.kind==10){98303+state::small_id(symbol.declaration)} else if (symbol.kind == 7) { 131071 + state::small_id(symbol.declaration) } else { nominal::resolve(parsed,symbol.type_root) };
        state::set_symbols(tables,id - 1,type);
        if (symbol.entry != 0) { state::set_entries(tables,symbol.entry - 1,id); }
        id += 1;
    }
    let state::Problem attributes=must_use::declarations(resolved,tables);if(attributes.code!=0){return attributes;}
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
            let u32 result_type = nominal::resolve(parsed,function.result_type);
            let state::Value actual = checking::block(resolved,tables,ref mut problem,function.body,result_type,result_type,0,0);
            if (problem.code != 0) { return problem; }
            let syntax::Statement body = syntax::statement_at(parsed,function.body);
            if (body.expression != 0) {
                if (actual.type != result_type) { let syntax::Expression tail = syntax::expression_at(parsed,body.expression); return state::fail(1,tail.start,tail.end,result_type,actual.type); }
            } else if (result_type != 7 && !control::block_terminates(parsed,function.body,0)) { return state::fail(7,body.start,body.end,result_type,7); }
        } else if(function.kind==6){
            let state::Value actual=checking::block(resolved,tables,ref mut problem,function.cleanup_body,7,7,0,0);
            if(problem.code!=0){return problem;}
            if(actual.type!=7){let body=syntax::statement_at(parsed,function.cleanup_body);let tail=syntax::expression_at(parsed,body.expression);return state::fail(1,tail.start,tail.end,7,actual.type);}
        }
        index += 1;
    }
    if(problem.code!=0){return problem;}let uses=must_use::check(resolved,tables);if(uses.code!=0){return uses;}let ownership=resource_use::classify(resolved,tables);if(ownership.code!=0){return ownership;}let flow=resource_flow::check(resolved,tables);return if(flow.code!=0){flow}else{result_flow::check(resolved,tables)};
}
