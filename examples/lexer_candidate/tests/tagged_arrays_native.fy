module tests::tagged_arrays_native;
import tests::hir_nominal as fixture;
import hir_model as h;
import hir_shapes as shapes;
import array_types as arrays;
import type_state as state;
import mir_lower as lower;
import mir_model as m;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn m::Program cloned(ref m::Program p,u32 mutation) {
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,m::source_view(p)).is_ok);
    var bytes::Buffer payload=bytes::empty();var usize end=0;var usize i=1;
    while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.payload_end>end){end=o.payload_end;}i+=1;}
    assert(bytes::resize_zeroed(ref mut payload,end).is_ok);i=1;
    while(i<=m::operation_count(p)){let o=m::operation_at(p,i);let data=m::literal_text(p,i);var usize j=0;
        while(j<data.len){bytes::view_mut(ref mut payload)[o.payload_start+j]=data[j];j+=1;}i+=1;}
    var v::Vec<m::Target> targets=v::empty<m::Target>();i=1;while(i<=m::target_count(p)){assert(v::push<m::Target>(ref mut targets,m::target_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Signature> signatures=v::empty<m::Signature>();i=0;while(i<m::signature_count(p)){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=m::signature_at(p,i)}).is_ok);i+=1;}
    var v::Vec<m::Function> functions=v::empty<m::Function>();i=1;while(i<=m::function_count(p)){assert(v::push<m::Function>(ref mut functions,m::function_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Local> locals=v::empty<m::Local>();var usize integer=0;i=1;
    while(i<=m::local_count(p)){let l=m::local_at(p,i);if(l.type==1 && integer==0){integer=i;}assert(v::push<m::Local>(ref mut locals,l).is_ok);i+=1;}
    var v::Vec<m::Block> blocks=v::empty<m::Block>();i=1;while(i<=m::block_count(p)){assert(v::push<m::Block>(ref mut blocks,m::block_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Operation> ops=v::empty<m::Operation>();var bool changed=false;var usize edge=0;i=1;
    while(i<=m::operation_count(p)){var o=m::operation_at(p,i);
        if(!changed){
            if(mutation==1 && o.kind==10){o.type=1;changed=true;}
            else if(mutation==2 && o.kind==12 && arrays::length(o.owner_type)==1){o.owner_type+=1;changed=true;}
            else if(mutation==3 && o.kind==11){edge=o.argument_start;changed=true;}
            else if(mutation==4 && o.kind==9 && arrays::length(o.type)==1){o.kind=6;o.boolean=false;changed=true;}
            else if(mutation==5 && o.kind==10){o.kind=7;o.right=0;o.operator=state::small_id(shapes::at(m::nominal(p),shapes::find(m::nominal(p),o.owner_type)-1).field_start)+1;changed=true;}
        }assert(v::push<m::Operation>(ref mut ops,o).is_ok);i+=1;
    }assert(changed && integer!=0);
    var v::Vec<usize> edges=v::empty<usize>();i=0;while(i<m::edge_count(p)){assert(v::push<usize>(ref mut edges,if(mutation==3 && i==edge){integer}else{m::edge_at(p,i)}).is_ok);i+=1;}
    let table=switch(shapes::copy(m::nominal(p))){case .ok(value):{value}case .err(_):{assert(false);return cloned(p,mutation);}};
    return switch(m::assemble_nominal(source,payload,targets,signatures,functions,locals,blocks,ops,edges,m::module_index(p),table)){case .ok(value):{value}case .err(_):{assert(false);return cloned(p,mutation);}};
}

@test
fn void array_mir_rejects_wrong_elements_indexes_and_nominal_operations_before_publication() {
    let p=switch(lower::lower(fixture::retained(b"module main;fn i32 main(){var Option<i32>[1] a=[.some(7)];let Option<i32>[2] b=[.none;2];discard b;a[0]=.some(8);return switch(a[0]){case .some(n):{n}case .none:{0}};}"))){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);var u32 mutation=1;
    while(mutation<=5){let bad=cloned(ref p,mutation);assert(verify::check(ref bad).code==if(mutation==2){1501}else{1001});
        var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0);
        assert(bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;
    }
}

@test
fn void rank_two_composite_identity_retains_both_lengths_and_child_layout() {
    let p=fixture::retained(b"module main;struct T{i32 n;}fn i32 main(){let a=[[T{.n=7},T{.n=8}],[T{.n=9},T{.n=10}]];return a[0][0].n;}");
    let ref shapes::Table table=h::nominal(ref p);var usize i=0;var bool found=false;
    while(i<shapes::count(table)){let s=shapes::at(table,i);if(arrays::rank_two(s.type)){let u32 child=shapes::array_element(table,s.type);
        assert(arrays::length(s.type)==2 && arrays::length(child)==2 && arrays::encoded_element(s.type)==child);found=true;}i+=1;
    }assert(found);let ir=switch(lower::lower(p)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(verify::check(ref ir).code==0);var bytes::Buffer unit=bytes::empty();assert(c::render(ref ir,ref mut unit)==0);
}
