module main;

enum Error {
    Failed;
}

fn Result<i32, Error> forward(Result<i32, Error> input) {
    return input;
}
