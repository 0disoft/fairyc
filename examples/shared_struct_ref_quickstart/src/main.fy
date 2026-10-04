module main;
import tokens as tokens;
import std::vec as vec;
import std::io as io;

@effects(console)
fn bool print_token(ref tokens::Token token) {
    return io::write("kind=").is_ok && io::write_u32(token.kind).is_ok
        && io::write(" value=").is_ok && io::write_i64(token.value).is_ok
        && io::write(" span=").is_ok && io::write_usize(token.position.start).is_ok
        && io::write("..").is_ok && io::write_usize(token.position.end).is_ok
        && io::write("\n").is_ok;
}

@effects(console)
fn i32 main() {
    var vec::Vec<tokens::Token> owner = vec::empty<tokens::Token>();
    for (var u32 kind = 1; kind <= 3; kind += 1) {
        let usize start = kind as usize;
        if (!vec::push<tokens::Token>(ref mut owner, tokens::Token {
            .kind = kind, .value = 4000000000 + kind as i64,
            .position = tokens::Span { .start = start, .end = start + 10 }
        }).is_ok) { return 1; }
    }
    let ref tokens::Token[] items = vec::view<tokens::Token>(ref owner);
    // The subview still points at the owner's original token storage.
    let ref tokens::Token[] tail = items[1..];
    switch (tokens::find(tail, 2)) {
        case .some(token): { if (!print_token(token)) { return 2; } }
        case .none: { return 3; }
    };
    switch (tokens::find(items, 99)) {
        case .some(_): { return 4; }
        case .none: {}
    };
    // References have reached their last use; the owner can now be cleared.
    assert(vec::clear<tokens::Token>(ref mut owner) == 3);
    assert(vec::len<tokens::Token>(ref owner) == 0);
    return 0;
}
