module examples::windows_file::main;

import examples::windows_file::windows_file as fs;

@effects(filesystem)
fn i32 write_position_probe(ref mut fs::File file) {
    let i32 current_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 9) { 0 } else { 14 } }
        case .err(_) : { 15 }
    };
    if (current_code != 0) {
        return current_code;
    }
    let i32 seek_code = switch (fs::seek_start(file, 5)) {
        case .ok(position) : { if (position == 5) { 0 } else { 16 } }
        case .err(_) : { 17 }
    };
    if (seek_code != 0) {
        return seek_code;
    }
    let i32 moved_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 5) { 0 } else { 18 } }
        case .err(_) : { 19 }
    };
    if (moved_code != 0) {
        return moved_code;
    }
    return switch (fs::seek_start(file, 0)) {
        case .ok(position) : { if (position == 0) { 0 } else { 25 } }
        case .err(_) : { 26 }
    };
}

@effects(filesystem)
fn i32 read_position_probe(ref mut fs::File file, ref mut u8[] buffer) {
    let i32 initial_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 0) { 0 } else { 35 } }
        case .err(_) : { 36 }
    };
    if (initial_code != 0) {
        return initial_code;
    }
    let i32 seek_code = switch (fs::seek_start(file, 5)) {
        case .ok(position) : { if (position == 5) { 0 } else { 37 } }
        case .err(_) : { 38 }
    };
    if (seek_code != 0) {
        return seek_code;
    }
    let i32 tail_code = switch (fs::read(file, buffer)) {
        case .ok(read) : { if (read == 4 &&
            buffer[0] == 67 && buffer[1] == 111 &&
            buffer[2] == 100 && buffer[3] == 101) { 0 } else { 39 } }
        case .err(_) : { 40 }
    };
    if (tail_code != 0) {
        return tail_code;
    }
    let i32 end_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 9) { 0 } else { 41 } }
        case .err(_) : { 43 }
    };
    if (end_code != 0) {
        return end_code;
    }
    return switch (fs::seek_start(file, 0)) {
        case .ok(position) : { if (position == 0) { 0 } else { 44 } }
        case .err(_) : { 45 }
    };
}

@effects(filesystem)
fn i32 update_then_close(fs::File input) {
    var fs::File file = input;
    let ref u8[] patch = b"Code";
    var u8[16] buffer = [
        0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0,
    ];
    let i32 seek_code = switch (fs::seek_start(ref mut file, 5)) {
        case .ok(position) : { if (position == 5) { 0 } else { 56 } }
        case .err(_) : { 57 }
    };
    let i32 operation_code = if (seek_code != 0) {
        seek_code
    } else {
        let i32 write_code = switch (fs::write_all(ref mut file, patch)) {
            case .ok(written) : { if (written == 4) { 0 } else { 58 } }
            case .err(_) : { 59 }
        };
        if (write_code != 0) {
            write_code
        } else {
            let i32 size_code = switch (fs::size(ref file)) {
                case .ok(size) : { if (size == 9) { 0 } else { 60 } }
                case .err(_) : { 61 }
            };
            if (size_code != 0) {
                size_code
            } else {
                let i32 position_code = write_position_probe(ref mut file);
                if (position_code != 0) {
                    position_code
                } else {
                    switch (fs::read(ref mut file, ref mut buffer)) {
                        case .ok(read) : { if (read == 9 &&
                            buffer[0] == 70 && buffer[1] == 97 && buffer[2] == 105 &&
                            buffer[3] == 114 && buffer[4] == 121 && buffer[5] == 67 &&
                            buffer[6] == 111 && buffer[7] == 100 && buffer[8] == 101) {
                            0
                        } else {
                            62
                        } }
                        case .err(_) : { 63 }
                    }
                }
            }
        }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 64 }
    };
    return if (operation_code != 0) {
        operation_code
    } else {
        close_code
    };
}

