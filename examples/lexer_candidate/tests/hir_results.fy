module tests::hir_results;
import tests::hir_nominal as fixture;
import hir_model as h;
import hir_shapes as shapes;
import hir_output as output;
import tagged_types as tagged;
import std::bytes as bytes;
import std::vec as v;

@test
fn void results_own_distinct_payloads_captures_and_source_spans() {
    let ref u8[] source=b"module main;fn Result<i64,bool> make(){return .ok(4000000000);}fn Result<i64,bool> fail(){return .err(true);}fn i64 read(Result<i64,bool> r){return switch(r){case .ok(n):{n}case .err(e):{assert(e);0}};}";
    let p=fixture::retained(source); let ref shapes::Table table=h::nominal(ref p);
    assert(bytes::equal(h::source_view(ref p),source));
    assert(shapes::has_results(table) && !shapes::has_options(table) && shapes::count(table)==1 && shapes::field_count(table)==4);
    let u32 type=tagged::make_result(2,6);assert(shapes::at(table,0).type==type && shapes::at(table,0).binding==0);
    assert(shapes::field_at(table,1).type==2 && shapes::field_at(table,3).type==6);
    assert(shapes::member_at(table,type,0)==2 && shapes::member_at(table,type,1)==4);
    var usize id=1;var usize variants=0;var usize captures=0;
    while(id<=h::expression_count(ref p)){
        let e=h::expression_at(ref p,id);
        assert(e.start<=e.end && e.end<=source.len);
        if(h::expression_code(e.kind)==18){
            assert(switch(e.type){case .Result{.id=id}:{id==type}case .Invalid:{false}case .I32:{false}case .I64:{false}case .U8:{false}case .U32:{false}case .Usize:{false}case .Bool:{false}case .Void:{false}case .Callable:{false}case .Resource{.id=_}:{false}case .Nominal{.id=_}:{false}case .Array{.id=_}:{false}case .Text:{false}case .Bytes:{false}case .MutableBytes:{false}case .Enumeration{.id=_}:{false}case .Option{.id=_}:{false}});
            assert(e.binding==2 || e.binding==4);variants+=1;
        }
        if(h::expression_code(e.kind)==21){assert(e.binding==1 || e.binding==3);assert(h::type_code(e.type)==if(e.binding==1){2}else{6});captures+=1;}
        id+=1;
    }
    assert(variants==2 && captures==2);
    var bytes::Buffer first=bytes::empty();var bytes::Buffer second=bytes::empty();
    assert(output::render(ref p,ref mut first)==0 && output::render(ref p,ref mut second)==0);
    assert(bytes::equal(bytes::view(ref first),bytes::view(ref second)));
}
fn shapes::Table malformed(u32 mutation) {
    let u32 type=tagged::make_result(1,6);
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type,.binding=if(mutation==1){1}else{0},.field_start=0,.field_count=4,.public=mutation==2}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=if(mutation==3){2}else{1},.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=0,.start=0,.end=1,.payload_start=0,.payload_count=1}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=if(mutation==4){1}else{6},.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=if(mutation==5){262145}else{type},.type=0,.start=0,.end=1,.payload_start=if(mutation==6){0}else{2},.payload_count=if(mutation==7){0}else{1}}).is_ok);
    return shapes::assemble(records,fields);
}
@test
fn void result_layouts_reject_wrong_components_members_and_owners() {
    let good=malformed(0);assert(shapes::valid(ref good,1,1));var u32 mutation=1;
    while(mutation<=7){let p=malformed(mutation);assert(!shapes::valid(ref p,1,1));mutation+=1;}
    assert(!h::represented_type(.Enumeration{.id=262150}) && !h::represented_type(.Result{.id=196609}));
}
