module std::num;

/// Reports empty input, invalid ASCII decimal syntax, or an exceeded i64 bound.
pub enum ParseI64Error {
    Empty;
    InvalidDecimal;
    OutOfRange;
}

/// Parses all bytes as -?[0-9]+, returning the first syntax or range failure.
pub fn Result<i64, ParseI64Error> parse_i64_decimal(ref u8[] raw) {
    if (raw.len == 0) { return .err(.Empty); }
    let bool negative = raw[0] == 45;
    var usize at = if (negative) { 1 } else { 0 };
    if (at == raw.len) { return .err(.InvalidDecimal); }
    let i64 limit = if (negative) { -9223372036854775808 } else { -9223372036854775807 };
    var i64 value = 0;
    while (at < raw.len) {
        let u8 byte = raw[at];
        if (byte < 48 || byte > 57) { return .err(.InvalidDecimal); }
        let i64 digit = (byte - 48) as i64;
        // Preflight both arithmetic intermediates; never form positive MIN.
        if (value < limit / 10 || (value == limit / 10 && digit > -(limit % 10))) {
            return .err(.OutOfRange);
        }
        value = value * 10 - digit;
        at += 1;
    }
    return .ok(if (negative) { value } else { -value });
}

/// Reports a value that cannot be represented by the requested integer type.
pub enum ConversionError {
    Negative;
    OutOfRange;
}

/// Returns the exact byte, Negative below zero, or OutOfRange above 255.
pub fn Result<u8, ConversionError> u8_from_i64(i64 value) {
    if (value < 0) { return .err(.Negative); }
    if (value > 255) { return .err(.OutOfRange); }
    var i64 remaining = value;
    var u8 result = 0;
    var u8 power = 1;
    while (remaining > 0) {
        if (remaining % 2 != 0) { result += power; }
        remaining /= 2;
        // The highest power is 128; never construct 256 in a u8.
        if (remaining > 0) { power *= 2; }
    }
    return .ok(result);
}

/// Returns the exact signed value, or OutOfRange above i64 maximum.
pub fn Result<i64, ConversionError> i64_from_usize(usize value) {
    if (value > 9223372036854775807) {
        return .err(.OutOfRange);
    }
    var usize remaining = value;
    var i64 result = 0;
    var i64 power = 1;
    while (remaining > 0) {
        if (remaining % 2 != 0) {
            result += power;
        }
        remaining /= 2;
        // Avoid constructing the unrepresentable signed power 2^63.
        if (remaining > 0) {
            power *= 2;
        }
    }
    return .ok(result);
}

/// Returns the exact unsigned value on x64, or Negative below zero.
pub fn Result<usize, ConversionError> usize_from_i64(i64 value) {
    if (value < 0) {
        return .err(.Negative);
    }
    var i64 remaining = value;
    var usize result = 0;
    var usize power = 1;
    while (remaining > 0) {
        if (remaining % 2 != 0) {
            result += power;
        }
        remaining /= 2;
        if (remaining > 0) {
            power *= 2;
        }
    }
    return .ok(result);
}
