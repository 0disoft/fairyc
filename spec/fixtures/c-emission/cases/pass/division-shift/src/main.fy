module main;

fn main() -> i32 {
    let quotient: i32 = 84 / 2;
    let shifted: i32 = -8 >> 1;
    let minimum: i32 = -0x8000_0000;
    let recovered: i32 = minimum + 2147483647 + 1;
    quotient + shifted + 4 + recovered
}
