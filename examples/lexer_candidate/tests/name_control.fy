module tests::name_control;
import name_resolution as names;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

@test
fn void loop_and_branch_scope_links() {
    var bytes::Buffer input = bytes::empty();
    assert(bytes::append(ref mut input,b"module main; fn i64 f(i64 i) { for(var i64 i=i; i<3; i+=1) { let i64 i=i+1; if(i>0) { i; } else { i; } } return i; }").is_ok);
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) {
        case .ok(value): { value } case .err(_): { assert(false); return; }
    };
    let syntax::Parsed parsed = switch(syntax::parse(stream,512)) {
        case .ok(value): { value } case .err(_): { assert(false); return; }
    };
    let names::Resolved resolved = names::resolve(parsed);
    assert(names::problem(ref resolved).code == 0 && names::symbol_count(ref resolved) == 4);
    assert(names::reference_count(ref resolved) == 8);
    let usize[8] expected = [2,3,3,3,4,4,4,2];
    var usize index = 0;
    while (index < 8) { assert(names::reference_at(ref resolved,index).symbol == expected[index]); index += 1; }
    assert(names::symbol_at(ref resolved,2).scope != names::symbol_at(ref resolved,3).scope);
    assert(names::symbol_at(ref resolved,3).scope != names::symbol_at(ref resolved,4).scope);
}
