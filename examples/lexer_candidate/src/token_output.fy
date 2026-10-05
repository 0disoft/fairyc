module token_output;
import scanner as scanner;
import std::bytes as bytes;
import std::text as text;
import std::io as io;

fn bool kind(ref mut bytes::Buffer out, u32 value) {
    let ref str label = if (value == 0) { "invalid" } else if (value == 1) { "eof" }
        else if (value == 2) { "trivia" } else if (value == 3) { "identifier" }
        else if (value == 4) { "keyword" } else if (value == 5) { "integer" }
        else if (value == 6) { "float" } else if (value == 7) { "string" }
        else if (value == 8) { "byte-string" } else if (value == 9) { "c-string" } else { "symbol" };
    return append(out, text::as_bytes(label));
}
fn bool trivia(ref mut bytes::Buffer out, u32 value) {
    let ref str label = if (value == 0) { "none" } else if (value == 1) { "preamble" }
        else if (value == 2) { "whitespace" } else if (value == 3) { "line-comment" }
        else if (value == 4) { "block-comment" } else { "documentation-line" };
    return append(out, text::as_bytes(label));
}
fn bool append(ref mut bytes::Buffer out, ref u8[] value) {
    if (value.len > 1048576 - bytes::len(out)) { return false; }
    return bytes::append(out, value).is_ok;
}
fn bool hex(ref mut bytes::Buffer out, ref u8[] value) {
    let ref u8[] digits = b"0123456789abcdef";
    for (let u8 byte in value) {
        let usize high = (byte / 16) as usize;
        let usize low = (byte % 16) as usize;
        if (!append(out, digits[high..high + 1]) || !append(out, digits[low..low + 1])) { return false; }
    }
    return true;
}

/// Confirm contiguous real spans and one terminal EOF before publishing TSV.
/// 3 allocation, 4 output cap, 5 invalid coverage. No console effects.
pub fn u32 tokens(ref scanner::Stream stream, ref mut bytes::Buffer out) {
    let usize count = scanner::token_count(stream);
    let ref u8[] source = scanner::source_view(stream);
    var usize position = 0;
    var usize index = 0;
    while (index < count) {
        let scanner::TokenInfo token = scanner::token_info(stream, index);
        if (token.start != position || token.end < token.start || token.end > source.len) { return 5; }
        if (token.kind == 1) {
            if (index + 1 != count || token.start != source.len || token.end != source.len) { return 5; }
        } else if (token.start == token.end || index + 1 == count) { return 5; }
        position = token.end; index += 1;
    }
    if (count == 0 || position != source.len) { return 5; }
    if (!append(out, b"kind\ttrivia\tstart\tend\traw_hex\tcooked_hex\n")) { return 3; }
    index = 0;
    while (index < count) {
        let scanner::TokenInfo token = scanner::token_info(stream, index);
        let ref u8[] raw = scanner::token_raw(stream, index);
        let ref u8[] cooked = scanner::token_cooked(stream, index);
        // Upper bound each row before appending, retaining a hard output cap.
        let usize maximum = 96 + raw.len * 2 + cooked.len * 2;
        if (maximum > 1048576 - bytes::len(out)) { return 4; }
        if (!kind(out, token.kind) || !append(out, b"\t") ||
            !trivia(out, token.trivia) || !append(out, b"\t") ||
            !bytes::append_usize(out, token.start).is_ok || !append(out, b"\t") ||
            !bytes::append_usize(out, token.end).is_ok || !append(out, b"\t") ||
            !hex(out, raw) || !append(out, b"\t") || !hex(out, cooked) || !append(out, b"\n")) { return 3; }
        index += 1;
    }
    return 0;
}

@effects(console)
pub fn i32 errors(ref scanner::Stream stream, ref u8[] path) {
    let ref u8[] source = scanner::source_view(stream);
    var usize index = 0;
    while (index < scanner::diagnostic_count(stream)) {
        let info = scanner::diagnostic_info(stream, index);
        prefix(source, path, info.start);
        discard io::write_stderr(if (info.code == 1) { "E0001" } else if (info.code == 4) { "E0004" }
            else if (info.code == 5) { "E0005" } else if (info.code == 6) { "E0006" } else { "E0010" });
        range(info.start, info.end); index += 1;
    }
    return 2;
}

@effects(console)
pub fn void prefix(ref u8[] source, ref u8[] path, usize offset) {
        var usize line = 1; var usize column = 1; var usize byte = 0;
        while (byte < offset) {
            if (source[byte] == 13) { line += 1; column = 1; }
            else if (source[byte] == 10) {
                if (byte == 0 || source[byte - 1] != 13) { line += 1; }
                column = 1;
            } else { column += 1; }
            byte += 1;
        }
        discard io::write_stderr_bytes(path); discard io::write_stderr(":");
        discard io::write_stderr_usize(line); discard io::write_stderr(":");
        discard io::write_stderr_usize(column); discard io::write_stderr(": error: ");
}

@effects(console)
pub fn void range(usize start, usize end) {
    discard io::write_stderr(" (bytes "); discard io::write_stderr_usize(start);
    discard io::write_stderr(".."); discard io::write_stderr_usize(end);
    discard io::write_stderr(")\n");
}
