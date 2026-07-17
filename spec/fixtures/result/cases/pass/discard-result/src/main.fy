module main;

enum Error {
    Failed;
}

fn ignore(input: Result<i32, Error>) -> i32 {
    discard input;
    0
}
