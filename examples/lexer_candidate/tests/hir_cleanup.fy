module tests::hir_cleanup;
import tests::hir_nominal as fixture;
import hir_model as h;
import hir_shapes as shapes;
import hir_output as output;
import mir_lower as mir;
import std::bytes as bytes;
import std::vec as v;

fn ref u8[] sample(){return b"module main;resource Empty{cleanup{}}resource R{i32 n;cleanup{let i32 x=this.n;this.n=x+1;return;}}fn void take(R r){}fn i32 main(){let R r=R{.n=2};take(r);return 0;}";}
@test
fn void private_drop_body_receiver_and_classification_are_owned(){
    let p=fixture::retained(sample());assert(h::drop_count(ref p)==2 && h::function_count(ref p)==3);
    let empty=h::drop_at(ref p,0);let d=h::drop_at(ref p,1);assert(!empty.nonempty && empty.function==0 && empty.receiver==0);
    assert(d.nonempty && d.function!=0 && d.receiver!=0);
    let f=h::function_at(ref p,0);let b=h::binding_at(ref p,d.function);let receiver=h::binding_at(ref p,d.receiver);
    assert(f.binding==d.function && !f.public && f.parameter_count==1 && h::edge_at(ref p,f.parameter_start)==d.receiver);
    assert(h::binding_code(b.kind)==9 && h::type_code(b.type)==7 && h::binding_code(receiver.kind)==10 && receiver.owner==d.function && h::type_code(receiver.type)==d.type);
    assert(bytes::equal(h::source_view(ref p)[d.start..d.start+7],b"cleanup"));
    var bytes::Buffer out=bytes::empty();assert(output::render(ref p,ref mut out)==0);
    assert(bytes::view(ref out).len>25 && bytes::equal(bytes::view(ref out)[..23],b"fairyc.resource-hir/v2\n"));
    let u32 code=switch(mir::lower(p)){case .ok(_):{0}case .err(e):{e.code}};assert(code==0);
}

fn u32 mutated(ref h::Program p,u32 mutation){
    let drop=h::drop_at(p,1);
    var bytes::Buffer source=bytes::empty();var bytes::Buffer payload=bytes::empty();assert(bytes::append(ref mut source,h::source_view(p)).is_ok);
    var v::Vec<h::Binding> bindings=v::empty<h::Binding>();var v::Vec<h::Function> functions=v::empty<h::Function>();
    var v::Vec<h::Expression> expressions=v::empty<h::Expression>();var v::Vec<h::Statement> statements=v::empty<h::Statement>();
    var v::Vec<h::Block> blocks=v::empty<h::Block>();var v::Vec<usize> edges=v::empty<usize>();var v::Vec<h::Signature> signatures=v::empty<h::Signature>();var v::Vec<h::Drop> drops=v::empty<h::Drop>();
    var usize i=1;while(i<=h::binding_count(p)){var h::Binding b=h::binding_at(p,i);
        let h::Binding changed=h::Binding{ .link_start=b.link_start,.link_end=b.link_end, .unsafe_call=b.unsafe_call,.must_use=b.must_use,.effects=b.effects,.kind=if(i==drop.function && mutation==13){.Function}else if(i==drop.receiver && mutation==5){.Parameter}else{b.kind},.type=if(i==drop.function && mutation==4){.I32}else{b.type},.start=b.start,.end=b.end,.owner=if(i==drop.receiver && mutation==6){1}else{b.owner},.scope=b.scope,.module_index=b.module_index,.declaration=b.declaration,.signature_start=b.signature_start,.signature_count=b.signature_count};
        assert(v::push<h::Binding>(ref mut bindings,changed).is_ok);i+=1;
    }
    i=0;while(i<h::function_count(p)){var h::Function f=h::function_at(p,i);if(f.binding==drop.function && mutation==7){f.public=true;}assert(v::push<h::Function>(ref mut functions,f).is_ok);i+=1;}
    i=1;while(i<=h::expression_count(p)){var h::Expression e=h::expression_at(p,i);
        if(mutation==10 && h::expression_code(e.kind)==6){e.binding=drop.function;}

        if(h::expression_code(e.kind)==1){let usize start=bytes::len(ref payload);assert(bytes::append(ref mut payload,h::integer_text(p,i)).is_ok);e.payload_start=start;e.payload_end=bytes::len(ref payload);}
        let h::Expression changed=h::Expression{.kind=if(mutation==12 && h::expression_code(e.kind)==3 && e.binding==drop.receiver){.Move}else{e.kind},.type=e.type,.operator=e.operator,.start=e.start,.end=e.end,.left=e.left,.right=e.right,.alternative=e.alternative,.block=e.block,.binding=e.binding,.edge_start=e.edge_start,.edge_count=e.edge_count,.payload_start=e.payload_start,.payload_end=e.payload_end,.boolean=e.boolean};
        assert(v::push<h::Expression>(ref mut expressions,changed).is_ok);i+=1;
    }
    i=1;while(i<=h::statement_count(p)){assert(v::push<h::Statement>(ref mut statements,h::statement_at(p,i)).is_ok);i+=1;}
    i=1;while(i<=h::block_count(p)){assert(v::push<h::Block>(ref mut blocks,h::block_at(p,i)).is_ok);i+=1;}
    i=0;while(i<h::edge_count(p)){assert(v::push<usize>(ref mut edges,h::edge_at(p,i)).is_ok);i+=1;}
    i=0;while(i<h::signature_count(p)){assert(v::push<h::Signature>(ref mut signatures,h::Signature{.type=h::signature_at(p,i)}).is_ok);i+=1;}
    i=0;while(i<h::drop_count(p)){var h::Drop d=h::drop_at(p,i);
        if(i==1){if(mutation==1){d.function=0;}if(mutation==2){d.receiver=0;}if(mutation==3){d.type=98307;}
            if(mutation==11){d.nonempty=false;d.function=0;d.receiver=0;}}
        if(mutation!=9){assert(v::push<h::Drop>(ref mut drops,d).is_ok);}i+=1;
    }
    if(mutation==8){assert(v::push<h::Drop>(ref mut drops,drop).is_ok);}
    let table=switch(shapes::copy(h::nominal(p))){case .ok(value):{value}case .err(_):{assert(false);return 0;}};
    return switch(h::assemble_resources(source,payload,functions,bindings,expressions,statements,blocks,edges,signatures,0,table,drops)){case .ok(_):{0}case .err(e):{e.code}};
}
@test
fn void forged_drop_links_and_owning_receivers_are_rejected(){
    let p=fixture::retained(sample());assert(mutated(ref p,0)==0);var u32 mutation=1;
    while(mutation<=13){assert(mutated(ref p,mutation)==4);mutation+=1;}
}
@test
fn void empty_nested_resources_also_keep_drop_definitions(){
    let p=fixture::retained(b"module main;resource Inner{cleanup{}}resource Outer{Inner item;cleanup{}}fn void take(Outer value){}");
    assert(h::drop_count(ref p)==2 && !h::drop_at(ref p,0).nonempty && !h::drop_at(ref p,1).nonempty);
}
