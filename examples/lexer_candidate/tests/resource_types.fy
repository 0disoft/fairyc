module tests::resource_types;
import resource_identity as resources;
import tagged_types as tagged;
import tagged_encoding as encoding;
import tests::hir_nominal as fixture;
import tests::result_types as types;
import type_check as typing;
import hir_model as h;
import hir_shapes as shapes;
import hir_output as output;
import mir_lower as lower;
import mir_model as m;
import std::bytes as bytes;
import std::vec as v;

@test
fn void resource_identity_stays_disjoint_and_tagged_payloads_keep_category() {
    var u32 i=0;while(i<512){let u32 r=98304+i;let u32 s=256+i;
        let u32 option=tagged::make_option(r);let u32 result=tagged::make_result(r,1);
        assert(resources::valid(r) && encoding::valid(option) && encoding::valid(result));
        assert(tagged::inner(option)==r && tagged::success(result)==r && tagged::failure(result)==1);
        assert(option!=tagged::make_option(s) && result!=tagged::make_result(s,1));
        assert(h::represented_type(.Resource{.id=r}) && !h::represented_type(.Nominal{.id=r}));i+=1;
    }
    assert(!resources::valid(98303) && !resources::valid(98816));
    assert(tagged::make_option(1)==196609 && tagged::make_result(1,6)==262150);
}

@test
fn void owned_resource_fields_and_properties_survive_source_owner() {
    let source=b"module main;struct Pair{i64 n;}resource R{i64 n;cleanup{}}resource Other{bool ready;cleanup{}}pub resource Outer{R owned;Option<Other> optional;Result<R,i32> result;Pair scalar;cleanup{}}fn R relay(R x){return x;}fn i32 main(){return 7;}";
    let p=fixture::retained(source);let ref shapes::Table table=h::nominal(ref p);
    let u32 r=98306;let u32 other=98307;let u32 outer=98308;
    assert(shapes::has_resources(table) && shapes::find(table,r)!=0 && shapes::find(table,other)!=0);
    assert(shapes::at(table,shapes::find(table,outer)-1).public);
    assert(shapes::needs_drop(r) && !shapes::copyable(table,r) && shapes::copyable(table,257));
    let shape=shapes::at(table,shapes::find(table,outer)-1);
    assert(shape.field_count==4 && shapes::field_at(table,shape.field_start+1).type==r);
    assert(shapes::field_at(table,shape.field_start+2).type==tagged::make_option(other));
    assert(shapes::field_at(table,shape.field_start+3).type==tagged::make_result(r,1));
    var usize i=1;while(i<=3){assert(shapes::field_owns_resource(table,shape.field_start+i));i+=1;}
    assert(!shapes::field_owns_resource(table,shape.field_start+4));
    let scalar=shapes::field_at(table,shape.field_start+4);assert(bytes::equal(h::source_view(ref p)[scalar.start..scalar.end],b"scalar"));
    var bytes::Buffer rendered=bytes::empty();assert(output::render(ref p,ref mut rendered)==0);
    assert(bytes::equal(bytes::view(ref rendered)[..23],b"fairyc.resource-hir/v1\n"));
    assert(switch(lower::lower(p)){case .ok(_):{false}case .err(problem):{problem.code==4}});
}

@test
fn void resource_identity_and_forbidden_storage_are_checked_in_source() {
    let mismatch=typing::check(types::resolved(b"module main;resource A{i32 n;cleanup{}}resource B{i32 n;cleanup{}}fn A wrong(B x){return x;}"));
    assert(typing::problem(ref mismatch).code==1);
    let direct=typing::check(types::resolved(b"module main;resource R{cleanup{}}struct Bad{R owner;}fn i32 main(){return 0;}"));
    assert(typing::problem(ref direct).code==43);
    let hidden=typing::check(types::resolved(b"module main;resource R{cleanup{}}struct Bad{Option<R> owner;}fn i32 main(){return 0;}"));
    assert(typing::problem(ref hidden).code==43);
    let cycle=typing::check(types::resolved(b"module main;resource R{Option<R> next;cleanup{}}fn i32 main(){return 0;}"));
    assert(typing::problem(ref cycle).code==20);
}

fn shapes::Table altered(ref shapes::Table table,u32 mutation) {
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    var usize i=0;while(i<shapes::count(table)){var s=shapes::at(table,i);
        if(mutation==1 && s.type==98306){s.type=700;}
        assert(v::push<shapes::Shape>(ref mut records,s).is_ok);i+=1;
    }
    i=1;while(i<=shapes::field_count(table)){var f=shapes::field_at(table,i);
        if(mutation==1 && f.owner==98306){f.owner=700;}
        if(mutation==2 && f.owner==98306){f.type=98815;}
        if(mutation==3 && f.owner==98305){f.type=98305;}
        if(mutation==4 && f.owner==98306){f.owner=98305;}
        assert(v::push<shapes::Field>(ref mut fields,f).is_ok);i+=1;
    }return shapes::assemble(records,fields);
}

@test
fn void owned_layout_verifier_rejects_erased_resource_ownership_missing_children_and_cycles() {
    let p=fixture::retained(b"module main;resource R{i32 n;cleanup{}}resource Box{R owned;cleanup{}}fn i32 main(){return 7;}");
    let ref shapes::Table table=h::nominal(ref p);assert(shapes::valid(table,h::source_view(ref p).len,h::binding_count(ref p)));
    var u32 mutation=1;while(mutation<=4){let bad=altered(table,mutation);
        assert(!shapes::valid(ref bad,h::source_view(ref p).len,h::binding_count(ref p)));mutation+=1;
    }
}

@test
fn void mir_factory_refuses_resource_tables_before_copy_only_backend() {
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=98305,.binding=1,.field_start=0,.field_count=0,.public=false}).is_ok);
    let table=shapes::assemble(records,v::empty<shapes::Field>());assert(shapes::valid(ref table,5,1152));
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"owned").is_ok);
    assert(switch(m::assemble_nominal(source,bytes::empty(),v::empty<m::Target>(),v::empty<m::Signature>(),v::empty<m::Function>(),v::empty<m::Local>(),v::empty<m::Block>(),v::empty<m::Operation>(),v::empty<usize>(),0,table)){case .ok(_):{false}case .err(problem):{problem.code==4}});
}
