module std::num;

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
