module main;

fn i32 main() {
    let Result<i32, i32> result = .err(-42);
    return switch (result) {
        case .ok(value) : { value }
        case .err(error) : { -error }
    };
}
