module tests::module_unit_casts;
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
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return checked(b"module empty;");}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return checked(b"module empty;");}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return checked(b"module empty;");}};return values::build(ref sources,ref metadata,ref ready);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}
@test fn void scalar_casts_reach_locals_operators_and_arguments(){
    let text=b"module main;fn i64 leaf(i64 n){return n;}fn i64 main(u8 n,u32 wide){let a=n as i64;let b=wide as i64;return leaf(4000000000+(a+b));}fn usize size(u8 n){return n as usize;}fn c::int foreign(i32 n){return n as c::int;}fn i32 back(c::int n){return n as i32;}";
    let original_source=parsed(text);let p=switch(checked(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==5 && values::checked_returns(ref p)==5 && values::inferred(ref p)==2 && values::checked_arguments(ref p)==1);var usize i=0;var usize casts=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);assert(model::checked_casts(c)==model::cast_count(c));casts+=model::cast_count(c);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==23){assert(e.target==e.type && e.target>0 && e.state==1 && e.right>0);let original=syntax::expression_at(ref original_source,e.source_index);assert(original.right==e.right && original.start==e.start && original.end==e.end);}j+=1;}i+=1;}assert(casts==5);
}
@test fn void cast_targets_do_not_contextualize_source_literals(){
    failure(b"module main;fn i64 main(){return 4000000000 as i64;}",1013,b"4000000000");
    failure(b"module main;fn i64 main(){return -9223372036854775808 as i64;}",1013,b"9223372036854775808");
    failure(b"module main;fn u8 main(i32 n){return n as u8;}",2,b"n as u8");
    failure(b"module main;fn i32 main(bool n){return n as i32;}",2,b"n as i32");
    failure(b"module main;fn i32 main(i32 n){return n as i32;}",2,b"n as i32");
    let p=switch(checked(b"module main;fn i64 main(){return (1+2) as i64;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==1 && model::checked_casts(values::at(ref p,0))==1);
}
@test fn void field_operands_and_unsafe_pointer_casts_resolve_exactly(){
    let p=switch(checked(b"module main;struct S{u8 n;}fn i64 field(ref S s){return s.n as i64;}unsafe fn ptr i32 pointer(ref i32 n){return n as ptr i32;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==2 && values::checked_returns(ref p)==2);var usize i=0;while(i<2){let c=values::at(ref p,i);assert(model::cast_count(c)==1 && model::checked_casts(c)==1);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==23){assert(e.target>0 && e.type==e.target && e.state==1 && (i==0 || e.builtin.unsafe_context));}j+=1;}i+=1;}
}
@test fn void specialized_cleanup_and_multi_module_ordinary_targets_stay_exact(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(){return p::go<i64>(1);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{u8 n;cleanup{let u8 local=1;let widened=local as i64;discard widened;return;}}fn i64 ordinary(u8 n){return n as i64;}pub fn T go<T>(i32 n){let Item item=Item{.n=1};discard item;return n as T;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,2)){case .ok(b):{b}case .err(_):{assert(false);return;}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize i=0;var usize casts=0;while(i<values::count(ref p)){let c=values::at(ref p,i);let f=model::frame(c);let source=v::at<syntax::Parsed>(ref sources,f.module_index);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==23){let n=syntax::expression_at(source,e.source_index);assert(e.target==e.type && e.state==1 && values::type_at(ref p,e.target).scalar==2 && n.start==e.start && n.end==e.end && n.right==e.right && f.module_index==1);casts+=1;}j+=1;}i+=1;}assert(casts==3 && values::checked_returns(ref p)==values::returns(ref p));
}

fn void complete(ref u8[] text,usize expected_casts){let p=switch(checked(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==values::returns(ref p));var usize casts=0;var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);assert(model::checked_casts(c)==model::cast_count(c));casts+=model::cast_count(c);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==23){assert(e.state==1 && e.type==e.target && e.builtin.unsafe_context);}j+=1;}i+=1;}assert(casts==expected_casts);}
@test fn void pointer_casts_preserve_exact_nominal_and_scalar_referents(){
    complete(b"module main;struct S{i32 n;}resource R{i32 n;cleanup{}}unsafe fn ptr i32 scalar(ref i32 n){return n as ptr i32;}unsafe fn ptr mut i32 mutable(ref mut i32 n){return n as ptr mut i32;}unsafe fn ptr i32 shared(ref mut i32 n){return n as ptr i32;}unsafe fn ptr S nominal(ref S n){return n as ptr S;}unsafe fn ptr mut S exclusive(ref mut S n){return n as ptr mut S;}unsafe fn ptr void opaque(ref S n){return n as ptr void;}fn ptr mut void opaque_resource(){var R n=R{.n=1};return unsafe{ref mut n as ptr mut void};}",7);
}
@test fn void pointer_casts_extract_borrowed_data_and_opaque_array_storage(){
    complete(b"module main;unsafe fn ptr u8 bytes(ref u8[] n){return n as ptr u8;}unsafe fn ptr mut u8 mutable_bytes(ref mut u8[] n){return n as ptr mut u8;}unsafe fn ptr u8 text(ref str n){return n as ptr u8;}unsafe fn ptr u8 c_string(ref CStr n){return n as ptr u8;}unsafe fn ptr void scalar_view(ref i64[] n){return n as ptr void;}fn ptr void array(){let u8[2] n=[1,2];return unsafe{ref n as ptr void};}",6);
}
@test fn void pointer_casts_require_the_casts_own_unsafe_boundary(){
    complete(b"module main;fn ptr i32 safe(ref i32 n){return unsafe{n as ptr i32};}unsafe fn ptr i32 inherited(ref i32 n){return n as ptr i32;}",2);
    failure(b"module main;fn ptr i32 f(ref i32 n){return n as ptr i32;}",5101,b"n as ptr i32");
    failure(b"module main;fn ptr i32 f(ref i32 n){return (unsafe{n}) as ptr i32;}",5101,b"(unsafe{n}) as ptr i32");
    failure(b"module main;fn ptr i32 f(ref i32 n){discard unsafe{n as ptr i32};return n as ptr i32;}",5101,b"n as ptr i32");
}
@test fn void pointer_casts_reject_strengthening_reinterpretation_and_opaque_temporaries(){
    failure(b"module main;unsafe fn ptr mut i32 f(ref i32 n){return n as ptr mut i32;}",2,b"n as ptr mut i32");
    failure(b"module main;unsafe fn ptr u32 f(ref i32 n){return n as ptr u32;}",2,b"n as ptr u32");
    failure(b"module main;unsafe fn ptr i32 f(ptr i32 n){return n as ptr i32;}",2,b"n as ptr i32");
    failure(b"module main;unsafe fn ptr i32 f(i32 n){return n as ptr i32;}",2,b"n as ptr i32");
    failure(b"module main;fn ref i32 identity(ref i32 n){return n;}unsafe fn ptr void f(ref i32 n){return identity(n) as ptr void;}",2,b"identity(n) as ptr void");
    failure(b"module main;struct S{i32 n;}unsafe fn ptr void f(ref S s){return ref s.n as ptr void;}",2,b"ref s.n as ptr void");
    failure(b"module main;unsafe fn ptr mut u8 f(ref u8[] n){return n as ptr mut u8;}",2,b"n as ptr mut u8");
}
