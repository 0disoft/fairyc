module main;

resource Error {
    code: i32;

    drop fn(self: &mut Self) {}
}

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn run(
    input: Result<i32, Error>,
    fallback: Result<i32, Error>,
) -> Result<i32, Error> {
    let token: Token = Token { value: 1 };
    let value: i32 = input?;
    fallback
}
