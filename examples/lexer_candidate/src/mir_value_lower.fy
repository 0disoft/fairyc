module mir_value_lower;
import reference_types as refs;
import type_state as ids;
import mir_arrays as array;
import mir_places as places;
import hir_model as h;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import mir_math as math;
import mir_nominal as nominal;
import std::bytes as bytes;
import std::vec as v;
import mir_lower as lowering;
import mir_control_lower as control;

// Shared owned context and source-depth budgets remain with the coordinator.
pub fn usize aggregate_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Expression e, u32 kind, u32 type, u32 operator, usize depth) {
    if (kind == 12) {
        var v::Vec<usize> values = v::empty<usize>(); var usize i = 0;
        while (i < e.edge_count) { let usize value = lowering::expression(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,e.edge_start+i),depth+1); if (c.current == 0 || c.problem.code != 0) { return 0; } if (!v::push<usize>(ref mut values,value).is_ok) { return state::fail(c,3,e.start,e.end); } i += 1; }
        let usize first = v::len<usize>(edges); i = 0;
        while (i < e.edge_count) { if (!state::edge(edges,*v::at<usize>(ref values,i),c)) { return 0; } i += 1; }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        var m::Operation o = state::operation(9,result,type,type,0,0,0,e.start,e.end); o.argument_start=first; o.argument_count=e.edge_count; o.boolean=e.boolean;
        if (state::emit(operations,blocks,c,o)==0) { return 0; } return result;
    }
    if (kind == 13 && places::root(input,id,false)!=0) {
        var v::Vec<places::Step> path=switch(places::collect(input,id)){case .ok(p):{p}case .err(code):{return state::fail(c,code,e.start,e.end);}};
        if(!lowering::prepare_place(input,ref mut path,locals,blocks,operations,edges,payload,c,depth)){return 0;}
        return places::read(input,ref path,c.locals[places::root(input,id,false)],locals,blocks,operations,c,e.start,e.end);
    }
    if (kind == 13 || kind == 14) {
        let h::Expression root = h::expression_at(input,e.left);
        let usize owner = if (kind == 13 && h::expression_code(root.kind) == 3) { c.locals[root.binding] } else { lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1) };
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 owner_type = h::type_code(h::expression_at(input,e.left).type);
        if (kind == 14) { let usize result = state::local(locals,c,5,0,false,false,e.start,e.end); if (state::emit(operations,blocks,c,state::operation(13,result,5,owner_type,0,owner,0,e.start,e.end))==0) { return 0; } return result; }
        let usize raw = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize index = array::index(locals,blocks,operations,c,owner_type,raw,e.start,e.end);
        return array::read(locals,blocks,operations,c,owner_type,type,owner,index,e.start,e.end);
    }
    if (kind == 10) {
        let usize owner = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 receiver_type=h::type_code(h::expression_at(input,e.left).type);if(refs::valid(receiver_type)){
            if(shapes::needs_drop(type)){return state::fail(c,4,e.start,e.end);}let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);return if(state::emit(operations,blocks,c,state::operation(31,result,type,receiver_type,ids::small_id(e.binding),owner,0,e.start,e.end))!=0){result}else{0};}
        return nominal::read(locals,blocks,operations,c,owner,type,e.binding,e.start,e.end);
    }
    if (kind == 9) {
        let ref shapes::Table table = h::nominal(input); let usize index = shapes::find(table,type);
        if (index == 0 || e.edge_count > 65536 - v::len<usize>(edges)) { return state::fail(c,4,e.start,e.end); }
        let shapes::Shape shape = shapes::at(table,index - 1);
        var v::Vec<usize> values = v::empty<usize>(); var usize i = 0;
        while (i < shape.field_count) { if (!v::push<usize>(ref mut values,0).is_ok) { return state::fail(c,3,e.start,e.end); } i += 1; }
        i = 0;
        while (i < e.edge_count) {
            let h::Expression initializer = h::expression_at(input,h::edge_at(input,e.edge_start + i));
            let usize value = lowering::expression(input,locals,blocks,operations,edges,payload,c,initializer.left,depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return 0; }
            { let ref mut usize slot = v::at_mut<usize>(ref mut values,initializer.binding - shape.field_start - 1); *slot = value; }
            i += 1;
        }
        let usize start = v::len<usize>(edges); i = 0;
        while (i < shape.field_count) { let usize value=*v::at<usize>(ref values,i);if (!state::edge(edges,value,c)) { return 0; }state::release(c,value);i += 1; }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        var m::Operation o = state::operation(6,result,type,type,0,0,0,e.start,e.end); o.argument_start = start; o.argument_count = shape.field_count;
        if (state::emit(operations,blocks,c,o) == 0) { return 0; } return result;
    }
    if (kind == 4 || kind == 5) {
        let usize left = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 owner_type = h::type_code(h::expression_at(input,e.left).type);
        if (kind == 5) {
            if (operator == 19 || operator == 20) { return control::logical_expression(input,locals,blocks,operations,edges,payload,c,e,operator,left,depth); }
            let usize right = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.right,depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return 0; }
            return math::binary(locals,blocks,operations,c,type,owner_type,operator,left,right,e.start,e.end);
        }
        let u32 mask = math::trap_mask(operator,owner_type,true);
        if (mask != 0 && !state::guard(operations,blocks,c,owner_type,operator,left,0,mask,e.start,e.end)) { return 0; }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(3,result,type,owner_type,operator,left,0,e.start,e.end)) == 0) { return 0; } return result;
    }
    return state::fail(c,4,e.start,e.end);
}
pub fn usize call_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Expression e, u32 kind, u32 type, u32 operator, usize depth) {
    if (kind == 6) {
        let usize callee = c.targets[e.binding]; if (callee == 0) { return state::fail(c,4,e.start,e.end); }
        // Materialize ordered arguments before publishing their contiguous
        // range. An argument may return, leaving no orphan parent-call edges.
        if (e.edge_count > 65536 - v::len<usize>(edges)) { return state::fail(c,2,e.start,e.end); }
        var v::Vec<usize> arguments = v::empty<usize>(); var usize j = 0;
        while (j < e.edge_count) {
            let usize argument = lowering::expression(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,e.edge_start + j),depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return 0; }
            if (!v::push<usize>(ref mut arguments,argument).is_ok) { return state::fail(c,3,e.start,e.end); } j += 1;
        }
        let usize start = v::len<usize>(edges); j = 0;
        while (j < e.edge_count) {
            let usize argument = *v::at<usize>(ref arguments,j);
            if (!state::edge(edges,argument,c)) { return 0; }state::release(c,argument);j += 1;
        }
        let usize destination = if (type == 7) { 0 } else { state::local(locals,c,type,0,false,false,e.start,e.end) };
        let usize continuation = state::new_block(blocks,c,e.start,e.end);
        var m::Block term = state::block(c.function,e.start,e.end); term.kind = 5; term.callee = callee; term.unsafe_call=e.boolean;
        term.destination = destination; term.argument_start = start; term.argument_count = e.edge_count; term.target = continuation;
        if (!state::terminate(blocks,c,term)) { return 0; } c.current = continuation;if(shapes::needs_drop(type)){state::hold(c,destination);}return destination;
    }
    if (kind == 7) { return lowering::block(input,locals,blocks,operations,edges,payload,c,e.block,depth + 1); }
    if (kind == 8) { return control::if_expression(input,locals,blocks,operations,edges,payload,c,e,type,depth); }
    return state::fail(c,4,e.start,e.end);
}
