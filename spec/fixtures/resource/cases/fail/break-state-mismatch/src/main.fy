module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn consume(token: Token) -> i32 {
    token.value
}

fn invalid(flag: bool) -> i32 {
    let token: Token = Token { value: 1 };
    while flag {
        consume(token);
        break;
    }
    0
}
