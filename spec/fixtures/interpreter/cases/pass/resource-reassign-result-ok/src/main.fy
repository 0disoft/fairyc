module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

resource Probe {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn replacement(fail: bool) -> Result<Token, i32> {
    let probe: Probe = Probe { value: 0 };
    if fail {
        Result::Err(7)
    } else {
        Result::Ok(Token { value: 42 })
    }
}

fn update(fail: bool) -> Result<i32, i32> {
    var token: Token = Token { value: 1 };
    token = replacement(fail)?;
    Result::Ok(token.value)
}

fn main() -> i32 {
    match update(false) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
