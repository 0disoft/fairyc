module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn read(token: &Token) -> i32 {
    token.value
}

fn run() -> i32 {
    let token: Token = Token { value: 1 };
    read(&token)
}
