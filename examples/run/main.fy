module examples::run::main;

fn Result<i32, i32> answer(bool enabled) {
    return if (enabled) {
        .ok(20 + 22)
    } else {
        .err(-1)
    };
}

fn i32 main() {
    return switch (answer(true && 20 < 22)) {
        case .ok(value) : { value }
        case .err(error) : { error }
    };
}
