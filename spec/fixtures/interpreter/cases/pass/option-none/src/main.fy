module main;

fn Option<i32> choose() {
    return .none;
}

fn i32 main() {
    return switch (choose()) {
        case .some(value) : { value }
        case .none : { 42 }
    };
}
