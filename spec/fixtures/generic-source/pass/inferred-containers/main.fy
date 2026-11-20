module main;

fn Option<Option<T>> wrap<T>(T value) { return .some(.some(value)); }

fn bool has_nested<T>(Option<Option<T>> value) {
    return switch (value) {
        case .none: { false }
        case .some(inner): {
            switch (inner) {
                case .none: { false }
                case .some(_): { true }
            }
        }
    };
}

fn T unwrap<T>(Option<Option<T>> value, T fallback) {
    return switch (value) {
        case .none: { fallback }
        case .some(inner): {
            switch (inner) {
                case .none: { fallback }
                case .some(actual): { actual }
            }
        }
    };
}

fn Result<T,E> relay<T,E>(Result<T,E> value) { return value; }

fn T read<T,E>(Result<T,E> value, T fallback) {
    return switch (value) {
        case .ok(actual): { actual }
        case .err(_): { fallback }
    };
}

fn i32 main() {
    assert(has_nested((.some(.some(42)))));
    let Result<i32,u32> value = .ok(unwrap(wrap(42), 0));
    return read(relay(value), 0);
}
