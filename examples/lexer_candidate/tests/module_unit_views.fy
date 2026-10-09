module tests::module_unit_views;
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
@test fn void array_views_slices_and_aliases_retain_original_storage(){
    let p=switch(checked(b"module main;fn i64 leaf(ref i64[] a){return a[0];}fn i64 main(){var i64[4] a=[4000000000;4];let ref mut i64[] exclusive=ref mut a;let child=exclusive[1..3];let alias=child;return leaf(alias)+alias[0];}fn i32 range(i32[4] a){let shared=ref ((a[1..]));let alias=shared;return alias[..][0];}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==3 && values::checked_arguments(ref p)==1 && values::shared_arguments(ref p)==1 && values::unresolved(ref p)==0);var usize i=1;var usize slices=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==26){assert(e.state==1 && e.place.root==1 && e.place.scope>0 && e.place.kind==0 && !e.place.borrowable);slices+=1;}if(e.kind==27){assert(e.state==2 && e.type==0);}j+=1;}j=2;while(j<=model::local_count(c)){let local=model::local_at(c,j);let t=values::type_at(ref p,local.type);if(t.kind==10 || t.kind==11 || t.kind==15){assert(local.origin.root==1);}j+=1;}i+=1;}assert(slices==3);
}
@test fn void all_scalar_views_keep_exact_element_and_mutability(){
    let p=switch(checked(b"module main;fn bool bit(){let bool[2] a=[true,false];let ref bool[] v=ref a;return v[..][0];}fn u8 byte(){var u8[2] a=[1,2];let ref mut u8[] v=ref mut a;return v[..][0];}fn i32 signed(){let i32[2] a=[1,2];let ref i32[] v=ref a;return v[..][0];}fn u32 word(){let u32[2] a=[1,2];let ref u32[] v=ref a;return v[..][0];}fn usize size(){let usize[2] a=[1,2];let ref usize[] v=ref a;return v[..][0];}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==5 && values::unresolved(ref p)==0);var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);let local=model::local_at(c,2);assert(local.type==local.expected && local.origin.root==1);i+=1;}
}
@test fn void array_field_views_and_grouped_subranges_keep_whole_struct(){
    let p=switch(checked(b"module main;struct S{i32[4] a;}fn i32 shared(ref S s){let ref i32[] whole=ref (s.a);let range=ref ((s.a[1..]));let value=whole[0];return value+range[0];}fn i32 mutable(ref mut S s){let range=ref mut s.a[..2];return range[0];}fn i32 borrowed_array(i32[4] a){let whole=ref a;return (*whole)[0];}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==3 && values::unresolved(ref p)==0);var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=2;while(j<=model::local_count(c)){let local=model::local_at(c,j);let t=values::type_at(ref p,local.type);if(t.kind!=1){assert(local.origin.root==1);}j+=1;}i+=1;}
}
@test fn void known_view_returns_and_length_bounds_keep_parameter_origin(){
    let p=switch(checked(b"module main;fn ref i32[] select(ref i32[] a){return a[1..];}fn ref mut i64[] select_mut(ref mut i64[] a){let child=a[..a.len];return child;}fn bool predicate(ref u8[] a){return a[..].is_empty;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==3 && values::unresolved(ref p)==0);var usize i=0;while(i<2){let c=values::at(ref p,i);let ret=model::return_at(c,0);assert(ret.checked && ret.expected==ret.actual);i+=1;}
}
@test fn void invalid_view_borrows_and_bounds_keep_reference_diagnostics(){
    failure(b"module main;fn i32 main(i32[4] a){let v=a[..];return 0;}",2,b"a");
    failure(b"module main;fn i32 main(ref i32[] a){let v=ref a[..];return 0;}",2,b"ref a[..]");
    failure(b"module main;fn i32 main(i32[4] a){let ref i32[] v=ref (a);return 0;}",2,b"ref (a)");
    failure(b"module main;fn i32 main(i32[4] a){let v=ref mut a[1..];return 0;}",1008,b"a");
    failure(b"module main;fn i32 main(i32[4] a){let ref mut i32[] v=ref mut a;return 0;}",1008,b"a");
    failure(b"module main;fn i32 main(i32[4] a){let ref u8[] v=ref a;return 0;}",1006,b"ref a");
    failure(b"module main;fn i32 main(ref mut i32[] a){let ref i32[] v=a;return 0;}",1006,b"a");
    failure(b"module main;fn i32 main(ref i32[] a,bool low){let v=a[low..];return 0;}",1006,b"low");
    failure(b"module main;fn i32 main(ref i32[] a,i32 high){let v=a[..high];return 0;}",1006,b"high");
    failure(b"module main;fn i32 main(ref str a){let v=a[..];return 0;}",2,b"a");
    failure(b"module main;struct S{ptr i32 p;i32[4] a;}fn i32 main(ref S s){let ref i32[] v=ref s.a;return 0;}",2,b"ref s.a");
    failure(b"module main;struct S{ptr i32 p;i32[4] a;}fn i32 main(ref S s){let v=ref s.a[..];return 0;}",1008,b"s.a");
}
@test fn void generic_and_cleanup_views_keep_provider_expression_ids(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(){return p::go<i64>(2);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{i32 n;cleanup{let i32[2] a=[1,2];let ref i32[] v=ref a;let part=v[..];discard part;return;}}pub fn T go<T>(T n){let Item item=Item{.n=1};let T[2] a=[n;2];let ref T[] v=ref a;discard item;return v[0];}")).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::checked_returns(ref p)==3);var usize i=0;var usize borrows=0;while(i<values::count(ref p)){let c=values::at(ref p,i);let frame=model::frame(c);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==21 || e.kind==26){assert(e.state==1 && e.place.root>0 && frame.module_index==1);let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(original.start==e.start && original.end==e.end);borrows+=1;}j+=1;}i+=1;}assert(borrows==3);
}
