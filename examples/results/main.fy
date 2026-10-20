module examples::results::main;

enum Error {
    Failed;
}

fn Result<i32, Error> forward(Result<i32, Error> input) {
    return input;
}

fn i32 ignore(Result<i32, Error> input) {
    discard input;
    return 0;
}

fn Result<i32, Error> propagate(Result<i32, Error> input, Error fallback) {
    let i32 value = input?;
    return .err(fallback);
}
