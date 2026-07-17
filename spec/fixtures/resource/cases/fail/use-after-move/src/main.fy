module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn consume(token: Token) -> i32 {
    token.value
}

fn invalid() -> i32 {
    let token: Token = Token { value: 1 };
    let first: i32 = consume(token);
    consume(token)
}
