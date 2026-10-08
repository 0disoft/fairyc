module mir_verify_flow;
import mir_model as m;
import mir_verify_shape as shape;
import std::bytes as bytes;
import std::vec as v;

struct Predecessor { usize block; usize next; }
pub struct Budget { usize work; }
fn bool charge(ref mut Budget budget, usize count) {
    if (budget.work > 67108864 || count > 67108864 - budget.work) { return false; } budget.work += count; return true;
}
fn u8 bit(usize index) { let u8[8] masks = [1,2,4,8,16,32,64,128]; return masks[index % 8]; }
fn bool has(ref bytes::Buffer data, usize index) { return (bytes::view(data)[index / 8] & bit(index)) != 0; }
fn void set(ref mut bytes::Buffer data, usize index) { let ref mut u8[] view = bytes::view_mut(data); view[index / 8] |= bit(index); }
fn bool enqueue(ref mut v::Vec<usize> queue, ref mut bytes::Buffer pending, usize id) {
    if (bytes::view(pending)[id] != 0) { return true; }
    if (!v::push<usize>(queue,id).is_ok) { return false; }
    let ref mut u8[] flags = bytes::view_mut(pending); flags[id] = 1; return true;
}
fn usize pop(ref mut v::Vec<usize> queue, ref mut bytes::Buffer pending) {
    let usize id = switch(v::pop<usize>(queue)) { case .some(value): { value } case .none: { 0 } };
    let ref mut u8[] flags = bytes::view_mut(pending); flags[id] = 0; return id;
}
fn bool successors(ref m::Program p, usize from, ref mut v::Vec<usize> queue, ref mut bytes::Buffer pending) {
    let m::Block b = m::block_at(p,from);
    return (b.target == 0 || enqueue(queue,pending,b.target)) && (b.alternative == 0 || enqueue(queue,pending,b.alternative));
}
fn bool read(ref bytes::Buffer initialized, usize id, usize base) { return id == 0 || has(initialized,id - base - 1); }
fn u32 transfer(ref m::Program p, m::Block b, usize base, ref mut bytes::Buffer initialized, bool inspect, ref mut Budget budget) {
    var usize op = b.first;
    while (op != 0) {
        let m::Operation o = m::operation_at(p,op);
        if (!charge(budget,1)) { return 2; }
        if (inspect && (!read(initialized,o.left,base) || !read(initialized,o.right,base))) { return 1002; }
        if (o.kind == 6 || o.kind == 21 || o.kind == 9 || o.kind == 11 || o.kind==38 || o.kind==39 || (o.kind == 19 || o.kind == 20)) {
            var usize j = 0; while (j < o.argument_count) {
                if (!charge(budget,1)) { return 2; }
                if (inspect && !read(initialized,m::edge_at(p,o.argument_start + j),base)) { return 1002; } j += 1;
            }
        }
        if (o.result != 0) { set(initialized,o.result - base - 1); } op = o.next;
    }
    if (inspect && (!read(initialized,b.value,base) || !read(initialized,b.left,base) || !read(initialized,b.right,base))) { return 1002; }
    if (b.kind == 5) {
        var usize j = 0; while (j < b.argument_count) {
            if (!charge(budget,1)) { return 2; }
            if (inspect && !read(initialized,m::edge_at(p,b.argument_start + j),base)) { return 1002; } j += 1;
        }
        if (b.destination != 0) { set(initialized,b.destination - base - 1); }
    }
    return 0;
}
fn bool incoming(ref m::Program p, usize id, m::Function f, usize width, ref bytes::Buffer reachable, ref bytes::Buffer states,
    ref v::Vec<usize> heads, ref v::Vec<Predecessor> predecessors, ref mut bytes::Buffer scratch, ref mut Budget budget) {
    if (!charge(budget,width)) { return false; }
    let ref mut u8[] data = bytes::view_mut(scratch); var usize j = 0;
    while (j < width) { data[j] = if (id == f.entry) { 0 } else { 255 }; j += 1; }
    if (id == f.entry) {
        j = 0; while (j < f.parameter_count) { let usize local = m::edge_at(p,f.parameter_start + j) - f.local_start - 1; data[local / 8] |= bit(local); j += 1; }
    } else {
        var usize pred = *v::at<usize>(heads,id - f.block_start - 1);
        while (pred != 0) {
            let Predecessor row = *v::at<Predecessor>(predecessors,pred - 1);
            if (bytes::view(reachable)[row.block] != 0) {
                if (!charge(budget,width)) { return false; }
                let usize offset = (row.block - f.block_start - 1) * width;
                j = 0; while (j < width) { data[j] &= bytes::view(states)[offset + j]; j += 1; }
            }
            pred = row.next;
        }
    }
    return true;
}

