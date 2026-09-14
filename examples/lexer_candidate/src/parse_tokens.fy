module parse_tokens;
// Statement keyword classification shared without changing parser arena ownership.
import scanner as scanner;

pub fn bool atom(ref scanner::Stream stream, usize index) {
    let u32 kind=scanner::token_info(stream,index).kind;
    return kind == 5 || kind == 6 || kind == 7 || kind == 8 || kind == 9 || kind == 3 || word(stream,index,b"true") || word(stream,index,b"false") || word(stream,index,b"null");
}
pub fn bool prefix(ref scanner::Stream stream, usize index) {
    return word(stream,index,b"-") || word(stream,index,b"+") || word(stream,index,b"!") || word(stream,index,b"~") || word(stream,index,b"*");
}

pub struct PatternBinding { bool ok; usize next; usize end; usize start; usize field; usize name; u32 expected; }
pub fn PatternBinding pattern_binding(ref scanner::Stream stream, usize index, bool named) {
    var PatternBinding r = PatternBinding { .ok=false, .next=index, .end=scanner::token_info(stream,index).start, .start=scanner::token_info(stream,index).start, .field=index, .name=index, .expected=0 };
    if (named) {
        r.expected=14; if (!word(stream,r.next,b".")) { return r; }
        r.end=scanner::token_info(stream,r.next).end; r.next=significant(stream,r.next+1);
        r.field=r.next; r.expected=1;
        if (scanner::token_info(stream,r.next).kind != 3) { return r; }
        r.end=scanner::token_info(stream,r.next).end; r.next=significant(stream,r.next+1);
        r.expected=9; if (!word(stream,r.next,b"=")) { return r; }
        r.end=scanner::token_info(stream,r.next).end; r.next=significant(stream,r.next+1);
    }
    r.name=r.next; r.expected=1;
    if (scanner::token_info(stream,r.next).kind != 3) { return r; }
    r.end=scanner::token_info(stream,r.next).end; r.next=significant(stream,r.next+1);
    r.ok=true; r.expected=0; return r;
}

pub fn bool inferred_binding(ref scanner::Stream stream, usize index) {
    if (scanner::token_info(stream,index).kind != 3) { return false; }
    var usize next = index + 1;
    while (next < scanner::token_count(stream) && scanner::token_info(stream,next).kind == 2) { next += 1; }
    return next < scanner::token_count(stream) && word(stream,next,b"=");
}

pub fn usize significant(ref scanner::Stream stream, usize start) {
    var usize index = start;
    while (scanner::token_info(stream,index).kind == 2 && index + 1 < scanner::token_count(stream)) { index += 1; }
    return index;
}

pub fn bool word(ref scanner::Stream stream, usize index, ref u8[] expected) {
    let ref u8[] raw = scanner::token_raw(stream, index);
    if (raw.len != expected.len) { return false; }
    var usize byte_index = 0;
    while (byte_index < raw.len) {
        if (raw[byte_index] != expected[byte_index]) { return false; }
        byte_index += 1;
    }
    return true;
}

pub fn usize precedence(ref scanner::Stream stream, usize index) {
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

pub fn bool jump(ref scanner::Stream stream, usize index) {
    return word(stream,index,b"break") || word(stream,index,b"continue");
}
pub fn bool block_start(ref scanner::Stream stream, usize index) {
    return word(stream,index,b"{") || word(stream,index,b"unsafe");
}
pub fn bool constructor_kind(u32 kind) { return kind == 2 || kind == 20 || kind == 28; }
pub fn bool block_kind(u32 kind) { return kind == 12 || kind == 13 || kind == 31; }
pub fn u32 operand_statement(ref scanner::Stream stream, usize index) {
    if (word(stream,index,b"discard")) { return 12; }
    if (word(stream,index,b"defer")) { return 13; }
    if (word(stream,index,b"burn")) { return 16; }
    return 0;
}
pub fn bool assignment(ref scanner::Stream stream, usize index) {
    return word(stream, index, b"=") || word(stream, index, b"+=") || word(stream, index, b"-=") || word(stream, index, b"*=") || word(stream, index, b"/=") || word(stream, index, b"%=") || word(stream, index, b"<<=") || word(stream, index, b">>=") || word(stream, index, b"&=") || word(stream, index, b"^=") || word(stream, index, b"|=");
}
