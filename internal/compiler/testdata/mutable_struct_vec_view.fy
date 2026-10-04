module main;
import std::vec as vec;

struct Span { usize start; usize end; }
struct Token { u32 kind; Span position; i64 value; bool active; }

fn Token token(i64 value) {
    return Token { .kind = 2, .position = Span { .start = 3, .end = 8 }, .value = value, .active = true };
}
fn i64 read(ref Token[] values) { return values[0].value; }

fn i32 main() {
    var vec::Vec<Token> owner = vec::empty<Token>();
    assert(switch(vec::push<Token>(ref mut owner, token(4000000000))) {
        case .ok(_): { true } case .err(_): { false }
    });
    assert(switch(vec::push<Token>(ref mut owner, token(7))) {
        case .ok(_): { true } case .err(_): { false }
    });
    let ref mut Token[] view = vec::view_mut<Token>(ref mut owner);
    assert(view.len == 2 && !view.is_empty);
    let Token old = view[0];
    var Token replacement = token(5000000000);
    view[0] = replacement;
    replacement.value = 9;
    assert(view[0].value == 5000000000 && view[0].position.end == 8 && view[0].active);
    assert(old.value == 4000000000);
    view[1] = view[0];
    assert(view[1].value == 5000000000);
    view[1] = token(-7);
    assert(view[1].value == -7 && read(view) == 5000000000);
    let ref Token[] shared = vec::view<Token>(ref owner);
    assert(shared[0].value == 5000000000 && shared[1].value == -7);
    return 0;
}
