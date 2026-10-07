module tests::foreign_calls;
import tests::hir_nominal as fixture;
import hir_model as h;
import mir_model as m;
import mir_lower as lower;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn m::Program changed(ref m::Program p,u32 mutation){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,m::source_view(p)).is_ok);
    var bytes::Buffer payload=bytes::empty();var v::Vec<m::Operation> ops=v::empty<m::Operation>();var usize i=1;
    while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.payload_end>o.payload_start){assert(bytes::append(ref mut payload,m::literal_text(p,i)).is_ok);}assert(v::push<m::Operation>(ref mut ops,o).is_ok);i+=1;}
    var v::Vec<m::Target> targets=v::empty<m::Target>();i=1;while(i<=m::target_count(p)){var t=m::target_at(p,i);if(t.foreign){if(mutation==1){t.link_start=7;t.link_end=11;}else if(mutation==2){t.result=6;}else if(mutation==3){t.foreign=false;}else if(mutation==4){t.unsafe_call=false;}}assert(v::push<m::Target>(ref mut targets,t).is_ok);i+=1;}
    var v::Vec<m::Signature> signatures=v::empty<m::Signature>();i=0;while(i<m::signature_count(p)){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=m::signature_at(p,i)}).is_ok);i+=1;}
    var v::Vec<m::Function> functions=v::empty<m::Function>();i=1;while(i<=m::function_count(p)){assert(v::push<m::Function>(ref mut functions,m::function_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Local> locals=v::empty<m::Local>();i=1;while(i<=m::local_count(p)){assert(v::push<m::Local>(ref mut locals,m::local_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Block> blocks=v::empty<m::Block>();i=1;while(i<=m::block_count(p)){var b=m::block_at(p,i);assert(v::push<m::Block>(ref mut blocks,b).is_ok);i+=1;}
    var v::Vec<usize> edges=v::empty<usize>();i=0;while(i<m::edge_count(p)){assert(v::push<usize>(ref mut edges,m::edge_at(p,i)).is_ok);i+=1;}
    return switch(m::assemble(source,payload,targets,signatures,functions,locals,blocks,ops,edges,0)){case .ok(p):{p}case .err(_):{assert(false);return changed(p,mutation);}};
}
@test fn void foreign_targets_reject_mutated_linker_identity(){
    let p=switch(lower::lower(fixture::retained(b"module main;@link_name(\"fixture_add\")unsafe extern C fn i32 foreign(i32 n);fn i32 main(){return unsafe{foreign(6)};}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0 && m::function_count(ref p)==1 && m::target_count(ref p)==2);
    let t=m::target_at(ref p,1);assert(t.foreign && t.external && t.unsafe_call && bytes::equal(m::foreign_text(ref p,1),b"fixture_add"));
    var bytes::Buffer original=bytes::empty();assert(c::render(ref p,ref mut original)==0);
    var u32 mutation=1;while(mutation<=4){let bad=changed(ref p,mutation);assert(verify::check(ref bad).code==if(mutation==3){1001}else{5106});
        var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;}
}
