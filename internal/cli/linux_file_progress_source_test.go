package cli

// Calls and observations go through the bundled SDK; only syscall scripting is foreign.
const linuxFileProgressSource = `module main;
import std::fs as fs;
import std::bytes as bytes;

@link_name("fairyc_file_progress_begin")
unsafe extern C fn void begin(u32 mode);
@link_name("fairyc_file_progress_end")
unsafe extern C fn void end();

fn bool matches(Result<usize, fs::FileError> result, u32 status, u32 os_code, usize completed) {
    return switch (result) {
        case .ok(count): { status == 0 && count == completed }
        case .err(error): {
            status != 0 && fs::error_status(error) == status && fs::error_os_code(error) == os_code &&
            fs::error_completed(error) == completed
        }
    };
}

fn bool control_matches(Result<i32, fs::FileError> result, u32 status, u32 os_code) {
    return switch (result) {
        case .ok(value): { status == 0 && value == 0 }
        case .err(error): {
            status != 0 && fs::error_status(error) == status && fs::error_os_code(error) == os_code &&
            fs::error_completed(error) == 0
        }
    };
}

@effects(filesystem)
fn void check_write(ref mut fs::File file, u32 mode, bool all, u32 status, u32 os_code, usize completed) {
    unsafe { begin(mode); }
    let ref u8[] payload = if (mode == 6) { b"" } else { b"A\0\xffbcd" };
    let bool correct = if (all) {
        matches(fs::write_all(file, payload), status, os_code, completed)
    } else {
        matches(fs::write(file, payload), status, os_code, completed)
    };
    unsafe { end(); }
    assert(correct);
}

@effects(filesystem)
fn void check_read(ref mut fs::File file, u32 mode, bool exact, u32 status, u32 os_code,
    usize completed, ref u8[] expected) {
    var u8[6] buffer = [165; 6];
    unsafe { begin(mode); }
    let bool correct = if (mode == 13) {
        matches(fs::read_exact(file, ref mut buffer[0..0]), status, os_code, completed)
    } else {
        if (exact) {
            matches(fs::read_exact(file, ref mut buffer), status, os_code, completed)
        } else {
            matches(fs::read(file, ref mut buffer), status, os_code, completed)
        }
    };
    unsafe { end(); }
    assert(correct && bytes::equal(ref buffer, expected));
}

@effects(filesystem)
fn void exercise(fs::File input) {
    var fs::File file = input;
    check_write(ref mut file, 1, true, 0, 0, 6);
    check_write(ref mut file, 2, true, 1, 28, 2);
    check_write(ref mut file, 3, true, 4, 0, 2);
    check_write(ref mut file, 4, true, 1, 5, 0);
    check_write(ref mut file, 5, true, 4, 0, 0);
    check_write(ref mut file, 6, true, 0, 0, 0);
    check_write(ref mut file, 7, false, 0, 0, 2);
    check_write(ref mut file, 1, true, 0, 0, 6);
    check_read(ref mut file, 8, true, 0, 0, 6, b"A\0\xffbcd");
    check_read(ref mut file, 9, true, 1, 5, 2, b"A\0\xa5\xa5\xa5\xa5");
    check_read(ref mut file, 10, true, 7, 0, 2, b"A\0\xa5\xa5\xa5\xa5");
    check_read(ref mut file, 11, true, 1, 5, 0, b"\xa5\xa5\xa5\xa5\xa5\xa5");
    check_read(ref mut file, 12, true, 7, 0, 0, b"\xa5\xa5\xa5\xa5\xa5\xa5");
    check_read(ref mut file, 13, true, 0, 0, 0, b"\xa5\xa5\xa5\xa5\xa5\xa5");
    check_read(ref mut file, 14, false, 0, 0, 2, b"A\0\xa5\xa5\xa5\xa5");
    check_read(ref mut file, 8, true, 0, 0, 6, b"A\0\xffbcd");
    unsafe { begin(15); }
    assert(control_matches(fs::sync(ref mut file), 0, 0));
    unsafe { end(); begin(16); }
    assert(control_matches(fs::sync(ref mut file), 1, 5));
    unsafe { end(); begin(17); }
    assert(control_matches(fs::close(file), 1, 4));
    unsafe { end(); }
}

@effects(filesystem)
fn i32 main() {
    return switch (fs::open_or_create(c"file-progress.tmp")) {
        case .ok(file): { exercise(file); 0 }
        case .err(_): { 1 }
    };
}
`
