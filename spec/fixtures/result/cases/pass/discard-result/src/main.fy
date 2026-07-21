module main;

enum Error {
    Failed;
}

fn i32 ignore(Result<i32, Error> input) {
    discard input;
    return 0;
}
