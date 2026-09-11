module expressions;
import scanner as scanner;
import std::vec as v;

pub struct Node { u32 kind; usize start; usize end; usize token; usize left; usize right; }
pub struct Attempt { usize next; usize end; usize count; usize root; i32 status; u32 expected; }
struct Cursor { usize index; usize previous_end; usize nodes; i32 status; u32 expected; }

fn scanner::TokenInfo current(ref scanner::Stream stream, ref mut Cursor cursor) {
    return scanner::token_info(stream, cursor.index);
}
fn void trivia(ref scanner::Stream stream, ref mut Cursor cursor) {
    while (scanner::token_info(stream, cursor.index).kind == 2 && cursor.index + 1 < scanner::token_count(stream)) {
        cursor.index = cursor.index + 1;
    }
}
fn void advance(ref scanner::Stream stream, ref mut Cursor cursor) {
    cursor.previous_end = scanner::token_info(stream, cursor.index).end;
    if (cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
    trivia(stream, cursor);
}
fn bool word(ref scanner::Stream stream, usize index, ref u8[] expected) {
    let ref u8[] raw = scanner::token_raw(stream, index);
    if (raw.len != expected.len) { return false; }
    var usize byte_index = 0;
    while (byte_index < raw.len) {
        if (raw[byte_index] != expected[byte_index]) { return false; }
        byte_index += 1;
    }
    return true;
}
fn bool take(ref scanner::Stream stream, ref mut Cursor cursor, ref u8[] expected) {
    if (!word(stream, cursor.index, expected)) { return false; }
    advance(stream, cursor);
    return true;
}
fn bool expect(ref scanner::Stream stream, ref mut Cursor cursor, ref u8[] text, u32 code) {
    if (take(stream, cursor, text)) { return true; }
    cursor.expected = code;
    return false;
}
fn usize precedence(ref scanner::Stream stream, usize index) {
    if (word(stream, index, b"||")) { return 1; }
    if (word(stream, index, b"&&")) { return 2; }
    if (word(stream, index, b"==") || word(stream, index, b"!=") || word(stream, index, b"<") || word(stream, index, b"<=") || word(stream, index, b">") || word(stream, index, b">=")) { return 3; }
    if (word(stream, index, b"|")) { return 4; }
    if (word(stream, index, b"^")) { return 5; }
    if (word(stream, index, b"&")) { return 6; }
    if (word(stream, index, b"<<") || word(stream, index, b">>")) { return 7; }
    if (word(stream, index, b"+") || word(stream, index, b"-")) { return 8; }
    if (word(stream, index, b"*") || word(stream, index, b"/") || word(stream, index, b"%")) { return 9; }
    return 0;
}
fn usize append(ref mut v::Vec<Node> nodes, ref mut Cursor cursor, Node node) {
    if (cursor.nodes >= 4096) { cursor.status = 4; return 0; }
    if (!switch (v::push<Node>(nodes, node)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return 0; }
    cursor.nodes = cursor.nodes + 1;
    return cursor.nodes;
}
fn usize expression(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, usize minimum, usize depth) {
    if (depth >= 64) { cursor.status = 4; return 0; }
    let usize token_index = cursor.index;
    let scanner::TokenInfo token = current(stream, cursor);
    var usize left = 0;
    if (token.kind == 5 || token.kind == 3 || word(stream, cursor.index, b"true") || word(stream, cursor.index, b"false")) {
        advance(stream, cursor);
        if (token.kind == 3) {
            while (take(stream, cursor, b"::")) {
                if (current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
                advance(stream, cursor);
            }
        }
        left = append(nodes, cursor, Node { .kind = if (token.kind == 3) { 2 } else { 1 }, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = 0, .right = 0 });
    } else if (take(stream, cursor, b"(")) {
        let usize child = expression(stream, cursor, nodes, 1, depth + 1);
        if (child == 0 || !expect(stream, cursor, b")", 5)) { return 0; }
        left = append(nodes, cursor, Node { .kind = 3, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = child, .right = 0 });
    } else if (word(stream, cursor.index, b"-") || word(stream, cursor.index, b"+") || word(stream, cursor.index, b"!") || word(stream, cursor.index, b"~") || word(stream, cursor.index, b"*")) {
        advance(stream, cursor);
        let usize child = expression(stream, cursor, nodes, 10, depth + 1);
        if (child == 0) { return 0; }
        left = append(nodes, cursor, Node { .kind = 4, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = child, .right = 0 });
    } else { cursor.expected = 3; return 0; }
    var bool comparison = false;
    while (left != 0) {
        let usize opener = cursor.index;
        if (take(stream, cursor, b".")) {
            let usize member = cursor.index;
            if (current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
            advance(stream, cursor);
            left = append(nodes, cursor, Node { .kind = 8, .start = token.start, .end = cursor.previous_end, .token = member, .left = left, .right = 0 });
            continue;
        }
        if (take(stream, cursor, b"[")) {
            let usize index = expression(stream, cursor, nodes, 1, depth + 1);
            if (index == 0 || !expect(stream, cursor, b"]", 11)) { return 0; }
            left = append(nodes, cursor, Node { .kind = 9, .start = token.start, .end = cursor.previous_end, .token = opener, .left = left, .right = index });
            continue;
        }
        if (take(stream, cursor, b"(")) {
            var usize arguments = 0;
            while (!word(stream, cursor.index, b")")) {
                let usize argument_token = cursor.index;
                let usize argument_start = current(stream, cursor).start;
                let usize value = expression(stream, cursor, nodes, 1, depth + 1);
                if (value == 0) { return 0; }
                arguments = append(nodes, cursor, Node { .kind = 7, .start = argument_start, .end = cursor.previous_end, .token = argument_token, .left = value, .right = arguments });
                if (arguments == 0) { return 0; }
                if (!take(stream, cursor, b",")) { break; }
            }
            if (!expect(stream, cursor, b")", 5)) { return 0; }
            left = append(nodes, cursor, Node { .kind = 6, .start = token.start, .end = cursor.previous_end, .token = opener, .left = left, .right = arguments });
            continue;
        }
        let usize power = precedence(stream, cursor.index);
        if (power < minimum) { return left; }
        if (power == 3 && comparison) { return 0; }
        if (power == 3) { comparison = true; }
        let usize operator = cursor.index;
        advance(stream, cursor);
        let usize right = expression(stream, cursor, nodes, power + 1, depth + 1);
        if (right == 0) { return 0; }
        left = append(nodes, cursor, Node { .kind = 5, .start = token.start, .end = cursor.previous_end, .token = operator, .left = left, .right = right });
    }
    return 0;
}

pub fn Attempt parse(ref scanner::Stream stream, usize start, usize previous_end, usize count, u32 expected, ref mut v::Vec<Node> nodes, usize minimum, usize depth) {
    var Cursor cursor = Cursor { .index = start, .previous_end = previous_end, .nodes = count, .status = 0, .expected = expected };
    let usize root = expression(stream, ref mut cursor, nodes, minimum, depth);
    return Attempt { .next = cursor.index, .end = cursor.previous_end, .count = cursor.nodes, .root = root, .status = cursor.status, .expected = cursor.expected };
}
