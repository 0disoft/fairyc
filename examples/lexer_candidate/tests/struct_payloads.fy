module tests::struct_payloads;
import tests::hir_nominal as fixture;
import tagged_types as tagged;
import tagged_encoding as encoding;
import hir_model as h;
import hir_shapes as shapes;
import mir_model as m;
import mir_lower as lower;
import mir_verify as verify;
import mir_verify_enums as enums;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

@test
fn void nominal_leaf_identities_roundtrip_without_structural_aliases() {
    var u32 type=256;
    while(type<=767){
        let u32 o=tagged::make_option(type);let u32 left=tagged::make_result(type,6);
        let u32 right=tagged::make_result(6,type);let u32 both=tagged::make_result(type,type);
        assert(o!=0 && left!=right && left!=both && right!=both);
        assert(encoding::valid(o) && encoding::valid(both));
        assert(tagged::inner(o)==type && tagged::success(left)==type && tagged::failure(right)==type);
        assert(tagged::success(both)==type && tagged::failure(both)==type);
        if(type<767){assert(o!=tagged::make_option(type+1));}type+=1;
    }
    assert(tagged::make_option(255)==0 && tagged::make_option(768)==0);
    assert(tagged::make_result(tagged::make_option(257),258)==0);
    assert(!encoding::valid(1073741849)); // A marker-root nominal is not a tagged type.
}

@test
fn void owned_struct_payloads_keep_fields_capture_spans_and_verified_mir() {
    let p=fixture::retained(b"module main;struct Token{i64 value;}struct Problem{u32 code;usize start;usize end;}fn i32 read(Result<Token,Problem> r){return switch(r){case .ok(token):{assert(token.value==4000000000);7}case .err(problem):{assert(problem.code==8);3}};}fn i32 main(){return read(.ok(Token{.value=4000000000}));}");
    let ref shapes::Table table=h::nominal(ref p);assert(shapes::count(table)==3 && shapes::field_count(table)==8);
    let u32 result=tagged::make_result(257,258);let usize at=shapes::find(table,result);assert(at!=0);
    let layout=shapes::at(table,at-1);assert(shapes::field_at(table,layout.field_start+1).type==257);
    assert(shapes::field_at(table,layout.field_start+3).type==258);
    var usize i=1;var usize captures=0;
    while(i<=h::expression_count(ref p)){let e=h::expression_at(ref p,i);
        if(h::expression_code(e.kind)==21){let u32 type=h::type_code(e.type);
            assert(type==257 || type==258);assert(bytes::equal(h::source_view(ref p)[e.start..e.end],if(type==257){b"token"}else{b"problem"}));captures+=1;}
        i+=1;
    }assert(captures==2);
    let ir=switch(lower::lower(p)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(verify::check(ref ir).code==0);var bytes::Buffer out=bytes::empty();assert(c::render(ref ir,ref mut out)==0);
}

@test
fn void struct_payload_reads_reject_wrong_mir_fields_and_declarations() {
    let ir=switch(lower::lower(fixture::retained(b"module main;struct A{i32 n;}struct B{i32 n;}fn i32 main(){let Result<A,B> r=.ok(A{.n=7});return switch(r){case .ok(a):{a.n}case .err(b):{b.n}};}"))){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(verify::check(ref ir).code==0);var usize i=1;var usize count=0;
    while(i<=m::operation_count(ref ir)){let o=m::operation_at(ref ir,i);
        if(o.kind==23){assert(enums::operation(ref ir,o));var wrong=o;wrong.operator=0;assert(!enums::operation(ref ir,wrong));
            wrong=o;wrong.type=if(o.type==257){258}else{257};assert(!enums::operation(ref ir,wrong));
            wrong=o;wrong.owner_type=tagged::make_result(258,257);assert(!enums::operation(ref ir,wrong));count+=1;}
        i+=1;
    }assert(count==2);
}

@test
fn void missing_nominal_payload_layout_and_wrong_nominal_identity_are_rejected() {
    let p=fixture::retained(b"module main;struct A{i32 n;}struct B{i32 n;}fn Option<A> f(){return .some(A{.n=7});}");
    let ref shapes::Table original=h::nominal(ref p);assert(shapes::count(original)==3);
    var usize mutation=0;while(mutation<2){
        var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
        var usize i=if(mutation==0){1}else{0};while(i<3){let old=shapes::at(original,i);let usize start=v::len<shapes::Field>(ref fields);
            assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=old.type,.binding=old.binding,.field_start=start,.field_count=old.field_count,.public=old.public}).is_ok);
            var usize j=0;while(j<old.field_count){let f=shapes::field_at(original,old.field_start+j+1);
                assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=f.owner,.type=if(mutation==1 && i==2 && j==0){258}else{f.type},.start=f.start,.end=f.end,.payload_start=if(f.payload_count==0){0}else{start+j-f.payload_count},.payload_count=f.payload_count}).is_ok);j+=1;}
            i+=1;
        }
        let bad=shapes::assemble(records,fields);assert(!shapes::valid(ref bad,h::source_view(ref p).len,h::binding_count(ref p)));mutation+=1;
    }
}
