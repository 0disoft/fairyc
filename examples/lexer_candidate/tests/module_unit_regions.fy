module tests::module_unit_regions;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import file_source as files;
import module_types as types;
import module_unit_value_storage as storage;
import module_unit_interfaces as interfaces;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let bound=switch(binding::bind(sources,ref metadata,first,stop)){case .ok(b):{b}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources,0,1);}
@effects(console) fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{discard io::write_u32(p.code);discard io::write(" ");discard io::write_usize(p.start);discard io::write(" ");discard io::write_usize(p.end);discard io::write("\n");assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}


@test @effects(console) fn void region_parameters_remain_canonical_across_call_units(){
    let text=b"module main;import provider as p;fn i32 local(Region scratch){return 1;}fn i32 f(Region scratch){discard local(scratch);return p::get(scratch);}";
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn i32 get(Region scratch){return 7;}")).is_ok);
    let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(p):{discard io::write_u32(p.code);discard io::write(" ");discard io::write_usize(p.start);discard io::write("\n");assert(false);return;}};assert(values::checked_calls(ref p)==2 && values::checked_arguments(ref p)==2);var usize capability_id=0;var usize found=0;var usize i=0;while(i<values::count(ref p)){let body=values::at(ref p,i);var usize j=1;while(j<=model::local_count(body)){let l=model::local_at(body,j);if(values::type_at(ref p,l.type).kind==19){assert(l.kind==1);if(capability_id==0){capability_id=l.type;}else{assert(l.type==capability_id);}found+=1;}j+=1;}i+=1;}assert(found==2);
}
@test @effects(console) fn void region_capabilities_cannot_be_manufactured_or_stored_as_values(){
    failure(b"module main;fn Region f(Region scratch){return scratch;}",3103,b"Region");
    failure(b"module main;fn i32 f(Region scratch){let Region copy=scratch;return 1;}",3103,b"copy");
    failure(b"module main;fn i32 f(Region scratch){var Region copy=scratch;return 1;}",3103,b"copy");
    failure(b"module main;fn i32 f(Region scratch){let copy=scratch;return 1;}",3103,b"copy");
    failure(b"module main;struct S{Region scratch;}fn i32 f(){return 1;}",3103,b"Region");
    failure(b"module main;enum S{Some{Region scratch;};}fn i32 f(){return 1;}",3103,b"Region");
    failure(b"module main;resource S{Region scratch;cleanup{}}fn i32 f(){return 1;}",3103,b"Region");
    failure(b"module main;fn i32 f(){region scratch in 3{discard 4;}return 1;}",2,b"3");
    failure(b"module main;fn i32 f(ref Region scratch){return 1;}",9,b"ref Region");
    failure(b"module main;fn i32 f(Option<Region> scratch){return 1;}",9,b"Option<Region>");
    failure(b"module main;fn i32 f(Result<i32,Region> scratch){return 1;}",9,b"Result<i32,Region>");
    failure(b"module main;fn i32 f(Region[2] scratch){return 1;}",9,b"Region[2]");
}

@effects(filesystem) fn syntax::Parsed allocator(){var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return allocator();}};let data=switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);return allocator();}};return parsed(bytes::view(ref data));}
@effects(filesystem,console) fn void source_case(usize index,ref u8[] text,u32 expected){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,allocator()).is_ok);var model::Problem result=model::Problem{.code=0,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};var usize regions=0;
    switch(projected(ref sources,0,1)){case .ok(p):{var usize i=0;while(i<values::count(ref p)){let body=values::at(ref p,i);var usize j=1;while(j<=model::local_count(body)){let local=model::local_at(body,j);if(local.kind==6){let t=values::type_at(ref p,local.type);assert(t.kind==19 && t.scalar==0 && t.first==0 && t.second==0);regions+=1;}j+=1;}j=0;while(j<model::consumer_count(body)){let consumer=model::consumer_at(body,j);if(consumer.kind==17){assert(consumer.checked && values::type_at(ref p,consumer.actual).kind==8);}j+=1;}i+=1;}assert(regions>0);}case .err(p):{result=p;}};
    discard io::write("sourcecase ");discard io::write_usize(index);discard io::write(" ");discard io::write_u32(result.code);discard io::write(" ");discard io::write_usize(result.module_index);discard io::write(" ");discard io::write_usize(result.start);discard io::write(" ");discard io::write_usize(result.end);discard io::write(" ");discard io::write_usize(regions);discard io::write("\n");assert(result.code==expected);
}
@test @effects(filesystem,console) fn void lexical_region_sources_match_authenticated_arena(){
    source_case(0,b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a){region r in a{discard r.place(3);}}",0);
    source_case(1,b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a){region r in a{}}",0);
    source_case(2,b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a){region r in a{discard r.place(3);}region s in a{discard s.place(4);}}",0);
    source_case(3,b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){region r in a{discard r.place(3);region s in b{discard s.place(4);}}}",0);
    source_case(4,b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a){region r in (a){discard r.place(3);}}",0);
    source_case(5,b"module main;import std::alloc as alloc;fn ref mut alloc::Arena source(ref mut alloc::Arena a){return a;}fn void f(ref mut alloc::Arena a){region r in source(a){discard r.place(3);}}",0);
    source_case(6,b"module main;import std::alloc as alloc;fn void use(Region r){discard r.place(3);}fn void f(ref mut alloc::Arena a){region r in a{use(r);}}",0);
    source_case(7,b"module main;import std::alloc as alloc;fn void use<T>(ref mut alloc::Arena a,T x){region r in a{discard r.place(x);}}fn void f(ref mut alloc::Arena a){use<i32>(a,3);}",0);
    source_case(8,b"module main;import std::alloc as alloc;fn ref i32 f(ref mut alloc::Arena a,ref i32 x){region r in a{discard r.place(3);return x;}}",0);
    source_case(9,b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,bool c){while(c){region r in a{discard r.place(3);break;}}}",0);
    source_case(10,b"module main;import std::alloc as alloc;fn void f(){region r in 3{}}",2);
    source_case(11,b"module main;import std::alloc as alloc;fn void f(ref alloc::Arena a){region r in a{}}",2);
    source_case(12,b"module main;import std::alloc as alloc;resource Arena{cleanup{}}fn void f(ref mut Arena a){region r in a{}}",2);
    source_case(13,b"module main;import std::alloc as alloc;fn void f(alloc::Arena a){region r in a{}}",2);
}
