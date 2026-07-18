module main;

fn add_overflow() -> i32 {
    2147483647 + 1
}

fn subtract_overflow() -> i32 {
    (-2147483647 - 1) - 1
}

fn multiply_overflow() -> i32 {
    50000 * 50000
}

fn negate_overflow() -> i32 {
    -(-2147483647 - 1)
}

fn subtract_boundary() -> i32 {
    -2147483647 - 1
}

fn multiply_boundary() -> i32 {
    46340 * 46340
}

fn negate_boundary() -> i32 {
    -2147483647
}

fn right_shift_invalid() -> i32 {
    1 >> 32
}

fn right_shift_negative() -> i32 {
    8 >> -1
}

fn right_shift_boundary() -> i32 {
    -8 >> 2
}

fn division_overflow() -> i32 {
    (-2147483647 - 1) / -1
}

fn remainder_zero() -> i32 {
    42 % 0
}

fn remainder_overflow() -> i32 {
    (-2147483647 - 1) % -1
}

fn remainder_boundary() -> i32 {
    43 % 5
}
