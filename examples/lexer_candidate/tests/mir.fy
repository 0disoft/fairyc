module tests::mir;
import mir_model as m;
import mir_lower as lower;
import mir_state as state;
import hir_lower as hir;
import hir_model as h;
import name_resolution as names;
import type_check as typing;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] source) {
    var bytes::Buffer input = bytes::empty(); assert(bytes::append(ref mut input,source).is_ok);
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) { case .ok(value): { value } case .err(_): { assert(false); return parsed(b"module main;"); } };
    let syntax::Parsed result = switch(syntax::parse(stream,512)) { case .ok(value): { value } case .err(_): { assert(false); return parsed(b"module main;"); } };
    assert(syntax::problem_count(ref result) == 0); return result;
}
fn Result<m::Program,m::Problem> lowered(ref u8[] source, usize module_index) {
    let typing::Typed typed = typing::check(names::resolve(parsed(source)));
    assert(typing::problem(ref typed).code == 0);
    let h::Program input = switch(hir::lower(typed,module_index)) { case .ok(value): { value } case .err(_): { assert(false); return lowered(b"module main;",0); } };
    return lower::lower(input);
}
fn m::Program retained(ref u8[] source, usize module_index) {
    return switch(lowered(source,module_index)) { case .ok(value): { value } case .err(_): { assert(false); return retained(b"module main;",0); } };
}

@test
fn void owned_constants_and_module_identity() {
    let ref u8[] source = b"module main; pub fn i64 f(){return -9223372036854775808;}";
    let m::Program p = retained(source,7);
    assert(bytes::equal(m::source_view(ref p),source)); assert(m::module_index(ref p) == 7);
    assert(m::function_count(ref p) == 1 && m::target_count(ref p) == 1);
    let m::Function f = m::function_at(ref p,1);
    assert(f.public && m::target_at(ref p,f.target).result == 2);
    assert(m::target_at(ref p,f.target).module_index == 7);
    let m::Block b = m::block_at(ref p,f.entry);
    assert(b.kind == 1 && b.target == 0 && b.alternative == 0);
    assert(m::operation_at(ref p,b.first).kind == 1);
    assert(bytes::equal(m::literal_text(ref p,b.first),b"-9223372036854775808"));
}

@test
fn void checked_arithmetic_has_explicit_trap_edges() {
    let m::Program p = retained(b"module main; fn i64 f(i64 x){return x/2;}",0);
    let m::Function f = m::function_at(ref p,1); let m::Block b = m::block_at(ref p,f.entry);
    assert(b.kind == 4 && b.trap_mask == 6 && b.check == b.last);
    assert(m::operation_at(ref p,b.check).kind == 5);
    let m::Block trap = m::block_at(ref p,b.alternative);
    assert(trap.kind == 6 && trap.trap_mask == 6 && trap.first == 0 && trap.target == 0);
    let m::Block normal = m::block_at(ref p,b.target);
    assert(normal.kind == 1 && m::operation_at(ref p,normal.first).kind == 4);
    assert(m::local_at(ref p,m::edge_at(ref p,f.parameter_start)).parameter);
}

@test
fn void nested_call_arguments_have_separate_ranges() {
    let m::Program p = retained(b"module main; fn i32 id(i32 x){return x;} fn i32 sum(i32 a,i32 b){return a+b;} fn i32 f(){return sum(id(1),id(2));}",0);
    let m::Function f = m::function_at(ref p,3); var usize count = 0; var usize i = f.block_start + 1;
    while (i <= f.block_start + f.block_count) {
        let m::Block b = m::block_at(ref p,i);
        if (b.kind == 5) {
            let m::Target target = m::target_at(ref p,b.callee);
            assert(b.argument_count == target.signature_count && b.target != 0);
            var usize j = 0;
            while (j < b.argument_count) { assert(m::local_at(ref p,m::edge_at(ref p,b.argument_start + j)).function == 3); j += 1; }
            count += 1;
        }
        i += 1;
    }
    assert(count == 3);
}

@test
fn void structured_control_and_defer_fail_closed() {
    assert(!lowered(b"module main; fn i32 f(){return if(true){1}else{2};}",0).is_ok);
    assert(!lowered(b"module main; fn bool f(){return false || true;}",0).is_ok);
    assert(!lowered(b"module main; fn void f(){while(true){break;}}",0).is_ok);
    assert(!lowered(b"module main; fn void f(){defer g();} fn void g(){}",0).is_ok);
    let m::Program empty = retained(b"module main;",0);
    assert(m::function_count(ref empty) == 0 && m::block_count(ref empty) == 0);
}

@test
fn void construction_rejects_unfinished_and_dangling_operations() {
    var state::Context c = state::empty(); c.function = 1;
    var v::Vec<m::Block> blocks = v::empty<m::Block>();
    var v::Vec<m::Operation> operations = v::empty<m::Operation>();
    c.current = state::new_block(ref mut blocks,ref mut c,0,1);
    var bytes::Buffer source = bytes::empty(); assert(bytes::append(ref mut source,b"x").is_ok);
    let Result<m::Program,m::Problem> result = m::assemble(source,bytes::empty(),v::empty<m::Target>(),v::empty<m::Signature>(),v::empty<m::Function>(),v::empty<m::Local>(),blocks,operations,v::empty<usize>(),0);
    assert(!result.is_ok);
    var state::Context invalid = state::empty();
    var v::Vec<m::Block> none = v::empty<m::Block>(); var v::Vec<m::Operation> ops = v::empty<m::Operation>();
    assert(state::emit(ref mut ops,ref mut none,ref mut invalid,state::operation(1,1,1,1,0,0,0,0,1)) == 0);
    assert(invalid.problem.code == 4 && v::len<m::Operation>(ref ops) == 0);
}
