module main;

import std::io as io;

@effects(console)
fn bool write_u32_to_stdout(u32 value, usize expected_length) {
    return switch (io::write_u32(value)) {
        case .ok(written) : { written == expected_length }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_usize_to_stdout(usize value, usize expected_length) {
    return switch (io::write_usize(value)) {
        case .ok(written) : { written == expected_length }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_bool_to_stdout(bool value, usize expected_length) {
    return switch (io::write_bool(value)) {
        case .ok(written) : { written == expected_length }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_text_to_stdout(ref str value) {
    return switch (io::write(value)) {
        case .ok(written) : { written == value.len }
        case .err(_) : { false }
    };
}

@effects(console)
fn i32 main() {
    let bool u32_zero_ok = write_u32_to_stdout(0, 1);
    let bool newline_1_ok = write_text_to_stdout("\n");
    let bool u32_one_ok = write_u32_to_stdout(1, 1);
    let bool newline_2_ok = write_text_to_stdout("\n");
    let bool u32_max_ok = write_u32_to_stdout(4294967295, 10);
    let bool newline_3_ok = write_text_to_stdout("\n");
    let bool usize_zero_ok = write_usize_to_stdout(0, 1);
    let bool newline_4_ok = write_text_to_stdout("\n");
    let bool usize_one_ok = write_usize_to_stdout(1, 1);
    let bool newline_5_ok = write_text_to_stdout("\n");
    let bool usize_max_ok = write_usize_to_stdout(18446744073709551615, 20);
    let bool newline_6_ok = write_text_to_stdout("\n");
    let bool bool_true_ok = write_bool_to_stdout(true, 4);
    let bool newline_7_ok = write_text_to_stdout("\n");
    let bool bool_false_ok = write_bool_to_stdout(false, 5);
    let bool newline_8_ok = write_text_to_stdout("\n");
    return if (u32_zero_ok && newline_1_ok && u32_one_ok && newline_2_ok && u32_max_ok && newline_3_ok &&
        usize_zero_ok && newline_4_ok && usize_one_ok && newline_5_ok && usize_max_ok && newline_6_ok &&
        bool_true_ok && newline_7_ok && bool_false_ok && newline_8_ok) { 0 } else { 1 };
}
