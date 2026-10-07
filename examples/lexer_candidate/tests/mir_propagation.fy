module tests::mir_propagation;
import tests::hir_nominal as fixture;
import hir_model as h;
import hir_shapes as shapes;
import hir_output as output;
import tagged_types as tagged;
import mir_model as m;
import mir_lower as lower;
import mir_verify as verify;
import c_flow as flow;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

// This source has only boolean payload literals, so its canonical byte arena is
// empty. Reassembly mutates retained nodes after the syntax/type owners are gone.
fn Result<h::Program,h::Problem> rebuilt(u32 mutation) {
    let p=fixture::retained(b"module main;fn Result<i64,bool> f(Result<i32,bool> r){discard r?;return .err(false);}");
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,h::source_view(ref p)).is_ok);
    let table=switch(shapes::copy(h::nominal(ref p))){case .ok(t):{t}case .err(_):{assert(false);return rebuilt(0);}};
    var v::Vec<h::Function> functions=v::empty<h::Function>();var v::Vec<h::Binding> bindings=v::empty<h::Binding>();
    var v::Vec<h::Expression> expressions=v::empty<h::Expression>();var v::Vec<h::Statement> statements=v::empty<h::Statement>();
    var v::Vec<h::Block> blocks=v::empty<h::Block>();var v::Vec<usize> edges=v::empty<usize>();var v::Vec<h::Signature> signatures=v::empty<h::Signature>();
    var usize i=0;while(i<h::function_count(ref p)){assert(v::push<h::Function>(ref mut functions,h::function_at(ref p,i)).is_ok);i+=1;}
    i=1;while(i<=h::binding_count(ref p)){
        let h::Binding b=h::binding_at(ref p,i);
        let h::Type type=if(mutation==8 && h::binding_code(b.kind)==1){.I32}else{b.type};
        assert(v::push<h::Binding>(ref mut bindings,h::Binding{.must_use=b.must_use,.effects=b.effects,.kind=b.kind,.type=type,.start=b.start,.end=b.end,.owner=b.owner,.scope=b.scope,.module_index=b.module_index,.declaration=b.declaration,.signature_start=b.signature_start,.signature_count=b.signature_count}).is_ok);i+=1;
    }
    i=1;while(i<=h::expression_count(ref p)){
        let h::Expression original=h::expression_at(ref p,i);
        let h::Type type=if(mutation==2 && h::expression_code(original.kind)==22){.Bool}else{original.type};
        var h::Expression e=h::Expression{.kind=original.kind,.type=type,.operator=original.operator,.start=original.start,.end=original.end,.left=original.left,.right=original.right,.alternative=original.alternative,.block=original.block,.binding=original.binding,.edge_start=original.edge_start,.edge_count=original.edge_count,.payload_start=original.payload_start,.payload_end=original.payload_end,.boolean=original.boolean};
        if(h::expression_code(e.kind)==22){
        if(mutation==1){e.left=0;}if(mutation==3){e.binding=0;}if(mutation==4){e.binding=2;}
        if(mutation==5){e.right=e.left;}if(mutation==6){e.boolean=true;}if(mutation==7){e.edge_count=1;}
    }assert(v::push<h::Expression>(ref mut expressions,e).is_ok);i+=1;}
    i=1;while(i<=h::statement_count(ref p)){assert(v::push<h::Statement>(ref mut statements,h::statement_at(ref p,i)).is_ok);i+=1;}
    i=1;while(i<=h::block_count(ref p)){assert(v::push<h::Block>(ref mut blocks,h::block_at(ref p,i)).is_ok);i+=1;}
    i=0;while(i<h::edge_count(ref p)){assert(v::push<usize>(ref mut edges,h::edge_at(ref p,i)).is_ok);i+=1;}
    i=0;while(i<h::signature_count(ref p)){assert(v::push<h::Signature>(ref mut signatures,h::Signature{.type=h::signature_at(ref p,i)}).is_ok);i+=1;}
    return h::assemble_nominal(source,bytes::empty(),functions,bindings,expressions,statements,blocks,edges,signatures,h::module_index(ref p),table);
}
@test
fn void propagation_hir_rejects_forged_operand_owner_and_success_type() {
    let good=switch(rebuilt(0)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var bytes::Buffer out=bytes::empty();assert(output::render(ref good,ref mut out)==0);
    var u32 mutation=1;while(mutation<=8){assert(switch(rebuilt(mutation)){case .ok(_):{false}case .err(p):{p.code==4}});mutation+=1;}
}
@test
fn void propagation_survives_hir_destruction_and_uses_only_verified_tagged_mir() {
    let input=fixture::retained(b"module main;fn Result<bool,bool> f(Result<i64,bool> r){let i64 n=r?;return .ok(n>0);}fn i32 main(){return switch(f(.err(true))){case .ok(_):{1}case .err(e):{if(e){2}else{3}}};}");
    var usize i=1;var usize count=0;
    while(i<=h::expression_count(ref input)){let e=h::expression_at(ref input,i);if(h::expression_code(e.kind)==22){assert(h::type_code(e.type)==2 && h::type_code(h::binding_at(ref input,e.binding).type)==tagged::make_result(6,6));count+=1;}i+=1;}assert(count==1);
    let p=switch(lower::lower(input)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);var usize constructs=0;var usize tests=0;var usize captures=0;i=1;
    while(i<=m::operation_count(ref p)){let o=m::operation_at(ref p,i);assert(o.kind<=23);if(o.kind==21){constructs+=1;}if(o.kind==22){tests+=1;}if(o.kind==23){captures+=1;}i+=1;}
    assert(constructs>=3 && tests>=2 && captures>=3);var bytes::Buffer out=bytes::empty();assert(c::render(ref p,ref mut out)==0);
}
fn bool stops(ref u8[] source) {
    let p=switch(lower::lower(fixture::retained(source))){case .ok(p):{p}case .err(_):{assert(false);return false;}};
    let plan=switch(flow::build(ref p)){case .ok(p):{p}case .err(_):{assert(false);return false;}};
    return flow::stops(ref plan,1);
}
@test
fn void literal_guard_traps_prune_nonreturning_call_continuations() {
    assert(stops(b"module main;fn i32 f(){return 2147483647+1;}"));
    assert(stops(b"module main;fn i64 f(){return -9223372036854775808-1;}"));
    assert(stops(b"module main;fn usize f(){return 18446744073709551615+1;}"));
    assert(stops(b"module main;fn u8 f(){return 0-1;}"));
    assert(stops(b"module main;fn i64 f(){return -9223372036854775808*-1;}"));
    assert(stops(b"module main;fn i32 f(){return -2147483648/-1;}"));
    assert(stops(b"module main;fn i64 f(){return 1<<64;}"));
    assert(stops(b"module main;fn i32 f(){return 2147483647<<1;}"));
    assert(!stops(b"module main;fn i64 f(){return -9223372036854775808+9223372036854775807;}"));
    assert(!stops(b"module main;fn usize f(){return 18446744073709551615-18446744073709551615;}"));
    assert(!stops(b"module main;fn i64 f(){return -9223372036854775808*0;}"));
    assert(!stops(b"module main;fn i32 f(i32 n){return n+1;}"));
    assert(stops(b"module main;fn i32 f(){let i32 n=2147483647;return n+1;}"));
    assert(!stops(b"module main;fn i32 f(){var i32 n=2147483647;return n+1;}"));
}