@effects(filesystem)
fn i32 write_then_close(fs::File input) {
    var fs::File file = input;
    let ref u8[] bytes = b"FairyFile";
    let i32 operation_code = switch (fs::write_all(ref mut file, bytes)) {
        case .ok(written) : { if (written == 9) {
            let i32 size_code = switch (fs::size(ref file)) {
                case .ok(size) : { if (size == 9) { 0 } else { 46 } }
                case .err(_) : { 47 }
            };
            if (size_code != 0) {
                size_code
            } else {
                let i32 position_code = write_position_probe(ref mut file);
                if (position_code != 0) {
                    position_code
                } else {
                    switch (fs::sync(ref mut file)) {
                        case .ok(_) : { 0 }
                        case .err(_) : { 12 }
                    }
                }
            }
        } else {
            11
        } }
        case .err(_) : { 10 }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 13 }
    };
    return if (operation_code != 0) {
        operation_code
    } else {
        close_code
    };
}

@effects(filesystem)
fn i32 read_then_close(fs::File input) {
    var fs::File file = input;
    var u8[16] buffer = [
        0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0,
    ];
    let i32 size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 9) { 0 } else { 50 } }
        case .err(_) : { 51 }
    };
    let i32 operation_code = if (size_code != 0) {
        size_code
    } else {
        let i32 position_code = read_position_probe(ref mut file, ref mut buffer);
        if (position_code != 0) {
            position_code
        } else {
            switch (fs::read(ref mut file, ref mut buffer)) {
                case .ok(read) : { if (read == 9 &&
                buffer[0] == 70 && buffer[1] == 97 && buffer[2] == 105 &&
                buffer[3] == 114 && buffer[4] == 121 && buffer[5] == 67 &&
                buffer[6] == 111 && buffer[7] == 100 && buffer[8] == 101) {
                    switch (fs::read(ref mut file, ref mut buffer)) {
                        case .ok(eof) : { if (eof == 0) { 0 } else { 23 } }
                        case .err(_) : { 22 }
                    }
                } else {
                    21
                } }
                case .err(_) : { 20 }
            }
        }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 24 }
    };
    return if (operation_code != 0) {
        operation_code
    } else {
        close_code
    };
}

@effects(filesystem)
fn i32 implicit_drop_probe(fs::File file) {
    return switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 0) { 0 } else { 54 } }
        case .err(_) : { 55 }
    };
}

@effects(filesystem)
fn i32 check_missing_update() {
    return switch (fs::open_update(c"fairyc-fy-file-missing.tmp")) {
        case .ok(_) : { 65 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 2 &&
                fs::error_completed(error) == 0) {
                0
            } else {
                66
            }
        }
    };
}

@effects(filesystem)
fn i32 main() {
    let ref CStr data_path = c"fairyc-fy-file-data.tmp";
    let ref CStr drop_path = c"fairyc-fy-file-drop.tmp";
    let i32 missing_update_code = check_missing_update();
    let i32 write_code = switch (fs::create_truncate(data_path)) {
        case .ok(file) : { write_then_close(file) }
        case .err(_) : { 30 }
    };
    return if (missing_update_code != 0) {
        missing_update_code
    } else if (write_code != 0) {
        write_code
    } else {
        let i32 update_code = switch (fs::open_update(data_path)) {
            case .ok(file) : { update_then_close(file) }
            case .err(_) : { 67 }
        };
        if (update_code != 0) {
            update_code
        } else {
            let i32 read_code = switch (fs::open_read(data_path)) {
                case .ok(file) : { read_then_close(file) }
                case .err(_) : { 31 }
            };
            if (read_code != 0) {
                read_code
            } else {
                let i32 drop_code = switch (fs::create_truncate(drop_path)) {
                    case .ok(file) : { implicit_drop_probe(file) }
                    case .err(_) : { 32 }
                };
                if (drop_code != 0) {
                    drop_code
                } else {
                    switch (fs::create_truncate(drop_path)) {
                        case .ok(file) : { switch (fs::close(file)) {
                            case .ok(_) : { 42 }
                            case .err(_) : { 34 }
                        } }
                        case .err(_) : { 33 }
                    }
                }
            }
        }
    };
}
