module mir_nominal;
import hir_model as h;
import mir_model as m;
import mir_state as state;
import type_state as ids;
import std::vec as v;

// Assignment places are static paths rooted in mutable source locals. Ordinary
// field expressions read frozen values; a store rereads current owner values
// after RHS evaluation, retaining every sibling field that RHS may have changed.
pub fn usize root(ref h::Program p, usize id) {
    var usize current = id; var usize depth = 0;
    while (h::expression_code(h::expression_at(p,current).kind) == 10) {
        if (depth >= 64) { return 0; } current = h::expression_at(p,current).left; depth += 1;
    }
    let h::Expression e = h::expression_at(p,current);
    return if (h::expression_code(e.kind) == 3 && h::binding_code(h::binding_at(p,e.binding).kind) == 4) { e.binding } else { 0 };
}
pub fn usize read(ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut state::Context c, usize owner, u32 type, usize field, usize start, usize end) {
    let usize result = state::local(locals,c,type,0,false,false,start,end);
    let u32 owner_type = v::at<m::Local>(locals,owner - 1).type;
    if (state::emit(operations,blocks,c,state::operation(7,result,type,owner_type,ids::small_id(field),owner,0,start,end)) == 0) { return 0; } return result;
}
fn usize update(ref h::Program input, ref v::Vec<usize> path, usize at, usize owner, usize value, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut state::Context c, usize start, usize end) {
    let usize expression = *v::at<usize>(path,at); let h::Expression field = h::expression_at(input,expression);
    var usize replacement = value;
    if (at != 0) {
        let usize child = read(locals,blocks,operations,c,owner,h::type_code(field.type),field.binding,start,end);
        if (child == 0) { return 0; }
        replacement = update(input,path,at - 1,child,value,locals,blocks,operations,c,start,end);
        if (replacement == 0) { return 0; }
    }
    let u32 type = v::at<m::Local>(locals,owner - 1).type; let usize result = state::local(locals,c,type,0,false,false,start,end);
    if (state::emit(operations,blocks,c,state::operation(8,result,type,type,ids::small_id(field.binding),owner,replacement,start,end)) == 0) { return 0; } return result;
}
pub fn usize store(ref h::Program input, usize target, usize owner, usize value, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut state::Context c, usize start, usize end) {
    var v::Vec<usize> path = v::empty<usize>(); var usize current = target;
    while (h::expression_code(h::expression_at(input,current).kind) == 10) {
        if (v::len<usize>(ref path) >= 64) { return state::fail(c,2,start,end); }
        if (!v::push<usize>(ref mut path,current).is_ok) { return state::fail(c,3,start,end); } current = h::expression_at(input,current).left;
    }
    if (v::len<usize>(ref path) == 0) { return value; }
    return update(input,ref path,v::len<usize>(ref path) - 1,owner,value,locals,blocks,operations,c,start,end);
}
