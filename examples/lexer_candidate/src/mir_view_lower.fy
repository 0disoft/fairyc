module mir_view_lower;
import reference_types as refs;
import hir_model as h;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import std::bytes as bytes;
import std::vec as v;
import mir_lower as lowering;

// Shared owned context and source-depth budgets remain with the coordinator.
pub fn usize view_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Expression e, u32 kind, u32 type, u32 operator, usize depth) {
    if (kind == 1 || kind == 2) {
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        var m::Operation o = state::operation(1,result,type,type,0,0,0,e.start,e.end); o.boolean = e.boolean;
        if (kind == 1) {
            let ref u8[] text = h::integer_text(input,id);
            if (text.len > 688128 - bytes::len(payload)) { return state::fail(c,2,e.start,e.end); }
            o.payload_start = bytes::len(payload); if (!bytes::append(payload,text).is_ok) { return state::fail(c,3,e.start,e.end); } o.payload_end = bytes::len(payload);
        }
        if (state::emit(operations,blocks,c,o) == 0) { return 0; } return result;
    }
    if (kind == 3) {
        let usize original = c.locals[e.binding];
        if (original == 0) { return state::fail(c,4,e.start,e.end); }
        if(shapes::needs_drop(type)){return original;}
        // Freeze source reads before later child expressions can assign/call.
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(2,result,type,type,0,original,0,e.start,e.end)) == 0) { return 0; } return result;
    }
    if(kind==4 && operator==5 && refs::valid(h::type_code(h::expression_at(input,e.left).type))){
        if(shapes::needs_drop(type)){return state::fail(c,4,e.start,e.end);}let usize owner=lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);let u32 owner_type=h::type_code(h::expression_at(input,e.left).type);return if(state::emit(operations,blocks,c,state::operation(29,result,type,owner_type,0,owner,0,e.start,e.end))!=0){result}else{0};
    }
    if(kind==25){let usize owner=lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);if(owner==0){return 0;}let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);return if(state::emit(operations,blocks,c,state::operation(34,result,type,h::type_code(h::expression_at(input,e.left).type),0,owner,0,e.start,e.end))!=0){result}else{0};}
    if (kind == 16) {
        if(refs::valid(type)){return lowering::address(input,locals,blocks,operations,edges,payload,c,e.left,e.boolean,depth+1);}
        let h::Expression source=h::expression_at(input,e.left);
        let bool projected=h::expression_code(source.kind)==10;
        let usize owner=if(projected){lowering::address(input,locals,blocks,operations,edges,payload,c,e.left,e.boolean,depth+1)}else{c.locals[source.binding]};
        if (owner == 0) { return state::fail(c,4,e.start,e.end); }
        let u32 array_type = h::type_code(h::expression_at(input,e.left).type);
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(18,result,type,if(projected){v::at<m::Local>(locals,owner-1).type}else{array_type},0,owner,0,e.start,e.end)) == 0) { return 0; } return result;
    }
    if (kind == 17) {
        let usize owner = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1); if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize raw = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.right,depth + 1); if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize start = state::local(locals,c,5,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(2,start,5,5,0,raw,0,e.start,e.end)) == 0) { return 0; }
        var usize end = 0;
        if (e.boolean) { end = state::local(locals,c,5,0,false,false,e.start,e.end); if (state::emit(operations,blocks,c,state::operation(15,end,5,type,0,owner,0,e.start,e.end)) == 0) { return 0; } }
        else { end = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.alternative,depth + 1); if (c.current == 0 || c.problem.code != 0) { return 0; } }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end); var m::Operation o = state::operation(19,result,type,type,0,owner,start,e.start,e.end);
        o.argument_start = v::len<usize>(edges); o.argument_count = 1;
        if (!state::edge(edges,end,c) || state::emit(operations,blocks,c,o) == 0) { return 0; } return result;
    }
    if (kind == 15) {
        let ref u8[] data = h::literal_bytes(input,id);
        if (bytes::len(payload) > 688128 || data.len > 688128 - bytes::len(payload)) { return state::fail(c,2,e.start,e.end); }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end); var m::Operation o = state::operation(14,result,type,type,0,0,0,e.start,e.end);
        o.payload_start = bytes::len(payload); if (!bytes::append(payload,data).is_ok) { return state::fail(c,3,e.start,e.end); } o.payload_end = bytes::len(payload);
        if (state::emit(operations,blocks,c,o) == 0) { return 0; } return result;
    }
    if ((kind == 13 || kind == 14) && h::type_code(h::expression_at(input,e.left).type) < 256) {
        let u32 owner_type = h::type_code(h::expression_at(input,e.left).type);
        let usize owner = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1); if (c.current == 0 || c.problem.code != 0) { return 0; }
        if (kind == 14) { let usize result = state::local(locals,c,type,0,false,false,e.start,e.end); var m::Operation o = state::operation(15,result,type,owner_type,0,owner,0,e.start,e.end); o.boolean = e.boolean; if (state::emit(operations,blocks,c,o) == 0) { return 0; } return result; }
        let usize raw = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1); if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize index = state::local(locals,c,5,0,false,false,e.start,e.end); if (state::emit(operations,blocks,c,state::operation(16,index,5,owner_type,0,owner,raw,e.start,e.end)) == 0) { return 0; }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end); if (state::emit(operations,blocks,c,state::operation(17,result,type,owner_type,0,owner,index,e.start,e.end)) == 0) { return 0; } return result;
    }
    return state::fail(c,4,e.start,e.end);
}