// Must-initialization is a greatest fixed point over reachable predecessor
// intersections. Entry is always seeded from parameters, including backedges.
// Uses are inspected only after convergence, so queue order cannot accept cycles
// that merely initialize each other or reject a valid delayed predecessor.
pub fn m::Problem function(ref m::Program p, usize fid, ref mut Budget budget) {
    let m::Function f = m::function_at(p,fid); let usize width = (f.local_count + 7) / 8;
    if (!charge(budget,f.block_count * width + f.block_count)) { return shape::problem(2,f.start,f.end); }
    var bytes::Buffer reachable = bytes::empty(); var bytes::Buffer pending = bytes::empty();
    var bytes::Buffer states = bytes::empty(); var bytes::Buffer scratch = bytes::empty();
    var v::Vec<usize> queue = v::empty<usize>(); var v::Vec<usize> heads = v::empty<usize>();
    var v::Vec<Predecessor> predecessors = v::empty<Predecessor>();
    if (!bytes::resize_zeroed(ref mut reachable,m::block_count(p) + 1).is_ok || !bytes::resize_zeroed(ref mut pending,m::block_count(p) + 1).is_ok
        || !bytes::resize_zeroed(ref mut states,f.block_count * width).is_ok || !bytes::resize_zeroed(ref mut scratch,width).is_ok) { return shape::problem(3,f.start,f.end); }
    var usize i = 0; while (i < f.block_count) { if (!v::push<usize>(ref mut heads,0).is_ok) { return shape::problem(3,f.start,f.end); } i += 1; }
    i = f.block_start + 1; while (i <= f.block_start + f.block_count) {
        let m::Block b = m::block_at(p,i); var usize edge = 0;
        while (edge < 2) {
            let usize to = if (edge == 0) { b.target } else { b.alternative };
            if (to != 0) {
                let usize at = to - f.block_start - 1; let usize previous = *v::at<usize>(ref heads,at);
                if (!v::push<Predecessor>(ref mut predecessors,Predecessor { .block = i, .next = previous }).is_ok) { return shape::problem(3,b.start,b.end); }
                let usize length = v::len<Predecessor>(ref predecessors);
                { let ref mut usize slot = v::at_mut<usize>(ref mut heads,at); *slot = length; }
            }
            edge += 1;
        }
        i += 1;
    }
    if (!enqueue(ref mut queue,ref mut pending,f.entry)) { return shape::problem(3,f.start,f.end); }
    while (v::len<usize>(ref queue) != 0) {
        let usize id = pop(ref mut queue,ref mut pending);
        if (bytes::view(ref reachable)[id] == 0) {
            let ref mut u8[] data = bytes::view_mut(ref mut reachable); data[id] = 1;
            if (!successors(p,id,ref mut queue,ref mut pending)) { return shape::problem(3,f.start,f.end); }
        }
    }
    // Start at top; only the entry boundary introduces absent initialization.
    { let ref mut u8[] data = bytes::view_mut(ref mut states); i = 0; while (i < data.len) { data[i] = 255; i += 1; } }
    i = f.block_start + 1; while (i <= f.block_start + f.block_count) {
        if (bytes::view(ref reachable)[i] != 0 && !enqueue(ref mut queue,ref mut pending,i)) { return shape::problem(3,f.start,f.end); } i += 1;
    }
    while (v::len<usize>(ref queue) != 0) {
        let usize id = pop(ref mut queue,ref mut pending); let m::Block b = m::block_at(p,id);
        if (!charge(budget,1) || !incoming(p,id,f,width,ref reachable,ref states,ref heads,ref predecessors,ref mut scratch,budget)) { return shape::problem(2,b.start,b.end); }
        let u32 code = transfer(p,b,f.local_start,ref mut scratch,false,budget);
        if (code != 0) { return shape::problem(code,b.start,b.end); }
        let usize offset = (id - f.block_start - 1) * width; var bool changed = false;
        { let ref mut u8[] data = bytes::view_mut(ref mut states); var usize j = 0;
          while (j < width) { let u8 value = bytes::view(ref scratch)[j]; changed = changed || data[offset + j] != value; data[offset + j] = value; j += 1; } }
        if (changed && !successors(p,id,ref mut queue,ref mut pending)) { return shape::problem(3,b.start,b.end); }
    }
    i = f.block_start + 1; while (i <= f.block_start + f.block_count) {
        if (bytes::view(ref reachable)[i] != 0) {
            let m::Block b = m::block_at(p,i);
            if (!incoming(p,i,f,width,ref reachable,ref states,ref heads,ref predecessors,ref mut scratch,budget)) { return shape::problem(2,b.start,b.end); }
            let u32 code = transfer(p,b,f.local_start,ref mut scratch,true,budget);
            if (code != 0) { return shape::problem(code,b.start,b.end); }
        }
        i += 1;
    }
    return shape::problem(0,0,0);
}
