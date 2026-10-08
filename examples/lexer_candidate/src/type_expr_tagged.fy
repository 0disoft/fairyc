module type_expr_tagged;
import scalar_views as views;
import tagged_types as tagged;
import tagged_check as tagged_check;
import enum_types as enums;
import name_resolution as names;
import syntax as syntax;
import type_state as state;
import std::bytes as bytes;
import std::vec as v;
import type_check as checking;

pub fn state::Value expression(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, ref mut state::Problem problem, usize id, u32 expected, u32 result_type, usize depth, usize loop_depth) {
    let ref syntax::Parsed parsed = names::parsed(resolved);
    let syntax::Expression node = syntax::expression_at(parsed,id);
    let ref u8[] op = syntax::expression_text(parsed,id);
    if (node.kind == 30) {
        // Match the reference: the operand has no contextual Result inference.
        // Only the error component must match the enclosing Result, exactly.
        let state::Value operand = checking::expression(resolved,tables,problem,node.left,0,result_type,depth+1,loop_depth);
        if (problem.code != 0) { return operand; }
        if (!tagged::result(operand.type)) { return checking::error(problem,40,node.end-1,node.end,0,operand.type); }
        let u32 actual_error = tagged::failure(operand.type);
        if (!tagged::result(result_type)) { return checking::error(problem,41,node.end-1,node.end,0,actual_error); }
        let u32 expected_error = tagged::failure(result_type);
        if (actual_error != expected_error) { return checking::error(problem,41,node.end-1,node.end,expected_error,actual_error); }
        return checking::recorded(tables,id,checking::value(tagged::success(operand.type),operand.terminates));
    }
    if(node.kind==28 && tagged::option(expected)){
        if(!bytes::equal(op,b"none")){return checking::error(problem,33,node.start,node.end,expected,0);}
        state::set_flags(tables,id-1,state::small_id(tagged::member(parsed,expected,false)));return checking::recorded(tables,id,checking::value(expected,false));
    }
    if(node.kind==28 && tagged::result(expected)){return checking::error(problem,37,node.start,node.end,expected,0);}
    if(node.kind==6 && syntax::expression_at(parsed,node.left).kind==28){
        let ref u8[] name=syntax::expression_text(parsed,node.left);
        if(!tagged::builtin(expected)){return checking::error(problem,if(bytes::equal(name,b"ok") || bytes::equal(name,b"err")){36}else{32},node.start,node.end,0,expected);}
        let bool first=tagged_check::first_name(expected,name);
        if((!first && !(tagged::result(expected) && tagged_check::second_name(expected,name))) || syntax::call_argument_count(parsed,id)!=1){return checking::error(problem,if(tagged::result(expected)){37}else{33},node.start,node.end,expected,0);}
        let usize argument=syntax::call_argument(parsed,id,0);let u32 child=tagged::payload_type(expected,first);
        let state::Value actual=checking::expression(resolved,tables,problem,argument,child,result_type,depth+1,loop_depth);
        if(problem.code!=0){return actual;}if(actual.type!=child){let a=syntax::expression_at(parsed,argument);return checking::error(problem,1,a.start,a.end,child,actual.type);}
        state::set_expressions(tables,node.left-1,expected);state::set_flags(tables,id-1,state::small_id(tagged::member(parsed,expected,first)));
        return checking::recorded(tables,id,checking::value(expected,actual.terminates));
    }
    if (node.kind == 28) {
        if(!enums::valid(expected)){return checking::error(problem,24,node.start,node.end,0,expected);}
        let usize member=enums::member(parsed,expected,op);if(member==0){return checking::error(problem,25,node.start,node.end,expected,0);}
        if(syntax::field_at(parsed,member-1).payload_count!=0){return checking::error(problem,31,node.start,node.end,expected,0);}
        state::set_flags(tables,id-1,state::small_id(member));return checking::recorded(tables,id,checking::value(expected,false));
    }
    if (node.kind == 15) {
        let state::Value subject=checking::expression(resolved,tables,problem,node.left,0,result_type,depth+1,loop_depth);if(problem.code!=0){return subject;}
        if(tagged::builtin(subject.type)){
            var usize link=node.right;var bool some=false;var bool none=false;var u32 join=expected;var bool known=false;var bool terminates=true;
            while(link!=0){let usize arm_id=syntax::expression_at(parsed,link).left;let arm=syntax::expression_at(parsed,arm_id);
                let pattern=tagged_check::pattern(resolved,tables,arm_id,subject.type);
                if(pattern.problem.code!=0){let p=pattern.problem;return checking::error(problem,p.code,p.start,p.end,p.expected,p.actual);}
                if(pattern.first && some || !pattern.first && none){let p=syntax::expression_at(parsed,arm.left);return checking::error(problem,26,p.start,p.end,0,0);}some=some || pattern.first;none=none || !pattern.first;
                let state::Value actual=checking::block(resolved,tables,problem,arm.right,join,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}
                if(!tagged_check::no_value_arm(parsed,arm.right,subject.type)){if(!known){join=actual.type;known=true;}else if(join!=actual.type){return checking::error(problem,1,arm.start,arm.end,join,actual.type);}}
                state::set_expressions(tables,arm_id-1,actual.type);terminates=terminates && actual.terminates;link=syntax::expression_at(parsed,link).right;
            }
            if(!some || !none){return checking::error(problem,if(tagged::result(subject.type)){39}else{35},node.start,node.end,0,subject.type);}
            if(!known){if(tagged::result(subject.type)){join=7;}else{return checking::error(problem,34,node.start,node.end,0,0);}}
            link=node.right;while(link!=0){let usize aid=syntax::expression_at(parsed,link).left;let arm=syntax::expression_at(parsed,aid);if(tagged_check::no_value_arm(parsed,arm.right,subject.type)){state::set_expressions(tables,aid-1,join);}link=syntax::expression_at(parsed,link).right;}
            return checking::recorded(tables,id,checking::value(join,subject.terminates || terminates));
        }
        if(!enums::valid(subject.type)){return checking::error(problem,24,node.start,node.end,0,subject.type);}
        let d=syntax::declaration_at(parsed,(subject.type-131072) as usize);var u8[1024] seen=[0;1024];var usize count=0;var usize link=node.right;
        var u32 join=expected;var bool known=false;var bool terminates=true;
        while(link!=0){let usize arm_id=syntax::expression_at(parsed,link).left;let arm=syntax::expression_at(parsed,arm_id);let pattern=syntax::expression_at(parsed,arm.left);
            if((pattern.kind!=16 || pattern.left!=0 || pattern.right!=0) && pattern.kind!=32){return checking::error(problem,28,pattern.start,pattern.end,0,0);}
            let usize member=enums::member(parsed,subject.type,syntax::expression_text(parsed,arm.left));if(member==0){return checking::error(problem,25,pattern.start,pattern.end,subject.type,0);}
            if(seen[member-1]!=0){return checking::error(problem,26,pattern.start,pattern.end,0,0);}seen[member-1]=1;count+=1;
            state::set_flags(tables,arm_id-1,state::small_id(member));
            let m=syntax::field_at(parsed,member-1);let usize fields=if(pattern.kind==32){syntax::pattern_field_count(parsed,arm.left)}else{0};
            if(fields!=m.payload_count || (pattern.kind==32)!=(m.payload_count!=0)){return checking::error(problem,31,pattern.start,pattern.end,0,0);}
            var u8[1024] bound=[0;1024];var usize n=0;while(n<fields){let usize field_id=syntax::pattern_field(parsed,arm.left,n);let f=syntax::expression_at(parsed,field_id);
                let usize target=enums::payload_field(parsed,member,syntax::expression_text(parsed,field_id));if(target==0){return checking::error(problem,18,f.start,f.end,0,0);}
                if(bound[target-1]!=0){return checking::error(problem,17,f.start,f.end,0,0);}bound[target-1]=1;state::set_flags(tables,field_id-1,state::small_id(target));
                let u32 type=state::primitive(parsed,syntax::field_at(parsed,target-1).type_root);state::set_expressions(tables,f.left-1,type);
                var usize symbol_id=1;while(symbol_id<=names::symbol_count(resolved)){let s=names::symbol_at(resolved,symbol_id);if(s.kind==8 && s.declaration==f.left){state::set_symbols(tables,symbol_id-1,type);}symbol_id+=1;}n+=1;
            }
            let state::Value actual=checking::block(resolved,tables,problem,arm.right,join,result_type,depth+1,loop_depth);if(problem.code!=0){return actual;}
            if(!known){join=actual.type;known=true;}else if(join!=actual.type){return checking::error(problem,1,arm.start,arm.end,join,actual.type);}
            state::set_expressions(tables,arm_id-1,actual.type);terminates=terminates && actual.terminates;link=syntax::expression_at(parsed,link).right;
        }
        if(count!=enums::member_count(parsed,subject.type)){return checking::error(problem,27,node.start,node.end,0,subject.type);}
        if(join==9 || views::valid(join)){return checking::error(problem,28,node.start,node.end,0,join);}
        return checking::recorded(tables,id,checking::value(join,subject.terminates || terminates));
    }
    return checking::error(problem,11,node.start,node.end,0,0);
}
