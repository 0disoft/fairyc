module tests::module_unit_interfaces;
import module_unit_interfaces as interfaces;
import module_unit_bind as binding;
import module_signatures as signatures;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn interfaces::Interfaces retained(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;import provider as q;@effects(console)fn i32 main(){var p::Owner owner=p::make();discard p::maybe();let i32 n=p::echo<i32>(7);discard unsafe{p::c_value(0)};return unsafe{q::work(ref mut owner,.Off)};}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub struct Leaf{i64 wide;}pub struct Box{Leaf[2] values;}pub resource Owner{Box item;cleanup{}}pub enum Kind{Off;Pair{i32 n;};}@repr(C)pub struct Packet{u32 code;}pub fn Owner make(){return Owner{.item=Box{.values=[Leaf{.wide=7};2]}};}@must_use @effects(console)pub unsafe fn i32 work(ref mut Owner owner,Kind kind){return 7;}pub fn Result<Box,i32> maybe(){return .ok(Box{.values=[Leaf{.wide=7};2]});}@link_name(\"fixture_packet\")pub unsafe extern C fn Packet c_value(u32 n);pub fn U echo<U>(U value){return value;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return retained();}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return retained();}};
    return switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);retained()}};
}
@test fn void owned_provider_fields_signatures_and_templates_survive_source_removal(){
    let p=retained();assert(interfaces::shape_count(ref p)==5 && interfaces::callable_count(ref p)==5 && interfaces::field_count(ref p)==7 && interfaces::work(ref p)<1048576);
    var usize structs=0;var usize resources=0;var usize enums=0;var usize reprs=0;var usize i=0;
    while(i<interfaces::shape_count(ref p)){let shape=interfaces::shape_at(ref p,i);let type=interfaces::type_at(ref p,shape.type);assert(shape.module_index==1 && type.module_index==1 && shape.declaration==type.declaration);
        if(type.kind==2){structs+=1;}else if(type.kind==3){resources+=1;assert(shape.abilities==1 && !shape.cleanup_nonempty && shape.cleanup_effects==0);}else if(type.kind==4){enums+=1;}
        if(shape.repr_c){reprs+=1;assert(type.kind==2 && shape.field_count==1 && bytes::equal(interfaces::field_name(ref p,shape.field_start),b"code"));}
        if(type.kind==4){assert(shape.field_count==3);let member=interfaces::field_at(ref p,shape.field_start+2);assert(member.kind==3 && member.payload_count==1 && member.payload_start==shape.field_start+1 && bytes::equal(interfaces::field_name(ref p,shape.field_start+2),b"Pair"));}
        i+=1;
    }assert(structs==3 && resources==1 && enums==1 && reprs==1);
    var usize arrays=0;var usize wide=0;i=0;while(i<interfaces::field_count(ref p)){let f=interfaces::field_at(ref p,i);if(f.type!=0){let t=interfaces::type_at(ref p,f.type);if(t.kind==16){arrays+=1;assert(t.second==2 && interfaces::type_at(ref p,t.first).kind==2);}if(t.kind==1 && t.scalar==2){wide+=1;}}i+=1;}assert(arrays==1 && wide==1);
    var usize effect_calls=0;var usize foreign=0;var usize templates=0;var usize tagged=0;i=0;
    while(i<interfaces::callable_count(ref p)){let c=interfaces::callable_at(ref p,i);assert(c.module_index==1 && c.declaration!=0);
        if(c.effects!=0){effect_calls+=1;assert(c.effects==1 && c.must_use && c.unsafe_call && !c.foreign && c.parameter_count==2);let borrow=interfaces::type_at(ref p,interfaces::parameter_at(ref p,c.parameter_start));assert(borrow.kind==8 && interfaces::type_at(ref p,borrow.first).kind==3);}
        if(c.foreign){foreign+=1;assert(c.unsafe_call && !c.template && bytes::equal(interfaces::linker_name(ref p,i),b"fixture_packet"));}
        if(c.template){templates+=1;assert(c.result==0 && c.parameter_count==0);}else if(interfaces::type_at(ref p,c.result).kind==6){tagged+=1;}
        i+=1;
    }assert(effect_calls==1 && foreign==1 && templates==1 && tagged==1);
}
@test fn void retained_bindings_recheck_changed_callers_and_private_provider_metadata(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::value();}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn i32 value(){return 7;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var v::Vec<syntax::Parsed> changed=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut changed,parsed(b"module main;import provider as p;fn i32 main(){return p::value();}")).is_ok);assert(v::push<syntax::Parsed>(ref mut changed,parsed(b"module provider;fn i32 value(){return 7;}")).is_ok);
    switch(interfaces::build(ref changed,ref metadata,ref bound)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12 && p.module_index==1);}};
    let fresh=switch(signatures::build(ref changed)){case .ok(p):{p}case .err(_):{assert(false);return;}};switch(interfaces::build(ref changed,ref fresh,ref bound)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==6 && p.module_index==0);}};
    var v::Vec<syntax::Parsed> caller=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut caller,parsed(b"module main;import provider as p;fn i32 main(){return p::value()+1;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut caller,parsed(b"module provider;pub fn i32 value(){return 7;}")).is_ok);let current=switch(signatures::build(ref caller)){case .ok(p):{p}case .err(_):{assert(false);return;}};switch(interfaces::build(ref caller,ref current,ref bound)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12 && p.module_index==0);}};
}
@test fn void provider_field_queries_return_canonical_types_and_keep_resource_privacy(){
    let p=retained();var usize found=0;var usize i=0;
    while(i<interfaces::shape_count(ref p)){let s=interfaces::shape_at(ref p,i);let t=interfaces::type_at(ref p,s.type);if(t.kind==3){
        switch(interfaces::field(ref p,s.type,0,b"item")){case .ok(_):{assert(false);}case .err(code):{assert(code==6201);}};
        let usize id=switch(interfaces::field(ref p,s.type,1,b"item")){case .ok(id):{id}case .err(_):{assert(false);return;}};let f=interfaces::field_at(ref p,id-1);assert(interfaces::type_at(ref p,f.type).kind==2);found+=1;
    }else if(t.kind==2){switch(interfaces::field(ref p,s.type,0,b"wide")){case .ok(id):{let f=interfaces::field_at(ref p,id-1);assert(interfaces::type_at(ref p,f.type).scalar==2);found+=1;}case .err(code):{assert(code==18);}};}else if(t.kind==4){switch(interfaces::field(ref p,s.type,1,b"Pair")){case .ok(_):{assert(false);}case .err(code):{assert(code==18);}};}i+=1;}
    assert(found==2);switch(interfaces::field(ref p,0,0,b"missing")){case .ok(_):{assert(false);}case .err(code):{assert(code==18);}};
}
