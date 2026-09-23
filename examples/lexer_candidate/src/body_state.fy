module body_state;
import std::vec as v;
import scanner as scanner;
import parse_tokens as tokens;

pub struct Node { u32 kind; usize start; usize end; usize token; usize left; usize right; }
pub fn Node node_value(u32 kind, usize start, usize end, usize token, usize left, usize right) {
    return Node { .kind, .start, .end, .token, .left, .right };
}
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
    if (cursor.nodes >= 4160) { cursor.status = 4; return 0; }
    if (!v::push<Node>(nodes, node).is_ok) { cursor.status = 3; return 0; }
    cursor.nodes = cursor.nodes + 1;
    cursor.expression_kind = node.kind;
    return cursor.nodes;
}

pub struct Cursor { usize index; usize previous_end; usize nodes; usize entries; usize statements; usize types; usize fields; usize expression; usize top_return; i32 status; u32 expected; usize active_expressions; usize block_depth; usize tail; usize last_entry; u32 expression_kind; u32 statement_kind; }
pub struct Statement { u32 kind; usize start; usize end; usize expression; usize target; usize first; usize second; usize entry; usize operator; }

pub fn Statement statement_value(u32 kind, usize start, usize end) {
    return Statement { .kind, .start, .end, .expression = 0, .target = 0, .first = 0, .second = 0, .entry = 0, .operator = 0 };
}
pub fn usize append_statement(ref mut v::Vec<Statement> statements, ref mut Cursor cursor, Statement value) {
    if (cursor.statements >= 4096) { cursor.status = 4; return 0; }
    if (!v::push<Statement>(statements, value).is_ok) { cursor.status = 3; return 0; }
    cursor.statements = cursor.statements + 1;
    cursor.statement_kind = value.kind;
    return cursor.statements;
}

pub fn usize pattern(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes) {
    let usize start = current(stream,cursor).start;
    if (!expect(stream,cursor,b".",14)) { return 0; }
    let usize token = cursor.index;
    if (current(stream,cursor).kind != 3) { cursor.expected = 1; return 0; }
    advance(stream,cursor);
    var usize binding = 0;
    let bool named = take(stream,cursor,b"{");
    if (named || take(stream,cursor,b"(")) {
      while (!named || !word(stream,cursor.index,b"}")) {
        if (named && word(stream,cursor.index,b":")) { cursor.expected = 25; return 0; }
        let tokens::PatternBinding item = tokens::pattern_binding(stream,cursor.index,named);
        cursor.index=item.next; cursor.previous_end=item.end; cursor.expected=item.expected;
        if (!item.ok) { return 0; }
        let usize value = append(nodes,cursor,node_value(17,scanner::token_info(stream,item.name).start,item.end,item.name,0,0));
        if (value == 0) { return 0; }
        binding = if (named) { append(nodes,cursor,node_value(33,item.start,item.end,item.field,value,binding)) } else { value };
        if (binding == 0) { return 0; }
        if (!named || !take(stream,cursor,b",")) { break; }
      }
      if (named && word(stream,cursor.index,b":")) { cursor.expected = 25; return 0; }
      if (!expect(stream,cursor,if (named) { b"}" } else { b")" },if (named) { 7 } else { 5 })) { return 0; }
    }
    return append(nodes,cursor,node_value(if (named) { 32 } else { 16 },start,cursor.previous_end,token,if (named) { 0 } else { binding },if (named) { binding } else { 0 }));
}
