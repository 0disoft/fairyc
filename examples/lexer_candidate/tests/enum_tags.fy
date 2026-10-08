module tests::enum_tags;
import tagged_encoding as encoding;
import tagged_types as tags;
import reference_types as refs;
import hir_shapes as shapes;
import std::vec as v;

@test fn void enum_leaves_preserve_identity_and_closed_bounds(){
    var u32 index=0;
    while(index<512){
        let u32 leaf=131072+index;
        let u32 option=tags::make_option(leaf);
        let u32 result=tags::make_result(2,leaf);
        assert(encoding::valid(option) && encoding::valid(result));
        assert(tags::inner(option)==leaf && tags::success(result)==2 && tags::failure(result)==leaf);
        assert(option!=tags::make_option(256+index) && option!=tags::make_option(98304+index));
        assert(tags::make_result(leaf,leaf)==0); // 16+16+2 exceeds the unchanged 30-bit budget.
        index+=1;
    }
    assert(tags::make_option(131584)==0);
    var u32 scalar=1;while(scalar<=6){
        let u32 reference=refs::make(scalar,false);
        let u32 option=tags::make_option(reference);
        assert(tags::inner(option)==reference && option!=tags::make_option(131072+scalar-1));
        scalar+=1;
    }
    assert(!encoding::valid(1073741824|2|(512<<7)<<2));
    assert(!encoding::valid(1073741824|2|(4|16|6<<5)<<2));
}

fn shapes::Table layout(bool missing_enum,bool wrong_payload){
    let u32 leaf=131073;let u32 result=tags::make_result(2,leaf);
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();
    var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    if(!missing_enum){
        assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=leaf,.binding=1,.field_start=0,.field_count=1,.public=true}).is_ok);
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=leaf,.type=0,.start=1,.end=2,.payload_start=0,.payload_count=0}).is_ok);
    }
    let usize offset=v::len<shapes::Field>(ref fields);
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=result,.binding=0,.field_start=offset,.field_count=4,.public=false}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=result,.type=2,.start=2,.end=3,.payload_start=0,.payload_count=0}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=result,.type=0,.start=2,.end=3,.payload_start=offset,.payload_count=1}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=result,.type=if(wrong_payload){131074}else{leaf},.start=2,.end=3,.payload_start=0,.payload_count=0}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=result,.type=0,.start=2,.end=3,.payload_start=offset+2,.payload_count=1}).is_ok);
    return shapes::assemble(records,fields);
}
@test fn void enum_payloads_need_the_exact_retained_layout(){
    let p=layout(false,false);assert(shapes::valid(ref p,3,1));
    let missing=layout(true,false);assert(!shapes::valid(ref missing,3,1));
    let wrong=layout(false,true);assert(!shapes::valid(ref wrong,3,1));
}
