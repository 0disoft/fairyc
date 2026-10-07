module mir_propagation;
import hir_model as h;
import hir_shapes as shapes;
import tagged_types as tagged;
import type_state as ids;
import mir_model as m;
import mir_state as state;
import mir_cleanup as finalize;
import mir_resource_lower as lifetime;
import std::vec as v;

// The subject is an already frozen value. Error return cleanup uses the same
// immutable registration metadata as direct return, leaving sibling paths intact.
pub fn usize lower(ref h::Program input, h::Expression e, usize subject,
    ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks,
    ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut state::Context c) {
    if (c.function == 0 || h::function_at(input,c.function-1).binding != e.binding) { return state::fail(c,4,e.start,e.end); }
    let u32 operand = h::type_code(h::expression_at(input,e.left).type);
    let u32 returned = h::type_code(h::binding_at(input,e.binding).type);
    let ref shapes::Table table = h::nominal(input);
    let usize ok_member = shapes::member_at(table,operand,0); let usize err_member = shapes::member_at(table,operand,1);
    let usize condition = state::local(locals,c,6,0,false,false,e.start,e.end);
    if (state::emit(operations,blocks,c,state::operation(22,condition,6,operand,0,subject,0,e.start,e.end)) == 0) { return 0; }
    let usize success = state::new_block(blocks,c,e.start,e.end); let usize failure = state::new_block(blocks,c,e.start,e.end);
    if (!state::branch(blocks,c,condition,success,failure,e.start,e.end)) { return 0; }
    let u8[1153] continuing=c.live;let u8[4096] pending=c.pending;c.current = failure;
    let u32 error_type = tagged::failure(operand);
    let usize error = state::local(locals,c,error_type,0,false,false,e.start,e.end);
    let usize field = shapes::field_at(table,err_member).payload_start+1;
    if (state::emit(operations,blocks,c,state::operation(23,error,error_type,operand,ids::small_id(field),subject,0,e.start,e.end)) == 0) { return 0; }
    if(shapes::needs_drop(operand)){state::release(c,subject);}
    let usize result = state::local(locals,c,returned,0,false,false,e.start,e.end);
    var m::Operation make = state::operation(21,result,returned,returned,1,0,0,e.start,e.end);
    make.argument_start = v::len<usize>(edges); make.argument_count = 1;
    state::release(c,error);c.keep=result;
    if (!state::edge(edges,error,c) || state::emit(operations,blocks,c,make) == 0 || !finalize::exit(input,locals,blocks,operations,edges,c,0,0,false,e.start,e.end)) { return 0; }
    var m::Block exit = state::block(c.function,e.start,e.end); exit.kind = 1; exit.value = result;
    if (!state::terminate(blocks,c,exit)) { return 0; }
    c.live=continuing;c.pending=pending;c.keep=0;c.current = success;
    let u32 success_type = tagged::success(operand); let usize value = state::local(locals,c,success_type,0,false,false,e.start,e.end);
    if (state::emit(operations,blocks,c,state::operation(23,value,success_type,operand,ids::small_id(shapes::field_at(table,ok_member).payload_start+1),subject,0,e.start,e.end)) == 0) { return 0; }
    if(shapes::needs_drop(operand)){state::release(c,subject);}
    return value;
}
