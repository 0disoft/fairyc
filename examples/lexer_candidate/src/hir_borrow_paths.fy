module hir_borrow_paths;
import hir_model as h;

// Outcome bits: 1=fallthrough, 2=return/break, 4=continue (backedge).
// Unknown/depth/budget exhaustion must not prove an exit or separate a loan.
fn bool charge(ref mut usize work) { *work = *work + 1; return *work <= 1048576; }
fn bool inside(usize start, usize end, usize a, usize b) { return start >= a && end <= b; }
fn usize expression_outcomes(ref h::Program p, usize id, usize depth, ref mut usize work) {
    if (id == 0) { return 1; }
    if (depth >= 32 || !charge(work)) { return 7; }
    let e = h::expression_at(p,id); let u32 kind = h::expression_code(e.kind);
    if (kind == 7 || kind == 20) { return outcomes(p,e.block,depth+1,work); }
    if (kind == 8) {
        return outcomes(p,e.block,depth+1,work) | expression_outcomes(p,e.alternative,depth+1,work);
    }
    if (kind == 19) {
        var usize result = 0; var usize i = 0;
        while (i < e.edge_count) {
            result = result | outcomes(p,h::expression_at(p,h::edge_at(p,e.edge_start+i)).block,depth+1,work);
            i += 1;
        }
        return if (e.edge_count == 0) { 1 } else { result };
    }
    return 1;
}
fn usize statement_outcomes(ref h::Program p, usize id, usize depth, ref mut usize work) {
    if (depth >= 32 || !charge(work)) { return 7; }
    let s = h::statement_at(p,id); let u32 kind = h::statement_code(s.kind);
    if (kind == 2 || kind == 7) { return 2; }
    if (kind == 8) { return 4; }
    if (kind == 4) { return expression_outcomes(p,s.expression,depth+1,work); }
    // Nested loops may run zero times. They cannot prove a definite exit here.
    return 1;
}
fn usize outcomes(ref h::Program p, usize id, usize depth, ref mut usize work) {
    if (id == 0) { return 1; }
    if (depth >= 32 || !charge(work)) { return 7; }
    let b = h::block_at(p,id); var usize result = 1; var usize i = 0;
    while (i < b.edge_count && (result & 1) != 0) {
        result = (result & 6) | statement_outcomes(p,h::edge_at(p,b.edge_start+i),depth+1,work);
        i += 1;
    }
    if ((result & 1) != 0) { result = (result & 6) | expression_outcomes(p,b.tail,depth+1,work); }
    return result;
}
pub fn bool reenters(ref h::Program p, h::Statement s, ref mut usize work) {
    return (outcomes(p,s.body,0,work) & 5) != 0;
}
pub fn bool reachable(ref h::Program p, usize start, usize end, ref mut usize work) {
    var usize i = 1;
    while (i <= h::block_count(p)) {
        if (!charge(work)) { return false; }
        let b = h::block_at(p,i);
        if (inside(start,end,b.start,b.end)) {
            var usize j = 0;
            while (j < b.edge_count) {
                let usize id = h::edge_at(p,b.edge_start+j); let s = h::statement_at(p,id);
                if (s.end <= start && (statement_outcomes(p,id,0,work) & 1) == 0) { return false; }
                j += 1;
            }
        }
        i += 1;
    }
    return true;
}
// Mutually exclusive arms and paths that leave before an access cannot retain
// a loan on that access path. A use after the join still retains the owner.
pub fn bool separate(ref h::Program p, usize point, usize start, usize end, ref mut usize work) {
    var usize i = 1;
    while (i <= h::expression_count(p)) {
        if (!charge(work)) { return false; }
        let e = h::expression_at(p,i); let u32 kind = h::expression_code(e.kind);
        if (kind == 8) {
            let b = h::block_at(p,e.block); let other = h::expression_at(p,e.alternative);
            if (inside(point,point+1,b.start,b.end) && inside(start,end,other.start,other.end) ||
                inside(start,end,b.start,b.end) && inside(point,point+1,other.start,other.end)) { return true; }
        }
        if (kind == 19) {
            var usize a = 0; var usize selected = 0; var usize accessed = 0;
            while (a < e.edge_count) {
                let b = h::block_at(p,h::expression_at(p,h::edge_at(p,e.edge_start+a)).block);
                if (inside(point,point+1,b.start,b.end)) { selected = a+1; }
                if (inside(start,end,b.start,b.end)) { accessed = a+1; }
                a += 1;
            }
            if (selected != 0 && accessed != 0 && selected != accessed) { return true; }
        }
        i += 1;
    }
    i = 1;
    while (i <= h::block_count(p)) {
        if (!charge(work)) { return false; }
        let b = h::block_at(p,i);
        if (inside(point,point+1,b.start,b.end) && b.end <= start && (outcomes(p,i,0,work) & 1) == 0) { return true; }
        i += 1;
    }
    return false;
}
