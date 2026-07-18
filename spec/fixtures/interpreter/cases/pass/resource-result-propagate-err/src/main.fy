module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn pass(input: Result<i32, Token>) -> Result<i32, Token> {
    let value: i32 = input?;
    Result::Ok(value)
}

fn main() -> i32 {
    match pass(Result::Err(Token { value: 42 })) {
        Result::Ok(value) => value,
        Result::Err(token) => token.value,
    }
}
