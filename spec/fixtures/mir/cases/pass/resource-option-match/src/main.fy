module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    let source: Token = Token { value: 42 };
    let wrapped: Option<Token> = Option::Some(source);
    match wrapped {
        Option::Some(token) => token.value,
        Option::None => 0,
    }
}
