module main;

fn Option<i64> read(bool available) {
    return if (available) { .some(4000000000) } else { .none };
}

fn i32 main() {
    let i64 value = switch (read(true)) {
        case .some(amount): { amount }
        case .none: { 0 }
    };
    assert(value == 4000000000);
    return 0;
}
