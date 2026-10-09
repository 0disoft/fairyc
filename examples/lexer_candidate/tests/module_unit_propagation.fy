module tests::module_unit_propagation;
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
@test fn void success_context_does_not_change_result_operand_or_enclosing_success(){
    let p=switch(checked(b"module main;fn Result<bool,bool> f(Result<i64,bool> input){let i64 n=input?;return .ok(n>0);}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::checked_returns(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,0);var usize i=1;var usize count=0;
    while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==30){let operand=values::type_at(ref p,e.target);assert(e.state==1 && e.expected==e.type && operand.kind==6 && operand.first==e.type && values::type_at(ref p,e.type).scalar==2 && e.place.root==0);count+=1;}i+=1;}assert(count==1);
}
@test fn void nested_results_and_nominal_payloads_feed_locals_fields_and_arguments(){
    let p=switch(checked(b"module main;struct Packet{i64 n;}resource Token{i32 n;cleanup{}}fn i64 take(i64 n){return n;}fn Result<i64,bool> packet(Result<Packet,bool> input){let item=input?;return .ok(take(item.n));}fn Result<i64,bool> nested(Result<Result<i64,bool>,bool> input){let i64 n=(input?)?;return .ok(n);}fn Result<i32,bool> token(Result<Token,bool> input){let Token item=input?;return .ok(item.n);}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::checked_returns(ref p)==4 && values::checked_arguments(ref p)==1 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==30){assert(e.state==1 && e.target>0 && values::type_at(ref p,e.target).first==e.type);count+=1;}j+=1;}i+=1;}assert(count==4);
}
@test fn void provider_generic_propagation_keeps_canonical_error_and_source_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn Result<bool,p::Error> main(Result<i64,p::Error> input){return p::go<i64>(input);}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub struct Error{i32 code;}pub fn Result<bool,Error> go<T>(Result<T,Error> input){let T n=input?;return .ok(n>0);}")).is_ok);
    let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==30){assert(model::frame(c).kind==1 && model::frame(c).module_index==1 && e.state==1);let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(original.start==e.start && original.end==e.end);let error=values::type_at(ref p,values::type_at(ref p,e.target).second);assert(error.kind==2 && error.module_index==1 && error.declaration==2);count+=1;}j+=1;}i+=1;}assert(count==1);
}
@test fn void propagation_rejects_nonresults_and_different_error_types_at_question(){
    failure(b"module main;fn Result<i32,bool> f(i32 input){let n=input?;return .ok(n);}",4102,b"?");
    failure(b"module main;fn Result<i32,bool> f(Option<i32> input){let n=input?;return .ok(n);}",4102,b"?");
    failure(b"module main;fn i32 f(Result<i32,bool> input){return input?;}",4103,b"?");
    failure(b"module main;fn Result<i32,i32> f(Result<i32,bool> input){let n=input?;return .ok(n);}",4103,b"?");
    failure(b"module main;struct A{i32 code;}struct B{i32 code;}fn Result<i32,A> f(Result<i32,B> input){let n=input?;return .ok(n);}",4103,b"?");
    failure(b"module main;fn Result<i32,Result<i32,bool>> f(Result<i32,Result<bool,i32>> input){let n=input?;return .ok(n);}",4103,b"?");
    failure(b"module main;fn Result<i32,bool> f(Result<i64,bool> input){let i32 n=input?;return .ok(n);}",1006,b"input?");
    failure(b"module main;fn Result<i32,bool> f(Result<bool,bool> input){let n=input?;return .ok(n);}",1006,b"n");
    failure(b"module main;fn Result<i32,bool> f(){let i32 n=(.ok(1))?;return .ok(n);}",4102,b"?");
    failure(b"module main;resource Item{i32 n;cleanup{let Result<i32,bool> input=.ok(1);let i32 n=input?;discard n;return;}}fn void main(){let Item item=Item{.n=1};return;}",4103,b"?");
}
