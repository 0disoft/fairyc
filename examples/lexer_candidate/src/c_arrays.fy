module c_arrays;
import mir_model as m;
import array_types as arrays;
import c_text as w;
import std::bytes as bytes;
import std::vec as v;
import hir_shapes as shapes;

pub fn bool present(ref m::Program p) { return m::has_arrays(p); }
fn bool mark(u32 type, ref mut bool[] flat, ref mut v::Vec<u32> nested, ref mut w::State s) {
    if(!arrays::valid(type)){return true;}
    if(arrays::flat(type)){flat[(type-1024) as usize]=true;return true;}
    flat[(arrays::element(type)-1024) as usize]=true;var usize i=0;
    while(i<v::len<u32>(nested)){if(*v::at<u32>(nested,i)==type){return true;}i+=1;}
    if(v::len<u32>(nested)>=1024){s.code=2;return false;}
    if(!v::push<u32>(nested,type).is_ok){s.code=3;return false;}return true;
}
fn bool definition(u32 type, ref mut bytes::Buffer out, ref mut w::State s) {
    return w::type(out,s,type) && w::text(out,s,b"{ ") && w::type(out,s,arrays::element(type)) && w::text(out,s,b" fy_items[") && w::number(out,s,arrays::length(type)) && w::text(out,s,b"]; };\n");
}
pub fn bool definitions(ref m::Program p, ref mut bytes::Buffer out, ref mut w::State s) {
    var bool[520] used=[false;520];var v::Vec<u32> nested=v::empty<u32>();var usize i=1;
    while(i<=m::local_count(p)){if(!mark(m::local_at(p,i).type,ref mut used[..],ref mut nested,s)){return false;}i+=1;}
    i=1;while(i<=m::target_count(p)){if(!mark(m::target_at(p,i).result,ref mut used[..],ref mut nested,s)){return false;}i+=1;}
    i=0;while(i<m::signature_count(p)){if(!mark(m::signature_at(p,i),ref mut used[..],ref mut nested,s)){return false;}i+=1;}
    i=1;while(i<=shapes::field_count(m::nominal(p))){if(!mark(shapes::field_at(m::nominal(p),i).type,ref mut used[..],ref mut nested,s)){return false;}i+=1;}
    var u32 t=1033;while(t<=1542){if(arrays::flat(t) && used[(t-1024) as usize] && !definition(t,out,s)){return false;}t+=1;}
    // Sorting numeric canonical IDs makes dependency-first output deterministic.
    i=1;while(i<v::len<u32>(ref nested)){let u32 value=*v::at<u32>(ref nested,i);var usize j=i;while(j>0 && *v::at<u32>(ref nested,j-1)>value){let u32 previous=*v::at<u32>(ref nested,j-1);{let ref mut u32 slot=v::at_mut<u32>(ref mut nested,j);*slot=previous;}j-=1;}{let ref mut u32 slot=v::at_mut<u32>(ref mut nested,j);*slot=value;}i+=1;}
    i=0;while(i<v::len<u32>(ref nested)){if(!definition(*v::at<u32>(ref nested,i),out,s)){return false;}i+=1;}return true;
}
pub fn bool operation(ref m::Program p, m::Operation o, ref mut bytes::Buffer out, ref mut w::State s) {
    if (o.kind == 12) {
        // Keep MSVC's reachability analysis from turning a known failing index
        // into C4702 after the trap. The source expression still executes once.
        return w::text(out,s,b"    volatile uintptr_t fy_index_") && w::number(out,s,o.result) && w::text(out,s,b"=") && w::local(out,s,o.left)
            && w::text(out,s,b";\n    if(fy_index_") && w::number(out,s,o.result) && w::text(out,s,b">=") && w::number(out,s,arrays::length(o.owner_type))
            && w::text(out,s,b")fy_runtime_trap(UINT32_C(6));\n    ") && w::local(out,s,o.result) && w::text(out,s,b"=fy_index_") && w::number(out,s,o.result) && w::text(out,s,b";\n");
    }
    if (!w::text(out,s,b"    ") || !w::local(out,s,o.result) || !w::text(out,s,b"=")) { return false; }
    if (o.kind == 9) {
        if (!w::text(out,s,b"(") || !w::type(out,s,o.type) || !w::text(out,s,b"){.fy_items={")) { return false; }
        var usize i = 0; while (i < arrays::length(o.type)) {
            if (i != 0 && !w::text(out,s,b",")) { return false; }
            if (!w::local(out,s,m::edge_at(p,o.argument_start + if (o.boolean) { 0 } else { i }))) { return false; } i += 1;
        }
        return w::text(out,s,b"}};\n");
    }
    if (o.kind == 13) { return w::text(out,s,b"((uintptr_t)") && w::number(out,s,arrays::length(o.owner_type)) && w::text(out,s,b");\n"); }
    if (!w::local(out,s,o.left)) { return false; }
    if (o.kind == 10) { return w::text(out,s,b".fy_items[") && w::local(out,s,o.right) && w::text(out,s,b"];\n"); }
    return w::text(out,s,b";\n    ") && w::local(out,s,o.result) && w::text(out,s,b".fy_items[") && w::local(out,s,o.right) && w::text(out,s,b"]=") && w::local(out,s,m::edge_at(p,o.argument_start)) && w::text(out,s,b";\n");
}
