module main;

fn valid(input: Result<i32, i32>) -> i32 {
    let forwarded: Result<i32, i32> = input;
    discard forwarded;
    0
}
