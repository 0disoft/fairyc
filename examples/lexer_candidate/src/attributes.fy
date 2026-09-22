module attributes;
import scanner as scanner;
import std::vec as v;
import types as types;

pub struct Attribute { usize start; usize end; usize name_start; usize name_end; usize argument_start; usize argument_count; }
pub struct Argument { u32 kind; usize start; usize end; usize token; }
pub struct Attempt { bool ok; usize next; usize end; usize count; usize arguments; i32 status; u32 expected; }
struct Cursor { usize index; usize end; usize count; usize arguments; i32 status; u32 expected; }

fn void advance(ref scanner::Stream stream, ref mut Cursor cursor) {
    cursor.end = scanner::token_info(stream,cursor.index).end;
    if (cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
    while (scanner::token_info(stream,cursor.index).kind == 2 && cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
}
fn bool take(ref scanner::Stream stream, ref mut Cursor cursor, ref u8[] text) {
    if (!types::word(stream,cursor.index,text)) { return false; }
    advance(stream,cursor); return true;
}
fn bool path(ref scanner::Stream stream, ref mut Cursor cursor) {
    if (scanner::token_info(stream,cursor.index).kind != 3) { cursor.expected = 1; return false; }
    advance(stream,cursor);
    while (take(stream,cursor,b"::")) {
        if (scanner::token_info(stream,cursor.index).kind != 3) { cursor.expected = 1; return false; }
        advance(stream,cursor);
    }
    return true;
}
fn bool read(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Attribute> attributes, ref mut v::Vec<Argument> arguments) {
    while (types::word(stream,cursor.index,b"@")) {
        if (cursor.count >= 256) { cursor.status = 4; return false; }
        let usize start = scanner::token_info(stream,cursor.index).start;
        advance(stream,cursor);
        let usize name_start = scanner::token_info(stream,cursor.index).start;
        if (!path(stream,cursor)) { return false; }
        let usize name_end = cursor.end; let usize argument_start = cursor.arguments;
        if (take(stream,cursor,b"(")) {
            while (!types::word(stream,cursor.index,b")")) {
                if (cursor.arguments >= 1024) { cursor.status = 4; return false; }
                let usize token = cursor.index; let scanner::TokenInfo info = scanner::token_info(stream,token);
                let u32 kind = if (info.kind == 3) { 1 } else if (info.kind == 5) { 2 } else if (info.kind == 7) { 3 } else { 0 };
                if (kind == 0) { cursor.expected = 15; return false; }
                if (kind == 1) { if (!path(stream,cursor)) { return false; } } else { advance(stream,cursor); }
                let Argument value = Argument { .kind, .start = info.start, .end = cursor.end, .token };
                if (!switch(v::push<Argument>(arguments,value)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return false; }
                cursor.arguments = cursor.arguments + 1;
                if (!take(stream,cursor,b",")) { break; }
            }
            if (!take(stream,cursor,b")")) { cursor.expected = 5; return false; }
        }
        let Attribute value = Attribute { .start, .end = cursor.end, .name_start, .name_end, .argument_start, .argument_count = cursor.arguments - argument_start };
        if (!switch(v::push<Attribute>(attributes,value)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return false; }
        cursor.count = cursor.count + 1;
    }
    return true;
}
pub fn Attempt parse(ref scanner::Stream stream, usize start, usize count, usize argument_count, ref mut v::Vec<Attribute> attributes, ref mut v::Vec<Argument> arguments) {
    var Cursor cursor = Cursor { .index = start, .end = 0, .count, .arguments = argument_count, .status = 0, .expected = 0 };
    let bool ok = read(stream,ref mut cursor,attributes,arguments);
    return Attempt { .ok, .next = cursor.index, .end = cursor.end, .count = cursor.count, .arguments = cursor.arguments, .status = cursor.status, .expected = cursor.expected };
}
