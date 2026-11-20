module main;

fn T answer<T>() { return 42; }

fn T forward<T>() { return { (answer()) }; }

fn Option<T> pack<T>(T value) { return .some(value); }

fn i32 finish() { return answer(); }

fn i32 main() {
    let Option<u32> packed = { pack(42) };
    let u32 right = forward();
    let u32 same = answer<u32>();
    return switch (packed) {
        case .none: { 0 }
        case .some(value): {
            if (value == right && right == same) { finish() } else { 0 }
        }
    };
}
