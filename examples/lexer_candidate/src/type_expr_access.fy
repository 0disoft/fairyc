module type_expr_access;
import scalar_views as views;
import reference_types as refs;
import array_types as arrays;
import tagged_types as tagged;
import name_resolution as names;
import syntax as syntax;
import type_state as state;
import nominal_types as nominal;
import resource_identity as resources;
import std::bytes as bytes;
import std::vec as v;
import type_check as tc;

pub fn state::Value expression(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected, u32 result_type, usize depth, usize loop_depth) {
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Expression node = syntax::expression_at(parsed,id);
    let ref u8[] op = syntax::expression_text(parsed,id);
    if (node.kind == 24 || node.kind == 29) {
        let usize count = if (node.kind == 29) { arrays::count(syntax::expression_text(parsed,node.right)) } else { syntax::array_element_count(parsed,id) };
        if (count == 0 || count > 4096) { return tc::error(problem,16,node.start,node.end,0,0); }
        var u32 element = if (arrays::valid(expected)) { arrays::source_element(parsed,expected) } else { 0 };
        if (arrays::valid(expected) && arrays::length(expected) != count) { return tc::error(problem,1,node.start,node.end,expected,0); }
        var usize i = 0; var bool terminates = false;
        while (i < if (node.kind == 29) { 1 } else { count }) {
            let usize child = if (node.kind == 29) { node.left } else { syntax::array_element(parsed,id,i) };
            let state::Value actual = tc::expression(resolved,tables,problem,child,element,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            if (i == 0 && element == 0) { element = actual.type; }
            if (actual.type != element || arrays::make_source(parsed,element,count) == 0) { return tc::error(problem,1,node.start,node.end,element,actual.type); }
            terminates = terminates || actual.terminates; i += 1;
        }
        return tc::recorded(tables,id,tc::value(arrays::make_source(parsed,element,count),terminates));
    }
    if (node.kind == 21 || node.kind == 22) {
        var u32 view_type = if (node.kind == 22) { 11 } else { 10 };
        let syntax::Expression selected=syntax::expression_at(parsed,node.left);
        let bool sliced=selected.kind==26;let usize target=if(sliced){selected.left}else{node.left};
        var usize root = target; while (syntax::expression_at(parsed,root).kind == 3 || (syntax::expression_at(parsed,root).kind==8 || syntax::expression_at(parsed,root).kind==9 || syntax::expression_at(parsed,root).kind==4 && bytes::equal(syntax::expression_text(parsed,root),b"*"))) { root = syntax::expression_at(parsed,root).left; }
        if (syntax::expression_at(parsed,root).kind != 2) { return tc::error(problem,16,node.start,node.end,10,0); }
        let state::Value owner = tc::expression(resolved,tables,problem,target,0,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return owner; }
        if(sliced && !arrays::valid(owner.type)){return tc::error(problem,16,node.start,node.end,10,owner.type);}
        if(refs::referent(owner.type) && !arrays::valid(owner.type)){
            let u32 type=refs::make(owner.type,node.kind==22);if(expected!=0 && expected!=type){return tc::error(problem,1,node.start,node.end,expected,type);}
            if(node.kind==22 && (names::symbol_at(resolved,state::uses(tables,root-1)).kind!=4 && names::symbol_at(resolved,state::uses(tables,root-1)).kind!=10 && !refs::mutable(state::expressions(tables,root-1)) || node.left!=root && (resources::valid(state::expressions(tables,root-1)) || refs::valid(state::expressions(tables,root-1)) && (!refs::mutable(state::expressions(tables,root-1)) || resources::valid(refs::inner(state::expressions(tables,root-1))))))){return tc::error(problem,4,node.start,node.end,0,owner.type);}return tc::recorded(tables,id,tc::value(type,owner.terminates));
        }
        view_type=if(arrays::valid(owner.type)){views::make(arrays::source_element(parsed,owner.type),node.kind==22)}else{0};
        if (view_type==0 || expected!=view_type || !arrays::valid(owner.type)) { return tc::error(problem,16,node.start,node.end,10,owner.type); }
        let u32 root_type=state::expressions(tables,root-1);
        if (views::mutable(view_type) && (names::symbol_at(resolved,state::uses(tables,root - 1)).kind != 4 && !refs::mutable(root_type) || resources::valid(root_type) || refs::valid(root_type) && resources::valid(refs::inner(root_type)))) { return tc::error(problem,4,node.start,node.end,11,owner.type); }
        var bool terminates=owner.terminates;
        if(sliced){
            let bounds=syntax::expression_at(parsed,selected.right);var usize i=0;
            while(i<2){let usize child=if(i==0){bounds.left}else{bounds.right};if(child!=0){
                let bound=tc::expression(resolved,tables,problem,child,5,result_type,depth+1,loop_depth);
                if(problem.code!=0){return bound;}if(bound.type!=5){return tc::error(problem,1,node.start,node.end,5,bound.type);}terminates=terminates || bound.terminates;
            }i+=1;}
            state::set_expressions(tables,node.left-1,view_type);
        }
        return tc::recorded(tables,id,tc::value(view_type,terminates));
    }
    if (node.kind == 26) {
        let state::Value owner = tc::expression(resolved,tables,problem,node.left,10,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return owner; }
        if (owner.type == 9) { return tc::error(problem,21,node.start,node.end,0,9); }
        if (!views::valid(owner.type)) { return tc::error(problem,16,node.start,node.end,10,owner.type); }
        let syntax::Expression bounds = syntax::expression_at(parsed,node.right); var bool terminates = owner.terminates; var usize i = 0;
        while (i < 2) { let usize child = if (i == 0) { bounds.left } else { bounds.right }; if (child != 0) {
            let state::Value bound = tc::expression(resolved,tables,problem,child,5,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return bound; } if (bound.type != 5) { return tc::error(problem,1,node.start,node.end,5,bound.type); } terminates = terminates || bound.terminates;
        } i += 1; }
        return tc::recorded(tables,id,tc::value(owner.type,terminates));
    }
    if (node.kind == 9) {
        let state::Value owner = tc::expression(resolved,tables,problem,node.left,0,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return owner; }
        if (owner.type == 9) { return tc::error(problem,21,node.start,node.end,0,9); }
        if (!arrays::valid(owner.type) && !views::valid(owner.type)) { return tc::error(problem,16,node.start,node.end,0,owner.type); }
        let state::Value index = tc::expression(resolved,tables,problem,node.right,5,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return index; }
        if (index.type != 5) { return tc::error(problem,1,node.start,node.end,5,index.type); }
        return tc::recorded(tables,id,tc::value(if (views::valid(owner.type)) { views::element(owner.type) } else { arrays::source_element(parsed,owner.type) },owner.terminates || index.terminates));
    }
    if (node.kind == 8) {
        let state::Value owner = tc::expression(resolved,tables,problem,node.left,0,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return owner; }
        if (tagged::result(owner.type) && bytes::equal(op,b"is_ok")) {
            state::set_flags(tables,id-1,2048); return tc::recorded(tables,id,tc::value(6,owner.terminates));
        }
        if ((owner.type == 9 || views::valid(owner.type)) && (bytes::equal(op,b"len") || bytes::equal(op,b"is_empty"))) { return tc::recorded(tables,id,tc::value(if (bytes::equal(op,b"len")) { 5 } else { 6 },owner.terminates)); }
        if (arrays::valid(owner.type) && bytes::equal(op,b"len")) { return tc::recorded(tables,id,tc::value(5,owner.terminates)); }
        let usize field = nominal::field(parsed,if(refs::valid(owner.type)){refs::inner(owner.type)}else{owner.type},op);
        if (field == 0) { return tc::error(problem,18,node.end - op.len,node.end,owner.type,0); }
        let u32 owner_type=if(refs::valid(owner.type)){refs::inner(owner.type)}else{owner.type};if(resources::valid(owner_type)){let d=syntax::declaration_at(parsed,(owner_type-98304) as usize);if(syntax::module_index(parsed,d.start)!=syntax::module_index(parsed,node.start)){return tc::error(problem,6201,node.start,node.end,0,0);}}
        let u32 type = nominal::resolve(parsed,syntax::field_at(parsed,field - 1).type_root);
        state::set_flags(tables,id - 1,state::small_id(field));
        return tc::recorded(tables,id,tc::value(type,owner.terminates));
    }
    return tc::error(problem,11,node.start,node.end,0,0);
}
