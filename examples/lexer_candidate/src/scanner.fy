module scanner;
import std::vec as v;
import std::io as io;
import std::bytes as bytes;
import numbers as numbers;
import diagnostics as diagnostics;
import utf8 as utf8;
import strings as strings;

// Private numeric observations mirror internal/lexer/token.go, not a stable ABI.
resource Token {
    u32 kind; u32 trivia; usize start; usize end;
    bytes::Buffer cooked;
    cleanup { }
}

fn bool emit(ref mut v::Vec<Token> tokens, bool admitted, u32 kind, u32 trivia, usize start, usize end) {
    if (!admitted) { return append(tokens, 0, 0, start, end); }
    return append(tokens, kind, trivia, start, end);
}

fn bool emit_cooked(ref mut v::Vec<Token> tokens, bool admitted, u32 kind, usize start, usize end, bytes::Buffer cooked) {
    if (!admitted) { return append(tokens, 0, 0, start, end); }
    return append_cooked(tokens, kind, 0, start, end, cooked);
}

fn i32 exhausted(ref u8[] input, usize position, ref mut v::Vec<Token> tokens, ref mut v::Vec<diagnostics::Entry> errors) {
    let i32 status = diagnostics::add(errors, 10, position, position, false, b"");
    if (status != 0) { return status; }
    if (position < input.len && !append(tokens, 0, 0, position, input.len)) { return 3; }
    if (!append(tokens, 1, 0, input.len, input.len)) { return 3; }
    return 0;
}

fn bool append(ref mut v::Vec<Token> tokens, u32 kind, u32 trivia, usize start, usize end) {
    return append_cooked(tokens, kind, trivia, start, end, bytes::empty());
}

