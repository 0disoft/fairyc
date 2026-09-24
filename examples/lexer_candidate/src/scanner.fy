module scanner;
import std::vec as v;
import std::io as io;
import std::bytes as bytes;
import numbers as numbers;
import diagnostics as diagnostics;
import utf8 as utf8;
import strings as strings;
import budget as budget;

// Private numeric observations mirror internal/lexer/token.go, not a stable ABI.
struct Token {
    u32 kind; u32 trivia; usize start; usize end;
    usize cooked_index;
}
// Only tokens with nonempty cooked bytes own a Buffer descriptor in this table.
// Index zero means an empty cooked view; all other indices are one-based.
fn bool emit(ref mut v::Vec<Token> tokens, bool admitted, u32 kind, u32 trivia, usize start, usize end) {
    if (!admitted) { return append(tokens, 0, 0, start, end); }
    return append(tokens, kind, trivia, start, end);
}

fn bool emit_cooked(ref mut v::Vec<Token> tokens, ref mut v::Vec<bytes::Buffer> cooked_values, bool admitted, u32 kind, usize start, usize end, bytes::Buffer cooked) {
    if (!admitted) { return append(tokens, 0, 0, start, end); }
    return append_cooked(tokens, cooked_values, kind, 0, start, end, cooked);
}

fn i32 exhausted(ref u8[] input, usize position, ref mut v::Vec<Token> tokens, ref mut v::Vec<diagnostics::Entry> errors) {
    let i32 status = diagnostics::add(errors, 10, position, position, false, b"");
    if (status != 0) { return status; }
    if (position < input.len && !append(tokens, 0, 0, position, input.len)) { return 3; }
    if (!append(tokens, 1, 0, input.len, input.len)) { return 3; }
    return 0;
}

fn i32 work_exhausted(ref u8[] input, usize start, usize position, ref mut v::Vec<Token> tokens, ref mut v::Vec<diagnostics::Entry> errors) {
    // The reference's progress guard retains one byte if recognition never advanced.
    let usize end = if (position > start) { position } else { start + 1 };
    if (!append(tokens, 0, 0, start, end)) { return 3; }
    return exhausted(input, end, tokens, errors);
}

fn bool append(ref mut v::Vec<Token> tokens, u32 kind, u32 trivia, usize start, usize end) {
    let Token value = Token { .kind, .trivia, .start, .end, .cooked_index = 0 };
    return switch (v::push<Token>(tokens, value)) { case .ok(_): { true } case .err(_): { false } };
}

fn bool append_cooked(ref mut v::Vec<Token> tokens, ref mut v::Vec<bytes::Buffer> cooked_values, u32 kind, u32 trivia, usize start, usize end, bytes::Buffer cooked) {
    if (bytes::len(ref cooked) == 0) { return append(tokens, kind, trivia, start, end); }
    let usize cooked_index = switch (v::push<bytes::Buffer>(cooked_values, cooked)) { case .ok(length): { length } case .err(_): { 0 } };
    if (cooked_index == 0) { return false; }
    let Token value = Token { .kind, .trivia, .start, .end, .cooked_index };
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
    if (start >= input.len) { return 0; }
    let u8 first = input[start];
    let u8 next = if (start + 1 < input.len) { input[start + 1] } else { 0 };
    if ((first == 60 || first == 62) && next == first) {
        if (start + 2 < input.len && input[start + 2] == 61) { return 3; }
        return 2;
    }
    if (next == 61 && (first == 43 || first == 45 || first == 42 || first == 47 ||
                       first == 37 || first == 38 || first == 94 || first == 124 ||
                       first == 61 || first == 33 || first == 60 || first == 62)) { return 2; }
    if ((first == 58 && next == 58) || (first == 46 && next == 46) ||
        (first == 38 && next == 38) || (first == 124 && next == 124) ||
        ((first == 45 || first == 61) && next == 62)) { return 2; }
    if (first == 33 || first == 37 || first == 38 ||
        (first >= 40 && first <= 47) || (first >= 58 && first <= 64) ||
        first == 91 || first == 93 || first == 94 ||
        (first >= 123 && first <= 126)) { return 1; }
    return 0;
}

