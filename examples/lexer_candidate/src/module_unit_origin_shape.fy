module module_unit_origin_shape;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_value_builtin as borrows;

// Stage0 admits at most one safe-borrow leaf through Option/Result wrappers.
// Ordinary nominal aggregates are checked by the owning shape contracts.
pub fn Result<u32,m::Problem> leaves(ref values::Plan p,usize id,m::Frame frame,ref mut usize work,usize depth){
    if(depth>=64 || !m::charge(work,1)){return .err(m::fail(frame,10,0,0,0,0));}if(id==0){return .ok(0);}let t=values::type_at(p,id);if(borrows::borrowed(t)){return .ok(1);}
    if(t.kind==5 || t.kind==6){let first=switch(leaves(p,t.first,frame,work,depth+1)){case .ok(n):{n}case .err(p):{return .err(p);}};if(t.kind==5){return .ok(first);}let second=switch(leaves(p,t.second,frame,work,depth+1)){case .ok(n):{n}case .err(p):{return .err(p);}};return .ok(if(first+second>1){2}else{first+second});}
    return .ok(0);
}
