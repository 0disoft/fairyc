module main;
import std::io as io;

@effects(console)
fn i32 main() {
    var i64[5] values = [-5, 0, 4000000000, 20, 7];
    for (let ref mut i64 value in values) {
        if (*value < 0) { *value = 0; }
        if (*value > 10) { *value = 10; }
    }
    for (let i64 value in values) {
        if (!switch (io::write_i64(value)) {
            case .ok(_): { true }
            case .err(_): { false }
        }) { return 1; }
        if (!switch (io::write("\n")) {
            case .ok(_): { true }
            case .err(_): { false }
        }) { return 1; }
    }
    return 0;
}
