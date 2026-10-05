module mir_state;
import mir_model as m;
import std::vec as v;

pub struct Context { m::Problem problem; usize function; usize current; usize[1153] locals; usize[1153] targets; }
pub fn Context empty() { return Context { .problem = m::Problem { .code = 0, .start = 0, .end = 0 }, .function = 0, .current = 0, .locals = [0;1153], .targets = [0;1153] }; }
pub fn usize fail(ref mut Context c, u32 code, usize start, usize end) { if (c.problem.code == 0) { c.problem = m::Problem { .code, .start, .end }; } return 0; }
pub fn bool edge(ref mut v::Vec<usize> edges, usize id, ref mut Context c) {
    if (c.problem.code != 0) { return false; }
    if (id == 0 || v::len<usize>(edges) >= 65536) { discard fail(c,if (id == 0) { 4 } else { 2 },0,0); return false; }
    if (!v::push<usize>(edges,id).is_ok) { discard fail(c,3,0,0); return false; } return true;
}
pub fn usize local(ref mut v::Vec<m::Local> locals, ref mut Context c, u32 type, usize binding, bool parameter, bool mutable, usize start, usize end) {
    if (c.problem.code != 0) { return 0; }
    if (type < 1 || type > 6 || c.function == 0 || start >= end) { return fail(c,4,start,end); }
    if (v::len<m::Local>(locals) >= 16384) { return fail(c,2,start,end); }
    if (!v::push<m::Local>(locals,m::Local { .function = c.function, .binding, .type, .parameter, .mutable, .start, .end }).is_ok) { return fail(c,3,start,end); }
    return v::len<m::Local>(locals);
}
pub fn m::Block block(usize function, usize start, usize end) { return m::Block { .function, .first = 0, .last = 0, .kind = 0, .value = 0, .left = 0, .right = 0, .operator = 0, .type = 0, .target = 0, .alternative = 0, .check = 0, .callee = 0, .destination = 0, .argument_start = 0, .argument_count = 0, .trap_mask = 0, .start, .end }; }
pub fn usize new_block(ref mut v::Vec<m::Block> blocks, ref mut Context c, usize start, usize end) {
    if (c.problem.code != 0) { return 0; }
    if (v::len<m::Block>(blocks) >= 16384) { return fail(c,2,start,end); }
    if (!v::push<m::Block>(blocks,block(c.function,start,end)).is_ok) { return fail(c,3,start,end); } return v::len<m::Block>(blocks);
}
pub fn m::Operation operation(u32 kind, usize result, u32 type, u32 owner_type, u32 operator, usize left, usize right, usize start, usize end) {
    return m::Operation { .function = 0, .block = 0, .kind, .result, .type, .owner_type, .operator, .left, .right, .next = 0, .payload_start = 0, .payload_end = 0, .boolean = false, .start, .end };
}
pub fn usize emit(ref mut v::Vec<m::Operation> operations, ref mut v::Vec<m::Block> blocks, ref mut Context c, m::Operation value) {
    if (c.problem.code != 0) { return 0; }
    if (c.current == 0 || c.current > v::len<m::Block>(blocks) || v::at<m::Block>(blocks,c.current - 1).kind != 0) { return fail(c,4,value.start,value.end); }
    if (v::len<m::Operation>(operations) >= 32768) { return fail(c,2,value.start,value.end); }
    let usize previous = v::at<m::Block>(blocks,c.current - 1).last;
    if (!v::push<m::Operation>(operations,m::Operation { .function = c.function, .block = c.current, .kind = value.kind, .result = value.result, .type = value.type, .owner_type = value.owner_type, .operator = value.operator, .left = value.left, .right = value.right, .next = 0, .payload_start = value.payload_start, .payload_end = value.payload_end, .boolean = value.boolean, .start = value.start, .end = value.end }).is_ok) { return fail(c,3,value.start,value.end); }
    let usize id = v::len<m::Operation>(operations);
    if (previous != 0) { let ref mut m::Operation p = v::at_mut<m::Operation>(operations,previous - 1); p.next = id; }
    let ref mut m::Block b = v::at_mut<m::Block>(blocks,c.current - 1);
    if (b.first == 0) { b.first = id; } b.last = id; return id;
}
pub fn bool terminate(ref mut v::Vec<m::Block> blocks, ref mut Context c, m::Block value) {
    if (c.problem.code != 0) { return false; }
    if (c.current == 0 || c.current > v::len<m::Block>(blocks) || value.kind == 0 || v::at<m::Block>(blocks,c.current - 1).kind != 0) { discard fail(c,4,value.start,value.end); return false; }
    let ref mut m::Block b = v::at_mut<m::Block>(blocks,c.current - 1);
    b.kind = value.kind; b.value = value.value; b.left = value.left; b.right = value.right;
    b.operator = value.operator; b.type = value.type; b.target = value.target; b.alternative = value.alternative;
    b.check = value.check; b.callee = value.callee; b.destination = value.destination;
    b.argument_start = value.argument_start; b.argument_count = value.argument_count;
    b.trap_mask = value.trap_mask; b.start = value.start; b.end = value.end;
    c.current = 0; return true;
}
pub fn bool jump(ref mut v::Vec<m::Block> blocks, ref mut Context c, usize target, usize start, usize end) {
    var m::Block term = block(c.function,start,end); term.kind = 2; term.target = target;
    return terminate(blocks,c,term);
}
pub fn bool branch(ref mut v::Vec<m::Block> blocks, ref mut Context c, usize condition, usize consequent, usize alternative, usize start, usize end) {
    var m::Block term = block(c.function,start,end); term.kind = 3; term.value = condition; term.type = 6;
    term.target = consequent; term.alternative = alternative; return terminate(blocks,c,term);
}
// A checked operator has a marker, a normal continuation and a non-unwinding
// trap block. The trap mask preserves every type-specific failure category.
pub fn bool guard(ref mut v::Vec<m::Operation> operations, ref mut v::Vec<m::Block> blocks, ref mut Context c, u32 type, u32 operator, usize left, usize right, u32 mask, usize start, usize end) {
    let usize check = emit(operations,blocks,c,operation(5,0,type,type,operator,left,right,start,end));
    if (check == 0) { return false; }
    let usize normal = new_block(blocks,c,start,end); let usize trap = new_block(blocks,c,start,end);
    if (c.problem.code != 0) { return false; }
    var m::Block branch = block(c.function,start,end);
    branch.kind = 4; branch.left = left; branch.right = right; branch.operator = operator; branch.type = type;
    branch.check = check; branch.target = normal; branch.alternative = trap; branch.trap_mask = mask;
    if (!terminate(blocks,c,branch)) { return false; }
    c.current = trap; var m::Block failed = block(c.function,start,end); failed.kind = 6; failed.trap_mask = mask;
    if (!terminate(blocks,c,failed)) { return false; } c.current = normal; return true;
}
