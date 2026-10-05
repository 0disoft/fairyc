module tests::nominal_types;
import name_resolution as names;
import type_check as typing;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;

fn typing::Typed checked(ref u8[] source) {
    var bytes::Buffer input = bytes::empty(); assert(bytes::append(ref mut input,source).is_ok);
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) { case .ok(s): { s } case .err(_): { assert(false); return checked(b"module main;"); } };
    let syntax::Parsed parsed = switch(syntax::parse(stream,512)) { case .ok(p): { p } case .err(_): { assert(false); return checked(b"module main;"); } };
    assert(syntax::problem_count(ref parsed) == 0);
    let names::Resolved resolved = names::resolve(parsed); assert(names::problem(ref resolved).code == 0);
    return typing::check(resolved);
}
fn u32 code(ref u8[] source) { let typing::Typed typed = checked(source); return typing::problem(ref typed).code; }
@test
fn void fields_nested_values_calls_and_nominal_identity() {
    let typing::Typed typed = checked(b"module main;struct Pair{i64 value;bool enabled;}struct Outer{Pair pair;}fn Pair identity(Pair p){return p;}fn i64 main(){var Outer outer=Outer{.pair=identity(Pair{.enabled=true,.value=4000000000})};outer.pair.value+=2;let copy=outer;return copy.pair.value;}");
    assert(typing::problem(ref typed).code == 0);
    let ref syntax::Parsed parsed = names::parsed(typing::resolved(ref typed));
    var usize id = 1; var usize fields = 0;
    while (id <= syntax::expression_count(parsed)) {
        if (syntax::expression_at(parsed,id).kind == 8) { assert(typing::field_target(ref typed,id) != 0); fields += 1; }
        id += 1;
    }
    assert(fields == 4);
    assert(code(b"module main;struct A{i32 x;}struct B{i32 x;}fn A f(B b){return b;}") == 1);
    assert(code(b"module main;struct Empty{}fn Empty f(){return Empty{};}") == 0);
    assert(code(b"module main;struct Pair{i32 x;i32 y;}fn Pair f(i32 x,i32 y){return Pair{.x,.y};}") == 0);
}
@test
fn void malformed_fields_cycles_and_immutable_places() {
    assert(code(b"module main;struct A{i32 x;}fn A f(){return A{};}") == 19);
    assert(code(b"module main;struct A{i32 x;}fn A f(){return A{.x=1,.x=2};}") == 17);
    assert(code(b"module main;struct A{i32 x;}fn A f(){return A{.z=1};}") == 18);
    assert(code(b"module main;struct A{i32 x;}fn i32 f(A a){return a.z;}") == 18);
    assert(code(b"module main;struct A{i32 x;}fn void f(A a){a.x=2;}") == 4);
    assert(code(b"module main;struct A{i32 x;}fn void f(){let a=A{.x=1};a.x=2;}") == 4);
    assert(code(b"module main;struct A{i32 x;}fn void f(){var a=A{.x=1};a.x=true;}") == 1);
    assert(code(b"module main;struct A{i32 x;i32 x;}") == 17);
    assert(code(b"module main;struct A{A x;}") == 20);
    assert(code(b"module main;struct A{B x;}struct B{A x;}") == 20);
    assert(code(b"module main;struct A{Missing x;}") == 16);
}
