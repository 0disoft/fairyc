module tests::mixed_origins;
import tests::hir_nominal as fixture;
import mir_lower as lower;
import mir_model as m;
import mir_verify as verify;
import hir_shapes as shapes;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

// Alter only owned MIR return facts. Independent provenance must reject escape
// even though the original Fairy source and HIR were a valid scalar consumer.
fn m::Program escaping(ref m::Program p){
    var bytes::Buffer source=switch(bytes::copy(m::source_view(p))){case .ok(b):{b}case .err(_):{assert(false);return escaping(p);}};
    var bytes::Buffer payload=bytes::empty();var usize size=0;var usize i=1;
    while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.payload_end>size){size=o.payload_end;}i+=1;}
    assert(bytes::resize_zeroed(ref mut payload,size).is_ok);i=1;
    while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.payload_end>o.payload_start){let raw=m::literal_text(p,i);let ref mut u8[] dest=bytes::view_mut(ref mut payload);var usize j=0;while(j<raw.len){dest[o.payload_start+j]=raw[j];j+=1;}}i+=1;}
    var v::Vec<m::Target> targets=v::empty<m::Target>();i=1;while(i<=m::target_count(p)){var m::Target t=m::target_at(p,i);t.result=10;assert(v::push<m::Target>(ref mut targets,t).is_ok);i+=1;}
    var v::Vec<m::Signature> signatures=v::empty<m::Signature>();i=0;while(i<m::signature_count(p)){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=m::signature_at(p,i)}).is_ok);i+=1;}
    var v::Vec<m::Function> functions=v::empty<m::Function>();i=1;while(i<=m::function_count(p)){assert(v::push<m::Function>(ref mut functions,m::function_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Local> locals=v::empty<m::Local>();var usize selected=0;i=1;
    while(i<=m::local_count(p)){let l=m::local_at(p,i);if(l.type==10 && l.binding!=0 && bytes::equal(m::source_view(p)[l.start..l.end],b"selected")){selected=i;}assert(v::push<m::Local>(ref mut locals,l).is_ok);i+=1;}
    assert(selected!=0);
    var v::Vec<m::Block> blocks=v::empty<m::Block>();i=1;while(i<=m::block_count(p)){var m::Block b=m::block_at(p,i);if(b.kind==1){b.value=selected;}assert(v::push<m::Block>(ref mut blocks,b).is_ok);i+=1;}
    var v::Vec<m::Operation> operations=v::empty<m::Operation>();i=1;while(i<=m::operation_count(p)){assert(v::push<m::Operation>(ref mut operations,m::operation_at(p,i)).is_ok);i+=1;}
    var v::Vec<usize> edges=v::empty<usize>();i=0;while(i<m::edge_count(p)){assert(v::push<usize>(ref mut edges,m::edge_at(p,i)).is_ok);i+=1;}
    let nominal=switch(shapes::copy(m::nominal(p))){case .ok(s):{s}case .err(_):{assert(false);return escaping(p);}};
    return switch(m::assemble_nominal(source,payload,targets,signatures,functions,locals,blocks,operations,edges,m::module_index(p),nominal)){case .ok(value):{value}case .err(_):{assert(false);return escaping(p);}};
}
@test fn void independent_mir_retains_union_and_rejects_forged_escape(){
    let hir=fixture::retained(b"module main;fn i32 main(){let u8[1] a=[1];let u8[1] b=[2];let ref u8[] selected=if(true){ref a}else{ref b};assert(selected[0]>=1);return 7;}");
    let p=switch(lower::lower(hir)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);
    let bad=escaping(ref p);assert(verify::check(ref bad).code==2103);
    var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);
    assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));
}
