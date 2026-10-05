module diagnostics;

pub struct Problem { u32 code; usize offset; }
pub struct Location { usize line; usize column; }

pub fn Problem problem(u32 code, usize offset) { return Problem { .code, .offset }; }

/// Byte positions are 1-based; CRLF is one newline and tabs occupy one column.
pub fn Location locate(ref u8[] source, usize offset) {
    assert(offset <= source.len);
    var usize line = 1;
    var usize column = 1;
    var usize index = 0;
    while (index < offset) {
        let u8 c = source[index];
        if (c == 13) {
            line += 1; column = 1;
        } else if (c == 10) {
            if (index == 0 || source[index - 1] != 13) { line += 1; }
            column = 1;
        } else { column += 1; }
        index += 1;
    }
    return Location { .line, .column };
}
