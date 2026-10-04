module tokens;
import std::vec as vec;

pub struct Span { usize start; usize end; }
pub struct Token { u32 kind; Span position; i64 value; }
pub struct Stats { usize identifiers; usize numbers; i64 sum; }
fn bool letter(u8 c) { return c >= 65 && c <= 90 || c >= 97 && c <= 122 || c == 95; }
fn bool digit(u8 c) { return c >= 48 && c <= 57; }

/// A bounded ASCII lexer; token offsets refer to the caller's source bytes.
/// 1 identifier, 2 decimal integer, 3 plus, 4 equal, 5 semicolon.
pub fn bool scan(ref u8[] input, ref mut vec::Vec<Token> output) {
    var usize i = 0;
    while (i < input.len) {
        let u8 c = input[i];
        if (c == 32 || c == 9 || c == 10 || c == 13) { i += 1; continue; }
        if (vec::len<Token>(output) >= 128) { return false; }
        let usize start = i;
        var u32 kind = 0;
        var i64 value = 0;
        if (letter(c)) {
            kind = 1; i += 1;
            while (i < input.len && (letter(input[i]) || digit(input[i]))) { i += 1; }
        } else if (digit(c)) {
            kind = 2;
            while (i < input.len && digit(input[i])) {
                let i64 next = (input[i] - 48) as i64;
                if (value > 922337203685477580 || value == 922337203685477580 && next > 7) { return false; }
                value = value * 10 + next; i += 1;
            }
        } else {
            kind = if (c == 43) { 3 } else if (c == 61) { 4 } else if (c == 59) { 5 } else { 0 };
            if (kind == 0) { return false; }
            i += 1;
        }
        let Token token = Token { .kind = kind, .position = Span { .start = start, .end = i }, .value = value };
        if (!switch(vec::push<Token>(output, token)) {case .ok(_): {true} case .err(_): {false}}) { return false; }
    }
    return true;
}

fn void rebase(ref mut Span position, usize origin) {
    let ref usize start = ref position.start;
    let ref usize end = ref position.end;
    let usize width = *end - *start;
    position.start -= origin;
    position.end -= origin;
    assert(position.end - position.start == width);
}

/// Normalizes the stored spans once; the first token becomes the segment origin.
/// Mutable item/field children retain the stored segment until their last use.
pub fn void rebase_segment(ref mut Token[] segment) {
    let usize origin = if (segment.is_empty) { 0 } else { segment[0].position.start };
    for (let ref mut Token token in segment) {
        let ref mut Span position = ref mut token.position;
        rebase(position, origin);
        assert(position.start < position.end);
    }
}

/// Reads the normalized stored token segment through a shared view.
pub fn Result<Stats, u32> analyze(ref Token[] segment) {
    var Stats stats = Stats { .identifiers = 0, .numbers = 0, .sum = 0 };
    for (let Token token in segment) {
        assert(token.position.start < token.position.end);
        if (token.kind == 1) { stats.identifiers += 1; }
        if (token.kind == 2) {
            if (token.value > 9223372036854775807 - stats.sum) { return .err(1); }
            stats.numbers += 1; stats.sum += token.value;
        }
    }
    return .ok(stats);
}

pub fn ref Token[] tail(ref vec::Vec<Token> owner, usize start) { return vec::view<Token>(owner)[start..]; }
pub fn Option<ref Token[]> tagged(ref vec::Vec<Token> owner) { return .some(vec::view<Token>(owner)[..]); }
