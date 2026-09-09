module utf8;

fn bool tail(u8 c) { return c >= 128 && c <= 191; }

// Zero means malformed/truncated encoding; callers consume one byte on failure.
pub fn usize width(ref u8[] input, usize start) {
    if (start >= input.len) { return 0; }
    let u8 a = input[start];
    if (a < 128) { return 1; }
    if (a < 194 || a > 244 || input.len - start < 2) { return 0; }
    let u8 b = input[start+1];
    if (!tail(b)) { return 0; }
    if (a <= 223) { return 2; }
    if (input.len - start < 3 || !tail(input[start+2])) { return 0; }
    if (a <= 239) {
        if ((a == 224 && b < 160) || (a == 237 && b >= 160)) { return 0; }
        return 3;
    }
    if ((a == 240 && b < 144) || (a == 244 && b > 143)) { return 0; }
    if (input.len - start < 4 || !tail(input[start+3])) { return 0; }
    return 4;
}
