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
    match read(Result::Ok(Token { value: 42 })) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
