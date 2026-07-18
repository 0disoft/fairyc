module main;

fn add(left: i32, right: i32) -> i32 {
    left + right
}

fn subtract(left: i32, right: i32) -> i32 {
    left - right
}

fn multiply(left: i32, right: i32) -> i32 {
    left * right
}

fn negate(value: i32) -> i32 {
    -value
}

fn shift_right(value: i32, amount: i32) -> i32 {
    value >> amount
}

fn remainder(left: i32, right: i32) -> i32 {
    left % right
}
