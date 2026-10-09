module tests::module_unit_fields;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(_):{assert(false);return checked(b"module empty;");}};let bound=switch(binding::bind(sources,ref metadata,first,stop)){case .ok(b):{b}case .err(_):{assert(false);return checked(b"module empty;");}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return checked(b"module empty;");}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources,0,1);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}
@test fn void nested_fields_reach_inference_casts_and_wide_argument_peers(){
    let text=b"module main;struct Inner{u8 byte;i64 wide;}struct Outer{Inner inner;}fn i64 leaf(i64 n){return n;}fn i64 main(ref Outer s){let inferred=s.inner.wide;let widened=s.inner.byte as i64;return leaf(4000000000+inferred+widened);}";
    let source=parsed(text);let p=switch(checked(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::inferred(ref p)==2 && values::checked_arguments(ref p)==1);var usize count=0;var usize i=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);assert(model::checked_fields(c)==model::field_count(c));var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==8){assert(e.field.id>0 && e.field.owner>0 && e.field.module_index==0 && e.field.declaration>0);let f=syntax::field_at(ref source,e.field.source_index);assert(bytes::equal(text[f.name_start..f.name_end],syntax::expression_text(ref source,e.source_index)));count+=1;}j+=1;}i+=1;}assert(count==4);
}
@test fn void builtin_view_array_and_result_fields_keep_distinct_metadata(){
    let p=switch(checked(b"module main;fn usize lengths(ref str text,ref u8[] data,ref mut i64[] numbers,i32[4] array){let totals=text.len+data.len+numbers.len+array.len;return totals;}fn bool predicate(Result<i32,u8> result,ref str text,ref u8[] data){return result.is_ok || text.is_empty || data.is_empty;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::inferred(ref p)==1);var usize count=0;var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);assert(model::checked_fields(c)==model::field_count(c));var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==8){assert(e.state==1 && e.field.id==0 && e.field.owner>0 && e.field.declaration==0);count+=1;}j+=1;}i+=1;}assert(count==7);
}
@test fn void invalid_members_and_field_type_mismatches_keep_original_spans(){
    failure(b"module main;struct S{i64 wide;}fn i64 main(ref S s){return s.missing;}",2,b"s.missing");
    failure(b"module main;struct S{i64 wide;}fn i32 main(ref S s){return s.wide;}",1009,b"s.wide");
    failure(b"module main;fn bool main(i32[2] array){return array.is_empty;}",2,b"array.is_empty");
    failure(b"module main;fn usize main(ref CStr text){return text.len;}",2,b"text.len");
}
@test fn void imported_struct_fields_retain_provider_and_resources_remain_private(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(ref p::Record s){return 4000000000+s.wide;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub struct Record{i64 wide;}")).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==1);let c=values::at(ref p,0);var usize i=1;var bool found=false;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==8){assert(e.field.id>0 && e.field.module_index==1 && e.field.declaration==2 && e.field.source_index==0 && values::type_at(ref p,e.field.owner).module_index==1);found=true;}i+=1;}assert(found);
    discard v::pop<syntax::Parsed>(ref mut sources);discard v::pop<syntax::Parsed>(ref mut sources);let text=b"module main;import provider as p;fn u8 main(ref p::Item s){return s.secret;}";assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub resource Item{u8 secret;cleanup{}}")).is_ok);switch(projected(ref sources,0,1)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==103 && p.module_index==0 && bytes::equal(text[p.start..p.end],b"s.secret"));}};
}
@test fn void generic_and_cleanup_fields_keep_private_provider_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn u8 main(){return p::go<u8>(1);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{u8 n;cleanup{let widened=this.n as i64;discard widened;return;}}pub fn T go<T>(T n){let Item item=Item{.n=1};let widened=item.n as i64;discard widened;discard item;return n;}")).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize count=0;var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==8){assert(e.state==1 && e.field.id>0 && e.field.module_index==1 && e.field.declaration==2 && values::type_at(ref p,e.type).scalar==3);count+=1;}j+=1;}i+=1;}assert(count==2 && values::checked_returns(ref p)==3);
}
