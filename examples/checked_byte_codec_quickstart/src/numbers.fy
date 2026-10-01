module numbers;

import std::process as process;

pub enum InputError {
    Unavailable;
    TooLong;
    Empty;
    InvalidDecimal;
    OutsideI64;
}

// Accumulate negative digits so i64 minimum never needs an invalid positive magnitude.
pub fn Result<i64, InputError> parse(ref u8[] raw) {
    if (raw.len == 0) { return .err(.Empty); }
    let bool negative = raw[0] == 45;
    var usize at = if (negative) { 1 } else { 0 };
    if (at == raw.len) { return .err(.InvalidDecimal); }
    let i64 limit = if (negative) { -9223372036854775808 } else { -9223372036854775807 };
    var i64 value = 0;
    while (at < raw.len) {
        if (raw[at] < 48 || raw[at] > 57) { return .err(.InvalidDecimal); }
        let i64 digit = (raw[at] - 48) as i64;
        if (value < limit / 10 || (value == limit / 10 && digit > -(limit % 10))) {
            return .err(.OutsideI64);
        }
        value = value * 10 - digit;
        at += 1;
    }
    return .ok(if (negative) { value } else { -value });
}

pub fn Result<i64, InputError> argument(usize index) {
    var u8[20] raw = [0; 20];
    let usize used = switch (process::arg(index, ref mut raw)) {
        case .ok(count): { count }
        case .err(error): {
            return .err(if (error.status == 3) { .TooLong } else { .Unavailable });
        }
    };
    let ref u8[] view = ref raw;
    return parse(view[..used]);
}
