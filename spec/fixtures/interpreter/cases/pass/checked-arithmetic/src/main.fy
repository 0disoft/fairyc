module main;

fn i32 add_overflow() {
    return 2147483647 + 1;
}

fn i32 subtract_overflow() {
    return (-2147483647 - 1) - 1;
}

fn i32 multiply_overflow() {
    return 50000 * 50000;
}

fn i32 negate_overflow() {
    return -(-2147483647 - 1);
}

fn i32 subtract_boundary() {
    return -2147483647 - 1;
}

fn i32 direct_minimum_literal() {
    return -0x8000_0000;
}

fn i32 multiply_boundary() {
    return 46340 * 46340;
}

fn i32 negate_boundary() {
    return -2147483647;
}

fn i32 right_shift_invalid() {
    return 1 >> 32;
}

fn i32 right_shift_negative() {
    return 8 >> -1;
}

fn i32 right_shift_boundary() {
    return -8 >> 2;
}

fn i32 division_overflow() {
    return (-2147483647 - 1) / -1;
}

fn i32 remainder_zero() {
    return 42 % 0;
}

fn i32 remainder_overflow() {
    return (-2147483647 - 1) % -1;
}

fn i32 remainder_boundary() {
    return 43 % 5;
}
