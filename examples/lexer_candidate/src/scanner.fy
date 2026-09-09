module scanner;
import std::vec as v;
import std::io as io;

// Private numeric observations mirror internal/lexer/token.go, not a stable ABI.
resource Token {
    u32 kind; u32 trivia; usize start; usize end;
    cleanup { }
}

fn bool append(ref mut v::Vec<Token> tokens, u32 kind, u32 trivia, usize start, usize end) {
    let Token value = Token { .kind = kind, .trivia = trivia, .start = start, .end = end };
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

fn bool keyword(ref u8[] input, usize start, usize end) {
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
fn i32 scan(ref u8[] input, ref mut v::Vec<Token> tokens) {
    if (input.len > 4096) { return 4; }
    var usize position = 0;
    if (matches(input, 0, b"\xEF\xBB\xBF")) {
        if (!append(tokens, 2, 1, 0, 3)) { return 3; }
        position = 3;
    }
    while (position < input.len) {
        let usize start = position;
        let u8 current = input[position];
        var u32 kind = 0;
        var u32 trivia = 0;
        if (space(current) || current == 13) {
            while (position < input.len) {
                if (space(input[position])) { position += 1; }
                else if (matches(input, position, b"\r\n")) { position += 2; }
                else { break; }
            }
            if (position == start) { return 2; }
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
            if (depth != 0) { return 2; }
            kind = 2; trivia = 4;
        } else if (letter(current)) {
            if (matches(input, position, b"b\"") || matches(input, position, b"c\"")) { return 2; }
            position += 1;
            while (position < input.len && (letter(input[position]) || digit(input[position]))) { position += 1; }
            kind = if (keyword(input, start, position)) { 4 } else { 3 };
        } else {
            let usize width = symbol_width(input, position);
            if (width == 0) { return 2; }
            position += width;
            kind = 10;
        }
        assert(position > start && position <= input.len);
        if (!append(tokens, kind, trivia, start, position)) { return 3; }
    }
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
    var v::Vec<Token> tokens = v::empty<Token>();
    let i32 status = scan(input, ref mut tokens);
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
        number(if (cooked) { width } else { 0 });
        if (cooked) {
            var usize byte_index = token.start;
            while (byte_index < token.end) { number(input[byte_index] as usize); byte_index += 1; }
        }
        position = token.end;
        index += 1;
    }
    assert(position == input.len);
    return 0;
}
