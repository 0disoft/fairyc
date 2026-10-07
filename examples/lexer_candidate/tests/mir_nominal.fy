module tests::mir_nominal;
import tests::hir_nominal as fixture;
import hir_shapes as shapes;
import mir_model as m;
import mir_lower as lower;
import mir_state as state;
import mir_verify as verify;
import mir_output as output;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn m::Program compiled(ref u8[] source) {
    return switch(lower::lower(fixture::retained(source))) { case .ok(p): { p } case .err(_): { assert(false); return compiled(source); } };
}
@test
fn void nominal_lowering_owns_shapes_and_ordered_operands() {
    let m::Program p = compiled(b"module main;struct P{i64 x;bool y;}struct B{P p;}fn B id(B b){return b;}fn i32 main(){var b=B{.p=P{.y=true,.x=4000000000}};b.p.x+={b.p.y=false;2};let c=id(b);assert(c.p.x==4000000002);return 0;}");
    assert(verify::check(ref p).code == 0 && shapes::count(m::nominal(ref p)) == 2);
    var usize id = 1; var usize constructs = 0; var usize reads = 0; var usize updates = 0;
    while (id <= m::operation_count(ref p)) {
        let m::Operation o = m::operation_at(ref p,id);
        if (o.kind == 6) {
            constructs += 1;
            if (o.type == 257) {
                assert(o.argument_count == 2);
                assert(m::local_at(ref p,m::edge_at(ref p,o.argument_start)).type == 2);
                assert(m::local_at(ref p,m::edge_at(ref p,o.argument_start + 1)).type == 6);
            }
        } else if (o.kind == 7) { reads += 1; } else if (o.kind == 8) { updates += 1; }
        id += 1;
    }
    assert(constructs == 2 && reads > 0 && updates == 4);
    var bytes::Buffer first = bytes::empty(); var bytes::Buffer second = bytes::empty();
    assert(output::render(ref p,ref mut first) == 0 && output::render(ref p,ref mut second) == 0);
    assert(bytes::equal(bytes::view(ref first),bytes::view(ref second)) && bytes::equal(bytes::view(ref first)[..22],b"fairyc.nominal-mir/v1\n"));
    var bytes::Buffer c_output = bytes::empty(); assert(c::render(ref p,ref mut c_output) == 0);
    let m::Program empty = compiled(b"module main;struct E{}fn E id(E x){return x;}fn i32 main(){discard id(E{});return 0;}");
    assert(verify::check(ref empty).code == 0 && shapes::field_count(m::nominal(ref empty)) == 0);
    var bytes::Buffer empty_c = bytes::empty(); assert(c::render(ref empty,ref mut empty_c) == 0);
}

fn m::Local local(u32 type) { return m::Local { .function = 1, .binding = 0, .type, .parameter = false, .mutable = true, .start = 0, .end = 1 }; }
fn m::Program mutated(u32 mutation) {
    var bytes::Buffer source = bytes::empty(); assert(bytes::append(ref mut source,b"fx").is_ok);
    var bytes::Buffer payload = bytes::empty(); assert(bytes::append(ref mut payload,b"1").is_ok);
    var v::Vec<shapes::Shape> records = v::empty<shapes::Shape>(); var v::Vec<shapes::Field> fields = v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape { .type = 257, .binding = 1, .field_start = 0, .field_count = 1, .public = false }).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field { .owner = 257, .type = 1, .start = 1, .end = 2, .payload_start = 0, .payload_count = 0 }).is_ok);
    var v::Vec<m::Target> targets = v::empty<m::Target>();
    assert(v::push<m::Target>(ref mut targets,m::Target { .unsafe_call=false, .effects = 0, .binding = 2, .module_index = 0, .declaration = 1, .result = 1, .signature_start = 0, .signature_count = 0, .start = 0, .end = 1, .external = false }).is_ok);
    var v::Vec<m::Function> functions = v::empty<m::Function>();
    assert(v::push<m::Function>(ref mut functions,m::Function { .target = 1, .entry = 1, .local_start = 0, .local_count = 5, .block_start = 0, .block_count = 1, .parameter_start = 0, .parameter_count = 0, .start = 0, .end = 2, .public = false }).is_ok);
    var v::Vec<m::Local> locals = v::empty<m::Local>(); var usize i = 0;
    while (i < 5) { assert(v::push<m::Local>(ref mut locals,local(if (i == 2 || i == 4) { 257 } else { 1 })).is_ok); i += 1; }
    var v::Vec<usize> edges = v::empty<usize>(); assert(v::push<usize>(ref mut edges,if (mutation == 1) { 2 } else if (mutation == 2) { 3 } else { 1 }).is_ok);
    if (mutation == 8) { assert(v::push<usize>(ref mut edges,1).is_ok); }
    var v::Vec<m::Operation> ops = v::empty<m::Operation>();
    var m::Operation constant = state::operation(1,1,1,1,0,0,0,0,1);constant.function = 1;constant.block = 1;constant.next = 2;constant.payload_end = 1;
    var m::Operation construct = state::operation(6,3,257,257,0,0,0,0,1);construct.function = 1;construct.block = 1;construct.next = 3;construct.argument_count = if (mutation == 3) { 0 } else { 1 };
    var m::Operation read = state::operation(7,4,1,257,if (mutation == 4) { 2 } else { 1 },3,0,0,1);read.function = 1;read.block = 1;read.next = 4;
    var m::Operation update = state::operation(8,5,257,257,1,3,if (mutation == 5) { 3 } else { 1 },0,1);update.function = 1;update.block = 1;update.next = 5;
    var m::Operation copy = state::operation(2,3,257,257,0,if (mutation == 6) { 1 } else { 5 },0,0,1);copy.function = 1;copy.block = 1;
    if (mutation == 7) { read.argument_count = 1; }
    if (mutation == 9) { construct.boolean = true; }
    if (mutation == 10) { update.owner_type = 1; }
    assert(v::push<m::Operation>(ref mut ops,constant).is_ok);assert(v::push<m::Operation>(ref mut ops,construct).is_ok);assert(v::push<m::Operation>(ref mut ops,read).is_ok);assert(v::push<m::Operation>(ref mut ops,update).is_ok);assert(v::push<m::Operation>(ref mut ops,copy).is_ok);
    var m::Block done = state::block(1,0,2);done.kind = 1;done.value = 4;done.first = 1;done.last = 5;
    var v::Vec<m::Block> blocks = v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    return switch(m::assemble_nominal(source,payload,targets,v::empty<m::Signature>(),functions,locals,blocks,ops,edges,0,shapes::assemble(records,fields))) { case .ok(p): { p } case .err(_): { assert(false);return mutated(0); } };
}
@test
fn void nominal_verifier_rejects_forged_operations_and_uninitialized_fields() {
    let m::Program good = mutated(0);assert(verify::check(ref good).code == 0);
    var u32 mutation = 1;
    while (mutation <= 10) {
        let m::Program p = mutated(mutation);assert(verify::check(ref p).code == if (mutation == 1) { 1002 } else { 1001 });
        var bytes::Buffer out = bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);
        assert(output::render(ref p,ref mut out) != 0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation += 1;
        assert(c::render(ref p,ref mut out) != 0 && bytes::equal(bytes::view(ref out),b"prefix"));
    }
}
