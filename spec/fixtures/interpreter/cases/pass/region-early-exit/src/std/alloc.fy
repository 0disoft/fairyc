module std::alloc;

pub resource Arena {
    opaque: i32;

    drop fn(self: &mut Self) {}
}

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn return_from_region() -> i32 {
    var arena: Arena = Arena { opaque: 0 };
    let outside: Token = Token { value: 20 };
    region scratch in &mut arena {
        let inner: Token = Token { value: 22 };
        let stored: &i32 = scratch.place(1);
        return outside.value + inner.value;
    }
}

fn propagate_from_region(input: Result<i32, i32>) -> Result<i32, i32> {
    var arena: Arena = Arena { opaque: 0 };
    let outside: Token = Token { value: 10 };
    region scratch in &mut arena {
        let inner: Token = Token { value: 32 };
        let stored: &i32 = scratch.place(2);
        let value: i32 = input?;
        return Result::Ok(value + outside.value + inner.value);
    }
}

fn propagate_main() -> i32 {
    match propagate_from_region(Result::Err(42)) {
        Result::Ok(value) => -value,
        Result::Err(error) => error,
    }
}

fn nested_return() -> i32 {
    var outer_arena: Arena = Arena { opaque: 0 };
    var inner_arena: Arena = Arena { opaque: 0 };
    let function_token: Token = Token { value: 2 };
    region outer in &mut outer_arena {
        let outer_token: Token = Token { value: 10 };
        let outer_stored: &i32 = outer.place(3);
        region inner in &mut inner_arena {
            let inner_token: Token = Token { value: 30 };
            let inner_stored: &i32 = inner.place(4);
            return function_token.value + outer_token.value + inner_token.value;
        }
    }
}

fn conditional_region(flag: bool) -> i32 {
    var arena: Arena = Arena { opaque: 0 };
    let outside: Token = Token { value: 40 };
    region scratch in &mut arena {
        let inner: Token = Token { value: 2 };
        if flag {
            return outside.value + inner.value;
        };
        let stored: &i32 = scratch.place(5);
    }
    outside.value
}

fn conditional_fallthrough_main() -> i32 {
    conditional_region(false)
}
