module main;

import std::process as process;

fn bool view_equals(ref u8[] actual, ref u8[] expected) {
    if (actual.len != expected.len) {
        return false;
    }
    let usize length = expected.len;
    return if (length == 0) {
        true
    } else if (length == 1) {
        actual[0] == expected[0]
    } else if (length == 2) {
        actual[0] == expected[0] && actual[1] == expected[1]
    } else if (length == 3) {
        actual[0] == expected[0] && actual[1] == expected[1] && actual[2] == expected[2]
    } else if (length == 4) {
        actual[0] == expected[0] && actual[1] == expected[1] && actual[2] == expected[2] &&
            actual[3] == expected[3]
    } else if (length == 5) {
        actual[0] == expected[0] && actual[1] == expected[1] && actual[2] == expected[2] &&
            actual[3] == expected[3] && actual[4] == expected[4]
    } else if (length == 6) {
        actual[0] == expected[0] && actual[1] == expected[1] && actual[2] == expected[2] &&
            actual[3] == expected[3] && actual[4] == expected[4] && actual[5] == expected[5]
    } else if (length == 7) {
        actual[0] == expected[0] && actual[1] == expected[1] && actual[2] == expected[2] &&
            actual[3] == expected[3] && actual[4] == expected[4] && actual[5] == expected[5] &&
            actual[6] == expected[6]
    } else if (length == 8) {
        actual[0] == expected[0] && actual[1] == expected[1] && actual[2] == expected[2] &&
            actual[3] == expected[3] && actual[4] == expected[4] && actual[5] == expected[5] &&
            actual[6] == expected[6] && actual[7] == expected[7]
    } else {
        false
    };
}

fn bool arg_matches(usize index, ref u8[] expected) {
    var u8[16] buffer = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
    return switch (process::arg(index, ref mut buffer)) {
        case .ok(read) : {
            let ref u8[] bytes = ref buffer;
            view_equals(bytes[..read], expected)
        }
        case .err(_) : { false }
    };
}

fn bool argv0_nonempty() {
    var u8[128] buffer = [
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    ];
    return switch (process::arg(0, ref mut buffer)) {
        case .ok(read) : { read > 0 }
        case .err(error) : { error.status == 3 && error.needed > 0 }
    };
}

fn bool out_of_range_rejected() {
    var u8[4] buffer = [0, 0, 0, 0];
    return switch (process::arg(6, ref mut buffer)) {
        case .ok(_) : { false }
        case .err(error) : { error.status == 1 }
    };
}

fn bool too_small_preserves() {
    var u8[3] buffer = [7, 8, 9];
    let bool rejected = switch (process::arg(2, ref mut buffer)) {
        case .ok(_) : { false }
        case .err(error) : { error.status == 3 && error.needed == 5 }
    };
    let ref u8[] bytes = ref buffer;
    return rejected && bytes[0] == 7 && bytes[1] == 8 && bytes[2] == 9;
}

fn i32 main() {
    if (process::arg_count() != 6) { return 10; }
    if (!argv0_nonempty()) { return 11; }
    if (!arg_matches(1, b"")) { return 12; }
    if (!arg_matches(2, b"Fairy")) { return 13; }
    if (!arg_matches(3, b"--locked")) { return 14; }
    if (!arg_matches(4, b"a b\"c\\d\\")) { return 15; }
    if (!arg_matches(5, b"\xed\x95\x9c\xea\xb8\x80\xce\xbb")) { return 16; }
    if (!out_of_range_rejected()) { return 17; }
    if (!too_small_preserves()) { return 18; }
    return 42;
}