// 0 success; 2 not implemented input; 3 allocation failure; 4 candidate size limit.
// No partial token stream escapes on any failure. Full Go budget parity is pending.
fn i32 scan(ref u8[] input, ref mut v::Vec<Token> tokens, ref mut v::Vec<bytes::Buffer> cooked_values, ref mut v::Vec<diagnostics::Entry> errors, bool previous, usize max_bytes, usize max_tokens, usize max_diagnostics, usize max_work) {
    if (input.len > 32768) { return 4; }
    if (input.len > max_bytes) { return exhausted(input, 0, tokens, errors); }
    var usize token_count = 0;
    var usize diagnostic_count = 0;
    var bool token_exhausted = false;
    var usize position = 0;
    var usize cooked_bytes = 0;
    var budget::Work work = budget::create(max_work);
    if (matches(input, 0, b"\xEF\xBB\xBF")) {
        work = budget::charge(work, 3);
        if (work.exhausted) { return exhausted(input, 0, tokens, errors); }
        token_exhausted = max_tokens == 0;
        if (!emit(tokens, !token_exhausted, 2, 1, 0, 3)) { return 3; }
        if (!token_exhausted) { token_count += 1; }
        position = 3;
    }
    while (position < input.len && !token_exhausted) {
        work = budget::charge(work, 1);
        if (work.exhausted) { return exhausted(input, position, tokens, errors); }
        token_exhausted = token_count >= max_tokens;
        if (!token_exhausted) { token_count += 1; }
        let usize start = position;
        let u8 current = input[position];
        var u32 kind = 0;
        var u32 trivia = 0;
        var bool unclosed_comment = false;
        if (space(current) || current == 13) {
            while (position < input.len) {
                let usize width = if (space(input[position])) { 1 } else if (matches(input, position, b"\r\n")) { 2 } else { 0 };
                if (width == 0) { break; }
                work = budget::charge(work, if (position == start) { width - 1 } else { width });
                if (work.exhausted) { return work_exhausted(input, start, position, tokens, errors); }
                position += width;
            }
            if (position == start) {
                position += 1;
                if (!emit(tokens, !token_exhausted, 0, 0, start, position)) { return 3; }
                if (token_exhausted) { break; }
                let i32 status = diagnostics::add_limited(errors, diagnostic_count, max_diagnostics, 1, start, position, false, b"");
                if (status == 5) { return exhausted(input, position, tokens, errors); }
                if (status != 0) { return status; }
                diagnostic_count += 1;
                continue;
            }
            kind = 2; trivia = 2;
        } else if (matches(input, position, b"//")) {
            trivia = if (matches(input, position, b"///")) { 5 } else { 3 };
            work = budget::charge(work, 1);
            if (work.exhausted) { return work_exhausted(input, start, position, tokens, errors); }
            position += 2;
            while (position < input.len && input[position] != 10 && input[position] != 13) {
                work = budget::charge(work, 1);
                if (work.exhausted) { return work_exhausted(input, start, position, tokens, errors); }
                position += 1;
            }
            kind = 2;
        } else if (matches(input, position, b"/*")) {
            work = budget::charge(work, 1);
            if (work.exhausted) { return work_exhausted(input, start, position, tokens, errors); }
            position += 2;
            var usize depth = 1;
            while (position < input.len && depth > 0) {
                let usize width = if (matches(input, position, b"/*") || matches(input, position, b"*/")) { 2 } else { 1 };
                work = budget::charge(work, width);
                if (work.exhausted) { return work_exhausted(input, start, position, tokens, errors); }
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
            let strings::Step step = strings::scan(input, start, kind, opener, ref mut cooked, errors, diagnostic_count, max_diagnostics, work);
            work = step.work;
            if (step.status == 5) {
                if (!append(tokens, 0, 0, start, step.end)) { return 3; }
                return exhausted(input, step.end, tokens, errors);
            }
            if (step.status != 0) { return step.status; }
            diagnostic_count = step.diagnostics;
            position = step.end;
            let usize length = bytes::len(ref cooked);
            let usize cooked_limit = 65536;
            if (length > cooked_limit - cooked_bytes) { return 4; }
            cooked_bytes += length;
            if (!emit_cooked(tokens, cooked_values, !token_exhausted, kind, start, position, cooked)) { return 3; }
            continue;
        } else if (letter(current)) {
            position += 1;
            while (position < input.len && (letter(input[position]) || digit(input[position]))) {
                work = budget::charge(work, 1);
                if (work.exhausted) { return work_exhausted(input, start, position, tokens, errors); }
                position += 1;
            }
            kind = if (keyword(input, start, position, previous)) { 4 } else { 3 };
        } else if (digit(current)) {
            var bytes::Buffer cooked = bytes::empty();
            let numbers::Scan result = numbers::scan(input, start, ref mut cooked, work);
            work = result.work;
            if (result.status == 5) { return work_exhausted(input, start, result.end, tokens, errors); }
            position = result.end; kind = result.kind;
            let i32 status = result.status;
            if (status == 3 || status == 4) { return status; }
            let usize length = bytes::len(ref cooked);
            let usize cooked_limit = 65536;
            if (length > cooked_limit - cooked_bytes) { return 4; }
            cooked_bytes += length;
            if (!emit_cooked(tokens, cooked_values, !token_exhausted, kind, start, position, cooked)) { return 3; }
            if (status == 1 && !token_exhausted) {
                let i32 added = diagnostics::add_limited(errors, diagnostic_count, max_diagnostics, 4, start, position, false, b"");
                if (added == 5) { return exhausted(input, position, tokens, errors); }
                if (added != 0) { return added; }
                diagnostic_count += 1;
            }
            continue;
        } else {
            let usize width = symbol_width(input, position);
            if (width == 0) {
                let usize decoded = utf8::width(input, position);
                work = budget::charge(work, if (decoded == 0) { 0 } else { decoded - 1 });
                if (work.exhausted) { return work_exhausted(input, start, position, tokens, errors); }
                position += if (decoded == 0) { 1 } else { decoded };
                if (!emit(tokens, !token_exhausted, 0, 0, start, position)) { return 3; }
                if (token_exhausted) { break; }
                let i32 status = diagnostics::add_limited(errors, diagnostic_count, max_diagnostics, 1, start, position, false, b"");
                if (status == 5) { return exhausted(input, position, tokens, errors); }
                if (status != 0) { return status; }
                diagnostic_count += 1;
                continue;
            } else {
                work = budget::charge(work, width - 1);
                if (work.exhausted) { return work_exhausted(input, start, position, tokens, errors); }
                position += width; kind = 10;
            }
        }
        assert(position > start && position <= input.len);
        if (!emit(tokens, !token_exhausted, kind, trivia, start, position)) { return 3; }
        if (unclosed_comment && !token_exhausted) {
            let i32 status = diagnostics::add_limited(errors, diagnostic_count, max_diagnostics, 1, start, position, true, b"*/");
            if (status == 5) { return exhausted(input, position, tokens, errors); }
            if (status != 0) { return status; }
            diagnostic_count += 1;
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
    return report_limits(input, previous, 32768, 1048576);
}

@effects(console)
pub fn i32 report_limits(ref u8[] input, bool previous, usize max_bytes, usize max_tokens) {
    return report_budget(input, previous, max_bytes, max_tokens, 256);
}

@effects(console)
pub fn i32 report_budget(ref u8[] input, bool previous, usize max_bytes, usize max_tokens, usize max_diagnostics) {
    return report_work(input, previous, max_bytes, max_tokens, max_diagnostics, 16777216);
}

@effects(console)
pub fn i32 report_work(ref u8[] input, bool previous, usize max_bytes, usize max_tokens, usize max_diagnostics, usize max_work) {
    var v::Vec<Token> tokens = v::empty<Token>();
    var v::Vec<bytes::Buffer> cooked_values = v::empty<bytes::Buffer>();
    var v::Vec<diagnostics::Entry> errors = v::empty<diagnostics::Entry>();
    let i32 status = scan(input, ref mut tokens, ref mut cooked_values, ref mut errors, previous, max_bytes, max_tokens, max_diagnostics, max_work);
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
        let ref u8[] numeric = if (token.cooked_index == 0) { input[0..0] } else { bytes::view(v::at<bytes::Buffer>(ref cooked_values, token.cooked_index - 1)) };
        number(if (cooked) { width } else { numeric.len });
        if (cooked) {
            var usize byte_index = token.start;
            while (byte_index < token.end) { number(input[byte_index] as usize); byte_index += 1; }
        }
        for (let u8 byte in numeric) { number(byte as usize); }
        position = token.end;
        index += 1;
    }
    assert(position == input.len);
    diagnostics::report(ref errors);
    return 0;
}

// Experimental owned recognition result; no console capability is required.
pub resource Stream {
    bytes::Buffer source;
    v::Vec<Token> tokens;
    v::Vec<bytes::Buffer> cooked;
    v::Vec<diagnostics::Entry> errors;
    cleanup { }
}

pub struct TokenInfo { u32 kind; u32 trivia; usize start; usize end; }

// Input is consumed on success and failure. Reference-budget exhaustion is a
// successful lossless stream with E0010, not an allocation/storage failure.
pub fn Result<Stream, i32> collect(bytes::Buffer input, bool previous, usize max_bytes, usize max_tokens, usize max_diagnostics, usize max_work) {
    var v::Vec<Token> tokens = v::empty<Token>();
    var v::Vec<bytes::Buffer> cooked_values = v::empty<bytes::Buffer>();
    var v::Vec<diagnostics::Entry> errors = v::empty<diagnostics::Entry>();
    let i32 status = scan(bytes::view(ref input), ref mut tokens, ref mut cooked_values, ref mut errors, previous, max_bytes, max_tokens, max_diagnostics, max_work);
    if (status != 0) { return .err(status); }
    return .ok(Stream { .source = input, .tokens = tokens, .cooked = cooked_values, .errors = errors });
}

pub fn usize token_count(ref Stream stream) { return v::len<Token>(ref stream.tokens); }
pub fn void release(Stream stream) { }
pub fn usize diagnostic_count(ref Stream stream) { return v::len<diagnostics::Entry>(ref stream.errors); }
pub fn ref u8[] source_view(ref Stream stream) { return bytes::view(ref stream.source); }

pub fn TokenInfo token_info(ref Stream stream, usize index) {
    let ref Token token = v::at<Token>(ref stream.tokens, index);
    return TokenInfo { .kind = token.kind, .trivia = token.trivia, .start = token.start, .end = token.end };
}

pub fn ref u8[] token_raw(ref Stream stream, usize index) {
    let ref Token token = v::at<Token>(ref stream.tokens, index);
    let ref u8[] input = bytes::view(ref stream.source);
    return input[token.start..token.end];
}

pub fn ref u8[] token_cooked(ref Stream stream, usize index) {
    let ref Token token = v::at<Token>(ref stream.tokens, index);
    let ref u8[] input = bytes::view(ref stream.source);
    let usize width = token.end - token.start;
    if (token.kind == 4 && ((width == 4 && (matches(input, token.start, b"true") || matches(input, token.start, b"null"))) || (width == 5 && matches(input, token.start, b"false")))) {
        return input[token.start..token.end];
    }
    if (token.cooked_index == 0) { return input[0..0]; }
    return bytes::view(v::at<bytes::Buffer>(ref stream.cooked, token.cooked_index - 1));
}

// Diagnostics are in recognition order; the legacy reporter sorts for display.
pub fn diagnostics::Info diagnostic_info(ref Stream stream, usize index) {
    return diagnostics::info(v::at<diagnostics::Entry>(ref stream.errors, index));
}
pub fn ref u8[] diagnostic_token(ref Stream stream, usize index) {
    return diagnostics::token_view(v::at<diagnostics::Entry>(ref stream.errors, index));
}

// Live logical payload sizes, not allocator capacity, heap/RSS or peak memory.
pub struct Storage { usize source_bytes; usize cooked_bytes; usize diagnostic_bytes; usize tokens; usize diagnostics; }
pub fn Storage storage(ref Stream stream) {
    var usize cooked = 0; var usize diagnostic = 0; var usize index = 0;
    while (index < v::len<bytes::Buffer>(ref stream.cooked)) {
        cooked += bytes::view(v::at<bytes::Buffer>(ref stream.cooked, index)).len; index += 1;
    }
    index = 0;
    while (index < v::len<diagnostics::Entry>(ref stream.errors)) {
        diagnostic += diagnostics::token_view(v::at<diagnostics::Entry>(ref stream.errors, index)).len;
        index += 1;
    }
    return Storage { .source_bytes = bytes::view(ref stream.source).len, .cooked_bytes = cooked,
        .diagnostic_bytes = diagnostic, .tokens = v::len<Token>(ref stream.tokens), .diagnostics = v::len<diagnostics::Entry>(ref stream.errors) };
}
