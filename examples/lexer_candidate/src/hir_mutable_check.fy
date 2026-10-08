module hir_mutable_check;
import hir_borrow_paths as paths;
import borrow_shapes as borrows;
import tagged_types as tagged;
import reference_types as refs;
import hir_model as h;
import borrow_origins as owners;

import array_types as arrays;
import std::vec as v;

// An immutable mutable descriptor keeps one exclusive generation. Aliases share
// it; slices, returned call reborrows and call arguments create child generations.
// The parent and its aliases resume only after all retained children expire.
struct Slot { usize generation; usize parent; usize alternate; usize start; usize end; usize receiver; bool fresh; bool argument; bool mutable; }
struct Context { usize work; h::Problem problem; }
fn usize fail(ref mut Context c, u32 code, usize start, usize end) {
    if (c.problem.code == 0) { c.problem = h::Problem { .code,.start,.end }; } return 0;
}
fn bool charge(ref mut Context c) { c.work += 1; return c.work <= 1048576; }
// Branch joins retain all possible incoming generations in a bounded DAG.
fn usize merge(ref mut Context c, ref mut v::Vec<Slot> slots, usize a, usize b, usize start, usize end, bool mutable) {
    if (a == b || b == 0) { return a; }
    if (a == 0) { return b; }
    if (!charge(c) || v::len<Slot>(slots) >= 16384) { return fail(c,2,start,end); }
    let usize id = v::len<Slot>(slots);
    if (!v::push<Slot>(slots,Slot { .generation=id,.parent=a,.alternate=b,.start,.end,.receiver=0,.fresh=true,.argument=false,.mutable }).is_ok) { return fail(c,3,start,end); }
    return id;
}
fn usize generation(ref h::Program p, ref mut Context c, ref mut v::Vec<Slot> slots, ref usize[] initial, ref usize[] summaries, usize id, usize depth) {
    if (id == 0 || c.problem.code != 0) { return 0; }
    let e = h::expression_at(p,id); let u32 kind = h::expression_code(e.kind);
    let bool mutable = borrows::mutable(h::type_code(e.type));
    if (borrows::count(h::type_code(e.type))==0) { return 0; }
    if (kind == 15 || kind==21) { return 0; }
    if(kind==18 && e.edge_count==0){return 0;}
    if (depth >= 64 || !charge(c)) { return fail(c,2,e.start,e.end); }
    let Slot old = *v::at<Slot>(slots,id); if (old.generation != 0) { return old.generation; }
    if(!paths::reachable(p,e.start,e.end,ref mut c.work)){return 0;}
    var usize base = 0; var usize parent = 0; var usize alternate = 0; var usize receiver = old.receiver;
    var bool fresh = old.fresh; var usize start = e.start;
    if(kind==18){receiver=h::edge_at(p,e.edge_start);base=generation(p,c,slots,initial,summaries,receiver,depth+1);if(base==0){return 0;}}
    else if(kind==22 || kind==11){receiver=e.left;base=generation(p,c,slots,initial,summaries,receiver,depth+1);}
    else if (kind == 16) { base = id; fresh = true; receiver = e.left;var usize root=e.left;var usize steps=0;while(steps<64 && (h::expression_code(h::expression_at(p,root).kind)==10 || h::expression_code(h::expression_at(p,root).kind)==13 || h::expression_code(h::expression_at(p,root).kind)==4 && h::operator_code(h::expression_at(p,root).operator)==5)){root=h::expression_at(p,root).left;steps+=1;}if(refs::valid(h::type_code(h::expression_at(p,root).type))){parent=generation(p,c,slots,initial,summaries,root,depth+1);} }
    else if(kind==25 || kind==29 || kind==30){parent=generation(p,c,slots,initial,summaries,e.left,depth+1);base=id;fresh=true;receiver=e.left;start=e.end;}
    else if (kind == 17) { parent = generation(p,c,slots,initial,summaries,e.left,depth+1); base = if (mutable) { id } else { parent }; fresh = mutable; receiver = e.left; start = e.end; }
    else if (kind == 3) {
        let b = h::binding_at(p,e.binding);
        base = if (h::binding_code(b.kind) == 2) { h::expression_count(p) + e.binding } else { generation(p,c,slots,initial,summaries,initial[e.binding],depth+1) };
    } else if ((kind == 7 || kind==20)) { receiver = h::block_at(p,e.block).tail; base = generation(p,c,slots,initial,summaries,receiver,depth+1); }
    else if (kind == 8) {
        let usize yes = generation(p,c,slots,initial,summaries,h::block_at(p,e.block).tail,depth+1);
        let usize no = generation(p,c,slots,initial,summaries,e.alternative,depth+1);
        if (yes == no || yes == 0 || no == 0) { base = if (yes != 0) { yes } else { no }; }
        else { base=id; parent=yes; alternate=no; fresh=true; receiver=0; start=e.end; }
    } else if (kind == 19) {
        var usize i = 0;
        while (i < e.edge_count) {
            let arm=h::expression_at(p,h::edge_at(p,e.edge_start+i));
            let usize tail=h::block_at(p,arm.block).tail;
            if (tail != 0) {
                let usize value=generation(p,c,slots,initial,summaries,tail,depth+1);
                base=merge(c,slots,base,value,e.start,e.end,mutable);
            }
            i+=1;
        }
        if (base != 0 && base != id) { parent=base; base=id; fresh=true; receiver=0; start=e.end; }
    } else if (kind == 6) {
        let usize returned = summaries[e.binding]; var usize i = 0;
        while (i < h::function_count(p)) {
            let f = h::function_at(p,i);
            if (f.binding == e.binding) { var usize j = 0; while (j < f.parameter_count) {
                if (h::edge_at(p,f.parameter_start+j) == returned) { receiver = h::edge_at(p,e.edge_start+j); parent = generation(p,c,slots,initial,summaries,receiver,depth+1); }
                j += 1;
            } } i += 1;
        }
        if(parent==0 && returned==0 && borrows::count(h::type_code(e.type))!=0 && !refs::valid(h::type_code(e.type)) && !borrows::mutable(h::type_code(e.type))){return 0;}
        if (parent == 0 && mutable) { return fail(c,2103,e.start,e.end); } base = id; fresh = true; start = e.end;
    } else { return fail(c,2103,e.start,e.end); }
    if (old.argument && base != id) { parent = base; base = id; receiver = id; start = e.start; }
    if(base==0 && kind==3 && initial[e.binding]!=0 && tagged::builtin(h::type_code(h::expression_at(p,initial[e.binding]).type))){return 0;}
    if (base == 0 && tagged::builtin(h::type_code(e.type))) { return 0; }
    if (base == 0) { return if (mutable) { fail(c,2103,e.start,e.end) } else { 0 }; }
    let ref mut Slot stored = v::at_mut<Slot>(slots,id);
    stored.generation = base; stored.parent = parent; stored.alternate = alternate; stored.receiver = receiver;
    stored.fresh = fresh; stored.mutable = mutable; stored.start = start;
    if (stored.end < e.end) { stored.end = e.end; }
    return base;
}
fn bool ancestor(ref v::Vec<Slot> slots, usize parent, usize child, usize depth, ref mut usize work) {
    if (child == 0) { return false; }
    if (parent == child) { return true; };
    *work = *work + 1;
    if (depth >= 64 || *work > 1048576) { return false; }
    let s = *v::at<Slot>(slots,child);
    return ancestor(slots,parent,s.parent,depth+1,work) || ancestor(slots,parent,s.alternate,depth+1,work);
}

