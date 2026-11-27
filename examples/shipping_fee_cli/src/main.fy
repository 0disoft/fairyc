module main;

import std::io as io;
import std::num as num;
import std::process as process;

const i64 FREE_SHIPPING_MINIMUM = 30000;
const i32 STANDARD_SHIPPING_FEE = 3000;

pub fn i32 shipping_fee(i64 subtotal) {
    return if (subtotal >= FREE_SHIPPING_MINIMUM) { 0 } else { STANDARD_SHIPPING_FEE };
}

@effects(console)
fn i32 input_error(ref str message) {
    return switch (io::write_stderr(message)) {
        case .ok(count): { if (count == message.len) { 2 } else { 3 } }
        case .err(_): { 3 }
    };
}

@effects(console)
fn i32 main() {
    if (process::arg_count() != 2) {
        return input_error("Usage: shipping-fee SUBTOTAL_WON\n");
    }
    var u8[20] raw = [0; 20];
    let usize size = switch (process::arg(1, ref mut raw)) {
        case .ok(count): { count }
        case .err(_): { return input_error("subtotal is too long or unreadable\n"); }
    };
    let ref u8[] view = ref raw;
    let i64 subtotal = switch (num::parse_i64_decimal(view[..size])) {
        case .ok(value): { value }
        case .err(_): { return input_error("subtotal must be a nonnegative i64 integer\n"); }
    };
    if (subtotal < 0) {
        return input_error("subtotal must be a nonnegative i64 integer\n");
    }
    if (!io::write_i32(shipping_fee(subtotal)).is_ok) {
        return 3;
    }
    return switch (io::write("\n")) {
        case .ok(count): { if (count == 1) { 0 } else { 3 } }
        case .err(_): { 3 }
    };
}
