module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn consume(token: Token) -> i32 {
    token.value
}

fn main() -> i32 {
    let token: Token = Token { value: 42 };
    consume(token)
}