fn usize held_end(ref h::Program p, usize id) {
    let e = h::expression_at(p,id); var usize end = e.end; var usize i = 1;
    while (i <= h::expression_count(p)) {
        let x = h::expression_at(p,i); let u32 kind = h::expression_code(x.kind);
        if ((kind == 13 || kind == 17 || kind==25 || kind==28 || kind==29 || kind==30 || kind==4 || kind==10 || kind==14) && x.left == id && x.end > end) { end = x.end; }
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
fn bool competing(ref h::Program p,usize id){let e=h::expression_at(p,id);var usize i=1;while(i<=h::statement_count(p)){let s=h::statement_at(p,i);if(h::statement_code(s.kind)==3){let t=h::expression_at(p,s.target);if(e.start>=t.start && e.end<=t.end){return true;}}i+=1;}i=1;while(i<=h::expression_count(p)){let x=h::expression_at(p,i);if(h::expression_code(x.kind)==6){var usize j=0;while(j<x.edge_count){let a=h::expression_at(p,h::edge_at(p,x.edge_start+j));if((borrows::mutable(h::type_code(a.type))) && e.start>=a.start && e.end<=a.end){return true;}j+=1;}}i+=1;}return false;}
pub fn h::Problem check(ref h::Program p, ref usize[] initial, ref usize[] last, ref usize[] summaries, ref v::Vec<usize> origins,ref v::Vec<owners::Entry> sets,ref v::Vec<usize> members) {
    var bool needed = false; var usize i = 1;
    while (i <= h::expression_count(p)) { needed = needed || borrows::mutable(h::type_code(h::expression_at(p,i).type)); i += 1; }
    if (!needed) { return h::Problem { .code=0,.start=0,.end=0 }; }
    var v::Vec<Slot> slots = v::empty<Slot>();
    var Context c = Context {.work=0,.problem=h::Problem { .code=0,.start=0,.end=0 } };
    i = 0; while (i <= h::expression_count(p) + h::binding_count(p)) {
        if (!v::push<Slot>(ref mut slots,Slot { .generation=0,.parent=0,.alternate=0,.start=0,.end=0,.receiver=0,.fresh=false,.argument=false,.mutable=false }).is_ok) { return h::Problem { .code=3,.start=0,.end=0 }; } i += 1;
    }
    // Exact mutable parameters create call-bounded reborrows even for void calls.
    i = 1; while (i <= h::expression_count(p)) {
        let e = h::expression_at(p,i); if (h::expression_code(e.kind) == 6) { var usize j = 0; while (j < e.edge_count) {
            let usize id = h::edge_at(p,e.edge_start+j); let arg = h::expression_at(p,id);
            if ((borrows::mutable(h::type_code(arg.type)))) { let ref mut Slot s = v::at_mut<Slot>(ref mut slots,id); s.fresh=true; s.argument=true; s.end=e.end; }
            j += 1;
        } } i += 1;
    }
    i = 1; while (i <= h::expression_count(p)) {
        let e = h::expression_at(p,i); let u32 type = h::type_code(e.type);
if (borrows::count(type)!=0) { discard generation(p,ref mut c,ref mut slots,initial,summaries,i,0); }
        if (c.problem.code != 0) { return c.problem; } i += 1;
    }
    // Every alias and direct temporary consumer extends its actual generation.
    i = 1; while (i <= h::expression_count(p)) {
        let s = *v::at<Slot>(ref slots,i);
        if (s.generation != 0) { let usize end = held_end(p,i); let ref mut Slot g = v::at_mut<Slot>(ref mut slots,s.generation); if (end > g.end) { g.end=end; } }
        i += 1;
    }
    i = 1; while (i <= h::binding_count(p)) {
        if (initial[i] != 0 && (borrows::count(h::type_code(h::binding_at(p,i).type))!=0)) {
            let usize gen = generation(p,ref mut c,ref mut slots,initial,summaries,initial[i],0);
            if (gen != 0) { let ref mut Slot g = v::at_mut<Slot>(ref mut slots,gen); if (last[i] > g.end) { g.end=last[i]; } }
        } i += 1;
    }
    // Returned descendants keep every ancestor's whole-owner hold alive.
    var bool changed = true;
    while (changed) { changed=false; i=1; while (i < v::len<Slot>(ref slots)) {
        if (!charge(ref mut c)) { return h::Problem { .code=2,.start=0,.end=0 }; }
        let s = *v::at<Slot>(ref slots,i);
        if (s.fresh && s.parent != 0) { let ref mut Slot parent = v::at_mut<Slot>(ref mut slots,s.parent); if (s.end > parent.end) { parent.end=s.end; changed=true; } }
        if (s.fresh && s.alternate != 0) { let ref mut Slot parent = v::at_mut<Slot>(ref mut slots,s.alternate); if (s.end > parent.end) { parent.end=s.end; changed=true; } }
        i += 1;
    } }
    i = 1; while (i <= h::expression_count(p)) {
        let loan = *v::at<Slot>(ref slots,i); let usize owner = *v::at<usize>(origins,i);
        if (loan.fresh && owner != 0) {
            var usize j = 1; while (j <= h::expression_count(p)) {
                if (!charge(ref mut c)) { return h::Problem { .code=2,.start=0,.end=0 }; }
                let other = *v::at<Slot>(ref slots,j);
                let e = h::expression_at(p,j);
                if (j > i && other.fresh && (loan.mutable || other.mutable) && owners::overlap(sets,members,*v::at<usize>(origins,j),owner,ref mut c.work) && loan.start < other.end && other.start < loan.end && !ancestor(ref slots,i,j,0,ref mut c.work) && !ancestor(ref slots,j,i,0,ref mut c.work) && !paths::separate(p,loan.start,e.start,e.end,ref mut c.work) && !paths::separate(p,loan.end-1,e.start,e.end,ref mut c.work)) { let e=h::expression_at(p,j); return h::Problem { .code=2101,.start=e.start,.end=e.end }; }
                if (h::expression_code(e.kind) == 3 && loan.start < e.end && e.start < loan.end) {
                    // The occurrence creating this loan is allowed; bounds and
                    // later arguments are separate occurrences and stay checked.
                    let bool creating = if (loan.receiver != 0) { let receiver = h::expression_at(p,loan.receiver); e.start >= receiver.start && e.end <= receiver.end } else { false };
                    let u32 type = h::type_code(e.type);
                    let bool access = owners::contains(sets,members,owner,e.binding,ref mut c.work) && loan.mutable || borrows::mutable(type) && owners::overlap(sets,members,*v::at<usize>(origins,j),owner,ref mut c.work) && !ancestor(ref slots,i,other.generation,0,ref mut c.work) && (loan.mutable || competing(p,j));
                    if (!creating && access && paths::reachable(p,e.start,e.end,ref mut c.work) && !paths::separate(p,loan.start,e.start,e.end,ref mut c.work) && !paths::separate(p,loan.end-1,e.start,e.end,ref mut c.work)) {
                        return h::Problem { .code=2101,.start=e.start,.end=e.end };
                    }
                }
                j += 1;
            }
        } i += 1;
    }
    return if(c.work>1048576){h::Problem{.code=2,.start=0,.end=0}}else{c.problem};
}
