module tests::nested_tagged;
import tagged_types as tagged;
import tagged_encoding as encoding;
import tests::hir_nominal as fixture;
import tests::result_types as types;
import type_check as typing;
import hir_model as h;
import hir_shapes as shapes;
import mir_lower as lower;
import mir_verify as verify;
import mir_model as m;
import c_emit as c;
import std::vec as v;
import std::bytes as bytes;

@test
fn void nested_identity_roundtrips_and_rejects_noncanonical_encodings() {
    var u32 leaf=1;while(leaf<=6){let u32 option=tagged::make_option(leaf);let u32 result=tagged::make_result(leaf,6);
        let u32 oo=tagged::make_option(option);let u32 or=tagged::make_option(result);
        let u32 ro=tagged::make_result(option,6);let u32 rr=tagged::make_result(result,6);
        assert(oo!=or && oo!=ro && oo!=rr && or!=ro && or!=rr && ro!=rr);
        assert(tagged::inner(oo)==option && tagged::inner(or)==result);
        assert(tagged::success(ro)==option && tagged::success(rr)==result && tagged::failure(rr)==6);
        assert(encoding::valid(oo) && encoding::valid(or) && encoding::valid(ro) && encoding::valid(rr));leaf+=1;
    }
    assert(!encoding::valid(1073741830)); // Marker alias of scalar Option<i32>.
    assert(!encoding::valid(1073741833)); // A scalar marker is not a tagged type.
    assert(tagged::make_result(7,1)==0 && tagged::make_option(0)==0);
    var u32 deep=1;var usize i=0;while(i<12){deep=tagged::make_option(deep);assert(deep!=0);i+=1;}
    assert(tagged::make_option(deep)==0);
}
@test
fn void nested_owned_layouts_survive_source_and_lower_to_verified_mir() {
    let p=fixture::retained(b"module main;fn i32 f(Result<Option<i64>,bool> r){return switch(r){case .ok(o):{switch(o){case .some(n):{assert(n==4000000000);7}case .none:{2}}}case .err(_):{3}};}fn i32 main(){return f(.ok(.some(4000000000)));}");
    let u32 option=tagged::make_option(2);let u32 result=tagged::make_result(option,6);
    let ref shapes::Table table=h::nominal(ref p);assert(shapes::count(table)==2 && shapes::field_count(table)==7);
    assert(shapes::at(table,0).type==option && shapes::at(table,1).type==result);
    assert(shapes::field_at(table,4).type==option);
    let ir=switch(lower::lower(p)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(verify::check(ref ir).code==0);var usize i=1;var bool found=false;
    while(i<=m::operation_count(ref ir)){let o=m::operation_at(ref ir,i);if(o.kind==23 && o.type==option){found=true;}i+=1;}assert(found);
    var bytes::Buffer output=bytes::empty();assert(c::render(ref ir,ref mut output)==0);
}
@test
fn void nested_capture_retains_its_must_use_obligation() {
    let rejected=typing::check(types::resolved(b"module main;fn i32 f(Option<Result<i32,bool>> r){return switch(r){case .some(inner):{7}case .none:{0}};}"));
    let problem=typing::problem(ref rejected);assert(problem.code==42);
    assert(bytes::equal(typing::source_view(ref rejected)[problem.start..problem.end],b"inner"));
    let accepted=typing::check(types::resolved(b"module main;fn i32 f(Option<Result<i32,bool>> r){return switch(r){case .some(inner):{discard inner;7}case .none:{0}};}"));
    assert(typing::problem(ref accepted).code==0);
}
@test
fn void nested_missing_child_layout_and_wrong_payload_are_rejected() {
    let p=fixture::retained(b"module main;fn Option<Option<i32>> f(){return .some(.none);}");
    let ref shapes::Table original=h::nominal(ref p);assert(shapes::count(original)==2);
    var usize mutation=0;while(mutation<2){var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
        var usize i=if(mutation==0){1}else{0};while(i<2){let old=shapes::at(original,i);let usize start=v::len<shapes::Field>(ref fields);
            assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=old.type,.binding=0,.field_start=start,.field_count=old.field_count,.public=false}).is_ok);
            var usize j=0;while(j<old.field_count){let f=shapes::field_at(original,old.field_start+j+1);
                assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=f.owner,.type=if(mutation==1 && i==1 && j==0){old.type}else{f.type},.start=f.start,.end=f.end,.payload_start=if(j==1){start}else{0},.payload_count=f.payload_count}).is_ok);j+=1;}i+=1;
        }
        let bad=shapes::assemble(records,fields);assert(!shapes::valid(ref bad,h::source_view(ref p).len,h::binding_count(ref p)));mutation+=1;
    }
}
