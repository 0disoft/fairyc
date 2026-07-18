module main;

fn invalid(input: Result<i32, i32>, flag: bool) -> i32 {
    if flag {
        discard input;
    } else {
        0;
    };
    0
}
