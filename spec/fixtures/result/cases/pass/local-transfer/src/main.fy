module main;

fn i32 valid(Result<i32, i32> input) {
    let Result<i32, i32> forwarded = input;
    discard forwarded;
    return 0;
}
