module main;

fn i32 add(i32 left, i32 right) {
    return left + right;
}

fn i32 subtract(i32 left, i32 right) {
    return left - right;
}

fn i32 multiply(i32 left, i32 right) {
    return left * right;
}

fn i32 negate(i32 value) {
    return -value;
}

fn i32 shift_right(i32 value, i32 amount) {
    return value >> amount;
}

fn i32 remainder(i32 left, i32 right) {
    return left % right;
}
