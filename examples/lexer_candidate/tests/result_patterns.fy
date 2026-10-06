module tests::result_patterns;
import tests::result_types as fixture;
import tagged_types as tagged;
import name_resolution as names;
import type_check as typing;
import syntax as syntax;
import std::bytes as bytes;

@test
fn void result_captures_preserve_success_error_types_and_local_scope() {
    let typed = typing::check(fixture::resolved(b"module main;fn i64 f(Result<i64,bool> r){return switch(r){case .err(e):{if(e){0}else{1}}case .ok(n):{n}};}"));
    assert(typing::problem(ref typed).code == 0);
    let ref names::Resolved resolved = typing::resolved(ref typed);
    let ref syntax::Parsed parsed = names::parsed(resolved);
    var usize id = 1; var usize captures = 0;
    while (id <= names::symbol_count(resolved)) {
        let symbol = names::symbol_at(resolved,id);
        if (symbol.kind == 8) {
            let bool error = bytes::equal(syntax::source_view(parsed)[symbol.start..symbol.end],b"e");
            assert(typing::symbol_type(ref typed,id) == if (error) { 6 } else { 2 });
            assert(typing::field_target(ref typed,symbol.declaration) == tagged::payload_for(parsed,tagged::make_result(2,6),!error));
            captures += 1;
        }
        id += 1;
    }
    assert(captures == 2);
}
@test
fn void result_terminating_arms_and_contextual_payloads_are_checked() {
    let returned = typing::check(fixture::resolved(b"module main;fn i32 f(Result<i64,bool> r){return switch(r){case .ok(n):{assert(n==4000000000);7}case .err(e):{return if(e){2}else{3};}};}"));
    assert(typing::problem(ref returned).code == 0);
    let literal = typing::check(fixture::resolved(b"module main;fn Result<i64,bool> make(){return .ok(4000000000);}fn i64 f(){return switch(make()){case .ok(n):{n}case .err(_):{assert(false);}};}"));
    assert(typing::problem(ref literal).code == 0);
    let wrong = typing::check(fixture::resolved(b"module main;fn Result<i64,bool> make(){return .err(1);}"));
    assert(typing::problem(ref wrong).code == 1);
    let incomplete = typing::check(fixture::resolved(b"module main;fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}};}"));
    assert(typing::problem(ref incomplete).code == 39);
}
