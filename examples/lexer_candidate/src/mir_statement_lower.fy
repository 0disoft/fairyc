module mir_statement_lower;
import scalar_views as slices;
import reference_types as refs;
import mir_places as places;
import hir_model as h;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import mir_math as math;
import mir_cleanup as finalize;
import mir_resource_lower as lifetime;
import mir_nominal as nominal;
import std::bytes as bytes;
import std::vec as v;
import mir_lower as lowering;

// Large defer captures and loop snapshots belong only to their selected
// statement helpers, rather than every recursive statement frame.
fn bool defer_statement(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Statement s, usize depth) {
    let h::Expression call = h::expression_at(input,s.expression);
    let usize callee = c.targets[call.binding];
    let u32 target_kind=h::binding_code(h::binding_at(input,call.binding).kind);
    if (h::expression_code(call.kind) != 6 || h::type_code(call.type) != 7 || callee == 0 || (target_kind!=1 && target_kind!=11)) { discard state::fail(c,4,s.start,s.end); return false; }
    if (c.defer_count >= 256) { discard state::fail(c,2,s.start,s.end); return false; }
    let usize captured = c.capture_count; var usize j = 0;
    while (j < call.edge_count) {
        let usize value = lowering::expression(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,call.edge_start + j),depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        if (c.capture_count >= 4096) { discard state::fail(c,2,s.start,s.end); return false; }
        let u32 capture_type=v::at<m::Local>(locals,value-1).type;if(capture_type<1 || capture_type>6){discard state::fail(c,4,s.start,s.end);return false;}
        let usize slot = c.capture_count; c.captures[slot] = value; c.capture_count += 1; j += 1;
    }
    let usize at = c.defer_count; c.defer_count += 1;
    c.deferred_unsafe[at]=call.boolean; c.deferred_targets[at] = callee; c.deferred_starts[at] = s.start; c.deferred_ends[at] = s.end;
    c.deferred_arg_starts[at] = captured; c.deferred_arg_counts[at] = call.edge_count;
    return true;
}

fn bool loop_statement(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Statement s, usize depth) {
    if (s.initial != 0 && !statement(input,locals,blocks,operations,edges,payload,c,s.initial,depth + 1)) { return false; }
    if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
    let usize header = state::new_block(blocks,c,s.start,s.end);
    if (!state::jump(blocks,c,header,s.start,s.end)) { return false; }
    c.current = header;
    var usize condition = 0;
    if (s.expression != 0) { condition = lowering::expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1); }
    else {
        condition = state::local(locals,c,6,0,false,false,s.start,s.end);
        var m::Operation truth = state::operation(1,condition,6,6,0,0,0,s.start,s.end); truth.boolean = true;
        if (state::emit(operations,blocks,c,truth) == 0) { return false; }
    }
    if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
    let usize body = state::new_block(blocks,c,s.start,s.end);
    let usize exit = state::new_block(blocks,c,s.start,s.end);
    if (!state::branch(blocks,c,condition,body,exit,s.start,s.end)) { return false; }
    if (c.loop_depth >= 64) { discard state::fail(c,2,s.start,s.end); return false; }
    let usize frame = c.loop_depth; c.loop_depth += 1;
    let u8[1153] incoming=c.live;let u8[4096] incoming_pending=c.pending;
    c.loop_ids[frame] = id; c.breaks[frame] = exit;
    let usize deferred_base = c.defer_count; c.loop_defer_base[frame] = deferred_base;
    c.continues[frame] = if (s.post == 0) { header } else { 0 };
    c.current = body;
    discard lowering::block(input,locals,blocks,operations,edges,payload,c,s.body,depth + 1);
    if (c.problem.code != 0) { return false; }
    if (c.current != 0) {
        if (c.continues[frame] == 0) { let usize post = state::new_block(blocks,c,s.start,s.end); c.continues[frame] = post; }
        let usize target = c.continues[frame];
        if (!state::jump(blocks,c,target,s.start,s.end)) { return false; }
    }
    let usize post = c.continues[frame]; c.loop_depth -= 1;
    if (s.post != 0 && post != 0) {
        c.current = post;
        if (!statement(input,locals,blocks,operations,edges,payload,c,s.post,depth + 1)) { return false; }
        if (c.current != 0 && !state::jump(blocks,c,header,s.start,s.end)) { return false; }
    }
    c.live=incoming;c.pending=incoming_pending;c.current = exit; return true;
}

