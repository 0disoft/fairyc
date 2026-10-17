package cbackendnative

const buildSmokeSource = `module main;
resource Token {
    i32 value;
    cleanup {}
}
fn Option<Token> choose() { return .some(Token { .value = 41 }); }
fn Result<Token, i32> increment(Result<Token, i32> input) {
	let Token token = input?;
	return .ok(Token { .value = token.value + 1 });
}
fn i32 main() {
	let Token token = switch (choose()) {
		case .some(token) : { token }
		case .none : { Token { .value = 0 } }
	};
	return switch (increment(.ok(token))) {
		case .ok(token) : { token.value }
		case .err(error) : { -error }
    };
}
`

const publicFileBuildSmokeSource = `module main;

import std::fs as fs;

@effects(filesystem)
fn i32 write_then_close(fs::File input) {
    var fs::File file = input;
    let ref u8[] payload = b"Fairy C";
    let i32 write_code = switch (fs::write_all(ref mut file, payload)) {
        case .ok(written) : { if (written == 7) { 0 } else { 1 } }
        case .err(_) : { 2 }
    };
    if (write_code != 0) {
        return write_code;
    }
    let i32 position_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 7) { 0 } else { 10 } }
        case .err(_) : { 11 }
    };
    if (position_code != 0) {
        return position_code;
    }
    let i32 size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 7) { 0 } else { 22 } }
        case .err(_) : { 23 }
    };
    if (size_code != 0) {
        return size_code;
    }
    let i32 seek_code = switch (fs::seek_start(ref mut file, 2)) {
        case .ok(position) : { if (position == 2) { 0 } else { 12 } }
        case .err(_) : { 13 }
    };
    if (seek_code != 0) {
        return seek_code;
    }
    let i32 observed_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 2) { 0 } else { 14 } }
        case .err(_) : { 15 }
    };
    if (observed_code != 0) {
        return observed_code;
    }
    let i32 preserved_size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 7) { 0 } else { 24 } }
        case .err(_) : { 25 }
    };
    if (preserved_size_code != 0) {
        return preserved_size_code;
    }
    let i32 size_position_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 2) { 0 } else { 26 } }
        case .err(_) : { 27 }
    };
    if (size_position_code != 0) {
        return size_position_code;
    }
    let i32 current_code = switch (fs::seek_current(ref mut file, -1)) {
        case .ok(position) : { if (position == 1) { 0 } else { 28 } }
        case .err(_) : { 29 }
    };
    if (current_code != 0) {
        return current_code;
    }
    let i32 end_code = switch (fs::seek_end(ref mut file, -2)) {
        case .ok(position) : { if (position == 5) { 0 } else { 30 } }
        case .err(_) : { 31 }
    };
    if (end_code != 0) {
        return end_code;
    }
    let i32 beyond_end_code = switch (fs::seek_end(ref mut file, 3)) {
        case .ok(position) : { if (position == 10) { 0 } else { 32 } }
        case .err(_) : { 33 }
    };
    if (beyond_end_code != 0) {
        return beyond_end_code;
    }
    let i32 beyond_end_size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 7) { 0 } else { 34 } }
        case .err(_) : { 35 }
    };
    if (beyond_end_size_code != 0) {
        return beyond_end_size_code;
    }
    let i32 beyond_end_position_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 10) { 0 } else { 36 } }
        case .err(_) : { 37 }
    };
    if (beyond_end_position_code != 0) {
        return beyond_end_position_code;
    }
    let i32 reset_code = switch (fs::seek_start(ref mut file, 2)) {
        case .ok(position) : { if (position == 2) { 0 } else { 38 } }
        case .err(_) : { 39 }
    };
    if (reset_code != 0) {
        return reset_code;
    }
    let usize invalid_offset = 18446744073709551615;
    let i32 rejected_code = switch (fs::seek_start(ref mut file, invalid_offset)) {
        case .ok(_) : { 18 }
        case .err(error) : { if (fs::error_status(error) == 2) { 0 } else { 19 } }
    };
    if (rejected_code != 0) {
        return rejected_code;
    }
    let i32 preserved_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 2) { 0 } else { 20 } }
        case .err(_) : { 21 }
    };
    if (preserved_code != 0) {
        return preserved_code;
    }
    let i32 restore_code = switch (fs::seek_start(ref mut file, 7)) {
        case .ok(position) : { if (position == 7) { 0 } else { 16 } }
        case .err(_) : { 17 }
    };
    if (restore_code != 0) {
        return restore_code;
    }
    let i32 sync_code = switch (fs::sync(ref mut file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 3 }
    };
    if (sync_code != 0) {
        return sync_code;
    }
    return switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 4 }
    };
}

@effects(filesystem)
fn i32 read_then_close(fs::File input) {
    var fs::File file = input;
    var u8[7] buffer = [0, 0, 0, 0, 0, 0, 0];
    let i32 read_code = switch (fs::read(ref mut file, ref mut buffer)) {
        case .ok(read) : {
            if (read == 7 &&
                buffer[0] == 70 && buffer[1] == 97 && buffer[2] == 105 &&
                buffer[3] == 114 && buffer[4] == 121 && buffer[5] == 32 &&
                buffer[6] == 67) { 0 } else { 5 }
        }
        case .err(_) : { 6 }
    };
    if (read_code != 0) {
        return read_code;
    }
    return switch (fs::close(file)) {
        case .ok(_) : { 42 }
        case .err(_) : { 7 }
    };
}

@effects(filesystem)
fn i32 utf8_round_trip() {
    let ref str path = "fairyc-public-file-한글.tmp";
    let i32 write_code = switch (fs::create_truncate_utf8(path)) {
        case .ok(file) : { write_then_close(file) }
        case .err(_) : { 40 }
    };
    if (write_code != 0) {
        return write_code;
    }
    return switch (fs::open_read_utf8(path)) {
        case .ok(file) : { if (read_then_close(file) == 42) { 0 } else { 41 } }
        case .err(_) : { 42 }
    };
}

@effects(filesystem)
fn i32 main() {
    let i32 utf8_code = utf8_round_trip();
    if (utf8_code != 0) {
        return utf8_code;
    }
    let ref CStr path = c"fairyc-public-file-smoke.tmp";
    let i32 write_code = switch (fs::create_truncate(path)) {
        case .ok(file) : { write_then_close(file) }
        case .err(_) : { 8 }
    };
    if (write_code != 0) {
        return write_code;
    }
    return switch (fs::open_read(path)) {
        case .ok(file) : { read_then_close(file) }
        case .err(_) : { 9 }
    };
}
`
