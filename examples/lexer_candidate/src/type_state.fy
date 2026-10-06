module type_state;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// 0 absent, 1 i32, 2 i64, 3 u8, 4 u32, 5 usize, 6 bool, 7 void, 8 callable.
pub struct Problem { u32 code; usize start; usize end; u32 expected; u32 actual; usize move_start; usize move_end; usize loop_start; usize loop_end; }
pub struct Value { u32 type; bool terminates; }
// Initialized, bounded heap slots keep large value copies off the native stack.
pub struct Slot { u32 symbol; u32 expression; u32 flag; usize use; usize entry; bool flexible; bool moving; }
pub fn v::Vec<Slot> empty() { return v::empty<Slot>(); }
pub fn bool initialize(ref mut v::Vec<Slot> tables, usize symbols, usize expressions, usize entries) {
    let usize length = if (symbols > expressions) { if (symbols > entries) { symbols } else { entries } }
        else { if (expressions > entries) { expressions } else { entries } };
    if (!v::reserve<Slot>(tables,length).is_ok) { return false; }
    while (v::len<Slot>(tables) < length) {
        if (!v::push<Slot>(tables,Slot { .symbol = 0, .expression = 0, .flag = 0, .use = 0, .entry = 0, .flexible = false, .moving = false }).is_ok) { return false; }
    }
    return true;
}
pub fn Problem ok() { return Problem { .code = 0, .start = 0, .end = 0, .expected = 0, .actual = 0, .move_start = 0, .move_end = 0, .loop_start=0, .loop_end=0 }; }
pub fn Problem fail(u32 code, usize start, usize end, u32 expected, u32 actual) {
    return Problem { .code, .start, .end, .expected, .actual, .move_start=0, .move_end=0, .loop_start=0, .loop_end=0 };
}
pub fn bool integer(u32 type) { return type >= 1 && type <= 5; }
// Declaration/field arenas have at most 1024 entries; no unchecked narrowing.
pub fn u32 small_id(usize value) {
    assert(value <= 1024); var usize left = value; var u32 result = 0;
    while (left != 0) { left -= 1; result += 1; } return result;
}
pub fn u32 primitive(ref syntax::Parsed parsed, usize root) {
    if (root == 0) { return 0; }
    let ref u8[] text = syntax::type_text(parsed,root);
    return if (bytes::equal(text,b"i32")) { 1 } else if (bytes::equal(text,b"i64")) { 2 }
        else if (bytes::equal(text,b"u8")) { 3 } else if (bytes::equal(text,b"u32")) { 4 }
        else if (bytes::equal(text,b"usize")) { 5 } else if (bytes::equal(text,b"bool")) { 6 } else { 7 };
}
pub fn bool operation(ref u8[] op) {
    return bytes::equal(op,b"+") || bytes::equal(op,b"-") || bytes::equal(op,b"*") ||
        bytes::equal(op,b"/") || bytes::equal(op,b"%") || bytes::equal(op,b"&") || bytes::equal(op,b"|") ||
        bytes::equal(op,b"^") || bytes::equal(op,b"<<") || bytes::equal(op,b">>");
}
pub fn bool comparison(ref u8[] op) {
    return bytes::equal(op,b"==") || bytes::equal(op,b"!=") || bytes::equal(op,b"<") ||
        bytes::equal(op,b"<=") || bytes::equal(op,b">") || bytes::equal(op,b">=");
}
pub fn bool equal(ref u8[] op) { return bytes::equal(op,b"==") || bytes::equal(op,b"!="); }

// Compare a lexer-admitted magnitude using checked bounded machine arithmetic;
// usize is 64 bits on this candidate's Windows/Linux AMD64 targets.
pub fn bool fits(ref u8[] raw, u32 type, bool minimum) {
    if (minimum && type != 1 && type != 2) { return false; }
    var usize maximum = if (type == 1) { 2147483647 } else if (type == 2) { 9223372036854775807 }
        else if (type == 3) { 255 } else if (type == 4) { 4294967295 } else { 18446744073709551615 };
    if (minimum) { maximum += 1; }
    var usize base = 10; var usize index = 0;
    if (raw.len >= 2 && raw[0] == 48) {
        if (raw[1] == 120 || raw[1] == 88) { base = 16; index = 2; }
        else if (raw[1] == 98 || raw[1] == 66) { base = 2; index = 2; }
    }
    var usize value = 0;
    while (index < raw.len) {
        let u8 byte = raw[index]; index += 1;
        if (byte == 95) { continue; }
        let usize digit = if (byte >= 48 && byte <= 57) { (byte - 48) as usize }
            else if (byte >= 65 && byte <= 70) { (byte - 65 + 10) as usize } else { (byte - 97 + 10) as usize };
        if (value > (maximum - digit) / base) { return false; }
        value = value * base + digit;
    }
    return true;
}
pub fn u32 symbols(ref v::Vec<Slot> tables, usize index) { return v::at<Slot>(tables,index).symbol; }
pub fn void set_symbols(ref mut v::Vec<Slot> tables, usize index, u32 value) { let ref mut Slot slot = v::at_mut<Slot>(tables,index); slot.symbol = value; }
pub fn u32 expressions(ref v::Vec<Slot> tables, usize index) { return v::at<Slot>(tables,index).expression; }
pub fn void set_expressions(ref mut v::Vec<Slot> tables, usize index, u32 value) { let ref mut Slot slot = v::at_mut<Slot>(tables,index); slot.expression = value; }
pub fn u32 flags(ref v::Vec<Slot> tables, usize index) { return v::at<Slot>(tables,index).flag; }
pub fn void set_flags(ref mut v::Vec<Slot> tables, usize index, u32 value) { let ref mut Slot slot = v::at_mut<Slot>(tables,index); slot.flag = value; }
pub fn usize uses(ref v::Vec<Slot> tables, usize index) { return v::at<Slot>(tables,index).use; }
pub fn void set_uses(ref mut v::Vec<Slot> tables, usize index, usize value) { let ref mut Slot slot = v::at_mut<Slot>(tables,index); slot.use = value; }
pub fn usize entries(ref v::Vec<Slot> tables, usize index) { return v::at<Slot>(tables,index).entry; }
pub fn void set_entries(ref mut v::Vec<Slot> tables, usize index, usize value) { let ref mut Slot slot = v::at_mut<Slot>(tables,index); slot.entry = value; }
pub fn bool flexible(ref v::Vec<Slot> tables, usize index) { return v::at<Slot>(tables,index).flexible; }
pub fn void set_flexible(ref mut v::Vec<Slot> tables, usize index, bool value) { let ref mut Slot slot = v::at_mut<Slot>(tables,index); slot.flexible = value; }
pub fn bool moving(ref v::Vec<Slot> tables, usize index) { return v::at<Slot>(tables,index).moving; }
pub fn void set_moving(ref mut v::Vec<Slot> tables, usize index, bool value) { let ref mut Slot slot = v::at_mut<Slot>(tables,index); slot.moving = value; }
