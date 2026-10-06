module tests::hir_nominal;
import hir_model as h;
import hir_shapes as shapes;
import hir_lower as lower;
import hir_output as output;
import mir_lower as mir;
import name_resolution as names;
import type_check as typing;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub fn h::Program retained(ref u8[] source) {
    var bytes::Buffer input = bytes::empty(); assert(bytes::append(ref mut input,source).is_ok);
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) { case .ok(value): { value } case .err(_): { assert(false); return retained(b"module main;"); } };
    let syntax::Parsed parsed = switch(syntax::parse(stream,512)) { case .ok(value): { value } case .err(_): { assert(false); return retained(b"module main;"); } };
    return switch(lower::lower(typing::check(names::resolve(parsed)),0)) { case .ok(value): { value } case .err(_): { assert(false); return retained(b"module main;"); } };
}
@test
fn void owned_layout_and_written_initializers_survive_input() {
    let ref u8[] source = b"module main;struct Pair{i64 x;bool y;}struct Box{Pair pair;}fn Box id(Box b){return b;}fn i64 main(){var b=Box{.pair=Pair{.y=true,.x=4000000000}};b.pair.x+=2;let c=id(b);return c.pair.x;}";
    let h::Program p = retained(source);
    assert(bytes::equal(h::source_view(ref p),source)); let ref shapes::Table table = h::nominal(ref p);
    assert(shapes::count(table) == 2 && shapes::field_count(table) == 3);
    assert(shapes::at(table,0).type == 257 && shapes::at(table,1).type == 258);
    assert(shapes::field_at(table,3).type == 257 && shapes::field_at(table,3).owner == 258);
    assert(h::type_code(h::binding_at(ref p,h::function_at(ref p,0).binding).type) == 258);
    var usize id = 1; var usize constructions = 0; var usize reads = 0;
    while (id <= h::expression_count(ref p)) {
        let h::Expression e = h::expression_at(ref p,id);
        if (h::expression_code(e.kind) == 9) {
            constructions += 1;
            if (h::type_code(e.type) == 257) {
                assert(e.edge_count == 2);
                let h::Expression y = h::expression_at(ref p,h::edge_at(ref p,e.edge_start));
                let h::Expression x = h::expression_at(ref p,h::edge_at(ref p,e.edge_start + 1));
                assert(y.binding == 2 && x.binding == 1 && h::expression_code(y.kind) == 11);
                assert(bytes::equal(h::integer_text(ref p,x.left),b"4000000000"));
            }
        } else if (h::expression_code(e.kind) == 10) { reads += 1; assert(e.binding != 0 && e.left < id); }
        id += 1;
    }
    assert(constructions == 2 && reads == 4);
    var bytes::Buffer first = bytes::empty(); var bytes::Buffer second = bytes::empty();
    assert(output::render(ref p,ref mut first) == 0 && output::render(ref p,ref mut second) == 0);
    assert(bytes::equal(bytes::view(ref first),bytes::view(ref second)));
    assert(bytes::equal(bytes::view(ref first)[..22],b"fairyc.nominal-hir/v1\n"));
    assert(switch(mir::lower(retained(source))) { case .ok(_): { true } case .err(_): { false } });
    let h::Program empty = retained(b"module main;struct Empty{}fn Empty make(){return Empty{};}");
    assert(shapes::count(h::nominal(ref empty)) == 1 && shapes::field_count(h::nominal(ref empty)) == 0);
}
fn shapes::Table malformed(u32 owner, u32 type) {
    var v::Vec<shapes::Shape> records = v::empty<shapes::Shape>(); var v::Vec<shapes::Field> fields = v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape { .type = 257, .binding = 1, .field_start = 0, .field_count = 1, .public = false }).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field { .owner, .type, .start = 1, .end = 2 }).is_ok);
    return shapes::assemble(records,fields);
}
@test
fn void malformed_layouts_are_rejected_before_indexing() {
    let shapes::Table wrong_owner = malformed(258,1); assert(!shapes::valid(ref wrong_owner,10,1));
    let shapes::Table missing = malformed(257,258); assert(!shapes::valid(ref missing,10,1));
    let shapes::Table cyclic = malformed(257,257); assert(!shapes::valid(ref cyclic,10,1));
    let shapes::Table valid = malformed(257,1); assert(shapes::valid(ref valid,10,1));
    assert(!shapes::valid(ref valid,1,1) && !shapes::valid(ref valid,10,0));
    assert(!h::represented_type(.Nominal { .id = 1 }) && !h::represented_type(.Nominal { .id = 768 }));
}
@test
fn void duplicate_declaration_identity_cannot_enter_program() {
    var bytes::Buffer source = bytes::empty(); assert(bytes::append(ref mut source,b"AB").is_ok);
    var v::Vec<h::Binding> bindings = v::empty<h::Binding>();
    var usize i = 0;
    while (i < 2) {
        assert(v::push<h::Binding>(ref mut bindings,h::Binding { .kind = .Struct, .type = .Nominal { .id = 257 }, .start = i, .end = i + 1, .owner = 0, .scope = 0, .module_index = 0, .declaration = 2, .signature_start = 0, .signature_count = 0 }).is_ok); i += 1;
    }
    var v::Vec<shapes::Shape> records = v::empty<shapes::Shape>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape { .type = 257, .binding = 1, .field_start = 0, .field_count = 0, .public = false }).is_ok);
    let shapes::Table table = shapes::assemble(records,v::empty<shapes::Field>());
    let u32 code = switch(h::assemble_nominal(source,bytes::empty(),v::empty<h::Function>(),bindings,v::empty<h::Expression>(),v::empty<h::Statement>(),v::empty<h::Block>(),v::empty<usize>(),v::empty<h::Signature>(),0,table)) { case .ok(_): { 0 } case .err(problem): { problem.code } };
    assert(code == 4);
}
