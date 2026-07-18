module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    let wrapped: Result<Token, i32> = Result::Ok(Token { value: 42 });
    match wrapped {
        Result::Ok(token) => token.value,
        Result::Err(error) => -error,
    }
}
