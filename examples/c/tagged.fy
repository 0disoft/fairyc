module examples::c::tagged;

fn Option<i32> choose() {
    return .some(41);
}

fn Result<i32, i32> increment(Result<i32, i32> input) {
    let i32 value = input?;
    return .ok(value + 1);
}

fn i32 main() {
    let i32 base = switch (choose()) {
        case .some(value) : { value }
        case .none : { 0 }
    };
    return switch (increment(.ok(base))) {
        case .ok(value) : { value }
        case .err(error) : { -error }
    };
}
