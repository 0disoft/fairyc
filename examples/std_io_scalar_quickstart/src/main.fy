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
fn bool write_i32_to_stderr(i32 value, usize expected_length) {
    return switch (io::write_stderr_i32(value)) {
        case .ok(written) : { written == expected_length }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_u32_to_stderr(u32 value, usize expected_length) {
    return switch (io::write_stderr_u32(value)) {
        case .ok(written) : { written == expected_length }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_usize_to_stderr(usize value, usize expected_length) {
    return switch (io::write_stderr_usize(value)) {
        case .ok(written) : { written == expected_length }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_bool_to_stderr(bool value, usize expected_length) {
    return switch (io::write_stderr_bool(value)) {
        case .ok(written) : { written == expected_length }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_text_to_stderr(ref str value) {
    return switch (io::write_stderr(value)) {
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

    let bool stderr_i32_zero_ok = write_i32_to_stderr(0, 1);
    let bool stderr_newline_1_ok = write_text_to_stderr("\n");
    let bool stderr_i32_negative_ok = write_i32_to_stderr(-42, 3);
    let bool stderr_newline_2_ok = write_text_to_stderr("\n");
    let bool stderr_i32_min_ok = write_i32_to_stderr(-2147483648, 11);
    let bool stderr_newline_3_ok = write_text_to_stderr("\n");
    let bool stderr_i32_max_ok = write_i32_to_stderr(2147483647, 10);
    let bool stderr_newline_4_ok = write_text_to_stderr("\n");
    let bool stderr_u32_zero_ok = write_u32_to_stderr(0, 1);
    let bool stderr_newline_5_ok = write_text_to_stderr("\n");
    let bool stderr_u32_max_ok = write_u32_to_stderr(4294967295, 10);
    let bool stderr_newline_6_ok = write_text_to_stderr("\n");
    let bool stderr_usize_zero_ok = write_usize_to_stderr(0, 1);
    let bool stderr_newline_7_ok = write_text_to_stderr("\n");
    let bool stderr_usize_max_ok = write_usize_to_stderr(18446744073709551615, 20);
    let bool stderr_newline_8_ok = write_text_to_stderr("\n");
    let bool stderr_bool_true_ok = write_bool_to_stderr(true, 4);
    let bool stderr_newline_9_ok = write_text_to_stderr("\n");
    let bool stderr_bool_false_ok = write_bool_to_stderr(false, 5);
    let bool stderr_newline_10_ok = write_text_to_stderr("\n");

    return if (u32_zero_ok && newline_1_ok && u32_one_ok && newline_2_ok && u32_max_ok && newline_3_ok &&
        usize_zero_ok && newline_4_ok && usize_one_ok && newline_5_ok && usize_max_ok && newline_6_ok &&
        bool_true_ok && newline_7_ok && bool_false_ok && newline_8_ok &&
        stderr_i32_zero_ok && stderr_newline_1_ok && stderr_i32_negative_ok && stderr_newline_2_ok &&
        stderr_i32_min_ok && stderr_newline_3_ok && stderr_i32_max_ok && stderr_newline_4_ok &&
        stderr_u32_zero_ok && stderr_newline_5_ok && stderr_u32_max_ok && stderr_newline_6_ok &&
        stderr_usize_zero_ok && stderr_newline_7_ok && stderr_usize_max_ok && stderr_newline_8_ok &&
        stderr_bool_true_ok && stderr_newline_9_ok && stderr_bool_false_ok && stderr_newline_10_ok) { 0 } else { 1 };
}
