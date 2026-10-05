module tests::types;
import name_resolution as names;
import type_check as typing;
import type_state as state;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

fn typing::Typed checked(ref u8[] source) {
    var bytes::Buffer input = bytes::empty();
    assert(bytes::append(ref mut input,source).is_ok);
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) {
        case .ok(value): { value } case .err(_): { assert(false); return checked(b"module main;"); }
    };
    let syntax::Parsed parsed = switch(syntax::parse(stream,512)) {
        case .ok(value): { value } case .err(_): { assert(false); return checked(b"module main;"); }
    };
    return typing::check(names::resolve(parsed));
}

@test
fn void retained_scalar_types() {
    let typing::Typed typed = checked(b"module main; fn i64 f(i64 x) { let y=1+x; return y; } fn i64 minimum() { return -9223372036854775808; }");
    assert(typing::problem(ref typed).code == 0);
    let ref names::Resolved resolved = typing::resolved(ref typed);
    var usize id = 1;
    while (id <= names::symbol_count(resolved)) { assert(typing::symbol_type(ref typed,id) == 2); id += 1; }
    let ref syntax::Parsed parsed = names::parsed(resolved);
    id = 1; var usize minima = 0;
    while (id <= syntax::expression_count(parsed)) {
        let syntax::Expression node = syntax::expression_at(parsed,id);
        if (node.kind == 1 || node.kind == 2 || node.kind == 4 || node.kind == 5) {
            assert(typing::expression_type(ref typed,id) == 2);
            if (typing::signed_minimum(ref typed,id)) { assert(node.kind == 4); minima += 1; }
        }
        id += 1;
    }
    assert(minima == 1);
}

@test
fn void literal_range_checks() {
    assert(state::fits(b"18446744073709551615",5,false));
    assert(!state::fits(b"18446744073709551616",5,false));
    assert(state::fits(b"0x8000000000000000",2,true));
    assert(!state::fits(b"0x8000000000000000",2,false));
    assert(!state::fits(b"9223372036854775809",2,true));
    let typing::Typed bad = checked(b"module main; fn i64 f() { return true; }");
    assert(typing::problem(ref bad).code == 1);
}
