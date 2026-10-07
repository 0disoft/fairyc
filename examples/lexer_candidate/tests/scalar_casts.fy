module tests::scalar_casts;
import tests::hir_nominal as fixture;
import hir_shapes as shapes;
import mir_model as m;
import mir_lower as lower;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;
fn m::Program compiled(){return switch(lower::lower(fixture::retained(b"module main;fn i32 main(){let u8 n=255;return n as i32;}"))){case .ok(p):{p}case .err(_):{assert(false);return compiled();}};}
fn m::Program changed(ref m::Program p,u32 mutation){var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,m::source_view(p)).is_ok);var bytes::Buffer payload=bytes::empty();var v::Vec<m::Operation> ops=v::empty<m::Operation>();var usize id=1;var bool altered=false;
while(id<=m::operation_count(p)){var o=m::operation_at(p,id);if(o.payload_end>o.payload_start){assert(bytes::len(ref payload)==o.payload_start);assert(bytes::append(ref mut payload,m::literal_text(p,id)).is_ok);}if(o.kind==35){if(mutation==1){o.owner_type=1;}else{o.operator=1;}altered=true;}assert(v::push<m::Operation>(ref mut ops,o).is_ok);id+=1;}assert(altered);
    var v::Vec<m::Target> targets=v::empty<m::Target>();id=1;while(id<=m::target_count(p)){assert(v::push<m::Target>(ref mut targets,m::target_at(p,id)).is_ok);id+=1;}
    var v::Vec<m::Signature> signatures=v::empty<m::Signature>();id=0;while(id<m::signature_count(p)){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=m::signature_at(p,id)}).is_ok);id+=1;}
    var v::Vec<m::Function> functions=v::empty<m::Function>();id=1;while(id<=m::function_count(p)){assert(v::push<m::Function>(ref mut functions,m::function_at(p,id)).is_ok);id+=1;}
    var v::Vec<m::Local> locals=v::empty<m::Local>();id=1;while(id<=m::local_count(p)){assert(v::push<m::Local>(ref mut locals,m::local_at(p,id)).is_ok);id+=1;}
    var v::Vec<m::Block> blocks=v::empty<m::Block>();id=1;while(id<=m::block_count(p)){assert(v::push<m::Block>(ref mut blocks,m::block_at(p,id)).is_ok);id+=1;}
    var v::Vec<usize> edges=v::empty<usize>();id=0;while(id<m::edge_count(p)){assert(v::push<usize>(ref mut edges,m::edge_at(p,id)).is_ok);id+=1;}
    let shapes::Table table=switch(shapes::copy(m::nominal(p))){case .ok(t):{t}case .err(_):{assert(false);return changed(p,mutation);}};
    return switch(m::assemble_nominal(source,payload,targets,signatures,functions,locals,blocks,ops,edges,0,table)){case .ok(p):{p}case .err(_):{assert(false);return changed(p,mutation);}};
}
@test fn void verified_cast_rejects_laundered_type_and_operator(){let p=compiled();assert(verify::check(ref p).code==0);var u32 mutation=1;while(mutation<=2){let bad=changed(ref p,mutation);assert(verify::check(ref bad).code==1001);var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;}}
