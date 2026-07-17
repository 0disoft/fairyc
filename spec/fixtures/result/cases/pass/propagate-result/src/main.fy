module main;

enum Error {
    Failed;
}

fn propagate(
    input: Result<i32, Error>,
    fallback: Result<i32, Error>,
) -> Result<i32, Error> {
    let value: i32 = input?;
    fallback
}
