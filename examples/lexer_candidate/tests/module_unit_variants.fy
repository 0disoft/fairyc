module tests::module_unit_variants;
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
@test fn void nested_tagged_constructors_keep_canonical_payload_contexts(){
    let p=switch(checked(b"module main;struct Packet{i64 n;}fn Option<i64> some(){return .some(4000000000);}fn Option<i64> none(){return .none;}fn Result<Option<i64>,bool> nested(){return .ok(.some(4000000001));}fn Result<Packet,bool> packet(){return .ok(Packet{.n=4000000002});}fn Result<Packet,bool> error(){return .err(false);}fn bool take(Option<i64> value){return true;}fn bool main(){return take(.some(4000000003));}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==7 && values::checked_arguments(ref p)==1 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==6 && e.target!=0){assert(e.state==1 && e.type==e.target && (values::type_at(ref p,e.type).kind==5 || values::type_at(ref p,e.type).kind==6) && e.place.root==0);count+=1;}if(e.kind==7 && e.expected!=0){assert(e.state==2 && e.type==0);}j+=1;}i+=1;}assert(count==6);
}
@test fn void enum_values_and_payloads_keep_source_order_and_field_provenance(){
    let p=switch(checked(b"module main;enum Code{Empty;More{i64 n;bool valid;};}fn Code main(){let i64 n=4000000000;let Code c=if(true){.More{.valid=true,.n}}else{.Empty};return c;}fn Code empty(){return .Empty{};}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==28 && e.state==1 || e.kind==10 && e.field.id!=0){assert(values::type_at(ref p,e.type).kind==4 && e.type==e.target && e.field.owner==e.type && e.field.module_index==0 && e.field.declaration==2 && e.place.root==0);count+=1;}if(e.kind==11){assert(e.state==2 && e.type==0 && e.expected!=0 && e.field.id!=0 && e.field.declaration==2);}j+=1;}i+=1;}assert(count==3);
}
@test fn void variant_errors_keep_reference_codes_and_spans(){
    failure(b"module main;fn Option<i32> main(){return .some(true);}",1006,b"true");
    failure(b"module main;fn Result<i32,bool> main(){return .err(1);}",1006,b"1");
    failure(b"module main;fn Option<u8> main(){return .some(256);}",1013,b"256");
    failure(b"module main;fn Option<i32> main(){return .some();}",2,b".some()");
    failure(b"module main;fn Option<i32> main(){return .some(1,2);}",2,b".some(1,2)");
    failure(b"module main;fn Option<i32> main(){return .none();}",2,b".none()");
    failure(b"module main;fn Result<i32,bool> main(){return .ok;}",2,b".ok");
    failure(b"module main;fn void main(){let value=.none;return;}",2,b".none");
    failure(b"module main;fn void main(){let value=.some(1);return;}",2,b".some(1)");
    failure(b"module main;fn Option<i32> main(){return .some(.value=1);}",1014,b".some(.value=1)");
    failure(b"module main;enum Code{Empty;More{i64 n;};}fn Code main(){return .Missing;}",2,b".Missing");
    failure(b"module main;enum Code{Empty;More{i64 n;};}fn Code main(){return .More;}",2,b".More");
    failure(b"module main;enum Code{Empty;More{i64 n;};}fn Code main(){return .More{};}",2,b".More{}");
    failure(b"module main;enum Code{Empty;More{i64 n;};}fn Code main(){return .More{.other=1};}",2,b".other=1");
    failure(b"module main;enum Code{Empty;More{i64 n;};}fn Code main(){return .More{.n=1,.n=2};}",2,b".n=2");
    failure(b"module main;enum Code{Empty;More{i64 n;};}fn Code main(){return .More{.n=true};}",1006,b"true");
    failure(b"module main;enum Code{Empty;More{u8 n;};}fn Code main(){return .More{.n=256};}",1013,b"256");
}
@test fn void imported_enum_values_use_original_provider_field_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn p::Code main(){return .More{.n=4000000000};}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub enum Code{Empty;More{i64 n;};}")).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,0);var usize i=1;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==10 || e.kind==11){assert(e.field.module_index==1 && e.field.declaration==2);let f=syntax::field_at(v::at<syntax::Parsed>(ref sources,1),e.field.source_index);if(e.kind==11){assert(bytes::equal(syntax::source_view(v::at<syntax::Parsed>(ref sources,1))[f.name_start..f.name_end],b"n"));}}i+=1;}
}
@test fn void generic_and_cleanup_constructors_keep_context_and_provider_ids(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn Result<i64,bool> main(){return p::go<i64>(4000000000);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;enum Code{Empty;More{i64 n;};}resource Item{i32 n;cleanup{let Code c=.More{.n=4000000000};discard c;return;}}pub fn Result<T,bool> go<T>(T n){let Item item=Item{.n=1};discard item;return .ok(n);}" )).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::checked_returns(ref p)==3 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if((e.kind==6 || e.kind==10) && e.target!=0 && (values::type_at(ref p,e.type).kind==4 || values::type_at(ref p,e.type).kind==6)){assert(model::frame(c).module_index==1 && e.state==1);let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(original.start==e.start && original.end==e.end);count+=1;}j+=1;}i+=1;}assert(count==2);
}
