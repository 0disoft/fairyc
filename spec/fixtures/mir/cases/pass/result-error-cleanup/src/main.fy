module main;

resource Error {
    i32 code;

    cleanup {}
}

resource Token {
    i32 value;

    cleanup {}
}

fn Result<i32, Error> run(Result<i32, Error> input, Error fallback) {
    let Token token = Token { .value = 1 };
    let i32 value = input?;
    return .err(fallback);
}
