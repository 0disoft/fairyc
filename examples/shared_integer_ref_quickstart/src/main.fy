module main;
import std::io as io;

fn Option<ref i64> find(ref i64[] values, i64 needle) {
    for (let ref i64 value in values) {
        if (*value == needle) { return .some(value); }
    }
    return .none;
}

@effects(console)
fn i32 main() {
    var i64[4] values = [-5, 4000000000, 20, 7];
    let ref i64[] view = ref values;
    switch (find(view, 4000000000)) {
        case .some(value): {
            if (!switch (io::write_i64(*value)) {
                case .ok(_): { true }
                case .err(_): { false }
            }) { return 1; }
            if (!switch (io::write("\n")) {
                case .ok(_): { true }
                case .err(_): { false }
            }) { return 1; }
        }
        case .none: { return 2; }
    };
    switch (find(view, 99)) {
        case .some(_): { return 3; }
        case .none: {}
    };
    for (let ref i64 value in values) { assert(*value != 99); }
    // All references have reached their last use; mutation is available again.
    values[0] = 0;
    assert(values[0] == 0);
    return 0;
}
