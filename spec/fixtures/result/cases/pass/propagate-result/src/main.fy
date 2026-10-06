module main;

enum Error {
    Failed;
}

fn Result<i32, Error> propagate(Result<i32, Error> input, Result<i32, Error> fallback) {
    discard fallback;
    let i32 value = input?;
    return fallback;
}
