module tests::unsafe_scopes;
import tests::hir_nominal as fixture;
import hir_model as h;
import mir_model as m;
import mir_lower as lower;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn u32 reassembled(ref h::Program p,u32 mutation){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,h::source_view(p)).is_ok);
    var v::Vec<h::Binding> bindings=v::empty<h::Binding>();var usize i=1;
    while(i<=h::binding_count(p)){var b=h::binding_at(p,i);if(mutation==2){b.unsafe_call=false;}assert(v::push<h::Binding>(ref mut bindings,b).is_ok);i+=1;}
    var v::Vec<h::Function> functions=v::empty<h::Function>();i=0;while(i<h::function_count(p)){assert(v::push<h::Function>(ref mut functions,h::function_at(p,i)).is_ok);i+=1;}
    var v::Vec<h::Expression> expressions=v::empty<h::Expression>();i=1;while(i<=h::expression_count(p)){var e=h::expression_at(p,i);if(mutation==3 && h::expression_code(e.kind)==6){e.boolean=false;}assert(v::push<h::Expression>(ref mut expressions,e).is_ok);i+=1;}
    var v::Vec<h::Statement> statements=v::empty<h::Statement>();i=1;while(i<=h::statement_count(p)){assert(v::push<h::Statement>(ref mut statements,h::statement_at(p,i)).is_ok);i+=1;}
    var v::Vec<h::Block> blocks=v::empty<h::Block>();i=1;while(i<=h::block_count(p)){var b=h::block_at(p,i);if(mutation==1){b.unsafe_body=false;}assert(v::push<h::Block>(ref mut blocks,b).is_ok);i+=1;}
    var v::Vec<usize> edges=v::empty<usize>();i=0;while(i<h::edge_count(p)){assert(v::push<usize>(ref mut edges,h::edge_at(p,i)).is_ok);i+=1;}
    var v::Vec<h::Signature> signatures=v::empty<h::Signature>();i=0;while(i<h::signature_count(p)){assert(v::push<h::Signature>(ref mut signatures,h::Signature{.type=h::signature_at(p,i)}).is_ok);i+=1;}
    var bytes::Buffer payload=bytes::empty();i=1;while(i<=h::expression_count(p)){let e=h::expression_at(p,i);if(e.payload_end>e.payload_start){assert(bytes::append(ref mut payload,h::integer_text(p,i)).is_ok);}i+=1;}
    return switch(h::assemble(source,payload,functions,bindings,expressions,statements,blocks,edges,signatures,0)){case .ok(_):{0}case .err(e):{e.code}};
}
@test fn void owned_hir_rechecks_lexical_permission(){
    let p=fixture::retained(b"module main;unsafe fn i32 host(){return 7;}fn i32 main(){return unsafe{host()};}");
    assert(reassembled(ref p,0)==0);var u32 mutation=1;while(mutation<=3){assert(reassembled(ref p,mutation)==5101);mutation+=1;}
}
fn m::Program changed(ref m::Program p){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,m::source_view(p)).is_ok);
    var bytes::Buffer payload=bytes::empty();var v::Vec<m::Operation> ops=v::empty<m::Operation>();var usize i=1;
    while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.payload_end>o.payload_start){assert(bytes::append(ref mut payload,m::literal_text(p,i)).is_ok);}assert(v::push<m::Operation>(ref mut ops,o).is_ok);i+=1;}
    var v::Vec<m::Target> targets=v::empty<m::Target>();i=1;while(i<=m::target_count(p)){assert(v::push<m::Target>(ref mut targets,m::target_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Signature> signatures=v::empty<m::Signature>();i=0;while(i<m::signature_count(p)){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=m::signature_at(p,i)}).is_ok);i+=1;}
    var v::Vec<m::Function> functions=v::empty<m::Function>();i=1;while(i<=m::function_count(p)){assert(v::push<m::Function>(ref mut functions,m::function_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Local> locals=v::empty<m::Local>();i=1;while(i<=m::local_count(p)){assert(v::push<m::Local>(ref mut locals,m::local_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Block> blocks=v::empty<m::Block>();i=1;while(i<=m::block_count(p)){var b=m::block_at(p,i);b.unsafe_call=false;assert(v::push<m::Block>(ref mut blocks,b).is_ok);i+=1;}
    var v::Vec<usize> edges=v::empty<usize>();i=0;while(i<m::edge_count(p)){assert(v::push<usize>(ref mut edges,m::edge_at(p,i)).is_ok);i+=1;}
    return switch(m::assemble(source,payload,targets,signatures,functions,locals,blocks,ops,edges,0)){case .ok(p):{p}case .err(_):{assert(false);return changed(p);}};
}
@test fn void mir_rejects_stripped_call_provenance_before_output(){
    let p=switch(lower::lower(fixture::retained(b"module main;unsafe fn i32 host(){return 7;}fn i32 main(){return unsafe{host()};}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);let bad=changed(ref p);assert(verify::check(ref bad).code==5101);
    var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));
}

@test fn void owned_hir_rechecks_pointer_cast_permission(){
    let p=fixture::retained(b"module main;fn i32 main(){let i32 n=7;let ptr i32 p=unsafe{ref n as ptr i32};assert(p!=null);return 7;}");
    assert(reassembled(ref p,0)==0 && reassembled(ref p,1)==5101);
}
