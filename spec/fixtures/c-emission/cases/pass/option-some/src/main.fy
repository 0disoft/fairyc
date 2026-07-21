module main;

fn Option<i32> choose() {
    return .some(42);
}

fn i32 main() {
    return switch (choose()) {
        case .some(value) : { value }
        case .none : { 0 }
    };
}
