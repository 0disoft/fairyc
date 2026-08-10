module main;

fn i32 main() {
    let Option<Option<i32>> nested = .some(.some(42));
    return switch (nested) {
        case .some(value) : { switch (value) {
            case .some(inner) : { inner }
            case .none : { 0 }
        } }
        case .none : { 0 }
    };
}
