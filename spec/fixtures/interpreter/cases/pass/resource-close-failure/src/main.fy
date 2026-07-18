module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn finish(token: Token) -> i32 {
    token.value
}

fn close(token: Token, fail: bool) -> Result<i32, Token> {
    if fail {
        Result::Err(token)
    } else {
        Result::Ok(finish(token))
    }
}

fn main() -> i32 {
    let token: Token = Token { value: 42 };
    match close(token, true) {
        Result::Ok(value) => value,
        Result::Err(returned) => returned.value,
    }
}
