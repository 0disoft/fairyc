module mir_verify_enums;
import borrow_shapes as borrows;
import mir_model as m;
import hir_shapes as shapes;
import enum_types as enums;
import tagged_types as tagged;
import nominal_identity as nominal;
import resource_identity as resources;

fn bool local(ref m::Program p, usize id, usize function, u32 type) {
    return id > 0 && id <= m::local_count(p) && m::local_at(p,id).function == function && m::local_at(p,id).type == type;
}
pub fn bool operation(ref m::Program p, m::Operation o) {
    let usize at = shapes::find(m::nominal(p),o.owner_type);
    if (!enums::valid(o.owner_type) || at == 0
        || o.payload_start != 0 || o.payload_end != 0 || o.boolean || o.right != 0
        || !local(p,o.result,o.function,o.type)) { return false; }
    if(o.kind==23){
        let usize field=o.operator as usize;
        if(field==0 || field>shapes::field_count(m::nominal(p)) || o.argument_start!=0 || o.argument_count!=0){return false;}
        let f=shapes::field_at(m::nominal(p),field);
        return f.owner==o.owner_type && f.type==o.type && (f.type>=1 && f.type<=6 || tagged::builtin(o.owner_type) && (tagged::builtin(f.type) || nominal::valid(f.type) || resources::valid(f.type) || borrows::count(f.type)==1))
            && shapes::payload_member(m::nominal(p),o.owner_type,field)!=0 && local(p,o.left,o.function,o.owner_type);
    }
    let usize member=shapes::member_at(m::nominal(p),o.owner_type,o.operator as usize);if(member==0){return false;}
    if (o.kind == 21) {
        let f=shapes::field_at(m::nominal(p),member);
        if(o.type!=o.owner_type || o.left!=0 || o.argument_count!=f.payload_count || o.argument_count==0 && o.argument_start!=0){return false;}
        var usize i=0;while(i<o.argument_count){
            if(!local(p,m::edge_at(p,o.argument_start+i),o.function,shapes::field_at(m::nominal(p),f.payload_start+i+1).type)){return false;}i+=1;
        }return true;
    }
    return o.kind == 22 && o.type == 6 && o.argument_start==0 && o.argument_count==0 && local(p,o.left,o.function,o.owner_type);
}