// Shared owned context and source-depth budgets remain with the coordinator.
pub fn bool statement(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, usize depth) {
    let h::Statement s = h::statement_at(input,id); let u32 kind = h::statement_code(s.kind);
    if (depth >= 64) { discard state::fail(c,2,s.start,s.end); return false; }
    if (kind == 11) { return defer_statement(input,locals,blocks,operations,edges,payload,c,id,s,depth); }
    if (kind == 7 || kind == 8) {
        if (c.loop_depth == 0 || c.loop_ids[c.loop_depth - 1] != s.loop_target) { discard state::fail(c,4,s.start,s.end); return false; }
        let usize frame = c.loop_depth - 1;
        if (kind == 8 && c.continues[frame] == 0) {
            // A for-post block is created only when a continue or fallthrough
            // reaches it, so an unconditional break/return leaves no orphan.
            let h::Statement loop = h::statement_at(input,s.loop_target);
            let usize post = state::new_block(blocks,c,loop.start,loop.end); c.continues[frame] = post;
        }
        let usize target = if (kind == 7) { c.breaks[frame] } else { c.continues[frame] };
        let usize from = c.loop_defer_base[frame];
        let h::Statement owner=h::statement_at(input,s.loop_target);
        if(!finalize::exit(input,locals,blocks,operations,edges,c,from,h::block_at(input,owner.body).scope,true,s.start,s.end)){return false;}
        return state::jump(blocks,c,target,s.start,s.end);
    }
    if(kind==12){let e=h::expression_at(input,s.expression);let usize binding=e.binding;let usize local=c.locals[binding];if(h::expression_code(e.kind)!=24 || local==0 || c.live[binding]==0){discard state::fail(c,4,s.start,s.end);return false;}c.live[binding]=0;state::release(c,local);let u32 type=h::type_code(e.type);return state::emit(operations,blocks,c,state::operation(27,0,type,type,0,local,0,s.start,s.end))!=0;}
    if (kind == 5 || kind == 6) { return loop_statement(input,locals,blocks,operations,edges,payload,c,id,s,depth); }
    if(kind==3){let target=h::expression_at(input,s.target);let usize root=places::root(input,s.target,false);let bool deref=h::expression_code(target.kind)==4 && h::operator_code(target.operator)==5;
        if(deref || root!=0 && refs::mutable(h::type_code(h::binding_at(input,root).type)) && s.target!=0 && h::expression_code(target.kind)==10){
            let usize pointer=if(deref){lowering::expression(input,locals,blocks,operations,edges,payload,c,target.left,depth+1)}else{lowering::address(input,locals,blocks,operations,edges,payload,c,s.target,true,depth+1)};if(pointer==0 || c.current==0){return c.problem.code==0;}
            let u32 type=h::type_code(target.type);let u32 owner_type=v::at<m::Local>(locals,pointer-1).type;let u32 op=h::operator_code(s.operator);var usize prior=0;
            if(op!=21){prior=state::local(locals,c,type,0,false,false,s.start,s.end);if(state::emit(operations,blocks,c,state::operation(29,prior,type,owner_type,0,pointer,0,s.start,s.end))==0){return false;}}
            var usize value=lowering::expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth+1);if(c.current==0 || c.problem.code!=0){return c.problem.code==0;}if(op!=21){value=math::binary(locals,blocks,operations,c,type,type,op,prior,value,s.start,s.end);}
            return state::emit(operations,blocks,c,state::operation(30,0,type,owner_type,0,pointer,value,s.start,s.end))!=0;
        }
    }
    if (kind == 3 && h::expression_code(h::expression_at(input,s.target).kind) == 13 && slices::mutable(h::type_code(h::expression_at(input,h::expression_at(input,s.target).left).type))) {
        let target = h::expression_at(input,s.target); let u32 operator = h::operator_code(s.operator);let u32 owner_type=h::type_code(h::expression_at(input,target.left).type);let u32 type=h::type_code(target.type);
        // Freeze the descriptor, then evaluate and check the index before RHS.
        let usize owner = lowering::expression(input,locals,blocks,operations,edges,payload,c,target.left,depth+1);
        let usize raw = lowering::expression(input,locals,blocks,operations,edges,payload,c,target.right,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        let usize index = state::local(locals,c,5,0,false,false,s.start,s.end);
        if (state::emit(operations,blocks,c,state::operation(16,index,5,owner_type,0,owner,raw,s.start,s.end)) == 0) { return false; }
        var usize prior = 0;
        if (operator != 21) { prior = state::local(locals,c,type,0,false,false,s.start,s.end); if (state::emit(operations,blocks,c,state::operation(17,prior,type,owner_type,0,owner,index,s.start,s.end)) == 0) { return false; } }
        var usize value = lowering::expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        if (operator != 21) { value = math::binary(locals,blocks,operations,c,type,type,operator,prior,value,s.start,s.end); }
        var m::Operation o = state::operation(20,0,type,owner_type,0,owner,index,s.start,s.end); o.argument_start=v::len<usize>(edges); o.argument_count=1;
        return state::edge(edges,value,c) && state::emit(operations,blocks,c,o) != 0;
    }
    if (kind == 3 && h::expression_code(h::expression_at(input,s.target).kind) == 13) {
        let usize binding=places::root(input,s.target,true);let usize destination=c.locals[binding];
        if(binding==0 || destination==0){discard state::fail(c,4,s.start,s.end);return false;}
        var v::Vec<places::Step> path=switch(places::collect(input,s.target)){case .ok(p):{p}case .err(code):{discard state::fail(c,code,s.start,s.end);return false;}};
        if(!lowering::prepare_place(input,ref mut path,locals,blocks,operations,edges,payload,c,depth)){return c.problem.code==0;}
        let u32 type=h::type_code(h::expression_at(input,s.target).type);let u32 operator=h::operator_code(s.operator);var usize prior=0;
        if(operator!=21){prior=places::read(input,ref path,destination,locals,blocks,operations,c,s.start,s.end);}
        var usize value=lowering::expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth+1);if(c.current==0 || c.problem.code!=0){return c.problem.code==0;}
        if(operator!=21){value=math::binary(locals,blocks,operations,c,type,type,operator,prior,value,s.start,s.end);}
        value=places::store(input,ref path,destination,value,locals,blocks,operations,edges,c,s.start,s.end);if(value==0){return false;}
        let u32 owner=v::at<m::Local>(locals,destination-1).type;
        return state::emit(operations,blocks,c,state::operation(2,destination,owner,owner,0,value,0,s.start,s.end))!=0;
    }
    if (kind == 1 || kind == 3) {
        let usize binding = if (kind == 1) { s.binding } else { nominal::root(input,s.target) };
        let usize destination = c.locals[binding]; if (destination == 0) { discard state::fail(c,4,s.start,s.end); return false; }
        let u32 type = if (kind == 1) { v::at<m::Local>(locals,destination - 1).type } else { h::type_code(h::expression_at(input,s.target).type) }; let u32 operator = h::operator_code(s.operator);
        var usize left = 0;
        if (kind == 3 && operator != 21) {
            let h::Expression target = h::expression_at(input,s.target);
            left = lowering::expression(input,locals,blocks,operations,edges,payload,c,s.target,depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        }
        var usize value = lowering::expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        if (kind == 3 && operator != 21) { value = math::binary(locals,blocks,operations,c,type,type,operator,left,value,s.start,s.end); }
        if (kind == 3) { value = nominal::store(input,s.target,destination,value,locals,blocks,operations,c,s.start,s.end); if (value == 0) { return false; } }
        let u32 destination_type = v::at<m::Local>(locals,destination - 1).type;
        if(shapes::needs_drop(destination_type)){
            if(kind==3 && h::expression_code(h::expression_at(input,s.target).kind)==10){return value==destination;}
            // The complete RHS is frozen above. A move of the old root has
            // already discharged it; otherwise retire its previous generation.
            if(c.live[binding]!=0){
                if(c.live[binding]!=1){discard state::fail(c,1104,s.start,s.end);return false;}
                c.live[binding]=0;if(!lifetime::drop(input,locals,blocks,operations,edges,c,destination,s.start,s.end)){return false;}
            }
            c.live[binding]=1;return state::emit(operations,blocks,c,state::operation(24,destination,destination_type,destination_type,0,value,0,s.start,s.end))!=0;
        }
        return state::emit(operations,blocks,c,state::operation(2,destination,destination_type,destination_type,0,value,0,s.start,s.end)) != 0;
    }
    if (kind == 2) {
        let usize value = lowering::expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        c.keep=value;
        if (!finalize::exit(input,locals,blocks,operations,edges,c,0,0,false,s.start,s.end)) { return false; }
        var m::Block term = state::block(c.function,s.start,s.end); term.kind = 1; term.value = value;
        return state::terminate(blocks,c,term);
    }
    if (kind == 4 || kind == 9 || kind == 10) {
        let usize value = lowering::expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        if (kind == 9) {
            if (!state::guard(operations,blocks,c,6,0,value,0,16,s.start,s.end)) { return false; }
            let h::Expression assertion = h::expression_at(input,s.expression);
            if (h::expression_code(assertion.kind) == 2 && !assertion.boolean) {
                var m::Block term = state::block(c.function,s.start,s.end); term.kind = 7;
                return state::terminate(blocks,c,term);
            }
            return true;
        }
        if(value!=0 && shapes::needs_drop(v::at<m::Local>(locals,value-1).type)){return lifetime::drop(input,locals,blocks,operations,edges,c,value,s.start,s.end);}
        return true;
    }
    discard state::fail(c,4,s.start,s.end); return false;
}
