module tests::module_unit_result_flow;
import module_project_compile as compile;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let raw=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let tokens=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn void check(ref u8[] text,bool accepted){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(compile::check_range(ref sources,ref metadata,0,1)){case .ok(_):{assert(accepted);}case .err(p):{assert(!accepted && p.stage==9 && p.code==4101 && p.module_index==0 && p.start<p.end);}};
    switch(compile::range(ref sources,ref metadata,0,1)){case .ok(_):{assert(accepted);}case .err(p):{assert(!accepted && p.stage==9 && p.code==4101 && p.module_index==0 && p.start<p.end);}};
}
@test fn void pending_parameters_values_overwrites_and_loop_edges_reject(){
    check(b"module main;fn void f(Result<i32,i32> value){}",false);
    check(b"module main;fn void f(){let Result<i32,i32> value=.ok(1);}",false);
    check(b"module main;fn void f(Result<i32,i32> value){value;discard value;}",false);
    check(b"module main;fn void f(){var Result<i32,i32> value=.ok(1);value=.ok(2);discard value;}",false);
    check(b"module main;fn void f(bool condition,Result<i32,i32> value){if(condition){discard value;};}",false);
    check(b"module main;fn void f(bool condition){while(condition){let Result<i32,i32> value=.ok(1);break;}}",false);
    check(b"module main;fn void f(bool condition){while(condition){let Result<i32,i32> value=.ok(1);continue;}}",false);
    check(b"module main;fn void f(bool condition){var Result<i32,i32> value=.ok(1);discard value;while(condition){value=.ok(2);continue;}discard value;}",false);
}
@test fn void returns_discards_matches_is_ok_and_transfers_discharge(){
    check(b"module main;fn Result<i32,i32> f(Result<i32,i32> value){return value;}",true);
    check(b"module main;fn void f(Result<i32,i32> value){discard value;}",true);
    check(b"module main;fn bool f(Result<i32,i32> value){return value.is_ok;}",true);
    check(b"module main;fn void f(Result<i32,i32> value){switch(value){case .ok(v):{discard v;}case .err(e):{discard e;}};}",true);
    check(b"module main;fn void take(Result<i32,i32> value){discard value;}fn void f(){take(.ok(1));}",true);
    check(b"module main;fn void f(bool condition,Result<i32,i32> value){if(condition){discard value;}else{discard value;};}",true);
    check(b"module main;fn void f(Result<i32,i32> value){assert(false);}",true);
    check(b"module main;fn void f(bool condition){while(condition){let Result<i32,i32> value=.ok(1);discard value;continue;}}",true);
}
@test fn void propagation_captures_shadowing_and_short_circuit_keep_obligations(){
    check(b"module main;fn Result<i32,i32> f(Result<i32,i32> value,Result<i32,i32> other){let i32 n=value?;discard other;return .ok(n);}",false);
    check(b"module main;fn void f(Result<Result<i32,i32>,i32> value){switch(value){case .ok(inner):{discard 1;}case .err(e):{discard e;}};}",false);
    check(b"module main;fn void f(bool condition,Result<i32,i32> value){discard condition && value.is_ok;}",false);
    check(b"module main;fn void f(Result<i32,i32> value){{let Result<i32,i32> value=.ok(1);discard value;};}",false);
    check(b"module main;fn Result<i32,i32> f(Result<i32,i32> value){let i32 n=value?;return .ok(n);}",true);
    check(b"module main;fn void f(Result<Result<i32,i32>,i32> value){switch(value){case .ok(inner):{discard inner;}case .err(e):{discard e;}};}",true);
    check(b"module main;fn void f(bool condition,Result<i32,i32> value){discard value.is_ok && condition;}",true);
}
@test fn void imported_generic_result_call_is_checked_in_the_consumer(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn void f(){p::wrap<i32>(1);}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn Result<T,i32> wrap<T>(T value){return .ok(value);}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(compile::check_range(ref sources,ref metadata,0,2)){case .ok(_):{assert(false);}case .err(p):{assert(p.stage==9 && p.code==4101 && p.module_index==0 && p.start==45 && p.end==60);}};
}
