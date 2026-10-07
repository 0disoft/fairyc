module hir_mutable_check;
import reference_types as refs;
import hir_model as h;

import array_types as arrays;
import std::vec as v;

// An immutable mutable descriptor keeps one exclusive generation. Aliases share
// it; slices, returned call reborrows and call arguments create child generations.
// The parent and its aliases resume only after all retained children expire.
struct Slot { usize generation; usize parent; usize start; usize end; usize receiver; bool fresh; bool argument; bool mutable; }
struct Context { usize work; h::Problem problem; }
fn usize fail(ref mut Context c, u32 code, usize start, usize end) {
    if (c.problem.code == 0) { c.problem = h::Problem { .code,.start,.end }; } return 0;
}
fn bool charge(ref mut Context c) { c.work += 1; return c.work <= 1048576; }
fn usize generation(ref h::Program p, ref mut Context c, ref mut v::Vec<Slot> slots, ref usize[] initial, ref usize[] summaries, usize id, usize depth) {
    if (id == 0 || c.problem.code != 0) { return 0; }
    let e = h::expression_at(p,id); let u32 kind = h::expression_code(e.kind);
    let bool mutable = h::type_code(e.type) == 11 || refs::mutable(h::type_code(e.type));
    if (!mutable && h::type_code(e.type) != 10 && !refs::valid(h::type_code(e.type))) { return 0; }
    if (kind == 15) { return 0; }
    if (depth >= 64 || !charge(c)) { return fail(c,2,e.start,e.end); }
    let Slot old = *v::at<Slot>(slots,id); if (old.generation != 0) { return old.generation; }
    var usize base = 0; var usize parent = 0; var usize receiver = old.receiver;
    var bool fresh = old.fresh; var usize start = e.start;
    if (kind == 16) { base = id; fresh = true; receiver = e.left;var usize root=e.left;var usize steps=0;while(steps<64 && (h::expression_code(h::expression_at(p,root).kind)==10 || h::expression_code(h::expression_at(p,root).kind)==13 || h::expression_code(h::expression_at(p,root).kind)==4 && h::operator_code(h::expression_at(p,root).operator)==5)){root=h::expression_at(p,root).left;steps+=1;}if(refs::valid(h::type_code(h::expression_at(p,root).type))){parent=generation(p,c,slots,initial,summaries,root,depth+1);} }
    else if(kind==25){parent=generation(p,c,slots,initial,summaries,e.left,depth+1);base=id;fresh=true;receiver=e.left;start=e.end;}
    else if (kind == 17) { parent = generation(p,c,slots,initial,summaries,e.left,depth+1); base = if (mutable) { id } else { parent }; fresh = mutable; receiver = e.left; start = e.end; }
    else if (kind == 3) {
        let b = h::binding_at(p,e.binding);
        base = if (h::binding_code(b.kind) == 2) { 4096 + e.binding } else { generation(p,c,slots,initial,summaries,initial[e.binding],depth+1) };
    } else if (kind == 7) { receiver = h::block_at(p,e.block).tail; base = generation(p,c,slots,initial,summaries,receiver,depth+1); }
    else if (kind == 8) {
        let usize yes = generation(p,c,slots,initial,summaries,h::block_at(p,e.block).tail,depth+1);
        let usize no = generation(p,c,slots,initial,summaries,e.alternative,depth+1);
        if (yes != no) { return fail(c,2103,e.start,e.end); } base = yes;
    } else if (kind == 6) {
        let usize returned = summaries[e.binding]; var usize i = 0;
        while (i < h::function_count(p)) {
            let f = h::function_at(p,i);
            if (f.binding == e.binding) { var usize j = 0; while (j < f.parameter_count) {
                if (h::edge_at(p,f.parameter_start+j) == returned) { receiver = h::edge_at(p,e.edge_start+j); parent = generation(p,c,slots,initial,summaries,receiver,depth+1); }
                j += 1;
            } } i += 1;
        }
        if (parent == 0 && mutable) { return fail(c,2103,e.start,e.end); } base = id; fresh = true; start = e.end;
    } else { return fail(c,2103,e.start,e.end); }
    if (old.argument && base != id) { parent = base; base = id; receiver = id; start = e.start; }
    if (base == 0) { return if (mutable) { fail(c,2103,e.start,e.end) } else { 0 }; }
    let ref mut Slot stored = v::at_mut<Slot>(slots,id);
    stored.generation = base; stored.parent = parent; stored.receiver = receiver;
    stored.fresh = fresh; stored.mutable = mutable; stored.start = start;
    if (stored.end < e.end) { stored.end = e.end; }
    return base;
}
fn bool ancestor(ref v::Vec<Slot> slots, usize parent, usize child) {
    var usize current = child; var usize depth = 0;
    while (current != 0 && depth < 64) { if (current == parent) { return true; } current = v::at<Slot>(slots,current).parent; depth += 1; }
    return false;
}
fn usize held_end(ref h::Program p, usize id) {
    let e = h::expression_at(p,id); var usize end = e.end; var usize i = 1;
    while (i <= h::expression_count(p)) {
        let x = h::expression_at(p,i); let u32 kind = h::expression_code(x.kind);
        if ((kind == 13 || kind == 17 || kind==25 || kind==4 || kind==10 || kind==14) && x.left == id && x.end > end) { end = x.end; }
        if (kind == 6) { var usize j = 0; while (j < x.edge_count) { if (h::edge_at(p,x.edge_start+j) == id && x.end > end) { end = x.end; } j += 1; } }
        i += 1;
    }
    i = 1; while (i <= h::statement_count(p)) {
        let s = h::statement_at(p,i); if (h::statement_code(s.kind) == 3) {
            let target = h::expression_at(p,s.target);
            if (e.start >= target.start && e.end <= target.end && s.end > end) { end = s.end; }
        } i += 1;
    } return end;
}
fn bool competing(ref h::Program p,usize id){let e=h::expression_at(p,id);var usize i=1;while(i<=h::statement_count(p)){let s=h::statement_at(p,i);if(h::statement_code(s.kind)==3){let t=h::expression_at(p,s.target);if(e.start>=t.start && e.end<=t.end){return true;}}i+=1;}i=1;while(i<=h::expression_count(p)){let x=h::expression_at(p,i);if(h::expression_code(x.kind)==6){var usize j=0;while(j<x.edge_count){let a=h::expression_at(p,h::edge_at(p,x.edge_start+j));if((h::type_code(a.type)==11 || refs::mutable(h::type_code(a.type))) && e.start>=a.start && e.end<=a.end){return true;}j+=1;}}i+=1;}return false;}
pub fn h::Problem check(ref h::Program p, ref usize[] initial, ref usize[] last, ref usize[] summaries, ref v::Vec<usize> origins) {
    var bool needed = false; var usize i = 1;
    while (i <= h::expression_count(p)) { needed = needed || (h::type_code(h::expression_at(p,i).type) == 11 || refs::mutable(h::type_code(h::expression_at(p,i).type))); i += 1; }
    if (!needed) { return h::Problem { .code=0,.start=0,.end=0 }; }
    var v::Vec<Slot> slots = v::empty<Slot>();
    var Context c = Context {.work=0,.problem=h::Problem { .code=0,.start=0,.end=0 } };
    i = 0; while (i <= 4096 + h::binding_count(p)) {
        if (!v::push<Slot>(ref mut slots,Slot { .generation=0,.parent=0,.start=0,.end=0,.receiver=0,.fresh=false,.argument=false,.mutable=false }).is_ok) { return h::Problem { .code=3,.start=0,.end=0 }; } i += 1;
    }
    // Exact mutable parameters create call-bounded reborrows even for void calls.
    i = 1; while (i <= h::expression_count(p)) {
        let e = h::expression_at(p,i); if (h::expression_code(e.kind) == 6) { var usize j = 0; while (j < e.edge_count) {
            let usize id = h::edge_at(p,e.edge_start+j); let arg = h::expression_at(p,id);
            if ((h::type_code(arg.type) == 11 || refs::mutable(h::type_code(arg.type)))) { let ref mut Slot s = v::at_mut<Slot>(ref mut slots,id); s.fresh=true; s.argument=true; s.end=e.end; }
            j += 1;
        } } i += 1;
    }
    i = 1; while (i <= h::expression_count(p)) {
        let e = h::expression_at(p,i); let u32 type = h::type_code(e.type);
if (type == 10 || type == 11 || refs::valid(type)) { discard generation(p,ref mut c,ref mut slots,initial,summaries,i,0); }
        if (c.problem.code != 0) { return c.problem; } i += 1;
    }
    // Every alias and direct temporary consumer extends its actual generation.
    i = 1; while (i <= h::expression_count(p)) {
        let s = *v::at<Slot>(ref slots,i);
        if (s.generation != 0) { let usize end = held_end(p,i); let ref mut Slot g = v::at_mut<Slot>(ref mut slots,s.generation); if (end > g.end) { g.end=end; } }
        i += 1;
    }
    i = 1; while (i <= h::binding_count(p)) {
        if (initial[i] != 0 && (h::type_code(h::binding_at(p,i).type) == 10 || h::type_code(h::binding_at(p,i).type) == 11 || refs::valid(h::type_code(h::binding_at(p,i).type)))) {
            let usize gen = generation(p,ref mut c,ref mut slots,initial,summaries,initial[i],0);
            if (gen != 0) { let ref mut Slot g = v::at_mut<Slot>(ref mut slots,gen); if (last[i] > g.end) { g.end=last[i]; } }
        } i += 1;
    }
    // Returned descendants keep every ancestor's whole-owner hold alive.
    var bool changed = true;
    while (changed) { changed=false; i=1; while (i <= h::expression_count(p)) {
        if (!charge(ref mut c)) { return h::Problem { .code=2,.start=0,.end=0 }; }
        let s = *v::at<Slot>(ref slots,i);
        if (s.fresh && s.parent != 0) { let ref mut Slot parent = v::at_mut<Slot>(ref mut slots,s.parent); if (s.end > parent.end) { parent.end=s.end; changed=true; } }
        i += 1;
    } }
    i = 1; while (i <= h::expression_count(p)) {
        let loan = *v::at<Slot>(ref slots,i); let usize owner = *v::at<usize>(origins,i);
        if (loan.fresh && owner != 0) {
            var usize j = 1; while (j <= h::expression_count(p)) {
                if (!charge(ref mut c)) { return h::Problem { .code=2,.start=0,.end=0 }; }
                let other = *v::at<Slot>(ref slots,j);
                if (j > i && other.fresh && (loan.mutable || other.mutable) && *v::at<usize>(origins,j) == owner && loan.start < other.end && other.start < loan.end && !ancestor(ref slots,i,j) && !ancestor(ref slots,j,i)) { let e=h::expression_at(p,j); return h::Problem { .code=2101,.start=e.start,.end=e.end }; }
                let e = h::expression_at(p,j);
                if (h::expression_code(e.kind) == 3 && loan.start < e.end && e.start < loan.end) {
                    // The occurrence creating this loan is allowed; bounds and
                    // later arguments are separate occurrences and stay checked.
                    let receiver = h::expression_at(p,loan.receiver);
                    let bool creating = e.start >= receiver.start && e.end <= receiver.end;
                    if (!creating) {
                        let u32 type = h::type_code(e.type);
                        if (e.binding == owner && loan.mutable || (type == 11 || refs::mutable(type)) && *v::at<usize>(origins,j) == owner && !ancestor(ref slots,i,other.generation) && (loan.mutable || competing(p,j))) { return h::Problem { .code=2101,.start=e.start,.end=e.end }; }
                    }
                }
                j += 1;
            }
        } i += 1;
    }
    return c.problem;
}
