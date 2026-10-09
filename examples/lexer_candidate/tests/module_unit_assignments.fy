module tests::module_unit_assignments;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import module_types as types;
import std::io as io;
import type_control as control;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let bound=switch(binding::bind(sources,ref metadata,0,1)){case .ok(b):{b}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources);}
@effects(console) fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{discard io::write("unexpected success\n");assert(false);}case .err(p):{if(p.code!=code || p.module_index!=0 || !bytes::equal(text[p.start..p.end],at)){discard io::write_u32(p.code);discard io::write(" at ");discard io::write_bytes(text[p.start..p.end]);discard io::write("\n");assert(false);}}};}
fn void assignments(ref values::Plan p,usize count){var usize total=0;var usize i=0;while(i<values::count(p)){let c=values::at(p,i);assert(model::assignment_count(c)==model::checked_assignments(c));var usize j=0;while(j<model::consumer_count(c)){let a=model::consumer_at(c,j);if(a.kind==4){assert(a.checked && a.target!=0 && a.expected==a.actual);let t=model::expression_at(c,a.target);let e=model::expression_at(c,a.expression);assert(a.expected==t.type && a.actual==e.type && t.place.mutable && t.place.kind!=0);total+=1;}j+=1;}i+=1;}assert(total==count && values::unresolved(p)==0 && values::work(p)<=1048576);}

@test fn void scalar_assignment_context_and_checked_compound_operators(){
    let p=switch(checked(b"module main;fn i64 main(){var i64 n=0;n=4000000000;n+=2;n-=1;n*=2;n/=2;n%=3;n&=7;n|=8;n^=1;n<<=1;n>>=1;var i32 small=0;small+=1;var u8 byte=0;byte+=1;var u32 word=0;word+=1;var usize size=0;size+=1;var bool flag=false;flag=true;return n;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assignments(ref p,16);let c=values::at(ref p,0);var usize i=0;while(i<model::consumer_count(c)){let a=model::consumer_at(c,i);assert(a.header==0 && values::type_at(ref p,a.actual).kind==1 && values::type_at(ref p,a.actual).scalar<=6);i+=1;}
}
@test fn void mutable_field_array_view_and_named_reference_places(){
    let p=switch(checked(b"module main;struct Inner{i32 n;}struct Outer{Inner value;i32[2] row;}fn void main(){var i32 n=0;let ref mut i32 r=ref mut n;*r=1;var Outer x=Outer{.value=Inner{.n=0},.row=[0,0]};x.value.n=2;x.row=[1,2];x.row[0]+=1;var i32[2] a=[0,0];a[0]=2;let ref mut i32[] s=ref mut a;s[1]+=1;var Inner[2] items=[Inner{.n=0},Inner{.n=0}];items[0].n=3;let ref mut Inner item=ref mut items[1];*item=Inner{.n=4};let ref mut Inner[] all=ref mut items;all[0].n=5;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assignments(ref p,9);
}
@test fn void for_initialization_and_post_keep_clause_flags_and_scalar_types(){
    let text=b"module main;fn i32 main(){var i32 n=0;for(n=0;(n as i64)<3;n+=1){n+=1;}for(var i64 j=4000000000;j<4000000002;j=j+(1 as i64)){}return n;}";let source=parsed(text);let p=switch(checked(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};assignments(ref p,4);let c=values::at(ref p,0);var usize init=0;var usize post=0;var usize body=0;var usize i=0;while(i<model::consumer_count(c)){let a=model::consumer_at(c,i);if(a.kind==14){assert(control::plain_header(ref source,a.source_index,0));}if(a.kind==4){if(a.header==1){init+=1;}else if(a.header==2){post+=1;}else{body+=1;}}i+=1;}assert(init==1 && post==2 && body==1);
}
@test @effects(console) fn void assignment_errors_keep_codes_and_original_spans(){
    failure(b"module main;fn void main(){var i32 n=0;n=true;}",1006,b"true");
    failure(b"module main;fn void main(){let i32 n=0;n=1;}",1008,b"n");
    failure(b"module main;fn void main(){var i32 n=0;let ref i32 r=ref n;*r=1;}",1008,b"*r");
    failure(b"module main;fn void main(){let i32[2] a=[0,0];a[0]=1;}",1008,b"a[0]");
    failure(b"module main;struct S{i32 n;}fn void main(){let S s=S{.n=0};s.n=1;}",1008,b"s.n");
    failure(b"module main;struct S{i32[2] row;}fn void main(){var S[2] items=[S{.row=[0,0]},S{.row=[0,0]}];items[0].row=[1,2];}",1008,b"items[0].row");
    failure(b"module main;struct Empty{}fn void edit(ref mut Empty p){*p=Empty{};}fn void main(){}",1008,b"*p");
    failure(b"module main;fn void main(){var bool b=false;b+=true;}",2,b"b+=true;");
    failure(b"module main;fn void main(){var u8 n=0;n=256;}",1013,b"256");
    failure(b"module main;fn i32 n(){return 1;}fn void main(){n()=2;}",1008,b"n()");
}
@test @effects(console) fn void for_clauses_reject_non_scalar_behavior_at_clause_spans(){
    failure(b"module main;fn i32 get(){return 1;}fn void main(){var i32 n=0;for(n=get();n<2;n+=1){}}",2,b"n=get()");
    failure(b"module main;fn i32 get(){return 1;}fn void main(){var i32 n=0;for(;n<2;n=get()){}}",2,b"n=get()");
    failure(b"module main;fn void main(){var i32[2] a=[0,0];for(a[0]=0;true;a[0]+=1){break;}}",2,b"a[0]=0");
    failure(b"module main;fn i32 get(){return 1;}fn void main(){for(var i32 n=get();n<2;n+=1){}}",2,b"var i32 n=get()");
    failure(b"module main;fn void main(){var i32 n=0;for(;true;*ref mut n=1){break;}}",1008,b"*ref mut n");
}
@test fn void provider_generic_and_cleanup_assignments_use_owned_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(){return p::go<i64>(4000000000);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{i32 n;cleanup{this.n=0;}}pub fn T go<T>(T n){let Item item=Item{.n=1};discard item;var T result=n;result=n;return result;}" )).is_ok);let p=switch(projected(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assignments(ref p,2);var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=0;while(j<model::consumer_count(c)){let a=model::consumer_at(c,j);if(a.kind==4){assert(model::frame(c).module_index==1);let e=model::expression_at(c,a.target);let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(e.start==original.start && e.end==original.end);}j+=1;}i+=1;}
}
