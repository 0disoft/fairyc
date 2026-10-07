module mir_reference;
import mir_model as m;
import reference_types as refs;
import hir_shapes as shapes;
pub fn bool operation(ref m::Program p,m::Operation o){
    if(o.right!=0 || o.argument_count!=0 || o.argument_start!=0 || o.boolean || o.payload_start!=0 || o.payload_end!=0 || o.result==0 || o.left==0){return false;}
    let left=m::local_at(p,o.left);let result=m::local_at(p,o.result);if(left.function!=o.function || result.function!=o.function || left.type!=o.owner_type || result.type!=o.type){return false;}
    if(o.kind==28){return refs::valid(o.type) && refs::inner(o.type)==o.owner_type && o.operator==0;}
    if(!refs::valid(o.owner_type) || shapes::needs_drop(o.type)){return false;}
    if(o.kind==29){return refs::inner(o.owner_type)==o.type && o.operator==0;}
    if(o.kind==31 && o.operator!=0 && (o.operator as usize)<=shapes::field_count(m::nominal(p))){let f=shapes::field_at(m::nominal(p),o.operator as usize);return f.owner==refs::inner(o.owner_type) && f.type==o.type;}return false;
}
