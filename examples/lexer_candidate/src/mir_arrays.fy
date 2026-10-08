module mir_arrays;
import mir_model as m;
import mir_state as state;
import std::vec as v;

// A checked index is an immutable temporary tied to one exact array type.
// Its bounds trap occurs here, before any RHS evaluation, load or store.
pub fn usize index(ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut state::Context c, u32 owner, usize raw, usize start, usize end) {
    let usize result = state::local(locals,c,5,0,false,false,start,end);
    if (state::emit(operations,blocks,c,state::operation(12,result,5,owner,0,raw,0,start,end)) == 0) { return 0; } return result;
}
pub fn usize read(ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut state::Context c, u32 owner, u32 element, usize array, usize index, usize start, usize end) {
    let usize result = state::local(locals,c,element,0,false,false,start,end);
    if (state::emit(operations,blocks,c,state::operation(10,result,element,owner,0,array,index,start,end)) == 0) { return 0; } return result;
}
