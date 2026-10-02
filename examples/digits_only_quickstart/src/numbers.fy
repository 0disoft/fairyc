module numbers;

import std::num as num;

// Application syntax policy: forbid a sign, including all negative zero spellings.
// The SDK still owns decimal accumulation, exact bounds and first-failure order.
pub fn Result<i64, num::ParseI64Error> parse(ref u8[] raw) {
    if (raw.len > 0) {
        if (raw[0] == 45) { return .err(.InvalidDecimal); }
    }
    return num::parse_i64_decimal(raw);
}
