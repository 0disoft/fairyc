module records;
import scanner as scanner;
import types as types;
import std::vec as v;

// kind 0 struct/resource, 1 state cell, 2 state map, 3 enum member, 4 enum payload field. Only maps have key_type.
// Payload fields precede their member in the shared arena; zero delimiter spans mean absent payload.
pub struct Field { u32 kind; usize start; usize end; usize name_start; usize name_end; usize type_root; usize key_type; bool public; usize public_start; usize public_end; usize payload_start; usize payload_count; usize payload_open; usize payload_end; }
pub struct Attempt { bool ok; usize next; usize end; usize fields; usize types; i32 status; u32 expected; usize cleanup_start; }
struct Cursor { usize index; usize end; usize fields; usize types; i32 status; u32 expected; usize cleanup_start; }
// Declaration kinds 4/6/10/11 select struct/resource/state/enum bodies.
// Payload fields use a separate internal mode, never a declaration kind.
const u32 PAYLOAD_BODY = 0;
fn void advance(ref scanner::Stream stream, ref mut Cursor cursor) {
    cursor.end = scanner::token_info(stream,cursor.index).end;
    if (cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
    while (scanner::token_info(stream,cursor.index).kind == 2 && cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
}
fn bool body(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<types::Node> nodes, ref mut v::Vec<Field> fields, u32 body_kind) {
    let bool is_resource = body_kind == 6;
    let bool is_state = body_kind == 10;
    let bool is_enum = body_kind == 11;
    let bool is_payload = body_kind == PAYLOAD_BODY;
    if (!types::word(stream,cursor.index,b"{")) { cursor.expected = 6; return false; }
    advance(stream,cursor);
    while (!types::word(stream,cursor.index,b"}")) {
        if (is_resource && (types::word(stream,cursor.index,b"cleanup") || types::word(stream,cursor.index,b"@"))) { break; }
        if (scanner::token_info(stream,cursor.index).kind == 1) { cursor.expected = 7; return false; }
        if (cursor.fields >= 1024) { cursor.status = 4; return false; }
        let usize start = scanner::token_info(stream,cursor.index).start;
        var u32 kind = if (is_enum) { 3 } else if (is_payload) { 4 } else { 0 };
        if (is_state) {
            kind = if (types::word(stream,cursor.index,b"cell")) { 1 } else if (types::word(stream,cursor.index,b"map")) { 2 } else { 0 };
            if (kind == 0) { cursor.expected = 22; return false; }
            advance(stream,cursor);
        }
        let bool public = types::word(stream,cursor.index,b"pub");
        if ((is_state || is_enum || is_payload) && public) { cursor.expected = if (is_state || is_payload) { 2 } else { 1 }; return false; }
        if (public) { advance(stream,cursor); }
        var usize value_type = 0; var usize key_type = 0;
        if (!is_enum) {
            let types::Attempt type = types::parse(stream,cursor.index,cursor.types,nodes);
            cursor.index = type.next; cursor.end = type.end; cursor.types = type.count;
            cursor.status = type.status; cursor.expected = type.expected;
            if (type.root == 0) { return false; }
            value_type = type.root;
            if (kind == 2) {
                let types::Attempt value = types::parse(stream,cursor.index,cursor.types,nodes);
                cursor.index = value.next; cursor.end = value.end; cursor.types = value.count;
                cursor.status = value.status; cursor.expected = value.expected;
                if (value.root == 0) { return false; }
                value_type = value.root; key_type = type.root;
            }
        }
        let scanner::TokenInfo name = scanner::token_info(stream,cursor.index);
        if (name.kind != 3) { cursor.expected = 1; return false; }
        advance(stream,cursor);
        var usize payload_start = 0; var usize payload_count = 0;
        var usize payload_open = 0; var usize payload_end = 0;
        if (is_enum && types::word(stream,cursor.index,b"{")) {
            payload_start = cursor.fields;
            payload_open = scanner::token_info(stream,cursor.index).start;
            if (!body(stream,cursor,nodes,fields,PAYLOAD_BODY)) { return false; }
            payload_count = cursor.fields - payload_start;
            payload_end = cursor.end;
        }
        if (!types::word(stream,cursor.index,b";")) { cursor.expected = 8; return false; }
        advance(stream,cursor);
        if (cursor.fields >= 1024) { cursor.status = 4; return false; }
        let Field field = Field { .kind, .start, .end = cursor.end, .name_start = name.start, .name_end = name.end, .type_root = value_type, .key_type, .public, .public_start = if (public) { start } else { 0 }, .public_end = if (public) { start + 3 } else { 0 }, .payload_start, .payload_count, .payload_open, .payload_end };
        if (!switch(v::push<Field>(fields,field)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return false; }
        cursor.fields = cursor.fields + 1;
    }
    if (is_resource && (types::word(stream,cursor.index,b"cleanup") || types::word(stream,cursor.index,b"@"))) {
        cursor.cleanup_start = scanner::token_info(stream,cursor.index).start;
        return true;
    }
    advance(stream,cursor); return true;
}
pub fn Attempt parse(ref scanner::Stream stream, usize index, usize type_count, usize field_count, ref mut v::Vec<types::Node> nodes, ref mut v::Vec<Field> fields, u32 declaration_kind) {
    var Cursor cursor = Cursor { .index, .end = scanner::token_info(stream,index).start, .fields = field_count, .types = type_count, .status = 0, .expected = 0, .cleanup_start = 0 };
    let bool ok = body(stream,ref mut cursor,nodes,fields,declaration_kind);
    return Attempt { .ok, .next = cursor.index, .end = cursor.end, .fields = cursor.fields, .types = cursor.types, .status = cursor.status, .expected = cursor.expected, .cleanup_start = cursor.cleanup_start };
}
