module mir_math;
import mir_model as m;
import mir_state as state;
import std::vec as v;

pub fn u32 trap_mask(u32 operator, u32 type, bool unary) {
    if (unary) { return if (operator == 2) { 1 } else { 0 }; }
    if (operator == 1 || operator == 2 || operator == 5) { return 1; }
    if (operator == 6 || operator == 7) { return if (type == 1 || type == 2) { 6 } else { 2 }; }
    if (operator == 11) { return 9; } if (operator == 12) { return 8; } return 0;
}
pub fn usize binary(ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut state::Context c, u32 type, u32 owner_type, u32 operator, usize left, usize right, usize start, usize end) {
    let u32 mask = trap_mask(operator,owner_type,false);
    if (mask != 0 && !state::guard(operations,blocks,c,owner_type,operator,left,right,mask,start,end)) { return 0; }
    let usize result = state::local(locals,c,type,0,false,false,start,end);
    if (result == 0 || state::emit(operations,blocks,c,state::operation(4,result,type,owner_type,operator,left,right,start,end)) == 0) { return 0; } return result;
}

