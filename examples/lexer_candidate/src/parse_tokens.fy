module parse_tokens;
import scanner as scanner;

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

pub fn bool assignment(ref scanner::Stream stream, usize index) {
    return word(stream, index, b"=") || word(stream, index, b"+=") || word(stream, index, b"-=") || word(stream, index, b"*=") || word(stream, index, b"/=") || word(stream, index, b"%=") || word(stream, index, b"<<=") || word(stream, index, b">>=") || word(stream, index, b"&=") || word(stream, index, b"^=") || word(stream, index, b"|=");
}
