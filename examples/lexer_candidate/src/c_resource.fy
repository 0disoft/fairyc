module c_resource;
import mir_model as m;
import hir_shapes as shapes;
import resource_identity as resources;
import c_text as w;
import std::bytes as bytes;

// Verified numeric paths preserve exact private glue and layout identities.
// Outer user cleanup runs before reverse-field recursion. Partial aggregates
// never reach this operation; their completed field temporaries drop separately.
fn bool emit(ref m::Program p,u32 type,ref u8[] value,ref mut bytes::Buffer out,ref mut w::State s,usize depth){
    if(depth>=64){s.code=2;return false;}
    if(!resources::valid(type)){return true;}
    let usize at=m::drop_index(p,type);if(at==0){s.code=4;return false;}let d=m::drop_at(p,at-1);
    if(d.nonempty && (!w::text(out,s,b"    ") || !w::function(out,s,d.target) || !w::text(out,s,b"(") || !w::text(out,s,value) || !w::text(out,s,b");\n"))){return false;}
    let shape=shapes::at(m::nominal(p),shapes::find(m::nominal(p),type)-1);var usize i=shape.field_count;
    while(i!=0){i-=1;let usize field=shape.field_start+i+1;let u32 child_type=shapes::field_at(m::nominal(p),field).type;
        if(shapes::needs_drop(child_type)){var bytes::Buffer child=bytes::empty();
            if(!bytes::append(ref mut child,value).is_ok || !bytes::append(ref mut child,b".fy_f_").is_ok || !bytes::append_usize(ref mut child,field).is_ok){s.code=3;return false;}
            if(!emit(p,child_type,bytes::view(ref child),out,s,depth+1)){return false;}}
    }return true;
}
pub fn bool operation(ref m::Program p,m::Operation o,ref mut bytes::Buffer out,ref mut w::State s){
    var bytes::Buffer value=bytes::empty();if(!bytes::append(ref mut value,b"fy_l_").is_ok || !bytes::append_usize(ref mut value,o.left).is_ok){s.code=3;return false;}
    return emit(p,o.type,bytes::view(ref value),out,s,0);
}
