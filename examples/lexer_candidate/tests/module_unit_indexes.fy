module tests::module_unit_indexes;
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
@test fn void scalar_views_and_fixed_arrays_feed_value_consumers(){
    let p=switch(checked(b"module main;fn i64 leaf(i64 n){return n;}fn i64 wide(ref i64[] a){let value=a[1];return leaf(4000000000+value);}fn u8 byte(ref u8[] a){return a[0];}fn u32 word(ref mut u32[] a){return a[0];}fn i32 signed(ref i32[] a){return a[0];}fn usize size(ref usize[] a){return a[0];}fn bool bit(ref bool[] a){return a[0];}fn i32 fixed(i32[4] a){return a[2];}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==8 && values::checked_arguments(ref p)==1 && values::inferred(ref p)==1 && values::unresolved(ref p)==0);var usize count=0;var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==9){assert(e.state==1 && e.place.kind==4 && e.place.root==1 && e.place.scope==1);count+=1;}j+=1;}i+=1;}assert(count==7);
}
@test fn void indexed_borrows_keep_whole_roots_and_mutability(){
    let p=switch(checked(b"module main;fn i64 shared(ref i64[] a){let r=ref (a[1]);return *r;}fn i64 mutable(ref mut i64[] a){let r=ref mut a[0];return *r;}fn i32 fixed(i32[4] a){let r=ref a[0];return *r;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==3 && values::inferred(ref p)==3 && values::unresolved(ref p)==0);var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);assert(model::local_at(c,2).origin.root==1);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==21 || e.kind==22){let t=values::type_at(ref p,e.type);assert(e.place.root==1 && t.kind==if(e.kind==21){7}else{8});}j+=1;}i+=1;}
}
@test fn void nominal_array_fields_and_index_field_borrows_retain_provider(){
    let p=switch(checked(b"module main;struct S{i64 n;i32[4] values;}fn i64 nominal(S[4] a){let r=ref a[0].n;return *r;}fn i32 field(ref S s){let value=s.values[0];return value;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::unresolved(ref p)==0);var usize i=0;var usize fields=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==8){assert(e.field.id>0 && e.field.module_index==0 && e.place.root==1 && e.state==1);fields+=1;}j+=1;}i+=1;}assert(fields==2);
}
@test fn void index_and_borrow_errors_keep_reference_spans(){
    failure(b"module main;fn i32 main(i32[4] a,bool n){return a[n];}",1006,b"n");
    failure(b"module main;fn i32 main(i32[4] a,i32 n){return a[n];}",1006,b"n");
    failure(b"module main;fn u8 main(ref str a){return a[0];}",2,b"a");
    failure(b"module main;fn i32 main(i32 a){return a[0];}",2,b"a");
    failure(b"module main;fn i64 main(ref i64[] a){let r=ref mut a[0];return *r;}",1008,b"a[0]");
    failure(b"module main;fn i32 main(i32[4] a){let r=ref mut a[0];return *r;}",1008,b"a[0]");
    failure(b"module main;fn i32 main(i32[2][3] a){let r=ref a[0][1];return *r;}",2,b"a");
    failure(b"module main;fn u8 main(ref i64[] a){return a[0];}",1009,b"a[0]");
}
@test fn void generic_provider_indexes_and_cleanup_fields_share_value_consumers(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(ref i64[] a){return p::go<i64>(a);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{i32 n;cleanup{let i32[4] values=[0;4];let value=values[0];discard value;return;}}pub fn T go<T>(ref T[] a){let Item item=Item{.n=1};discard item;return a[0];}")).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::checked_returns(ref p)==3);var usize i=0;var usize count=0;while(i<values::count(ref p)){let c=values::at(ref p,i);let f=model::frame(c);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==9){assert(f.module_index==1 && e.state==1 && e.place.root>0);count+=1;}j+=1;}i+=1;}assert(count==2);
}
