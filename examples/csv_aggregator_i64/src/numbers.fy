module numbers;

// Error 1: empty, 2: syntax, 3: i64 range. Accumulate negatively to admit i64 minimum.
pub fn Result<i64, i32> parse(ref u8[] input) {
    var usize start = 0;
    var usize end = input.len;
    while (start < end && (input[start] == 32 || input[start] == 9)) { start += 1; }
    while (end > start && (input[end - 1] == 32 || input[end - 1] == 9)) { end -= 1; }
    if (start == end) { return .err(1); }
    var bool negative = false;
    if (input[start] == 45 || input[start] == 43) {
        negative = input[start] == 45;
        start += 1;
    }
    if (start == end) { return .err(2); }
    let i64 limit = if (negative) { -9223372036854775808 } else { -9223372036854775807 };
    var i64 value = 0;
    while (start < end) {
        let u8 character = input[start];
        if (character < 48 || character > 57) { return .err(2); }
        let i64 digit = (character as i64) - 48;
        if (value < -922337203685477580) { return .err(3); }
        let i64 multiplied = value * 10;
        if (multiplied < limit + digit) { return .err(3); }
        value = multiplied - digit;
        start += 1;
    }
    return .ok(if (negative) { value } else { -value });
}

pub fn bool sum_fits(i64 sum, i64 value) {
    let i64 maximum = 9223372036854775807;
    let i64 minimum = -9223372036854775808;
    return !(value > 0 && sum > maximum - value) &&
        !(value < 0 && sum < minimum - value);
}
