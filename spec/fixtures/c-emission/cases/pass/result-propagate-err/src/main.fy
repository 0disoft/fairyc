module main;

fn Result<i32, i32> increment(Result<i32, i32> input) {
    let i32 value = input?;
    return .ok(value + 1);
}

fn i32 main() {
    return switch (increment(.err(-42))) {
        case .ok(value) : { value }
        case .err(error) : { -error }
    };
}
