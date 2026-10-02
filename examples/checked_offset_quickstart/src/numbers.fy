module numbers;

import std::num as num;
import std::process as process;

pub enum InputError {
    Unavailable;
    TooLong;
    Empty;
    InvalidDecimal;
    OutsideI64;
    Negative;
    OutsideUsize;
}

// Keep caller diagnostics while delegating decimal syntax and bounds to the SDK.
pub fn Result<i64, InputError> parse(ref u8[] raw) {
    return switch (num::parse_i64_decimal(raw)) {
        case .ok(value): { .ok(value) }
        case .err(error): { switch (error) {
            case .Empty: { .err(.Empty) }
            case .InvalidDecimal: { .err(.InvalidDecimal) }
            case .OutOfRange: { .err(.OutsideI64) }
        } }
    };
}

pub fn Result<usize, InputError> argument(usize index) {
    var u8[20] raw = [0; 20];
    let usize used = switch (process::arg(index, ref mut raw)) {
        case .ok(count): { count }
        case .err(error): {
            return .err(if (error.status == 3) { .TooLong } else { .Unavailable });
        }
    };
    let ref u8[] view = ref raw;
    let i64 value = parse(view[..used])?;
    return switch (num::usize_from_i64(value)) {
        case .ok(converted): { .ok(converted) }
        case .err(error): {
            switch (error) {
                case .Negative: { .err(.Negative) }
                case .OutOfRange: { .err(.OutsideUsize) }
            }
        }
    };
}
