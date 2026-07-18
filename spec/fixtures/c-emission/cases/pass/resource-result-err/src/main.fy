module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    let wrapped: Result<i32, Token> = Result::Err(Token { value: 42 });
    match wrapped {
        Result::Ok(value) => value,
        Result::Err(token) => token.value,
    }
}
