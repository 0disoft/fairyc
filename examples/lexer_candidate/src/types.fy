module types;
import scanner as scanner;
import std::vec as v;

// Private postorder nodes: 1 path, 2 ref, 3 ref mut, 4 slice, 5 group.
// 6 generic application (left base, right argument tail), 7 argument link.
// 8 fixed array (left element, right length), 9 integer length, 10 path length.
// 11 raw pointer, 12 mutable raw pointer; prefix constructors retain the child.
// Shared across the enclosing Parsed owner; IDs are one-based, zero absent.
pub struct Node { u32 kind; usize start; usize end; usize left; usize right; }
pub struct Attempt { usize next; usize end; usize count; usize root; i32 status; u32 expected; }
struct Cursor { usize index; usize end; usize count; i32 status; u32 expected; bool half; }

pub fn bool word(ref scanner::Stream stream, usize index, ref u8[] text) {
    let ref u8[] raw = scanner::token_raw(stream, index);
    if (raw.len != text.len) { return false; }
    var usize i = 0;
    while (i < raw.len) { if (raw[i] != text[i]) { return false; } i += 1; }
    return true;
}
fn void trivia(ref scanner::Stream stream, ref mut Cursor cursor) {
    while (scanner::token_info(stream,cursor.index).kind == 2 && cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
}
fn void advance(ref scanner::Stream stream, ref mut Cursor cursor) {
    cursor.end = scanner::token_info(stream,cursor.index).end;
    if (cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
    trivia(stream,cursor);
}
fn bool take(ref scanner::Stream stream, ref mut Cursor cursor, ref u8[] text) {
    if (cursor.half) { return false; }
    if (!word(stream,cursor.index,text)) { return false; }
    advance(stream,cursor); return true;
}
// Consume one greater-than in type context without mutating retained tokens.
fn bool close_argument(ref scanner::Stream stream, ref mut Cursor cursor) {
    if (cursor.half) { cursor.half = false; advance(stream,cursor); return true; }
    if (take(stream,cursor,b">")) { return true; }
    if (word(stream,cursor.index,b">>")) {
        cursor.half = true; cursor.end = scanner::token_info(stream,cursor.index).start + 1;
        return true;
    }
    cursor.expected = 13; return false;
}
fn bool named(ref scanner::Stream stream, usize index) {
    return scanner::token_info(stream,index).kind == 3 || word(stream,index,b"void") || word(stream,index,b"bool") || word(stream,index,b"i32") || word(stream,index,b"usize") || word(stream,index,b"u8") || word(stream,index,b"u32");
}
fn usize append(ref mut v::Vec<Node> nodes, ref mut Cursor cursor, u32 kind, usize start, usize left, usize right) {
    if (cursor.count >= 4096) { cursor.status = 4; return 0; }
    let Node node = Node { .kind = kind, .start = start, .end = cursor.end, .left = left, .right = right };
    if (!switch(v::push<Node>(nodes,node)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return 0; }
    cursor.count = cursor.count + 1; return cursor.count;
}
fn usize type_node(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, usize depth) {
    if (depth >= 32) { cursor.status = 4; return 0; }
    let usize start = scanner::token_info(stream,cursor.index).start;
    if (word(stream,cursor.index,b"ref") || word(stream,cursor.index,b"ptr")) {
        let bool pointer = take(stream,cursor,b"ptr");
        if (!pointer) { discard take(stream,cursor,b"ref"); }
        let bool mutable = take(stream,cursor,b"mut");
        let usize child = type_node(stream,cursor,nodes,depth + 1);
        if (child == 0) { return 0; }
        return append(nodes,cursor,if (pointer) { if (mutable) { 12 } else { 11 } } else { if (mutable) { 3 } else { 2 } },start,child,0);
    }
    var usize root = 0;
    if (take(stream,cursor,b"(")) {
        let usize child = type_node(stream,cursor,nodes,depth + 1);
        if (child == 0) { return 0; }
        if (!take(stream,cursor,b")")) { cursor.expected = 5; return 0; }
        root = append(nodes,cursor,5,start,child,0);
    } else if (named(stream,cursor.index)) {
        advance(stream,cursor);
        while (take(stream,cursor,b"::")) {
            if (scanner::token_info(stream,cursor.index).kind != 3) { cursor.expected = 1; return 0; }
            advance(stream,cursor);
        }
        root = append(nodes,cursor,1,start,0,0);
        if (root != 0 && take(stream,cursor,b"<")) {
            let usize tail = arguments(stream,cursor,nodes,depth + 1);
            if (tail == 0) { return 0; }
            root = append(nodes,cursor,6,start,root,tail);
        }
    } else { cursor.expected = 2; return 0; }
    while (root != 0 && take(stream,cursor,b"[")) {
        var usize length = 0;
        if (!word(stream,cursor.index,b"]")) {
            let usize length_start = scanner::token_info(stream,cursor.index).start;
            let u32 kind = scanner::token_info(stream,cursor.index).kind;
            if (kind != 5 && kind != 3) { cursor.expected = 16; return 0; }
            advance(stream,cursor);
            if (kind == 3) {
                while (take(stream,cursor,b"::")) {
                    if (scanner::token_info(stream,cursor.index).kind != 3) { cursor.expected = 1; return 0; }
                    advance(stream,cursor);
                }
            }
            length = append(nodes,cursor,if (kind == 5) { 9 } else { 10 },length_start,0,0);
            if (length == 0) { return 0; }
        }
        if (!take(stream,cursor,b"]")) { cursor.expected = 11; return 0; }
        root = append(nodes,cursor,if (length == 0) { 4 } else { 8 },start,root,length);
    }
    return root;
}
pub fn Attempt parse(ref scanner::Stream stream, usize index, usize count, ref mut v::Vec<Node> nodes) {
    var Cursor cursor = Cursor { .index = index, .end = scanner::token_info(stream,index).start, .count = count, .status = 0, .expected = 0, .half = false };
    trivia(stream,ref mut cursor);
    var usize root = type_node(stream,ref mut cursor,nodes,0);
    if (cursor.half) { root = 0; cursor.expected = 0; }
    return Attempt { .next = cursor.index, .end = cursor.end, .count = cursor.count, .root = root, .status = cursor.status, .expected = cursor.expected };
}
fn usize arguments(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, usize depth) {
    var usize tail = 0;
    while (true) {
        let usize start = scanner::token_info(stream,cursor.index).start;
        let usize value = type_node(stream,cursor,nodes,depth);
        if (value == 0) { return 0; }
        tail = append(nodes,cursor,7,start,value,tail);
        if (tail == 0) { return 0; }
        if (!take(stream,cursor,b",")) { break; }
        if (word(stream,cursor.index,b">") || word(stream,cursor.index,b">>")) { break; }
    }
    if (!close_argument(stream,cursor)) { return 0; }
    return tail;
}
pub fn Attempt parse_arguments(ref scanner::Stream stream, usize index, usize count, ref mut v::Vec<Node> nodes) {
    var Cursor cursor = Cursor { .index = index, .end = scanner::token_info(stream,index).start, .count = count, .status = 0, .expected = 0, .half = false };
    var usize root = 0;
    if (take(stream,ref mut cursor,b"<")) { root = arguments(stream,ref mut cursor,nodes,0); }
    if (cursor.half) { root = 0; cursor.expected = 0; }
    return Attempt { .next = cursor.index, .end = cursor.end, .count = cursor.count, .root = root, .status = cursor.status, .expected = cursor.expected };
}
pub fn bool call_ahead(ref scanner::Stream stream, usize index) {
    var usize next = index; var usize offset = 0; var i32 depth = 0;
    while (offset < 256) {
        let ref u8[] raw = scanner::token_raw(stream,next);
        if (raw.len == 0 || word(stream,next,b";") || word(stream,next,b"{") || word(stream,next,b"}") || word(stream,next,b"=") || word(stream,next,b"+") || word(stream,next,b"-") || word(stream,next,b"/") || word(stream,next,b"%") || word(stream,next,b"&&") || word(stream,next,b"||")) { return false; }
        if (word(stream,next,b"<")) { depth += 1; }
        else if (word(stream,next,b">")) { depth -= 1; }
        else if (word(stream,next,b">>")) { depth -= 2; }
        if (next + 1 < scanner::token_count(stream)) { next += 1; }
        while (scanner::token_info(stream,next).kind == 2 && next + 1 < scanner::token_count(stream)) { next += 1; }
        if (depth <= 0) { return depth == 0 && offset > 1 && word(stream,next,b"("); }
        offset += 1;
    }
    return false;
}
