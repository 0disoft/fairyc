module tests::module_unit_operators;
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
@test fn void integer_peer_context_and_signed_minimum_are_preserved(){
    let p=switch(checked(b"module main;fn u8 first(u8 n){return (1+n);}fn u8 last(u8 n){let result=n+1;return result;}fn i64 wide(i64 n){return (4000000000+n);}fn bool order(i64 n){return 4000000000<n;}fn bool logic(bool n){return !n || (n && true);}fn i32 low(){return -2147483648;}fn i64 low64(){return -9223372036854775808;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::count(ref p)==7 && values::checked_returns(ref p)==7 && values::inferred(ref p)==1 && values::work(ref p)<=1048576);var usize i=0;var usize minimum=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);let r=model::return_at(c,0);assert(r.checked && r.expected==r.actual);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.minimum){assert(e.kind==4 && e.state==1 && (values::type_at(ref p,e.type).scalar==1 || values::type_at(ref p,e.type).scalar==2));minimum+=1;}j+=1;}i+=1;}assert(minimum==2);
}
@test fn void operator_errors_keep_the_original_operand_span(){
    failure(b"module main;fn bool f(){return true && 1;}",1007,b"1");failure(b"module main;fn bool f(){return !1;}",1007,b"1");failure(b"module main;fn bool f(){return true+false;}",2,b"true+false");failure(b"module main;fn i32 f(u8 n){return +n;}",1006,b"n");failure(b"module main;fn u8 f(){return 256+1;}",1013,b"256");failure(b"module main;fn u8 f(u8 n,i64 wide){return n+wide;}",1006,b"wide");failure(b"module main;fn i64 f(){return -9223372036854775809;}",1013,b"9223372036854775809");
}
@test fn void field_and_cast_anchors_contextualize_wide_literal_peers(){
    let p=switch(checked(b"module main;struct S{i64 n;}fn bool field(ref S s){return 4000000000<s.n;}fn i64 cast(){return 1+(2 as i64);}fn i64 leaf(i64 n){return n;}fn i64 argument(){return leaf(4000000000+1);}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::count(ref p)==4 && values::checked_returns(ref p)==4 && values::unresolved(ref p)==0);let c=values::at(ref p,0);let r=model::return_at(c,0);assert(r.checked);var usize i=1;var bool contextual=false;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==1){assert(e.state==1 && values::type_at(ref p,e.type).scalar==2);contextual=true;}i+=1;}assert(contextual);
}
@test fn void scalar_operators_share_generic_cleanup_and_ordinary_contexts(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn u8 main(){return p::go<u8>(1)+1;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{u8 n;cleanup{let inferred=helper()+1;discard inferred;return;}}fn u8 helper(){return 2;}pub fn T go<T>(T n){let Item item=Item{.n=1};let inferred=1+n;discard item;return inferred;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::inferred(ref p)==2 && values::checked_returns(ref p)==3);var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::local_count(c)){let s=model::local_at(c,j);if(s.expected==0 && s.type!=0){assert(values::type_at(ref p,s.type).scalar==3 && bytes::equal(syntax::source_view(v::at<syntax::Parsed>(ref sources,1))[s.start..s.end],b"inferred"));}j+=1;}i+=1;}
}
