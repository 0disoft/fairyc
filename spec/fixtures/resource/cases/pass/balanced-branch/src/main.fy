module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn consume(token: Token) -> i32 {
    token.value
}

fn run(flag: bool) -> i32 {
    let token: Token = Token { value: 1 };
    let result: i32 = if flag { consume(token) } else { consume(token) };
    result
}
