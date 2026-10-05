module tests::bindings;
import name_resolution as names;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

@test
fn void retained_symbol_links() {
    var bytes::Buffer input = bytes::empty();
    assert(bytes::append(ref mut input,b"module main; fn i64 f(i64 x) { let i64 y=x; { let i64 y=y+1; y; } return y; }").is_ok);
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) {
        case .ok(value): { value } case .err(_): { assert(false); return; }
    };
    let syntax::Parsed parsed = switch(syntax::parse(stream,512)) {
        case .ok(value): { value } case .err(_): { assert(false); return; }
    };
    let names::Resolved resolved = names::resolve(parsed);
    assert(names::problem(ref resolved).code == 0);
    assert(names::symbol_count(ref resolved) == 4 && names::reference_count(ref resolved) == 4);
    let ref syntax::Parsed tree = names::parsed(ref resolved);
    var usize id = 1;
    while (id <= names::symbol_count(ref resolved)) {
        let names::Symbol symbol = names::symbol_at(ref resolved,id);
        if (symbol.kind == 1) {
            let syntax::Declaration item = syntax::declaration_at(tree,symbol.declaration - 1);
            assert(symbol.entry == 0 && item.name_start == symbol.start && item.name_end == symbol.end && item.result_type == symbol.type_root);
        } else {
            let syntax::Entry item = syntax::entry_at(tree,symbol.entry - 1);
            assert(symbol.declaration == 0 && item.name_start == symbol.start && item.name_end == symbol.end && item.type_root == symbol.type_root);
        }
        id += 1;
    }
    var usize index = 0;
    while (index < names::reference_count(ref resolved)) {
        let names::Reference use = names::reference_at(ref resolved,index);
        let syntax::Expression expression = syntax::expression_at(tree,use.expression);
        assert(expression.kind == 2 && expression.start == use.start && expression.end == use.end);
        index += 1;
    }
    assert(names::reference_at(ref resolved,0).symbol == 2);
    assert(names::reference_at(ref resolved,1).symbol == 3);
    assert(names::reference_at(ref resolved,2).symbol == 4);
    assert(names::reference_at(ref resolved,3).symbol == 3);
    assert(names::symbol_at(ref resolved,3).scope != names::symbol_at(ref resolved,4).scope);
}
