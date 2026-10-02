module main;

import numbers as numbers;
import std::num as num;

fn Result<i64, num::ParseI64Error> forward(ref u8[] raw) {
    let i64 value = numbers::parse(raw)?;
    return .ok(value);
}

fn bool expected(Result<i64, num::ParseI64Error> result, i32 code, i64 value) {
    return switch (result) {
        case .ok(actual): { code == 0 && actual == value }
        case .err(error): { switch (error) {
            case .Empty: { code == 1 }
            case .InvalidDecimal: { code == 2 }
            case .OutOfRange: { code == 3 }
        } }
    };
}

fn void check(ref u8[] raw, i32 code, i64 value) {
    assert(expected(numbers::parse(raw), code, value));
    assert(expected(forward(raw), code, value));
    // 128 bytes is this fixture's capacity, not a parser input limit.
    assert(raw.len <= 126);
    var u8[128] owner = [165; 128];
    var usize at = 0;
    while (at < raw.len) { owner[at + 1] = raw[at]; at += 1; }
    let ref u8[] full = ref owner;
    let ref u8[] field = full[1..raw.len + 1];
    assert(expected(numbers::parse(field), code, value));
    assert(expected(forward(field), code, value));
    // Sentinels outside the field must not affect parsing; the owner is preserved.
    assert(full[0] == 165);
    at = 0;
    while (at < raw.len) { assert(full[at + 1] == raw[at]); at += 1; }
    at = raw.len + 1;
    while (at < full.len) { assert(full[at] == 165); at += 1; }
}

fn i32 main() {
    // Fixed contract results: success=0, Empty=1, InvalidDecimal=2, OutOfRange=3.
    check(b"", 1, 0);
    check(b"0", 0, 0);
    check(b"000256", 0, 256);
    check(b"2147483648", 0, 2147483648);
    check(b"9223372036854775807", 0, 9223372036854775807);
    check(b"0009223372036854775807", 0, 9223372036854775807);
    check(b"00000000000000000000000000000000000000000000000000123", 0, 123);
    check(b"-", 2, 0);
    check(b"-0", 2, 0);
    check(b"-0000", 2, 0);
    check(b"-1", 2, 0);
    check(b"-9223372036854775808", 2, 0);
    check(b"-9223372036854775809", 2, 0);
    check(b"-9223372036854775809x", 2, 0);
    check(b"+1", 2, 0);
    check(b"1-0", 2, 0);
    check(b" 1", 2, 0);
    check(b"1 ", 2, 0);
    check(b"1\t0", 2, 0);
    check(b"1\n", 2, 0);
    check(b"0x10", 2, 0);
    check(b"1_0", 2, 0);
    check(b"1.0", 2, 0);
    check(b"9223372036854775808", 3, 0);
    check(b"9223372036854775808x", 3, 0);
    check(b"922337203685477580x8", 2, 0);
    check(b"99999999999999999999999999999999999999999999", 3, 0);
    { let u8[2] invalid_utf8 = [255, 48]; check(ref invalid_utf8, 2, 0); }
    { let u8[2] nul = [49, 0]; check(ref nul, 2, 0); }
    { let u8[3] bom = [239, 187, 191]; check(ref bom, 2, 0); }
    { let u8[3] unicode_digit = [239, 188, 145]; check(ref unicode_digit, 2, 0); }
    return 0;
}
