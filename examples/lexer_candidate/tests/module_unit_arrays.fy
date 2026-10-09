module tests::module_unit_arrays;
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

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(_):{assert(false);return checked(b"module empty;");}};let bound=switch(binding::bind(sources,ref metadata,first,stop)){case .ok(b):{b}case .err(_):{assert(false);return checked(b"module empty;");}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return checked(b"module empty;");}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources,0,1);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}
@test fn void list_and_repeat_types_feed_inference_indexes_calls_and_returns(){
    let p=switch(checked(b"module main;fn u8 leaf(u8[2] a){return a[0];}fn u8 main(){let a=[1,2];let u8[3] bytes=[255;3];let repeated=[1;4];discard a;discard repeated;return leaf([1,2])+bytes[0];}fn i64 wide(){let i64[2] a=[4000000000,2];return a[0];}fn bool bit(){let flags=[true,false];return flags[0];}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==4 && values::checked_arguments(ref p)==1 && values::inferred(ref p)==3 && values::unresolved(ref p)==0);var usize i=0;var usize arrays=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==24 || e.kind==29){let t=values::type_at(ref p,e.type);assert(e.state==1 && t.kind==16 && t.first>0 && t.second>=2 && t.second<=4);arrays+=1;}j+=1;}i+=1;}assert(arrays==6);
}
@test fn void known_nominal_array_elements_validate_plain_shapes_and_identity(){
    let p=switch(checked(b"module main;struct Leaf{i32 n;}struct S{Leaf leaf;i32[2] cells;}fn S[2] copies(S value){return [value;2];}fn i32 main(S value){let a=[value,value];return a[0].leaf.n;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::inferred(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,0);let r=model::return_at(c,0);let t=values::type_at(ref p,r.actual);assert(t.kind==16 && t.second==2 && values::type_at(ref p,t.first).kind==2);
}
@test fn void invalid_array_values_and_counts_keep_reference_spans(){
    failure(b"module main;fn i32 main(){let a=[];return 0;}",2,b"[]");
    failure(b"module main;fn i32 main(){let a=[1;0];return 0;}",2,b"0");
    failure(b"module main;fn i32 main(){let a=[1;4097];return 0;}",2,b"4097");
    failure(b"module main;fn i32 main(usize n){let a=[1;n];return 0;}",2,b"n");
    failure(b"module main;fn i32 main(){let a=[1,true];return 0;}",1006,b"true");
    failure(b"module main;fn i32 main(){let u8[2] a=[1,256];return 0;}",1013,b"256");
    failure(b"module main;fn i32 main(){let u8[3] a=[1,2];return 0;}",1006,b"[1,2]");
    failure(b"module main;fn i32[3] main(){return [1,2];}",1009,b"[1,2]");
    failure(b"module main;fn i32 main(){let a=[[1,2],[3,4]];return 0;}",2,b"[[1,2],[3,4]]");
    failure(b"module main;struct Empty{}fn i32 main(Empty value){let a=[value;2];return 0;}",2,b"[value;2]");
    failure(b"module main;struct S{ptr i32 p;}fn i32 main(S value){let a=[value;2];return 0;}",2,b"[value;2]");
    failure(b"module main;struct S{i32 n;}struct T{i32 n;}fn i32 main(S a,T b){let values=[a,b];return 0;}",1006,b"b");
}
@test fn void original_provider_array_literals_keep_generic_and_cleanup_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn u8 main(){return p::go<u8>(2);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{i32 n;cleanup{let inferred=[1,2];discard inferred;return;}}pub fn T go<T>(T n){let Item item=Item{.n=1};let T[2] values=[n;2];discard item;return values[0];}")).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::checked_returns(ref p)==3 && values::inferred(ref p)==1);var usize i=0;var usize arrays=0;while(i<values::count(ref p)){let c=values::at(ref p,i);let f=model::frame(c);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==24 || e.kind==29){assert(f.module_index==1 && e.state==1 && values::type_at(ref p,e.type).kind==16);let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(e.start==original.start && e.end==original.end);arrays+=1;}j+=1;}i+=1;}assert(arrays==2);
}
