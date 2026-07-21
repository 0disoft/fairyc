module main;

resource Token {
    i32 value;

    cleanup {}
}

resource Probe {
    i32 value;

    cleanup {}
}

fn Result<Token, i32> replacement(bool fail) {
    let Probe probe = Probe { .value = 0 };
    return if (fail) {
        .err(7)
    } else {
        .ok(Token { .value = 42 })
    };
}

fn Result<i32, i32> update(bool fail) {
    var Token token = Token { .value = 1 };
    token = replacement(fail)?;
    return .ok(token.value);
}

fn i32 main() {
    return switch (update(true)) {
        case .ok(value) : { value }
        case .err(error) : { error }
    };
}
