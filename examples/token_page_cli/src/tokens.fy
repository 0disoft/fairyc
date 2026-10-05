module tokens;

pub struct Span { usize start; usize end; }
pub struct Token { u32 kind; Span position; i64 value; }
pub struct TokenPage { Token[128] tokens; usize used; usize error_offset; }
pub struct Stats { usize identifiers; usize numbers; i64 sum; }

pub fn TokenPage empty() {
    return TokenPage {
        .tokens = [Token { .kind = 0, .position = Span { .start = 0, .end = 0 }, .value = 0 }; 128],
        .used = 0, .error_offset = 0
    };
}
fn bool letter(u8 c) { return c >= 65 && c <= 90 || c >= 97 && c <= 122 || c == 95; }
fn bool digit(u8 c) { return c >= 48 && c <= 57; }

/// 1 identifier, 2 decimal integer, 3 plus, 4 equal, 5 semicolon.
/// Evaluation only: 6 multiply, 7 opening parenthesis, 8 closing parenthesis.
/// Errors: 1 invalid ASCII, 2 integer overflow, 3 token limit.
pub fn Result<usize, u32> scan(ref u8[] input, ref mut TokenPage page) {
    return scan_mode(input, page, false);
}
pub fn Result<usize, u32> scan_eval(ref u8[] input, ref mut TokenPage page) {
    return scan_mode(input, page, true);
}
fn Result<usize, u32> scan_mode(ref u8[] input, ref mut TokenPage page, bool evaluation) {
    page.used = 0;
    page.error_offset = 0;
    var usize index = 0;
    while (index < input.len) {
        let u8 c = input[index];
        if (c == 32 || c == 9 || c == 10 || c == 13) { index += 1; continue; }
        let usize slot = page.used;
        if (slot == 128) { page.error_offset = index; return .err(3); }
        let usize start = index;
        var u32 kind = 0;
        var i64 value = 0;
        if (letter(c)) {
            kind = 1; index += 1;
            while (index < input.len && (letter(input[index]) || digit(input[index]))) { index += 1; }
        } else if (digit(c)) {
            kind = 2;
            while (index < input.len && digit(input[index])) {
                let i64 next = (input[index] - 48) as i64;
                if (value > 922337203685477580 || value == 922337203685477580 && next > 7) { page.error_offset = index; return .err(2); }
                value = value * 10 + next; index += 1;
            }
        } else {
            kind = if (c == 43) { 3 } else if (c == 61) { 4 } else if (c == 59) { 5 } else { 0 };
            if (evaluation && kind == 0) {
                kind = if (c == 42) { 6 } else if (c == 40) { 7 } else if (c == 41) { 8 } else { 0 };
            }
            if (kind == 0) { page.error_offset = index; return .err(1); }
            index += 1;
        }
        // Snapshot the owner metadata before holding its exclusive element loan.
        page.tokens[slot] = Token { .kind, .position = Span { .start, .end = index }, .value };
        page.used = slot + 1;
    }
    return .ok(page.used);
}

/// Normalize retained offsets without reading or copying the whole array field.
pub fn void rebase(ref mut TokenPage page) {
    let usize used = page.used;
    if (used == 0) { return; }
    let usize origin = page.tokens[0].position.start;
    page.tokens[0].position.start -= origin;
    page.tokens[0].position.end -= origin;
    for (let ref mut Token token in page.tokens[1..used]) {
        token.position.start -= origin;
        token.position.end -= origin;
    }
}

pub fn Result<Stats, u32> analyze(ref Token[] segment) {
    var Stats stats = Stats { .identifiers = 0, .numbers = 0, .sum = 0 };
    for (let ref Token token in segment) {
        assert(token.position.start < token.position.end);
        if (token.kind == 1) { stats.identifiers += 1; }
        if (token.kind == 2) {
            if (token.value > 9223372036854775807 - stats.sum) { return .err(4); }
            stats.numbers += 1; stats.sum += token.value;
        }
    }
    return .ok(stats);
}

pub fn usize first_end(ref Token[] segment) {
    var usize count = 0;
    for (let ref Token token in segment) {
        count += 1;
        if (token.kind == 5) { break; }
    }
    return count;
}

pub fn ref Token[] valid(ref TokenPage page) {
    let usize used = page.used;
    return ref page.tokens[..used];
}
