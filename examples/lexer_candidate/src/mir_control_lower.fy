module mir_control_lower;
import type_state as ids;
import hir_model as h;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import mir_ownership_snapshot as snapshots;
import mir_resource_lower as lifetime;
import std::bytes as bytes;
import std::vec as v;
import mir_lower as lowering;

// Shared owned context and source-depth budgets remain with the coordinator.
pub fn usize switch_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, h::Expression e, u32 type, usize depth) {
        // Evaluate the subject once. The final exhaustive arm needs no test.
        let usize subject = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 owner = h::type_code(h::expression_at(input,e.left).type);
        let shapes::Shape shape = shapes::at(h::nominal(input),shapes::find(h::nominal(input),owner) - 1);
        let usize result = if (type == 7) { 0 } else { state::local(locals,c,type,0,false,false,e.start,e.end) };
        let u8[1153] incoming=c.live;let u8[4096] incoming_pending=c.pending;var u8[1153] merged=[0;1153];var bool continuing=false;
        var v::Vec<usize> ends = v::empty<usize>(); var usize i = 0;
        while (i < e.edge_count) {
            c.live=incoming;c.pending=incoming_pending;
            let h::Expression arm = h::expression_at(input,h::edge_at(input,e.edge_start + i)); var usize next = 0;
            if (i + 1 < e.edge_count) {
                let usize condition = state::local(locals,c,6,0,false,false,arm.start,arm.end);
                let u32 tag = ids::small_id(shapes::member_tag(h::nominal(input),owner,arm.binding));
                if (state::emit(operations,blocks,c,state::operation(22,condition,6,owner,tag,subject,0,arm.start,arm.end)) == 0) { return 0; }
                let usize body = state::new_block(blocks,c,arm.start,arm.end); next = state::new_block(blocks,c,e.start,e.end);
                if (!state::branch(blocks,c,condition,body,next,arm.start,arm.end)) { return 0; } c.current = body;
            }
            var usize capture=0;while(capture<arm.edge_count){let field=h::expression_at(input,h::edge_at(input,arm.edge_start+capture));
                if(field.edge_count==1){let usize binding=h::edge_at(input,field.edge_start);let usize local=c.locals[binding];let u32 field_type=h::type_code(field.type);
                    if(local==0 || state::emit(operations,blocks,c,state::operation(23,local,field_type,owner,ids::small_id(field.binding),subject,0,field.start,field.end))==0){return 0;}
                    if(shapes::needs_drop(owner)){state::release(c,subject);}if(shapes::needs_drop(field_type)){c.live[binding]=1;}
                }else if(shapes::needs_drop(owner)){
                    let u32 field_type=h::type_code(field.type);let usize ignored=state::local(locals,c,field_type,0,false,false,field.start,field.end);
                    if(state::emit(operations,blocks,c,state::operation(23,ignored,field_type,owner,ids::small_id(field.binding),subject,0,field.start,field.end))==0){return 0;}
                    state::release(c,subject);if(shapes::needs_drop(field_type)){state::in_scope(c,ignored,h::block_at(input,arm.block).scope);}
                }capture+=1;
            }
            if(shapes::needs_drop(owner)){var bool extracted=arm.edge_count!=0;var usize j=0;while(j<arm.edge_count){extracted=extracted || h::expression_at(input,h::edge_at(input,arm.edge_start+j)).edge_count==1;j+=1;}
                if(!extracted && !lifetime::drop(input,locals,blocks,operations,edges,c,subject,arm.start,arm.end)){return 0;}}
            let usize value = lowering::block(input,locals,blocks,operations,edges,payload,c,arm.block,depth + 1);
            if (c.problem.code != 0) { return 0; }
            if (c.current != 0) {
                if (result != 0 && state::emit(operations,blocks,c,state::operation(if(shapes::needs_drop(type)){24}else{2},result,type,type,0,value,0,arm.start,arm.end)) == 0) { return 0; }
                if(!continuing){merged=c.live;continuing=true;}
                if (!v::push<usize>(ref mut ends,c.current).is_ok) { return state::fail(c,3,arm.start,arm.end); }
            }
            c.current = next; i += 1;
        }
        if (v::len<usize>(ref ends) == 0) { c.current = 0; return 0; }
        let usize join = state::new_block(blocks,c,e.start,e.end); i = 0;
        while (i < v::len<usize>(ref ends)) { c.current = *v::at<usize>(ref ends,i); if (!state::jump(blocks,c,join,e.start,e.end)) { return 0; } i += 1; }
        c.live=merged;c.current = join; return result;
}

