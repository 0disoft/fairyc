module tests::output;

import std::io as io;

@test
@effects(console)
fn void alpha_passes() {
    var u8[2] output = [0, 255];
    let ref u8[] bytes = ref output;
    let bool wrote = switch (io::write_bytes(bytes)) {
        case .ok(count) : { count == 2 }
        case .err(_) : { false }
    };
    assert(wrote);
}

@test
@effects(console)
fn void zeta_fails() {
    var u8[2] output = [10, 128];
    let ref u8[] bytes = ref output;
    let bool unexpected_count = switch (io::write_stderr_bytes(bytes)) {
        case .ok(count) : { count == 1 }
        case .err(_) : { false }
    };
    assert(unexpected_count);
}
