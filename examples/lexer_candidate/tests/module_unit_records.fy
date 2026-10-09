module tests::module_unit_records;
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
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(_):{assert(false);return checked(b"module empty;");}};let bound=switch(binding::bind(sources,ref metadata,first,stop)){case .ok(b):{b}case .err(_):{assert(false);return checked(b"module empty;");}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return checked(b"module empty;");}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources,0,1);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}
@test fn void records_contextualize_fields_in_source_order(){
    let p=switch(checked(b"module main;struct Inner{i64 n;}struct Packet{Inner inner;usize size;bool valid;}fn Packet make(){let i64 n=4000000000;let packet=Packet{.valid=true,.inner=Inner{.n},.size=2};return packet;}fn i64 use(){return make().inner.n;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::unresolved(ref p)==0);let c=values::at(ref p,0);assert(model::checked_records(c)==2 && model::record_fields(c)==4);let local=model::local_at(c,2);assert(local.type!=0 && local.expected==0);var usize i=1;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==11){assert(e.state==2 && e.type==0 && e.field.id>0 && e.expected>0);}if(e.kind==10){assert(e.state==1 && e.type==e.target && e.place.root==0);}i+=1;}
}
@test fn void record_arrays_feed_indexes_views_calls_and_returns(){
    let p=switch(checked(b"module main;struct Token{i64 x;}fn i64 leaf(Token t){return t.x;}fn i64 main(){let Token[2] a=[Token{.x=4000000000},Token{.x=4000000001}];let ref Token[] view=ref a;return leaf(view[0])+view[1].x;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::checked_arguments(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,1);assert(model::checked_records(c)==2 && model::record_fields(c)==2);
}
@test fn void resources_and_empty_records_keep_owned_constructor_types(){
    let p=switch(checked(b"module main;struct Empty{}resource Item{i64 n;cleanup{}}fn Item make(){return Item{.n=4000000000};}fn void empty(){let item=Empty{};discard item;return;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::unresolved(ref p)==0);var usize count=0;var usize i=0;while(i<values::count(ref p)){count+=model::checked_records(values::at(ref p,i));i+=1;}assert(count==2);
}
@test fn void record_field_errors_keep_reference_codes_and_spans(){
    failure(b"module main;struct S{i64 n;bool b;}fn S main(){return S{.n=1};}",2,b"S{.n=1}");
    failure(b"module main;struct S{i64 n;}fn S main(){return S{.n=1,.n=2};}",2,b".n=2");
    failure(b"module main;struct S{i64 n;}fn S main(){return S{.other=1};}",2,b".other=1");
    failure(b"module main;struct S{i64 n;}fn S main(){return S{.n=true};}",1006,b"true");
    failure(b"module main;struct S{u8 n;}fn S main(){return S{.n=256};}",1013,b"256");
    failure(b"module main;struct A{i32 n;}struct B{i32 n;}fn void main(){let A a=B{.n=1};}",1006,b"B{.n=1}");
    failure(b"module main;struct A{i32 n;}struct B{i32 n;}fn A main(){return B{.n=1};}",1009,b"B{.n=1}");
    failure(b"module main;fn i32 value(){return 0;}fn i32 main(){let x=value{};return 0;}",2,b"value{}");
    failure(b"module main;struct S{i32 first;u8 second;}fn S main(){return S{.second=256,.first=true};}",1013,b"256");
}
@test fn void imported_records_preserve_provider_fields_and_privacy(){
    let ref u8[] main=b"module main;import provider as p;fn p::Packet main(){return p::Packet{.n=4000000000};}";var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(main)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub struct Packet{i64 n;}")).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};let c=values::at(ref p,0);assert(model::checked_records(c)==1 && model::record_fields(c)==1 && values::checked_returns(ref p)==1);var usize i=1;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==11){assert(e.field.module_index==1 && e.field.declaration==2 && e.field.source_index==0);}i+=1;}
    var v::Vec<syntax::Parsed> private=v::empty<syntax::Parsed>();let ref u8[] bad=b"module main;import provider as p;fn p::Item main(){return p::Item{.n=1};}";assert(v::push<syntax::Parsed>(ref mut private,parsed(bad)).is_ok);assert(v::push<syntax::Parsed>(ref mut private,parsed(b"module provider;pub resource Item{i32 n;cleanup{}}")).is_ok);switch(projected(ref private,0,1)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==103 && e.module_index==0 && bytes::equal(bad[e.start..e.end],b"p::Item"));}};
}
@test fn void generic_and_cleanup_records_use_provider_field_contexts(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(){return p::go<i64>(4000000000);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;struct Packet{i64 n;}resource Item{i32 n;cleanup{let Packet p=Packet{.n=4000000000};discard p;return;}}pub fn T go<T>(T n){let Packet p=Packet{.n};let Item item=Item{.n=1};discard item;return n;}" )).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::checked_returns(ref p)==3);var usize i=0;var usize count=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==10){assert(model::frame(c).module_index==1 && e.state==1 && e.type==e.target);let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(original.start==e.start && original.end==e.end);count+=1;}j+=1;}i+=1;}assert(count==3);
}

@test @effects(filesystem) fn void sdk_vector_records_use_owned_specialization_targets(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return;}};let sdk=switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import std::vec as v;fn usize main(){let v::Vec<i32> value=v::empty<i32>();return v::len<i32>(ref value);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref sdk))).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize i=0;var usize count=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==10){assert(model::frame(c).module_index==1 && e.state==1 && e.type==e.target && values::type_at(ref p,e.type).kind==13);assert(model::record_fields(c)==6);count+=1;}j+=1;}i+=1;}assert(count==1);
}
