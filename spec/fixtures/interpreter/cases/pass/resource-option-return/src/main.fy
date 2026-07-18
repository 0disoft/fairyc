module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn unwrap(input: Option<Token>) -> Token {
    match input {
        Option::Some(token) => token,
        Option::None => Token { value: 0 },
    }
}

fn main() -> i32 {
    let token: Token = unwrap(Option::Some(Token { value: 42 }));
    token.value
}
