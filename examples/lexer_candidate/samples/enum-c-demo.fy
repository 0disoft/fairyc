module main;
enum State { Cold; Ready; Done; }
fn State next(State state) {
    return switch (state) {
        case .Done: { .Cold }
        case .Cold: { .Ready }
        case .Ready: { .Done }
    };
}
fn i32 main() {
    var State state = .Cold;
    state = next(state);
    return switch (state) {
        case .Done: { 2 }
        case .Ready: { 1 }
        case .Cold: { 0 }
    };
}
