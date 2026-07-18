module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn read(input: Result<Token, i32>) -> Result<i32, i32> {
    let token: Token = input?;
    Result::Ok(token.value)
}

fn main() -> i32 {
    let wrapped: Option<Token> = Option::Some(Token { value: 42 });
    let token: Token = match wrapped {
        Option::Some(token) => token,
        Option::None => Token { value: 0 },
    };
    match read(Result::Ok(token)) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
