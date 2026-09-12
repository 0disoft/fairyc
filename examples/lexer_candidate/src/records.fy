module records;
import scanner as scanner;
import types as types;
import std::vec as v;

// kind 0 struct/resource, 1 state cell, 2 state map. Only maps have key_type.
pub struct Field { u32 kind; usize start; usize end; usize name_start; usize name_end; usize type_root; usize key_type; bool public; usize public_start; usize public_end; }
pub struct Attempt { bool ok; usize next; usize end; usize fields; usize types; i32 status; u32 expected; usize cleanup_start; }
struct Cursor { usize index; usize end; usize fields; usize types; i32 status; u32 expected; usize cleanup_start; }
fn void advance(ref scanner::Stream stream, ref mut Cursor cursor) {
    cursor.end = scanner::token_info(stream,cursor.index).end;
    if (cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
    while (scanner::token_info(stream,cursor.index).kind == 2 && cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
}
fn bool body(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<types::Node> nodes, ref mut v::Vec<Field> fields, bool is_resource, bool is_state) {
    if (!types::word(stream,cursor.index,b"{")) { cursor.expected = 6; return false; }
    advance(stream,cursor);
    while (!types::word(stream,cursor.index,b"}")) {
        if (is_resource && types::word(stream,cursor.index,b"cleanup")) { break; }
        if (scanner::token_info(stream,cursor.index).kind == 1) { cursor.expected = 7; return false; }
        if (cursor.fields >= 1024) { cursor.status = 4; return false; }
        let usize start = scanner::token_info(stream,cursor.index).start;
        var u32 kind = 0;
        if (is_state) {
            kind = if (types::word(stream,cursor.index,b"cell")) { 1 } else if (types::word(stream,cursor.index,b"map")) { 2 } else { 0 };
            if (kind == 0) { cursor.expected = 22; return false; }
            advance(stream,cursor);
        }
        let bool public = types::word(stream,cursor.index,b"pub");
        if (is_state && public) { cursor.expected = 2; return false; }
        if (public) { advance(stream,cursor); }
        let types::Attempt type = types::parse(stream,cursor.index,cursor.types,nodes);
        cursor.index = type.next; cursor.end = type.end; cursor.types = type.count;
        cursor.status = type.status; cursor.expected = type.expected;
        if (type.root == 0) { return false; }
        var usize value_type = type.root;
        if (kind == 2) {
            let types::Attempt value = types::parse(stream,cursor.index,cursor.types,nodes);
            cursor.index = value.next; cursor.end = value.end; cursor.types = value.count;
            cursor.status = value.status; cursor.expected = value.expected;
            if (value.root == 0) { return false; }
            value_type = value.root;
        }
        let scanner::TokenInfo name = scanner::token_info(stream,cursor.index);
        if (name.kind != 3) { cursor.expected = 1; return false; }
        advance(stream,cursor);
        if (!types::word(stream,cursor.index,b";")) { cursor.expected = 8; return false; }
        advance(stream,cursor);
        let Field field = Field { .kind = kind, .start = start, .end = cursor.end, .name_start = name.start, .name_end = name.end, .type_root = value_type, .key_type = if (kind == 2) { type.root } else { 0 }, .public = public, .public_start = if (public) { start } else { 0 }, .public_end = if (public) { start + 3 } else { 0 } };
        if (!switch(v::push<Field>(fields,field)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return false; }
        cursor.fields = cursor.fields + 1;
    }
    if (is_resource && types::word(stream,cursor.index,b"cleanup")) {
        cursor.cleanup_start = scanner::token_info(stream,cursor.index).start;
        return true;
    }
    advance(stream,cursor); return true;
}
pub fn Attempt parse(ref scanner::Stream stream, usize index, usize type_count, usize field_count, ref mut v::Vec<types::Node> nodes, ref mut v::Vec<Field> fields, bool is_resource, bool is_state) {
    var Cursor cursor = Cursor { .index = index, .end = scanner::token_info(stream,index).start, .fields = field_count, .types = type_count, .status = 0, .expected = 0, .cleanup_start = 0 };
    let bool ok = body(stream,ref mut cursor,nodes,fields,is_resource,is_state);
    return Attempt { .ok = ok, .next = cursor.index, .end = cursor.end, .fields = cursor.fields, .types = cursor.types, .status = cursor.status, .expected = cursor.expected, .cleanup_start = cursor.cleanup_start };
}
