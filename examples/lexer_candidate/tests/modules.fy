module tests::modules;
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

@test
fn void retained_import_links_and_types() {
    let ref u8[] main_source = b"module main; import util /* path */ :: math as lib; fn i64 f(i64 x){ let lib=1; return lib /* call */ :: identity(1+x); }";
    let ref u8[] library = b"module util::math; fn i32 helper(){return 0;} pub fn i64 identity(i64 x){return x;}";
    let modules::Registry source_set = registry(main_source,library);
    assert(modules::module_count(ref source_set) == 2);
    // This helper's registry has already left scope; target IDs and copied
    // scalar signatures must remain owned by Resolved and then Typed.
    let names::Resolved resolved = resolved_after_registry_drop(main_source,library);
    assert(names::problem(ref resolved).code == 0);
    var usize index = 0; var usize external_symbol = 0;
    while (index < names::reference_count(ref resolved)) {
        let names::Reference use = names::reference_at(ref resolved,index);
        assert(use.symbol > 0 && use.expression > 0);
        let names::Symbol symbol = names::symbol_at(ref resolved,use.symbol);
        if (symbol.kind == 5) {
            external_symbol = use.symbol;
            let modules::Item signature = names::external_signature(ref resolved,use.symbol);
            assert(signature.module_index == 1 && signature.declaration == 3 && signature.public);
            assert(signature.result == 2 && signature.parameter_count == 1);
            assert(names::external_parameter(ref resolved,signature.parameter_start) == 2);
            assert(syntax::expression_at(names::parsed(ref resolved),use.expression).kind == 2);
            assert(use.start == symbol.start && use.end == symbol.end);
        }
        index += 1;
    }
    assert(external_symbol != 0);
    let typing::Typed typed = typing::check(resolved); assert(typing::problem(ref typed).code == 0);
    assert(typing::symbol_type(ref typed,external_symbol) == 2);
    let ref syntax::Parsed ast = names::parsed(typing::resolved(ref typed));
    index = 1; var usize calls = 0;
    while (index <= syntax::expression_count(ast)) {
        let syntax::Expression node = syntax::expression_at(ast,index);
        if (node.kind == 6) { calls += 1; assert(typing::expression_type(ref typed,index) == 2); assert(typing::expression_type(ref typed,node.left) == 8); }
        if (node.kind == 5) { assert(typing::expression_type(ref typed,index) == 2); }
        index += 1;
    }
    assert(calls == 1);
}

@test
fn void registry_rejections() {
    assert(rejection(b"module a;",b"module a;") == 1);
    assert(rejection(b"module a; import b as x; import b as x;",b"module b;") == 7);
    assert(rejection(b"module a; import b as x; fn i32 x(){return 0;}",b"module b;") == 2);
    assert(rejection(b"module a; import absent as x;",b"module b;") == 3);
    assert(rejection(b"module a; import b as source_location;",b"module b;") == 9);
    let modules::Registry source_set = registry(b"module a; import b as x;",b"module b; fn i32 private(){return 1;} pub fn i32 visible(){return private();}");
    assert(modules::lookup(ref source_set,0,b"missing::visible").code == 4);
    assert(modules::lookup(ref source_set,0,b"x::absent").code == 5);
    assert(modules::lookup(ref source_set,0,b"x::private").code == 6);
    assert(modules::lookup(ref source_set,0,b"x::visible").code == 0);
}

@test
fn void qualified_call_type_errors() {
    assert(typed_error(b"module a; import b as x; fn i32 f(){return x::f();}",b"module b; pub fn i32 f(i32 y){return y;}") == 5);
    assert(typed_error(b"module a; import b as x; fn i32 f(){return x::f(true);}",b"module b; pub fn i32 f(i32 y){return y;}") == 1);
    assert(typed_error(b"module a; import b as x; fn i32 f(){return x::f();}",b"module b; pub fn bool f(){return true;}") == 1);
    assert(typed_error(b"module a; import b as x; fn u8 f(){return x::f(256);}",b"module b; pub fn u8 f(u8 y){return y;}") == 2);
    let modules::Registry source_set = registry(b"module a; import b as x;",b"module b; pub fn i32 f(){return missing;}");
    let names::Resolved bad_body = names::resolve_modules(parsed(b"module b; pub fn i32 f(){return missing;}"),ref source_set,1);
    assert(names::problem(ref bad_body).code == 1);
}

@test
fn void legacy_single_file_preserved() {
    let names::Resolved imports = names::resolve(parsed(b"module a; import b as x;"));
    assert(names::problem(ref imports).code == 3);
    let names::Resolved qualified = names::resolve(parsed(b"module a; fn i32 f(){return x::f();}"));
    assert(names::problem(ref qualified).code == 3);
    let names::Resolved valid = names::resolve(parsed(b"module a; fn i32 f(i32 x){return 1+x;}"));
    assert(names::problem(ref valid).code == 0);
    let typing::Typed typed = typing::check(valid); assert(typing::problem(ref typed).code == 0);
}

@test
fn void manifest_lines_and_limits() {
    var v::Vec<cli::Path> paths = v::empty<cli::Path>();
    assert(cli::paths(b"a.fy\r\nb.fy\nc.fy",ref mut paths) == 0 && v::len<cli::Path>(ref paths) == 3);
    assert(v::at<cli::Path>(ref paths,0).end == 4 && v::at<cli::Path>(ref paths,1).start == 6);
    discard v::clear<cli::Path>(ref mut paths);
    assert(cli::paths(b"a.fy\n",ref mut paths) == 0 && v::len<cli::Path>(ref paths) == 1);
    discard v::clear<cli::Path>(ref mut paths);
    assert(cli::paths(b"",ref mut paths) == 1);
    assert(cli::paths(b"a.fy\n\nb.fy",ref mut paths) == 1);
    assert(cli::paths(b"/a.fy",ref mut paths) == 1);
    assert(cli::paths(b"C:\\a.fy",ref mut paths) == 1);
    assert(cli::paths(b"a.fy\r",ref mut paths) == 1);
    assert(cli::paths(b"\xef\xbb\xbfa.fy",ref mut paths) == 1);
    assert(cli::paths(b"\xff.fy",ref mut paths) == 1);
    var bytes::Buffer manifest = bytes::empty(); var usize index = 0;
    while (index < 512) { assert(bytes::append(ref mut manifest,b"a.fy\n").is_ok); index += 1; }
    discard v::clear<cli::Path>(ref mut paths);
    assert(cli::paths(bytes::view(ref manifest),ref mut paths) == 0 && v::len<cli::Path>(ref paths) == 512);
    assert(bytes::append(ref mut manifest,b"a.fy\n").is_ok);
    discard v::clear<cli::Path>(ref mut paths);
    assert(cli::paths(bytes::view(ref manifest),ref mut paths) == 5);
}
