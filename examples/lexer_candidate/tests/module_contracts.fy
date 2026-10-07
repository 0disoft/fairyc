module tests::module_contracts;
import module_registry as modules;
import module_cli as cli;
import name_resolution as names;
import type_check as typing;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] source) {
    var bytes::Buffer input = bytes::empty(); assert(bytes::append(ref mut input,source).is_ok);
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) { case .ok(value): { value } case .err(_): { assert(false); return parsed(b"module main;"); } };
    let syntax::Parsed result = switch(syntax::parse(stream,512)) { case .ok(value): { value } case .err(_): { assert(false); return parsed(b"module main;"); } };
    assert(syntax::problem_count(ref result) == 0); return result;
}
fn modules::Registry registry(ref u8[] first, ref u8[] second) {
    var v::Vec<syntax::Parsed> files = v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut files,parsed(first)).is_ok);
    assert(v::push<syntax::Parsed>(ref mut files,parsed(second)).is_ok);
    return switch(modules::build(ref files)) { case .ok(value): { value } case .err(_): { assert(false); return registry(b"module a;",b"module b;"); } };
}
fn u32 rejection(ref u8[] first, ref u8[] second) {
    var v::Vec<syntax::Parsed> files = v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut files,parsed(first)).is_ok);
    assert(v::push<syntax::Parsed>(ref mut files,parsed(second)).is_ok);
    return switch(modules::build(ref files)) { case .ok(_): { 0 } case .err(problem): { problem.code } };
}
fn u32 typed_error(ref u8[] first, ref u8[] second) {
    let modules::Registry source_set = registry(first,second);
    let names::Resolved resolved = names::resolve_modules(parsed(first),ref source_set,0);
    assert(names::problem(ref resolved).code == 0);
    let typing::Typed typed = typing::check(resolved); return typing::problem(ref typed).code;
}
fn names::Resolved resolved_after_registry_drop(ref u8[] first, ref u8[] second) {
    let modules::Registry source_set = registry(first,second);
    return names::resolve_modules(parsed(first),ref source_set,0);
}

@test fn void imported_contracts_check_actual_targets(){
    let ref u8[] provider=b"module provider;@must_use @effects(console)pub fn i32 token(){return 7;}";
    assert(typed_error(b"module consumer;import provider as p;@effects(console)fn i32 main(){return p::token();}",provider)==0);
    assert(typed_error(b"module consumer;import provider as p;fn i32 main(){return p::token();}",provider)==7102);
    assert(typed_error(b"module consumer;import provider as p;@effects(console)fn i32 main(){p::token();return 0;}",provider)==4202);
    assert(typed_error(b"module consumer;import provider as p;@effects(console)fn i32 main(){discard p::token();return 0;}",provider)==0);
    let r=resolved_after_registry_drop(b"module consumer;import provider as p;@effects(console)fn i32 main(){return p::token();}",provider);
    var usize id=1;var bool found=false;while(id<=names::symbol_count(ref r)){let s=names::symbol_at(ref r,id);if(s.kind==5){let signature=names::external_signature(ref r,id);assert(signature.effects==1 && signature.must_use && signature.module_index==1);found=true;}id+=1;}assert(found);
}
@test fn void imported_metadata_is_validated_not_ignored(){
    assert(rejection(b"module consumer;import provider as p;fn i32 main(){return 0;}",b"module provider;@effects(unknown)pub fn i32 token(){return 7;}")==7101);
    assert(rejection(b"module consumer;import provider as p;fn i32 main(){return 0;}",b"module provider;@must_use pub fn void token(){}")==4201);
    assert(typed_error(b"module consumer;import provider as p;@effects(storage_write)fn i32 main(){return p::token();}",b"module provider;@effects(storage_read)pub fn i32 token(){return 7;}")==7102);
}
