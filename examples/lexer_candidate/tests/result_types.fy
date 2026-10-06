module tests::result_types;
import tagged_types as tagged;
import nominal_types as nominal;
import name_resolution as names;
import type_check as typing;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

fn names::Resolved resolved(ref u8[] source) {
    var bytes::Buffer input = bytes::empty(); assert(bytes::append(ref mut input,source).is_ok);
    let stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) {
        case .ok(p): { p } case .err(_): { assert(false); return resolved(b"module main;"); }
    };
    let parsed = switch(syntax::parse(stream,512)) {
        case .ok(p): { p } case .err(_): { assert(false); return resolved(b"module main;"); }
    };
    assert(syntax::problem_count(ref parsed) == 0); return names::resolve(parsed);
}
@test
fn void result_pairs_are_ordered_and_distinct_from_other_types() {
    var u32 success = 1;
    while (success <= 6) {
        var u32 failure = 1;
        while (failure <= 6) {
            let u32 id = tagged::make_result(success,failure);
            assert(tagged::result(id) && !tagged::option(id));
            assert(tagged::success(id) == success && tagged::failure(id) == failure);
            assert(id == 262144 + (success - 1) * 6 + failure);
            failure += 1;
        }
        success += 1;
    }
    assert(tagged::make_result(0,1) == 0 && tagged::make_result(1,7) == 0);
    let r = resolved(b"module main;fn Result<i64,bool> id(Result<i64,bool> x){let Result<i64,bool> y=x;return y;}");
    assert(names::problem(ref r).code == 0);
    let typed = typing::check(r); assert(typing::problem(ref typed).code == 0);
    var usize i = 1;
    while (i <= names::symbol_count(typing::resolved(ref typed))) {
        assert(typing::symbol_type(ref typed,i) == tagged::make_result(2,6)); i += 1;
    }
}
@test
fn void result_identity_mismatches_and_closed_storage_are_rejected() {
    let different_error = typing::check(resolved(b"module main;fn Result<i32,bool> f(Result<i32,i64> x){return x;}"));
    assert(typing::problem(ref different_error).code == 1);
    let different_success = typing::check(resolved(b"module main;fn Result<i64,bool> f(Result<i32,bool> x){return x;}"));
    assert(typing::problem(ref different_success).code == 1);
    let field = typing::check(resolved(b"module main;struct Box{Result<i32,bool> value;}fn i32 main(){return 0;}"));
    assert(typing::problem(ref field).code == 16);
    let invalid = resolved(b"module main;fn Result<void,bool> f(Result<void,bool> x){return x;}");
    assert(names::problem(ref invalid).code != 0);
}
