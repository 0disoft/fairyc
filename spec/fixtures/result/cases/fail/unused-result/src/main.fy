module main;

enum Error {
    Failed;
}

fn invalid(input: Result<i32, Error>) -> i32 {
    input;
    0
}
