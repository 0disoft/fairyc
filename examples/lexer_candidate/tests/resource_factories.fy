module tests::resource_factories;
import tests::hir_nominal as fixture;
import hir_model as h;
import hir_shapes as shapes;
import resource_identity as resources;
import tagged_types as tagged;
import mir_lower as mir;
import std::bytes as bytes;
import std::vec as v;

@test
fn void factories_keep_resource_identity_and_written_initializer_order() {
    let p=fixture::retained(b"module main;resource Pair{i64 x;bool y;cleanup{}}fn i64 first(){return 4000000000;}fn bool second(){return true;}pub fn Pair make(){return Pair{.y=second(),.x=first()};}fn Pair forward(){return make();}");
    var usize id=1;var usize constructions=0;var usize calls=0;
    while(id<=h::expression_count(ref p)){let e=h::expression_at(ref p,id);
        if(h::expression_code(e.kind)==9){
            constructions+=1;assert(resources::valid(h::type_code(e.type)) && h::binding_code(h::binding_at(ref p,e.binding).kind)==8 && e.edge_count==2);
            let y=h::expression_at(ref p,h::edge_at(ref p,e.edge_start));let x=h::expression_at(ref p,h::edge_at(ref p,e.edge_start+1));
            assert(y.binding==2 && x.binding==1 && y.left<x.left && x.left<id);
            assert(h::expression_code(h::expression_at(ref p,y.left).kind)==6 && bytes::equal(h::binding_text(ref p,h::expression_at(ref p,y.left).binding),b"second"));
            assert(bytes::equal(h::binding_text(ref p,h::expression_at(ref p,x.left).binding),b"first"));
        }else if(h::expression_code(e.kind)==6){calls+=1;}
        id+=1;
    }
    assert(constructions==1 && calls==3);
    assert(switch(mir::lower(p)){case .ok(_):{false}case .err(problem):{problem.code==4}});
}

@test
fn void fallible_factory_retains_both_alternatives_and_nested_ownership() {
    let p=fixture::retained(b"module main;resource R{i64 n;cleanup{}}resource Box{R child;Option<R> optional;cleanup{}}fn R make(){return R{.n=4000000000};}fn Result<Box,i32> checked(bool ok){if(!ok){return .err(9);}return .ok(Box{.optional=.none,.child=make()});}");
    var usize id=1;var usize constructions=0;var usize ok=0;var usize err=0;
    while(id<=h::expression_count(ref p)){let e=h::expression_at(ref p,id);let u32 type=h::type_code(e.type);
        if(h::expression_code(e.kind)==9){constructions+=1;assert(resources::valid(type));}
        if(h::expression_code(e.kind)==18 && tagged::result(type)){let shape=shapes::at(h::nominal(ref p),shapes::find(h::nominal(ref p),type)-1);if(e.binding==shape.field_start+2){ok+=1;}else{assert(e.binding==shape.field_start+4);err+=1;}}
        id+=1;
    }
    assert(constructions==2 && ok==1 && err==1);
    let ref shapes::Table table=h::nominal(ref p);assert(shapes::field_owns_resource(table,2) && shapes::field_owns_resource(table,3));
}

