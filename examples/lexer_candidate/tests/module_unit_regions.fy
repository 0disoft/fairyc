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
    failure(b"module main;fn i32 f(){region scratch in 3{discard 4;}return 1;}",2,b"scratch");
    failure(b"module main;fn i32 f(ref Region scratch){return 1;}",9,b"ref Region");
    failure(b"module main;fn i32 f(Option<Region> scratch){return 1;}",9,b"Option<Region>");
    failure(b"module main;fn i32 f(Result<i32,Region> scratch){return 1;}",9,b"Result<i32,Region>");
    failure(b"module main;fn i32 f(Region[2] scratch){return 1;}",9,b"Region[2]");
}
