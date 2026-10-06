module c_flow;
import mir_model as m;
import c_guard_facts as guards;
import std::bytes as bytes;
import std::vec as v;

pub resource Plan { bytes::Buffer nonreturn; bytes::Buffer reachable; cleanup { } }
fn void mark(ref mut bytes::Buffer flags, usize id) { let ref mut u8[] data = bytes::view_mut(flags); data[id] = 1; }
pub fn bool stops(ref Plan plan, usize target) { return bytes::view(ref plan.nonreturn)[target] != 0; }
pub fn bool live(ref Plan plan, usize block) { return bytes::view(ref plan.reachable)[block] != 0; }
// Unique immutable literals/copies can prove a checked trap before the operation.
// Mutable/parameter/join values remain governed by the generated runtime guard.
pub fn u32 failure(ref m::Program p, m::Block b) {
    return guards::failure(p,b);
}
// Only the final definition in this block can prove its bool successor.
pub fn usize selected(ref m::Program p, m::Block b) {
    if (failure(p,b) != 0) { return b.alternative; }
    let usize local = if (b.kind == 3) { b.value } else if (b.kind == 4 && b.operator == 0) { b.left } else { 0 };
    if (local == 0) { return 0; }
    var usize op = b.first; var bool known = false; var bool value = false;
    while (op != 0) {
        let m::Operation o = m::operation_at(p,op);
        if (o.result == local) { known = o.kind == 1 && o.type == 6; value = o.boolean; }
        op = o.next;
    }
    return if (!known) { 0 } else if (value) { b.target } else { b.alternative };
}
fn bool enqueue(ref mut bytes::Buffer seen, ref mut v::Vec<usize> queue, usize id) {
    if (id == 0 || bytes::view(seen)[id] != 0) { return true; }
    mark(seen,id); return v::push<usize>(queue,id).is_ok;
}
pub fn Result<Plan,u32> build(ref m::Program p) {
    var bytes::Buffer stopped = bytes::empty(); var bytes::Buffer reached = bytes::empty();
    if (!bytes::resize_zeroed(ref mut stopped,m::target_count(p) + 1).is_ok || !bytes::resize_zeroed(ref mut reached,m::block_count(p) + 1).is_ok) { return .err(3); }
    var v::Vec<usize> queue = v::empty<usize>(); var usize work = 0; var bool changed = true;
    // Unknown calls may return. Remove a continuation only after its closed
    // callee has no reachable return. Recursive cycles stay conservative.
    while (changed) {
        changed = false;
        { let ref mut u8[] data = bytes::view_mut(ref mut reached); var usize i = 0; while (i < data.len) { data[i] = 0; i += 1; } }
        var usize fid = 1; while (fid <= m::function_count(p)) {
            let m::Function f = m::function_at(p,fid); var bool returns = false;
            if (!enqueue(ref mut reached,ref mut queue,f.entry)) { return .err(3); }
            while (v::len<usize>(ref queue) != 0) {
                if (work == 67108864) { return .err(2); } work += 1;
                let usize id = switch(v::pop<usize>(ref mut queue)) { case .some(value): { value } case .none: { 0 } };
                if (id == 0) { return .err(1); }
                let m::Block b = m::block_at(p,id); returns = returns || b.kind == 1;
                if (b.kind != 5 || bytes::view(ref stopped)[b.callee] == 0) {
                    let usize edge = selected(p,b);
                    if (edge != 0) { if (!enqueue(ref mut reached,ref mut queue,edge)) { return .err(3); } }
                    else if (!enqueue(ref mut reached,ref mut queue,b.target) || !enqueue(ref mut reached,ref mut queue,b.alternative)) { return .err(3); }
                }
            }
            if (!returns && bytes::view(ref stopped)[f.target] == 0) { mark(ref mut stopped,f.target); changed = true; }
            fid += 1;
        }
    }
    return .ok(Plan { .nonreturn = stopped, .reachable = reached });
}