// Reassemble all owned arenas; altered construction metadata must fail without
// rereading or trusting the discarded syntax/type owner.
fn u32 mutated(ref h::Program p,u32 mutation) {
    var bytes::Buffer source=bytes::empty();var bytes::Buffer payload=bytes::empty();
    assert(bytes::append(ref mut source,h::source_view(p)).is_ok);
    var v::Vec<h::Function> functions=v::empty<h::Function>();var v::Vec<h::Binding> bindings=v::empty<h::Binding>();
    var v::Vec<h::Expression> expressions=v::empty<h::Expression>();var v::Vec<h::Statement> statements=v::empty<h::Statement>();
    var v::Vec<h::Block> blocks=v::empty<h::Block>();var v::Vec<usize> edges=v::empty<usize>();var v::Vec<h::Signature> signatures=v::empty<h::Signature>();
    var usize i=0;while(i<h::function_count(p)){assert(v::push<h::Function>(ref mut functions,h::function_at(p,i)).is_ok);i+=1;}
    i=1;while(i<=h::binding_count(p)){let original=h::binding_at(p,i);let b=h::Binding{ .link_start=0,.link_end=0, .unsafe_call=false,.must_use=false,.effects=0,.kind=if(mutation==1 && h::binding_code(original.kind)==8){.Struct}else{original.kind},.type=original.type,.start=original.start,.end=original.end,.owner=original.owner,.scope=original.scope,.module_index=original.module_index,.declaration=original.declaration,.signature_start=original.signature_start,.signature_count=original.signature_count};assert(v::push<h::Binding>(ref mut bindings,b).is_ok);i+=1;}
    var usize duplicate_edge=0;i=1;
    while(i<=h::expression_count(p)){var e=h::expression_at(p,i);
        if(h::expression_code(e.kind)==1){e.payload_start=bytes::len(ref payload);assert(bytes::append(ref mut payload,h::integer_text(p,i)).is_ok);e.payload_end=bytes::len(ref payload);}
        if(h::expression_code(e.kind)==9){if(mutation==2){e.edge_count-=1;}if(mutation==3){duplicate_edge=e.edge_start+2;}if(mutation==4){e.binding=1;}}
        if(mutation==6 && h::expression_code(e.kind)==11){e.binding=0;}
        let adjusted=h::Expression{.kind=e.kind,.type=if(mutation==5 && h::expression_code(e.kind)==9){.Nominal{.id=257}}else{e.type},.operator=e.operator,.start=e.start,.end=e.end,.left=e.left,.right=e.right,.alternative=e.alternative,.block=e.block,.binding=e.binding,.edge_start=e.edge_start,.edge_count=e.edge_count,.payload_start=e.payload_start,.payload_end=e.payload_end,.boolean=e.boolean};assert(v::push<h::Expression>(ref mut expressions,adjusted).is_ok);i+=1;
    }
    i=1;while(i<=h::statement_count(p)){assert(v::push<h::Statement>(ref mut statements,h::statement_at(p,i)).is_ok);i+=1;}
    i=1;while(i<=h::block_count(p)){assert(v::push<h::Block>(ref mut blocks,h::block_at(p,i)).is_ok);i+=1;}
    i=0;while(i<h::edge_count(p)){let usize edge=if(i+1==duplicate_edge){h::edge_at(p,i-1)}else{h::edge_at(p,i)};assert(v::push<usize>(ref mut edges,edge).is_ok);i+=1;}
    i=0;while(i<h::signature_count(p)){assert(v::push<h::Signature>(ref mut signatures,h::Signature{.type=h::signature_at(p,i)}).is_ok);i+=1;}
    let ref shapes::Table table=h::nominal(p);var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    i=0;while(i<shapes::count(table)){assert(v::push<shapes::Shape>(ref mut records,shapes::at(table,i)).is_ok);i+=1;}
    i=1;while(i<=shapes::field_count(table)){assert(v::push<shapes::Field>(ref mut fields,shapes::field_at(table,i)).is_ok);i+=1;}
    return switch(h::assemble_nominal(source,payload,functions,bindings,expressions,statements,blocks,edges,signatures,0,shapes::assemble(records,fields))){case .ok(_):{0}case .err(problem):{problem.code}};
}

@test
fn void forged_resource_construction_and_initializers_cannot_enter_owned_hir() {
    let p=fixture::retained(b"module main;struct Empty{}resource R{i32 x;i64 y;cleanup{}}fn R make(){return R{.y=4000000000,.x=7};}");
    assert(mutated(ref p,0)==0);var u32 mutation=1;while(mutation<=6){assert(mutated(ref p,mutation)==4);mutation+=1;}
}
