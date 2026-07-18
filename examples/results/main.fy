module examples::results::main;

enum Error {
    Failed;
}

fn forward(input: Result<i32, Error>) -> Result<i32, Error> {
    input
}

fn ignore(input: Result<i32, Error>) -> i32 {
    discard input;
    0
}

fn propagate(
    input: Result<i32, Error>,
    fallback: Result<i32, Error>,
) -> Result<i32, Error> {
    let value: i32 = input?;
    fallback
}