pub fn usize if_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, h::Expression e, u32 type, usize depth) {
        let usize condition = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize consequent = state::new_block(blocks,c,e.start,e.end); let usize alternative = state::new_block(blocks,c,e.start,e.end);
        if (!state::branch(blocks,c,condition,consequent,alternative,e.start,e.end)) { return 0; }
        let usize result = if (type == 7) { 0 } else { state::local(locals,c,type,0,false,false,e.start,e.end) };
        let bytes::Buffer incoming=snapshots::capture(c,e.start,e.end);if(c.problem.code!=0){return 0;}
        c.current = consequent; let usize yes = lowering::block(input,locals,blocks,operations,edges,payload,c,e.block,depth + 1);
        if (c.problem.code != 0) { return 0; }
        if (c.current != 0 && result != 0 && state::emit(operations,blocks,c,state::operation(if(shapes::needs_drop(type)){24}else{2},result,type,type,0,yes,0,e.start,e.end)) == 0) { return 0; }
        let usize yes_end = c.current;
        let bytes::Buffer yes_state=snapshots::capture(c,e.start,e.end);if(c.problem.code!=0){return 0;}snapshots::restore(c,ref incoming);
        c.current = alternative; let usize no = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.alternative,depth + 1);
        if (c.problem.code != 0) { return 0; }
        if (c.current != 0 && result != 0 && state::emit(operations,blocks,c,state::operation(if(shapes::needs_drop(type)){24}else{2},result,type,type,0,no,0,e.start,e.end)) == 0) { return 0; }
        let usize no_end = c.current;
        if(yes_end!=0){snapshots::restore(c,ref yes_state);}
        // Allocate a join only for live fallthrough; both returning arms leave
        // no orphan unfinished block and suppress enclosing RHS/store/return.
        if (yes_end == 0 && no_end == 0) { c.current = 0; return 0; }
        let usize join = state::new_block(blocks,c,e.start,e.end);
        if (yes_end != 0) { c.current = yes_end; if (!state::jump(blocks,c,join,e.start,e.end)) { return 0; } }
        if (no_end != 0) { c.current = no_end; if (!state::jump(blocks,c,join,e.start,e.end)) { return 0; } }
        c.current = join; return result;
}

pub fn usize logical_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, h::Expression e, u32 operator, usize left, usize depth) {
                // Default result is the frozen left value. Only the required
                // RHS path overwrites it; its return/trap never reaches join.
                let usize result = state::local(locals,c,6,0,false,false,e.start,e.end);
                if (state::emit(operations,blocks,c,state::operation(2,result,6,6,0,left,0,e.start,e.end)) == 0) { return 0; }
                let usize rhs = state::new_block(blocks,c,e.start,e.end); let usize join = state::new_block(blocks,c,e.start,e.end);
                let usize consequent = if (operator == 19) { rhs } else { join };
                let usize alternative = if (operator == 19) { join } else { rhs };
                if (!state::branch(blocks,c,left,consequent,alternative,e.start,e.end)) { return 0; }
                let u8[1153] skipped=c.live;let u8[4096] skipped_pending=c.pending;
                c.current = rhs; let usize right = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.right,depth + 1);
                if (c.problem.code != 0) { return 0; }
                if (c.current != 0) {
                    if (state::emit(operations,blocks,c,state::operation(2,result,6,6,0,right,0,e.start,e.end)) == 0) { return 0; }
                    // RHS-only owned temporaries cannot cross the skipped edge.
                    // Freeze the boolean, then discharge those values in reverse
                    // creation order before both paths meet with identical states.
                    var usize temporary=v::len<m::Local>(locals);
                    while(temporary>c.base){if(state::pending(c,temporary)==1 && skipped_pending[temporary-c.base-1]==0){if(!lifetime::drop(input,locals,blocks,operations,edges,c,temporary,e.start,e.end)){return 0;}}temporary-=1;}
                    if (!state::jump(blocks,c,join,e.start,e.end)) { return 0; }
                }
                c.live=skipped;c.pending=skipped_pending;c.current = join; return result;
}
