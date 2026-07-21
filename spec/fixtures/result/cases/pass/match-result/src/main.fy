module main;

fn i32 unwrap(Result<i32, i32> input) {
    return switch (input) {
        case .ok(value) : { value }
        case .err(error) : { -error }
    };
}