fn bool append_cooked(ref mut v::Vec<Token> tokens, u32 kind, u32 trivia, usize start, usize end, bytes::Buffer cooked) {
    let Token value = Token { .kind = kind, .trivia = trivia, .start = start, .end = end, .cooked = cooked };
    return switch (v::push<Token>(tokens, value)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
}

fn bool matches(ref u8[] input, usize position, ref u8[] word) {
    if (position > input.len || word.len > input.len - position) { return false; }
    var usize index = 0;
    while (index < word.len) {
        if (input[position + index] != word[index]) { return false; }
        index += 1;
    }
    return true;
}

fn bool letter(u8 value) {
    return value == 95 || (value >= 65 && value <= 90) || (value >= 97 && value <= 122);
}

fn bool digit(u8 value) { return value >= 48 && value <= 57; }

fn bool space(u8 value) { return value == 32 || value == 9 || value == 10; }

fn bool keyword(ref u8[] input, usize start, usize end, bool previous) {
    if (previous && end - start == 5 && matches(input, start, b"defer")) { return false; }
    let ref u8[] words = b"as assert break case cleanup const continue discard else enum extern false fn for if import in let module mut null pub ptr ref region resource return struct switch true unsafe var while defer";
    var usize first = 0;
    while (first < words.len) {
        var usize last = first;
        while (last < words.len && words[last] != 32) { last += 1; }
        if (last - first == end - start && matches(input, start, words[first..last])) { return true; }
        first = last + 1;
    }
    return false;
}

fn usize symbol_width(ref u8[] input, usize start) {
    let ref u8[] words = b"<<= >>= += -= *= /= %= &= ^= |= :: -> => == != <= >= << >> && || .. @ ( ) { } [ ] , ; : . ? = + - * / % ! ~ & | ^ < >";
    var usize first = 0;
    var usize best = 0;
    while (first < words.len) {
        var usize last = first;
        while (last < words.len && words[last] != 32) { last += 1; }
        let usize width = last - first;
        if (width > best && matches(input, start, words[first..last])) { best = width; }
        first = last + 1;
    }
    return best;
}

// 0 success; 2 not implemented input; 3 allocation failure; 4 candidate size limit.
// No partial token stream escapes on any failure. Full Go budget parity is pending.
fn i32 scan(ref u8[] input, ref mut v::Vec<Token> tokens, ref mut v::Vec<diagnostics::Entry> errors, bool previous, usize max_bytes, usize max_tokens) {
    if (input.len > 4096) { return 4; }
    if (input.len > max_bytes) { return exhausted(input, 0, tokens, errors); }
    var usize token_count = 0;
    var bool token_exhausted = false;
    var usize position = 0;
    var usize cooked_bytes = 0;
    if (matches(input, 0, b"\xEF\xBB\xBF")) {
        token_exhausted = max_tokens == 0;
        if (!emit(tokens, !token_exhausted, 2, 1, 0, 3)) { return 3; }
        if (!token_exhausted) { token_count += 1; }
        position = 3;
    }
    while (position < input.len && !token_exhausted) {
        token_exhausted = token_count >= max_tokens;
        if (!token_exhausted) { token_count += 1; }
        let usize start = position;
        let u8 current = input[position];
        var u32 kind = 0;
        var u32 trivia = 0;
        var bool unclosed_comment = false;
        if (space(current) || current == 13) {
            while (position < input.len) {
                if (space(input[position])) { position += 1; }
                else if (matches(input, position, b"\r\n")) { position += 2; }
                else { break; }
            }
            if (position == start) {
                position += 1;
                if (!emit(tokens, !token_exhausted, 0, 0, start, position)) { return 3; }
                if (token_exhausted) { break; }
                let i32 status = diagnostics::add(errors, 1, start, position, false, b"");
                if (status != 0) { return status; }
                continue;
            }
            kind = 2; trivia = 2;
        } else if (matches(input, position, b"//")) {
            trivia = if (matches(input, position, b"///")) { 5 } else { 3 };
            position += 2;
            while (position < input.len && input[position] != 10 && input[position] != 13) { position += 1; }
            kind = 2;
        } else if (matches(input, position, b"/*")) {
            position += 2;
            var usize depth = 1;
            while (position < input.len && depth > 0) {
                if (matches(input, position, b"/*")) { depth += 1; position += 2; }
                else if (matches(input, position, b"*/")) { depth -= 1; position += 2; }
                else { position += 1; }
            }
            unclosed_comment = depth != 0;
            kind = 2; trivia = 4;
        } else if (current == 34 || matches(input, position, b"b\"") || matches(input, position, b"c\"")) {
            kind = if (current == 34) { 7 } else if (current == 98) { 8 } else { 9 };
            let usize opener = if (current == 34) { 1 } else { 2 };
            var bytes::Buffer cooked = bytes::empty();
            let strings::Step step = strings::scan(input, start, kind, opener, ref mut cooked, errors);
            if (step.status != 0) { return step.status; }
            position = step.end;
            let usize length = bytes::len(ref cooked);
            let usize cooked_limit = 65536;
            if (length > cooked_limit - cooked_bytes) { return 4; }
            cooked_bytes += length;
            if (!emit_cooked(tokens, !token_exhausted, kind, start, position, cooked)) { return 3; }
            continue;
        } else if (letter(current)) {
            position += 1;
            while (position < input.len && (letter(input[position]) || digit(input[position]))) { position += 1; }
            kind = if (keyword(input, start, position, previous)) { 4 } else { 3 };
        } else if (digit(current)) {
            var bytes::Buffer cooked = bytes::empty();
            let numbers::Scan result = numbers::scan(input, start, ref mut cooked);
            position = result.end; kind = result.kind;
            let i32 status = result.status;
            if (status == 3 || status == 4) { return status; }
            let usize length = bytes::len(ref cooked);
            let usize cooked_limit = 65536;
            if (length > cooked_limit - cooked_bytes) { return 4; }
            cooked_bytes += length;
            if (!emit_cooked(tokens, !token_exhausted, kind, start, position, cooked)) { return 3; }
            if (status == 1 && !token_exhausted) {
                let i32 added = diagnostics::add(errors, 4, start, position, false, b"");
                if (added != 0) { return added; }
            }
            continue;
        } else {
            let usize width = symbol_width(input, position);
            if (width == 0) {
                let usize decoded = utf8::width(input, position);
                position += if (decoded == 0) { 1 } else { decoded };
                if (!emit(tokens, !token_exhausted, 0, 0, start, position)) { return 3; }
                if (token_exhausted) { break; }
                let i32 status = diagnostics::add(errors, 1, start, position, false, b"");
                if (status != 0) { return status; }
                continue;
            } else { position += width; kind = 10; }
        }
        assert(position > start && position <= input.len);
        if (!emit(tokens, !token_exhausted, kind, trivia, start, position)) { return 3; }
        if (unclosed_comment && !token_exhausted) {
            let i32 status = diagnostics::add(errors, 1, start, position, true, b"*/");
            if (status != 0) { return status; }
        }
    }
    if (token_exhausted) { return exhausted(input, position, tokens, errors); }
    if (!append(tokens, 1, 0, position, position)) { return 3; }
    return 0;
}

@effects(console)
fn void number(usize value) {
    assert(switch (io::write_usize(value)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (io::write(" ")) { case .ok(_) : { true } case .err(_) : { false } });
}

@effects(console)
pub fn i32 report(ref u8[] input) {
    return report_mode(input, false);
}

@effects(console)
pub fn i32 report_mode(ref u8[] input, bool previous) {
    return report_limits(input, previous, 4096, 1048576);
}

@effects(console)
pub fn i32 report_limits(ref u8[] input, bool previous, usize max_bytes, usize max_tokens) {
    var v::Vec<Token> tokens = v::empty<Token>();
    var v::Vec<diagnostics::Entry> errors = v::empty<diagnostics::Entry>();
    let i32 status = scan(input, ref mut tokens, ref mut errors, previous, max_bytes, max_tokens);
    if (status != 0) { return status; }
    var usize index = 0;
    var usize position = 0;
    while (index < v::len<Token>(ref tokens)) {
        let ref Token token = v::at<Token>(ref tokens, index);
        assert(token.start == position && token.end >= token.start && token.end <= input.len);
        number(token.kind as usize); number(token.trivia as usize);
        number(token.start); number(token.end);
        let usize width = token.end - token.start;
        let bool cooked = token.kind == 4 && ((width == 4 && (matches(input, token.start, b"true") || matches(input, token.start, b"null"))) || (width == 5 && matches(input, token.start, b"false")));
        let ref u8[] numeric = bytes::view(ref token.cooked);
        number(if (cooked) { width } else { numeric.len });
        if (cooked) {
            var usize byte_index = token.start;
            while (byte_index < token.end) { number(input[byte_index] as usize); byte_index += 1; }
        }
        var usize n = 0;
        while (n < numeric.len) { number(numeric[n] as usize); n += 1; }
        position = token.end;
        index += 1;
    }
    assert(position == input.len);
    diagnostics::report(ref errors);
    return 0;
}
