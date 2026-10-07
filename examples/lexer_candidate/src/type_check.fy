module type_check;
import reference_types as refs;
import resource_abilities as abilities;
import array_types as arrays;
import tagged_types as tagged;
import tagged_check as tagged_check;
import enum_types as enums;
import name_resolution as names;
import syntax as syntax;
import scalar_casts as casts;
import effect_declarations as effects;
import type_state as state;
import nominal_types as nominal;
import resource_identity as resources;
import type_control as control;
import result_flow as result_flow;
import resource_use as resource_use;
import resource_flow as resource_flow;
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
fn state::Value expression(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected, u32 result_type, usize depth, usize loop_depth) {
    if (id == 0) { return value(7,false); }
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (depth >= 64) { return error(problem,10,node.start,node.end,0,0); }
    let ref u8[] op = syntax::expression_text(parsed,id);
    if (node.kind == 30) {
        // Match the reference: the operand has no contextual Result inference.
        // Only the error component must match the enclosing Result, exactly.
        let state::Value operand = expression(resolved,tables,problem,node.left,0,result_type,depth+1,loop_depth);
        if (problem.code != 0) { return operand; }
        if (!tagged::result(operand.type)) { return error(problem,40,node.end-1,node.end,0,operand.type); }
        let u32 actual_error = tagged::failure(operand.type);
        if (!tagged::result(result_type)) { return error(problem,41,node.end-1,node.end,0,actual_error); }
        let u32 expected_error = tagged::failure(result_type);
        if (actual_error != expected_error) { return error(problem,41,node.end-1,node.end,expected_error,actual_error); }
        return recorded(tables,id,value(tagged::success(operand.type),operand.terminates));
    }
    if(node.kind==28 && tagged::option(expected)){
        if(!bytes::equal(op,b"none")){return error(problem,33,node.start,node.end,expected,0);}
        state::set_flags(tables,id-1,state::small_id(tagged::member(parsed,expected,false)));return recorded(tables,id,value(expected,false));
    }
    if(node.kind==28 && tagged::result(expected)){return error(problem,37,node.start,node.end,expected,0);}
    if(node.kind==6 && syntax::expression_at(parsed,node.left).kind==28){
        let ref u8[] name=syntax::expression_text(parsed,node.left);
        if(!tagged::builtin(expected)){return error(problem,if(bytes::equal(name,b"ok") || bytes::equal(name,b"err")){36}else{32},node.start,node.end,0,expected);}
        let bool first=tagged_check::first_name(expected,name);
        if((!first && !(tagged::result(expected) && tagged_check::second_name(expected,name))) || syntax::call_argument_count(parsed,id)!=1){return error(problem,if(tagged::result(expected)){37}else{33},node.start,node.end,expected,0);}
        let usize argument=syntax::call_argument(parsed,id,0);let u32 child=tagged::payload_type(expected,first);
        let state::Value actual=expression(resolved,tables,problem,argument,child,result_type,depth+1,loop_depth);
        if(problem.code!=0){return actual;}if(actual.type!=child){let a=syntax::expression_at(parsed,argument);return error(problem,1,a.start,a.end,child,actual.type);}
        state::set_expressions(tables,node.left-1,expected);state::set_flags(tables,id-1,state::small_id(tagged::member(parsed,expected,first)));
        return recorded(tables,id,value(expected,actual.terminates));
    }
    if (node.kind == 1) {
        let u32 literal = syntax::literal_kind(parsed,id);
        if (literal == 7 || literal == 8) { return recorded(tables,id,value(if (literal == 7) { 9 } else { 10 },false)); }
        let u32 type = if (syntax::literal_kind(parsed,id) == 5) { if (state::integer(expected)) { expected } else { 1 } } else { 6 };
        if (state::integer(type) && !state::fits(op,type,false)) { return error(problem,2,node.start,node.end,type,0); }
        return recorded(tables,id,value(type,false));
    }
    if (node.kind == 2) {
        let usize symbol_id = state::uses(tables,id - 1); assert(symbol_id != 0);
        let names::Symbol symbol = names::symbol_at(resolved,symbol_id);
        if (symbol.kind == 6 || symbol.kind == 7 || symbol.kind==9) { return error(problem,16,node.start,node.end,0,0); }
        let u32 type = if (symbol.kind == 1 || symbol.kind == 5) { 8 } else { state::symbols(tables,symbol_id - 1) };
        return recorded(tables,id,value(type,false));
    }
    if (node.kind == 28) {
        if(!enums::valid(expected)){return error(problem,24,node.start,node.end,0,expected);}
        let usize member=enums::member(parsed,expected,op);if(member==0){return error(problem,25,node.start,node.end,expected,0);}
        if(syntax::field_at(parsed,member-1).payload_count!=0){return error(problem,31,node.start,node.end,expected,0);}
        state::set_flags(tables,id-1,state::small_id(member));return recorded(tables,id,value(expected,false));
    }
    if (node.kind == 15) {
        let state::Value subject=expression(resolved,tables,problem,node.left,0,result_type,depth+1,loop_depth);if(problem.code!=0){return subject;}
        if(tagged::builtin(subject.type)){
            var usize link=node.right;var bool some=false;var bool none=false;var u32 join=expected;var bool known=false;var bool terminates=true;
            while(link!=0){let usize arm_id=syntax::expression_at(parsed,link).left;let arm=syntax::expression_at(parsed,arm_id);
                let pattern=tagged_check::pattern(resolved,tables,arm_id,subject.type);
                if(pattern.problem.code!=0){let p=pattern.problem;return error(problem,p.code,p.start,p.end,p.expected,p.actual);}
                if(pattern.first && some || !pattern.first && none){let p=syntax::expression_at(parsed,arm.left);return error(problem,26,p.start,p.end,0,0);}some=some || pattern.first;none=none || !pattern.first;
                let state::Value actual=block(resolved,tables,problem,arm.right,join,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}
                if(!tagged_check::no_value_arm(parsed,arm.right,subject.type)){if(!known){join=actual.type;known=true;}else if(join!=actual.type){return error(problem,1,arm.start,arm.end,join,actual.type);}}
                state::set_expressions(tables,arm_id-1,actual.type);terminates=terminates && actual.terminates;link=syntax::expression_at(parsed,link).right;
            }
            if(!some || !none){return error(problem,if(tagged::result(subject.type)){39}else{35},node.start,node.end,0,subject.type);}
            if(!known){if(tagged::result(subject.type)){join=7;}else{return error(problem,34,node.start,node.end,0,0);}}
            link=node.right;while(link!=0){let usize aid=syntax::expression_at(parsed,link).left;let arm=syntax::expression_at(parsed,aid);if(tagged_check::no_value_arm(parsed,arm.right,subject.type)){state::set_expressions(tables,aid-1,join);}link=syntax::expression_at(parsed,link).right;}if(join==9 || join==10 || join==11){return error(problem,if(tagged::result(subject.type)){38}else{34},node.start,node.end,0,join);}
            return recorded(tables,id,value(join,subject.terminates || terminates));
        }
        if(!enums::valid(subject.type)){return error(problem,24,node.start,node.end,0,subject.type);}
        let d=syntax::declaration_at(parsed,(subject.type-131072) as usize);var u8[1024] seen=[0;1024];var usize count=0;var usize link=node.right;
        var u32 join=expected;var bool known=false;var bool terminates=true;
        while(link!=0){let usize arm_id=syntax::expression_at(parsed,link).left;let arm=syntax::expression_at(parsed,arm_id);let pattern=syntax::expression_at(parsed,arm.left);
            if((pattern.kind!=16 || pattern.left!=0 || pattern.right!=0) && pattern.kind!=32){return error(problem,28,pattern.start,pattern.end,0,0);}
            let usize member=enums::member(parsed,subject.type,syntax::expression_text(parsed,arm.left));if(member==0){return error(problem,25,pattern.start,pattern.end,subject.type,0);}
            if(seen[member-1]!=0){return error(problem,26,pattern.start,pattern.end,0,0);}seen[member-1]=1;count+=1;
            state::set_flags(tables,arm_id-1,state::small_id(member));
            let m=syntax::field_at(parsed,member-1);let usize fields=if(pattern.kind==32){syntax::pattern_field_count(parsed,arm.left)}else{0};
            if(fields!=m.payload_count || (pattern.kind==32)!=(m.payload_count!=0)){return error(problem,31,pattern.start,pattern.end,0,0);}
            var u8[1024] bound=[0;1024];var usize n=0;while(n<fields){let usize field_id=syntax::pattern_field(parsed,arm.left,n);let f=syntax::expression_at(parsed,field_id);
                let usize target=enums::payload_field(parsed,member,syntax::expression_text(parsed,field_id));if(target==0){return error(problem,18,f.start,f.end,0,0);}
                if(bound[target-1]!=0){return error(problem,17,f.start,f.end,0,0);}bound[target-1]=1;state::set_flags(tables,field_id-1,state::small_id(target));
                let u32 type=state::primitive(parsed,syntax::field_at(parsed,target-1).type_root);state::set_expressions(tables,f.left-1,type);
                var usize symbol_id=1;while(symbol_id<=names::symbol_count(resolved)){let s=names::symbol_at(resolved,symbol_id);if(s.kind==8 && s.declaration==f.left){state::set_symbols(tables,symbol_id-1,type);}symbol_id+=1;}n+=1;
            }
            let state::Value actual=block(resolved,tables,problem,arm.right,join,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}
            if(!known){join=actual.type;known=true;}else if(join!=actual.type){return error(problem,1,arm.start,arm.end,join,actual.type);}
            state::set_expressions(tables,arm_id-1,actual.type);terminates=terminates && actual.terminates;link=syntax::expression_at(parsed,link).right;
        }
        if(count!=enums::member_count(parsed,subject.type)){return error(problem,27,node.start,node.end,0,subject.type);}
        if(join==9 || join==10 || join==11){return error(problem,28,node.start,node.end,0,join);}
        return recorded(tables,id,value(join,subject.terminates || terminates));
    }
    if(node.kind==23){let actual=expression(resolved,tables,problem,node.left,0,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}let u32 target=nominal::resolve(parsed,node.right);if(!casts::allowed(actual.type,target)){return error(problem,52,node.start,node.end,target,actual.type);}return recorded(tables,id,value(target,actual.terminates));}
    if (node.kind == 3) { let state::Value actual = expression(resolved,tables,problem,node.left,expected,result_type,depth + 1,loop_depth); return recorded(tables,id,actual); }
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
        let state::Value actual = expression(resolved,tables,problem,node.left,child_context,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return actual; }
        if(bytes::equal(op,b"*")){if(!refs::valid(actual.type)){return error(problem,16,node.start,node.end,0,actual.type);}return recorded(tables,id,value(refs::inner(actual.type),actual.terminates));}
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
            right = expression(resolved,tables,problem,node.right,context,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return right; }
            left = expression(resolved,tables,problem,node.left,if (state::integer(right.type)) { right.type } else { context },result_type,depth + 1,loop_depth);
        } else {
            left = expression(resolved,tables,problem,node.left,context,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return left; }
            right = expression(resolved,tables,problem,node.right,left.type,result_type,depth + 1,loop_depth);
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
        if (callee.kind != 1 && callee.kind != 5) { let syntax::Expression source = syntax::expression_at(parsed,node.left); return error(problem,6,source.start,source.end,8,state::symbols(tables,callee_id - 1)); }
        state::set_expressions(tables,node.left - 1,8);
        if (callee.kind == 5) {
            let usize count = names::external_signature(resolved,callee_id).parameter_count;
            let usize start = names::external_signature(resolved,callee_id).parameter_start;
            if (count != syntax::call_argument_count(parsed,id)) { return error(problem,5,node.start,node.end,0,0); }
            var usize argument = 0; var bool terminates = false;
            while (argument < count) {
                let usize argument_id = syntax::call_argument(parsed,id,argument);
                let u32 type = names::external_parameter(resolved,start + argument);
                let state::Value actual = expression(resolved,tables,problem,argument_id,type,result_type,depth + 1,loop_depth);
                if (problem.code != 0) { return actual; }
                if(actual.type!=type){if(refs::valid(type) && !refs::mutable(type) && refs::mutable(actual.type) && refs::inner(type)==refs::inner(actual.type)){state::set_flags(tables,argument_id-1,4096);}else{let source=syntax::expression_at(parsed,argument_id);return error(problem,1,source.start,source.end,type,actual.type);}}
                terminates = terminates || actual.terminates; argument += 1;
            }
            let u32 type = state::symbols(tables,callee_id - 1);
            return recorded(tables,id,value(type,terminates));
        }
        let required=effects::read(parsed,callee.declaration-1);
        if(!effects::includes(effects::caller(parsed,node.start,node.end),required.bits)){return error(problem,7102,node.start,node.end,required.bits,0);}
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
                let usize argument_id = syntax::call_argument(parsed,id,argument); let u32 type = nominal::resolve(parsed,entry.type_root);
                let state::Value actual = expression(resolved,tables,problem,argument_id,type,result_type,depth + 1,loop_depth);
                if (problem.code != 0) { return actual; }
                if(actual.type!=type){if(refs::valid(type) && !refs::mutable(type) && refs::mutable(actual.type) && refs::inner(type)==refs::inner(actual.type)){state::set_flags(tables,argument_id-1,4096);}else{let source=syntax::expression_at(parsed,argument_id);return error(problem,1,source.start,source.end,type,actual.type);}}
                terminates = terminates || actual.terminates; argument += 1;
            }
            entry_index += 1;
        }
        let u32 type = state::symbols(tables,callee_id - 1); return recorded(tables,id,value(type,terminates));
    }
    if (node.kind == 24 || node.kind == 29) {
        let usize count = if (node.kind == 29) { arrays::count(syntax::expression_text(parsed,node.right)) } else { syntax::array_element_count(parsed,id) };
        if (count == 0 || count > 64) { return error(problem,16,node.start,node.end,0,0); }
        var u32 element = if (arrays::valid(expected)) { arrays::source_element(parsed,expected) } else { 0 };
        if (arrays::valid(expected) && arrays::length(expected) != count) { return error(problem,1,node.start,node.end,expected,0); }
        var usize i = 0; var bool terminates = false;
        while (i < if (node.kind == 29) { 1 } else { count }) {
            let usize child = if (node.kind == 29) { node.left } else { syntax::array_element(parsed,id,i) };
            let state::Value actual = expression(resolved,tables,problem,child,element,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            if (i == 0 && element == 0) { element = actual.type; }
            if (actual.type != element || arrays::make_source(parsed,element,count) == 0) { return error(problem,1,node.start,node.end,element,actual.type); }
            terminates = terminates || actual.terminates; i += 1;
        }
        return recorded(tables,id,value(arrays::make_source(parsed,element,count),terminates));
    }
    if (node.kind == 21 || node.kind == 22) {
        let u32 view_type = if (node.kind == 22) { 11 } else { 10 };
        var usize root = node.left; while (syntax::expression_at(parsed,root).kind == 3 || (syntax::expression_at(parsed,root).kind==8 || syntax::expression_at(parsed,root).kind==9 || syntax::expression_at(parsed,root).kind==4 && bytes::equal(syntax::expression_text(parsed,root),b"*"))) { root = syntax::expression_at(parsed,root).left; }
        if (syntax::expression_at(parsed,root).kind != 2) { return error(problem,16,node.start,node.end,10,0); }
        let state::Value owner = expression(resolved,tables,problem,node.left,0,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return owner; }
        if(refs::referent(owner.type) && !arrays::valid(owner.type)){
            let u32 type=refs::make(owner.type,node.kind==22);if(expected!=0 && expected!=type){return error(problem,1,node.start,node.end,expected,type);}
            if(node.kind==22 && (names::symbol_at(resolved,state::uses(tables,root-1)).kind!=4 && !refs::mutable(state::expressions(tables,root-1)) || node.left!=root && (resources::valid(state::expressions(tables,root-1)) || refs::valid(state::expressions(tables,root-1)) && (!refs::mutable(state::expressions(tables,root-1)) || resources::valid(refs::inner(state::expressions(tables,root-1))))))){return error(problem,4,node.start,node.end,0,owner.type);}return recorded(tables,id,value(type,owner.terminates));
        }
        if (expected!=view_type || !arrays::valid(owner.type) || arrays::source_element(parsed,owner.type) != 3) { return error(problem,16,node.start,node.end,10,owner.type); }
        if (view_type == 11 && (names::symbol_at(resolved,state::uses(tables,root - 1)).kind != 4 || resources::valid(state::expressions(tables,root-1)))) { return error(problem,4,node.start,node.end,11,owner.type); }
        return recorded(tables,id,value(view_type,owner.terminates));
    }
    if (node.kind == 26) {
        let state::Value owner = expression(resolved,tables,problem,node.left,10,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return owner; }
        if (owner.type == 9) { return error(problem,21,node.start,node.end,0,9); }
        if (owner.type != 10 && owner.type != 11) { return error(problem,16,node.start,node.end,10,owner.type); }
        let syntax::Expression bounds = syntax::expression_at(parsed,node.right); var bool terminates = owner.terminates; var usize i = 0;
        while (i < 2) { let usize child = if (i == 0) { bounds.left } else { bounds.right }; if (child != 0) {
            let state::Value bound = expression(resolved,tables,problem,child,5,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return bound; } if (bound.type != 5) { return error(problem,1,node.start,node.end,5,bound.type); } terminates = terminates || bound.terminates;
        } i += 1; }
        return recorded(tables,id,value(owner.type,terminates));
    }
    if (node.kind == 9) {
        let state::Value owner = expression(resolved,tables,problem,node.left,0,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return owner; }
        if (owner.type == 9) { return error(problem,21,node.start,node.end,0,9); }
        if (!arrays::valid(owner.type) && owner.type != 10 && owner.type != 11) { return error(problem,16,node.start,node.end,0,owner.type); }
        let state::Value index = expression(resolved,tables,problem,node.right,5,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return index; }
        if (index.type != 5) { return error(problem,1,node.start,node.end,5,index.type); }
        return recorded(tables,id,value(if (owner.type == 10 || owner.type == 11) { 3 } else { arrays::source_element(parsed,owner.type) },owner.terminates || index.terminates));
    }
    if (node.kind == 8) {
        let state::Value owner = expression(resolved,tables,problem,node.left,0,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return owner; }
        if (tagged::result(owner.type) && bytes::equal(op,b"is_ok")) {
            state::set_flags(tables,id-1,2048); return recorded(tables,id,value(6,owner.terminates));
        }
        if ((owner.type == 9 || owner.type == 10 || owner.type == 11) && (bytes::equal(op,b"len") || bytes::equal(op,b"is_empty"))) { return recorded(tables,id,value(if (bytes::equal(op,b"len")) { 5 } else { 6 },owner.terminates)); }
        if (arrays::valid(owner.type) && bytes::equal(op,b"len")) { return recorded(tables,id,value(5,owner.terminates)); }
        let usize field = nominal::field(parsed,if(refs::valid(owner.type)){refs::inner(owner.type)}else{owner.type},op);
        if (field == 0) { return error(problem,18,node.end - op.len,node.end,owner.type,0); }
        let u32 type = nominal::resolve(parsed,syntax::field_at(parsed,field - 1).type_root);
        state::set_flags(tables,id - 1,state::small_id(field));
        return recorded(tables,id,value(type,owner.terminates));
    }
    if (node.kind == 10 && syntax::expression_at(parsed,node.left).kind==28) {
        if(!enums::valid(expected)){return error(problem,24,node.start,node.end,0,expected);}
        let usize member=enums::member(parsed,expected,syntax::expression_text(parsed,node.left));if(member==0){return error(problem,25,node.start,node.end,expected,0);}
        let m=syntax::field_at(parsed,member-1);let usize count=syntax::record_field_count(parsed,id);
        if(m.payload_count==0 || count!=m.payload_count){return error(problem,31,node.start,node.end,expected,0);}
        var u8[1024] seen=[0;1024];var usize i=0;var bool terminates=false;
        while(i<count){let usize entry=syntax::record_field(parsed,id,i);let f=syntax::expression_at(parsed,entry);
            let usize target=enums::payload_field(parsed,member,syntax::expression_text(parsed,entry));if(target==0){return error(problem,18,f.start,f.end,expected,0);}
            if(seen[target-1]!=0){return error(problem,17,f.start,f.end,0,0);}seen[target-1]=1;
            let u32 type=state::primitive(parsed,syntax::field_at(parsed,target-1).type_root);let state::Value actual=expression(resolved,tables,problem,f.left,type,result_type,depth+1,loop_depth);
            if(problem.code!=0){return actual;}if(actual.type!=type){return error(problem,1,f.start,f.end,type,actual.type);}
            state::set_flags(tables,entry-1,state::small_id(target));state::set_expressions(tables,entry-1,type);terminates=terminates || actual.terminates;i+=1;
        }
        state::set_flags(tables,id-1,state::small_id(member));return recorded(tables,id,value(expected,terminates));
    }
    if (node.kind == 10) {
        let syntax::Expression target = syntax::expression_at(parsed,node.left);
        if (target.kind != 2) { return error(problem,16,target.start,target.end,0,0); }
        let usize symbol_id = state::uses(tables,node.left - 1);
        if (symbol_id == 0 || (names::symbol_at(resolved,symbol_id).kind != 6 && names::symbol_at(resolved,symbol_id).kind != 9)) { return error(problem,16,target.start,target.end,0,0); }
        let u32 type = state::symbols(tables,symbol_id - 1);
        let syntax::Declaration declaration = syntax::declaration_at(parsed,(type - if(resources::valid(type)){98304}else{256}) as usize);
        var u8[1024] seen = [0;1024]; var bool terminates = false; var usize i = 0;
        while (i < syntax::record_field_count(parsed,id)) {
            let usize entry = syntax::record_field(parsed,id,i); let syntax::Expression field = syntax::expression_at(parsed,entry);
            let ref u8[] name = syntax::expression_text(parsed,entry); let usize at = nominal::field(parsed,type,name);
            if (at == 0) { return error(problem,18,field.start,field.end,type,0); }
            if (seen[at - 1] != 0) { return error(problem,17,field.start,field.end,0,0); } seen[at - 1] = 1;
            let u32 expected_field = nominal::resolve(parsed,syntax::field_at(parsed,at - 1).type_root);
            let state::Value actual = expression(resolved,tables,problem,field.left,expected_field,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            if (actual.type != expected_field) { let syntax::Expression source = syntax::expression_at(parsed,field.left); return error(problem,1,source.start,source.end,expected_field,actual.type); }
            state::set_flags(tables,entry - 1,state::small_id(at)); state::set_expressions(tables,entry - 1,actual.type);
            terminates = terminates || actual.terminates; i += 1;
        }
        if (i != declaration.field_count) { return error(problem,19,node.start,node.end,type,0); }
        state::set_expressions(tables,node.left - 1,type);
        return recorded(tables,id,value(type,terminates));
    }
    if (node.kind == 13) {
        let state::Value condition = expression(resolved,tables,problem,node.left,6,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return condition; }
        if (condition.type != 6) { let syntax::Expression source = syntax::expression_at(parsed,node.left); return error(problem,3,source.start,source.end,6,condition.type); }
        let syntax::Expression branches = syntax::expression_at(parsed,node.right);
        let state::Value consequent = block(resolved,tables,problem,branches.left,expected,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return consequent; }
        if (branches.right == 0) { return recorded(tables,id,value(7,false)); }
        let state::Value alternative = expression(resolved,tables,problem,branches.right,expected,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return alternative; }
        if (consequent.type != alternative.type) {
            let syntax::Expression source = syntax::expression_at(parsed,branches.right); return error(problem,1,source.start,source.end,consequent.type,alternative.type);
        }
        return recorded(tables,id,value(consequent.type,consequent.terminates && alternative.terminates));
    }
    if (node.kind == 12) { let state::Value actual = block(resolved,tables,problem,node.left,expected,result_type,depth + 1,loop_depth); return recorded(tables,id,actual); }
    return error(problem,11,node.start,node.end,0,0);
}
fn state::Value statement(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 result_type, usize depth, usize loop_depth) {
    if (id == 0) { return value(7,false); }
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Statement item = syntax::statement_at(parsed,id);
    if (depth >= 64) { return error(problem,10,item.start,item.end,0,0); }
        if (item.kind == 2) {
            let usize symbol_id = state::entries(tables,item.entry - 1); let names::Symbol symbol = names::symbol_at(resolved,symbol_id);
            let u32 expected = nominal::resolve(parsed,symbol.type_root);
            let state::Value actual = expression(resolved,tables,problem,item.expression,expected,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            let syntax::Expression source = syntax::expression_at(parsed,item.expression);
            if (actual.type == 7 || actual.type == 8) { return error(problem,8,source.start,source.end,expected,actual.type); }
            if (expected != 0 && expected != actual.type) { return error(problem,1,source.start,source.end,expected,actual.type); }
            state::set_symbols(tables,symbol_id - 1,actual.type); return value(7,actual.terminates);
        } else if (item.kind == 3) {
            let state::Value actual = expression(resolved,tables,problem,item.expression,result_type,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            if (actual.type != result_type) {
                let syntax::Expression source = if (item.expression == 0) { syntax::Expression { .kind = 0, .start = item.start, .end = item.end, .token = 0, .left = 0, .right = 0 } } else { syntax::expression_at(parsed,item.expression) };
                return error(problem,1,source.start,source.end,result_type,actual.type);
            }
            return value(7,true);
        } else if (item.kind == 4) {
            let state::Value target = expression(resolved,tables,problem,item.target,0,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return target; }
            var usize place_id = item.target;
            while (syntax::expression_at(parsed,place_id).kind == 3) { place_id = syntax::expression_at(parsed,place_id).left; }
            let syntax::Expression checked_place = syntax::expression_at(parsed,place_id);
            if (checked_place.kind == 9 && state::expressions(tables,checked_place.left - 1) == 10 || checked_place.kind == 8 && state::flags(tables,place_id - 1) == 0) { return error(problem,4,checked_place.start,checked_place.end,0,target.type); }
            var usize root = item.target; var usize fields = 0;
            while (syntax::expression_at(parsed,root).kind == 8 || syntax::expression_at(parsed,root).kind == 3 || syntax::expression_at(parsed,root).kind == 9 || syntax::expression_at(parsed,root).kind==4 && bytes::equal(syntax::expression_text(parsed,root),b"*")) {
                if (syntax::expression_at(parsed,root).kind == 8) { fields += 1; }
                root = syntax::expression_at(parsed,root).left;
            }
            if (fields > 1 && nominal::contains_tagged(parsed,state::expressions(tables,root - 1),0)) { return error(problem,4,checked_place.start,checked_place.end,0,target.type); }
            let usize symbol_id = if (syntax::expression_at(parsed,root).kind == 2) { state::uses(tables,root - 1) } else { 0 };
            let bool indirect=refs::mutable(state::expressions(tables,root-1)) && item.target!=root;
            if(checked_place.kind==4 && (!indirect || !bytes::equal(syntax::assignment_operator(parsed,id),b"="))){return error(problem,4,checked_place.start,checked_place.end,0,target.type);}
            let u32 expected = target.type;
            let state::Value actual = expression(resolved,tables,problem,item.expression,expected,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            let syntax::Expression place = syntax::expression_at(parsed,item.target);
            if (symbol_id == 0 || (names::symbol_at(resolved,symbol_id).kind != 4 && names::symbol_at(resolved,symbol_id).kind != 10 && !indirect && !(checked_place.kind == 9 && state::expressions(tables,checked_place.left - 1) == 11))) { return error(problem,4,place.start,place.end,0,target.type); }
            if(names::symbol_at(resolved,symbol_id).kind==10 && fields>1){return error(problem,4,place.start,place.end,0,target.type);}
            if(names::symbol_at(resolved,symbol_id).kind==10 && (fields==0 || nominal::contains_resource(parsed,expected,0))){return error(problem,48,place.start,place.end,0,target.type);}
            if (expected != actual.type) { let syntax::Expression source = syntax::expression_at(parsed,item.expression); return error(problem,1,source.start,source.end,expected,actual.type); }
            if (!bytes::equal(syntax::assignment_operator(parsed,id),b"=") && !state::integer(expected)) { return error(problem,9,item.start,item.end,0,expected); }
            return value(7,actual.terminates);
        } else if(item.kind==16){
            let actual=expression(resolved,tables,problem,item.expression,0,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}
            let e=syntax::expression_at(parsed,item.expression);let usize local=state::uses(tables,item.expression-1);
            if(e.kind!=2 || local==0 || (names::symbol_at(resolved,local).kind!=2 && names::symbol_at(resolved,local).kind!=3 && names::symbol_at(resolved,local).kind!=4) || !abilities::burnable(parsed,actual.type)){return error(problem,51,item.start,item.end,0,actual.type);}return value(7,false);
        } else if (item.kind == 6 || item.kind == 12 || item.kind == 13) {
            let state::Value actual = expression(resolved,tables,problem,item.expression,0,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return actual; }
            if (item.kind == 13 && (syntax::expression_at(parsed,item.expression).kind != 6 || actual.type != 7)) {
                return error(problem,15,item.start,item.end,7,actual.type);
            }
            if (item.kind == 13) {
                var usize i = 0; while (i < syntax::call_argument_count(parsed,item.expression)) {
                    let usize argument = syntax::call_argument(parsed,item.expression,i);
                    if (state::expressions(tables,argument-1)<1 || state::expressions(tables,argument-1)>6) { return error(problem,29,item.start,item.end,0,0); } i += 1;
                }
            }
            return value(7,actual.terminates);
        }

    if (item.kind == 9 || item.kind == 10) {
        if (loop_depth == 0) { return error(problem,13,item.start,item.end,0,0); }
        return value(7,false);
    }
    if (item.kind == 11 || item.kind == 8) {
        let state::Value condition = expression(resolved,tables,problem,item.expression,6,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return condition; }
        if (condition.type != 6) { let syntax::Expression source = syntax::expression_at(parsed,item.expression); return error(problem,3,source.start,source.end,6,condition.type); }
        if (item.kind == 8) {
            let state::Value body = block(resolved,tables,problem,item.first,7,result_type,depth + 1,loop_depth + 1);
            if (problem.code != 0) { return body; }
        }
        return value(7,false);
    }
    if (item.kind == 14) {
        let syntax::Statement header = syntax::statement_at(parsed,item.second);
        let state::Value initial = statement(resolved,tables,problem,header.first,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return initial; }
        if (header.first != 0) {
            let syntax::Statement clause = syntax::statement_at(parsed,header.first);
            if (!control::plain_header(parsed,clause.expression,0)) { return error(problem,14,clause.start,syntax::expression_at(parsed,clause.expression).end,0,0); }
        }
        if (item.expression != 0) {
            let state::Value condition = expression(resolved,tables,problem,item.expression,6,result_type,depth + 1,loop_depth);
            if (problem.code != 0) { return condition; }
            let syntax::Expression source = syntax::expression_at(parsed,item.expression);
            if (!control::plain_header(parsed,item.expression,0)) { return error(problem,14,source.start,source.end,0,0); }
            if (condition.type != 6) { return error(problem,3,source.start,source.end,6,condition.type); }
        }
        let state::Value body = block(resolved,tables,problem,item.first,7,result_type,depth + 1,loop_depth + 1);
        if (problem.code != 0) { return body; }
        let state::Value post = statement(resolved,tables,problem,header.second,result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return post; }
        if (header.second != 0) {
            let syntax::Statement clause = syntax::statement_at(parsed,header.second);
            if (!control::plain_header(parsed,clause.expression,0)) { return error(problem,14,clause.start,syntax::expression_at(parsed,clause.expression).end,0,0); }
        }
        return value(7,false);
    }
    return error(problem,11,item.start,item.end,0,0);
}
fn state::Value block(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected_tail, u32 result_type, usize depth, usize loop_depth) {
    let ref syntax::Parsed parsed = names::parsed(resolved); let syntax::Statement body = syntax::statement_at(parsed,id);
    if (depth >= 64) { return error(problem,10,body.start,body.end,0,0); }
    var bool terminates = false; var usize index = 0;
    while (index < syntax::block_statement_count(parsed,id)) {
        let state::Value checked = statement(resolved,tables,problem,syntax::block_statement(parsed,id,index),result_type,depth + 1,loop_depth);
        if (problem.code != 0) { return checked; }
        terminates = terminates || checked.terminates; index += 1;
    }
    let state::Value tail = expression(resolved,tables,problem,body.expression,expected_tail,result_type,depth + 1,loop_depth);
    return value(tail.type,terminates || tail.terminates);
}
fn state::Problem collect(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables) {
    let ref syntax::Parsed parsed = names::parsed(resolved);
    if (names::symbol_count(resolved) > 1152 || syntax::expression_count(parsed) > 4096 || syntax::entry_count(parsed) > 1024 || syntax::declaration_count(parsed) > 512) { return state::fail(10,0,syntax::source_view(parsed).len,0,0); }
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
            let state::Value actual = block(resolved,tables,ref mut problem,function.body,result_type,result_type,0,0);
            if (problem.code != 0) { return problem; }
            let syntax::Statement body = syntax::statement_at(parsed,function.body);
            if (body.expression != 0) {
                if (actual.type != result_type) { let syntax::Expression tail = syntax::expression_at(parsed,body.expression); return state::fail(1,tail.start,tail.end,result_type,actual.type); }
            } else if (result_type != 7 && !control::block_terminates(parsed,function.body,0)) { return state::fail(7,body.start,body.end,result_type,7); }
        } else if(function.kind==6){
            let state::Value actual=block(resolved,tables,ref mut problem,function.cleanup_body,7,7,0,0);
            if(problem.code!=0){return problem;}
            if(actual.type!=7){let body=syntax::statement_at(parsed,function.cleanup_body);let tail=syntax::expression_at(parsed,body.expression);return state::fail(1,tail.start,tail.end,7,actual.type);}
        }
        index += 1;
    }
    if(problem.code!=0){return problem;}let ownership=resource_use::classify(resolved,tables);if(ownership.code!=0){return ownership;}let flow=resource_flow::check(resolved,tables);return if(flow.code!=0){flow}else{result_flow::check(resolved,tables)};
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
pub fn usize field_target(ref Typed typed, usize id) { assert(typed.problem.code == 0); let u32 flag=state::flags(ref typed.tables,id-1); return if(flag==2048){0}else{flag as usize}; }
pub fn bool result_is_ok(ref Typed typed, usize id) { assert(typed.problem.code==0); return state::flags(ref typed.tables,id-1)==2048; }
pub fn bool moving(ref Typed typed, usize id) { assert(typed.problem.code==0); return state::moving(ref typed.tables,id-1); }
pub fn bool signed_minimum(ref Typed typed, usize id) { assert(typed.problem.code == 0 && id > 0 && id <= syntax::expression_count(names::parsed(ref typed.source))); return syntax::expression_at(names::parsed(ref typed.source),id).kind == 4 && state::flags(ref typed.tables,id - 1) == 1; }

pub fn bool shared_argument(ref Typed p,usize id){return state::flags(ref p.tables,id-1)==4096;}
