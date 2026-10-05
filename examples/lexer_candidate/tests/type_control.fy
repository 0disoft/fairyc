module tests::type_control;
import name_resolution as names;
import type_check as typing;
import type_control as control;
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
fn void branch_and_loop_facts() {
    let typing::Typed typed = checked(b"module main; fn i64 f(i64 limit) { var i64 total=0; for(var i64 i=0; i<limit; i+=1) { if(i%2==0) { total+=i; } else { continue; } } if(limit>0) { return total; } else { return 0; } }");
    assert(typing::problem(ref typed).code == 0);
    let ref syntax::Parsed parsed = names::parsed(typing::resolved(ref typed));
    assert(control::block_terminates(parsed,syntax::declaration_at(parsed,1).body,0));
    var usize id = 1; var usize comparisons = 0; var usize branches = 0;
    while (id <= syntax::expression_count(parsed)) {
        let syntax::Expression node = syntax::expression_at(parsed,id);
        if (node.kind == 13) { assert(typing::expression_type(ref typed,id) == 7); branches += 1; }
        if (node.kind == 5 && (bytes::equal(syntax::expression_text(parsed,id),b"<") ||
            bytes::equal(syntax::expression_text(parsed,id),b">") || bytes::equal(syntax::expression_text(parsed,id),b"=="))) {
            assert(typing::expression_type(ref typed,id) == 6); comparisons += 1;
        }
        id += 1;
    }
    assert(branches == 2 && comparisons == 3);
}

@test
fn void branch_value_context() {
    let typing::Typed typed = checked(b"module main; fn u8 f(bool b) { return if(b) { 255 } else { 0 }; }");
    assert(typing::problem(ref typed).code == 0);
    let ref syntax::Parsed parsed = names::parsed(typing::resolved(ref typed));
    var usize id = 1; var usize branches = 0;
    while (id <= syntax::expression_count(parsed)) {
        if (syntax::expression_at(parsed,id).kind == 13) { assert(typing::expression_type(ref typed,id) == 3); branches += 1; }
        id += 1;
    }
    assert(branches == 1);
}

@test
fn void return_and_loop_rejections() {
    let typing::Typed partial = checked(b"module main; fn i64 f(bool b) { if(b) { return 1; } }");
    assert(typing::problem(ref partial).code == 7);
    let typing::Typed loop = checked(b"module main; fn void f() { for(;;) { break; } continue; }");
    assert(typing::problem(ref loop).code == 13);
    let typing::Typed terminal = checked(b"module main; fn i64 f() { assert(false); }");
    assert(typing::problem(ref terminal).code == 0);
}
