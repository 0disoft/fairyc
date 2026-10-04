module main;
import tokens as tokens;
import std::vec as vec;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

@effects(console)
fn i32 error() { discard io::write_stderr("invalid ASCII input, integer overflow, token limit or allocation failure\n"); return 2; }
@effects(console)
fn void row(ref str label, usize count) { discard io::write(label); discard io::write_usize(count); discard io::write("\n"); }
@effects(console)
fn i32 analyze_source(ref u8[] source) {
    var vec::Vec<tokens::Token> owned = vec::empty<tokens::Token>();
    if (!tokens::scan(source, ref mut owned)) { return error(); }
    let ref mut tokens::Token[] writable = vec::view_mut<tokens::Token>(ref mut owned);
    tokens::rebase_segment(writable);
    let ref tokens::Token[] view = vec::view<tokens::Token>(ref owned);
    assert(view.is_empty || view[0].position.start == 0);
    let tokens::Stats all = switch(tokens::analyze(view)) {case .ok(s): {s} case .err(_): {return error();}};
    var usize first_end = 0;
    for (let tokens::Token token in view) {
        first_end += 1;
        if (token.kind == 5) { break; }
    }
    let ref tokens::Token[] first = view[..first_end];
    let tokens::Stats section = switch(tokens::analyze(first)) {case .ok(s): {s} case .err(_): {return error();}};
    let ref tokens::Token[] empty = view[view.len..];
    assert(empty.is_empty && empty.len == 0);
    let ref tokens::Token[] rest = tokens::tail(ref owned, first_end);
    assert(rest.len + first_end == view.len);
    let Option<ref tokens::Token[]> tagged = tokens::tagged(ref owned);
    switch(tagged) {case .some(alias): {assert(alias.len == view.len);} case .none: {assert(false);}};
    var bytes::Buffer sum = bytes::empty();
    if (!switch(bytes::append_i64(ref mut sum, all.sum)) {case .ok(_): {true} case .err(_): {false}}) {return error();}
    row("tokens\t", view.len);
    row("identifiers\t", all.identifiers);
    row("numbers\t", all.numbers);
    discard io::write("sum\t"); discard io::write_bytes(bytes::view(ref sum)); discard io::write("\n");
    row("first_tokens\t", first.len);
    row("first_identifiers\t", section.identifiers);
    row("first_numbers\t", section.numbers);
    return 0;
}

@effects(console)
fn i32 main() {
    let usize argc = process::arg_count();
    if (argc == 1) { return analyze_source(b"price = 4000000000 + fee; other = 7;"); }
    if (argc != 2) { discard io::write_stderr("usage: struct-vec-view [ASCII-source]\n"); return 2; }
    var bytes::Buffer input = bytes::empty();
    if (!switch(bytes::resize_zeroed(ref mut input, 4096)) {case .ok(_): {true} case .err(_): {false}}) {return error();}
    let usize length = switch(process::arg(1, bytes::view_mut(ref mut input))) {case .ok(n): {n} case .err(_): {return error();}};
    let ref u8[] source = bytes::view(ref input)[..length];
    return analyze_source(source);
}
