module body_state;
import std::vec as v;
import scanner as scanner;
import parse_tokens as tokens;

pub struct Node { u32 kind; usize start; usize end; usize token; usize left; usize right; }
pub fn scanner::TokenInfo current(ref scanner::Stream stream, ref mut Cursor cursor) {
    return scanner::token_info(stream, cursor.index);
}
pub fn void trivia(ref scanner::Stream stream, ref mut Cursor cursor) {
    cursor.index = tokens::significant(stream,cursor.index);
}
pub fn void advance(ref scanner::Stream stream, ref mut Cursor cursor) {
    cursor.previous_end = scanner::token_info(stream, cursor.index).end;
    if (cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
    trivia(stream, cursor);
}
pub fn bool word(ref scanner::Stream stream, usize index, ref u8[] expected) { return tokens::word(stream,index,expected); }

pub fn bool take(ref scanner::Stream stream, ref mut Cursor cursor, ref u8[] expected) {
    if (!word(stream, cursor.index, expected)) { return false; }
    advance(stream, cursor);
    return true;
}
pub fn bool expect(ref scanner::Stream stream, ref mut Cursor cursor, ref u8[] text, u32 code) {
    if (take(stream, cursor, text)) { return true; }
    cursor.expected = code;
    return false;
}

pub fn usize append(ref mut v::Vec<Node> nodes, ref mut Cursor cursor, Node node) {
    if (cursor.nodes >= 4096) { cursor.status = 4; return 0; }
    if (!switch (v::push<Node>(nodes, node)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return 0; }
    cursor.nodes = cursor.nodes + 1;
    cursor.expression_kind = node.kind;
    return cursor.nodes;
}

pub struct Cursor { usize index; usize previous_end; usize nodes; usize entries; usize statements; usize types; usize fields; usize expression; usize top_return; i32 status; u32 expected; usize active_expressions; usize block_depth; usize tail; usize last_entry; u32 expression_kind; u32 statement_kind; }
pub struct Statement { u32 kind; usize start; usize end; usize expression; usize target; usize first; usize second; usize entry; usize operator; }

pub fn Statement statement_value(u32 kind, usize start, usize end) {
    return Statement { .kind = kind, .start = start, .end = end, .expression = 0, .target = 0, .first = 0, .second = 0, .entry = 0, .operator = 0 };
}
pub fn usize append_statement(ref mut v::Vec<Statement> statements, ref mut Cursor cursor, Statement value) {
    if (cursor.statements >= 4096) { cursor.status = 4; return 0; }
    if (!switch (v::push<Statement>(statements, value)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return 0; }
    cursor.statements = cursor.statements + 1;
    cursor.statement_kind = value.kind;
    return cursor.statements;
}
