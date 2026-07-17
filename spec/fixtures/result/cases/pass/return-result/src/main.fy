module main;

enum Error {
    Failed;
}

fn forward(input: Result<i32, Error>) -> Result<i32, Error> {
    input
}
