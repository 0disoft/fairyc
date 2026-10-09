module tests::module_unit_builtins;
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

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let bound=switch(binding::bind(sources,ref metadata,first,stop)){case .ok(b):{b}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources,0,1);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}
@test fn void raw_views_preserve_safe_owner_and_lexical_unsafe_facts(){
    let p=switch(checked(b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn ref str text(ref Buffer value){return unsafe{str::from_raw_parts(value,value.data,value.length)};}fn ref u8[] bytes(ref Buffer value){return unsafe{byte_slice::from_raw_parts(value,value.data,value.length)};}fn ref mut u8[] mutable(ref mut Buffer value){return unsafe{byte_slice::from_raw_parts_mut(value,value.data,value.length)};}fn ref CStr cstr(ref Buffer value){return unsafe{CStr::from_raw_parts(value,value.data,value.length)};}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::checked_returns(ref p)==4 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==6 && e.builtin.kind!=0){assert(e.state==1 && e.builtin.kind>=9 && e.builtin.kind<=12 && e.builtin.unsafe_context && e.place.root==1 && e.place.kind==0 && e.place.indirect && !e.place.borrowable);assert(values::type_at(ref p,e.type).kind==if(e.builtin.kind==12){14}else{e.builtin.kind} && e.place.mutable==(e.builtin.kind==11));count+=1;}j+=1;}i+=1;}assert(count==4);
}
@test fn void inferred_raw_text_aliases_keep_the_original_parameter_root(){
    let p=switch(checked(b"module main;resource Buffer{ptr u8 data;usize length;cleanup{}}fn ref str text(ref Buffer value,ptr u8 data,usize length){let owner=value;let view=unsafe{str::from_raw_parts(owner,data,length)};return view;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::checked_returns(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,values::count(ref p)-1);assert(model::local_count(c)==5 && model::local_at(c,4).origin.root==1 && model::local_at(c,5).origin.root==1);var usize i=1;
    while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==6 && e.builtin.kind==9){assert(e.place.root==1 && e.place.scope==1);}i+=1;}
}
@test fn void source_location_has_static_text_type_and_feeds_local_arguments(){
    let p=switch(checked(b"module main;fn usize length(ref str text){return text.len;}fn ref str site(){return source_location();}fn usize main(){let location=source_location();return length(location);}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::checked_returns(ref p)==3 && values::checked_arguments(ref p)==1 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==6 && e.builtin.kind==200){assert(e.state==1 && values::type_at(ref p,e.type).kind==9 && e.place.root==0);count+=1;}j+=1;}i+=1;}assert(count==2);
}
@test fn void builtin_errors_preserve_arity_types_reserved_names_and_unsafe_boundaries(){
    failure(b"module main;resource Buffer{ptr u8 data;usize length;cleanup{}}fn ref str text(ref Buffer value){return str::from_raw_parts(value,value.data,value.length);}",5101,b"str::from_raw_parts(value,value.data,value.length)");
    failure(b"module main;resource Buffer{ptr u8 data;usize length;cleanup{}}fn ref str text(ref Buffer value){return unsafe{str::from_raw_parts(value,value.data)};}",1010,b"str::from_raw_parts(value,value.data)");
    failure(b"module main;resource Buffer{ptr u8 data;usize length;cleanup{}}fn ref str text(ref Buffer value){return unsafe{str::from_raw_parts(1,value.data,value.length)};}",2,b"1");
    failure(b"module main;resource Buffer{ptr u8 data;usize length;cleanup{}}fn ref str text(ref Buffer value){return unsafe{str::from_raw_parts(value,1,value.length)};}",2,b"1");
    failure(b"module main;resource Buffer{ptr u8 data;usize length;cleanup{}}fn ref str text(ref Buffer value){return unsafe{str::from_raw_parts(value,value.data,true)};}",1006,b"true");
    failure(b"module main;resource Buffer{ptr u8 data;usize length;cleanup{}}fn ref mut u8[] text(ref mut Buffer value){return unsafe{byte_slice::from_raw_parts_mut(value,value.data,value.length)};}",2,b"value.data");
    failure(b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn ref mut u8[] text(ref Buffer value){return unsafe{byte_slice::from_raw_parts_mut(value,value.data,value.length)};}",2,b"value");
    failure(b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn ref str text(ref mut Buffer value){return unsafe{str::from_raw_parts(value,value.data,value.length)};}",2,b"value");
    failure(b"module main;resource Buffer{ptr u8 data;usize length;cleanup{}}fn ref str text(ref Buffer value){let view=unsafe{str::from_raw_parts(value,value.data,value.length)};discard view;return str::from_raw_parts(value,value.data,value.length+1);}",5101,b"str::from_raw_parts(value,value.data,value.length+1)");
    failure(b"module main;fn ref str main(){return source_location(1);}",1010,b"source_location(1)");
    failure(b"module main;fn ref str main(){return source_location(.value=1);}",1014,b"source_location(.value=1)");
    failure(b"module main;fn ref str main(){return source_location;}",4401,b"source_location");
    failure(b"module main;fn i32 main(){let i32 source_location=1;return 0;}",4401,b"source_location");
    failure(b"module main;fn i32 main(i32 source_location){return 0;}",4401,b"source_location");
    failure(b"module main;fn ref str source_location(){return \"bad\";}",4401,b"source_location");
}
@test fn void provider_generic_raw_views_and_cleanup_location_keep_original_frames(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn ref u8[] main(ref i32 owner,ptr u8 data,usize length){return p::view<i32>(owner,data,length);}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{i32 n;cleanup{let location=source_location();discard location;return;}}pub fn ref u8[] view<T>(ref T owner,ptr u8 data,usize length){let Item item=Item{.n=1};discard item;return unsafe{byte_slice::from_raw_parts(owner,data,length)};}")).is_ok);
    let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==3 && values::checked_arguments(ref p)==3 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==6 && e.builtin.kind!=0){assert(model::frame(c).module_index==1 && e.state==1);assert(model::frame(c).kind==if(e.builtin.kind==200){2}else{1});let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(e.start==original.start && e.end==original.end);count+=1;}j+=1;}i+=1;}assert(count==2);
}
