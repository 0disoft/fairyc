module main;

import std::fs as fs;

@effects(filesystem)
fn i32 write_position_probe(ref mut fs::File file) {
    let i32 current_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 9) { 0 } else { 14 } }
        case .err(_) : { 15 }
    };
    if (current_code != 0) {
        return current_code;
    }
    let usize invalid_offset = 18446744073709551615;
    let i32 invalid_code = switch (fs::seek_start(file, invalid_offset)) {
        case .ok(_) : { 16 }
        case .err(error) : { if (fs::error_status(error) == 2 &&
            fs::error_os_code(error) == 0 && fs::error_completed(error) == 0) {
            0
        } else {
            17
        } }
    };
    if (invalid_code != 0) {
        return invalid_code;
    }
    let i32 preserved_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 9) { 0 } else { 18 } }
        case .err(_) : { 19 }
    };
    if (preserved_code != 0) {
        return preserved_code;
    }
    let i32 seek_code = switch (fs::seek_start(file, 5)) {
        case .ok(position) : { if (position == 5) { 0 } else { 25 } }
        case .err(_) : { 26 }
    };
    if (seek_code != 0) {
        return seek_code;
    }
    let i32 moved_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 5) { 0 } else { 27 } }
        case .err(_) : { 28 }
    };
    if (moved_code != 0) {
        return moved_code;
    }
    return switch (fs::seek_start(file, 0)) {
        case .ok(position) : { if (position == 0) { 0 } else { 29 } }
        case .err(_) : { 35 }
    };
}

@effects(filesystem)
fn i32 resize_round_trip(ref mut fs::File file, ref u8[] replacement) {
    let i32 shrink_code = switch (fs::set_len(file, 5)) {
        case .ok(size) : { if (size == 5) { 0 } else { 80 } }
        case .err(_) : { 81 }
    };
    if (shrink_code != 0) {
        return shrink_code;
    }
    let i32 shrink_position_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 9) { 0 } else { 82 } }
        case .err(_) : { 83 }
    };
    if (shrink_position_code != 0) {
        return shrink_position_code;
    }
    let i32 extend_code = switch (fs::set_len(file, 9)) {
        case .ok(size) : { if (size == 9) { 0 } else { 84 } }
        case .err(_) : { 85 }
    };
    if (extend_code != 0) {
        return extend_code;
    }
    let i32 extend_position_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 9) { 0 } else { 86 } }
        case .err(_) : { 87 }
    };
    if (extend_position_code != 0) {
        return extend_position_code;
    }
    let i32 seek_code = switch (fs::seek_start(file, 5)) {
        case .ok(position) : { if (position == 5) { 0 } else { 88 } }
        case .err(_) : { 89 }
    };
    if (seek_code != 0) {
        return seek_code;
    }
    return switch (fs::write_all(file, replacement)) {
        case .ok(written) : { if (written == 4) { 0 } else { 90 } }
        case .err(_) : { 91 }
    };
}

@effects(filesystem)
fn i32 read_position_probe(ref mut fs::File file, ref mut u8[] buffer) {
    let i32 initial_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 0) { 0 } else { 52 } }
        case .err(_) : { 43 }
    };
    if (initial_code != 0) {
        return initial_code;
    }
    let i32 seek_code = switch (fs::seek_start(file, 5)) {
        case .ok(position) : { if (position == 5) { 0 } else { 44 } }
        case .err(_) : { 45 }
    };
    if (seek_code != 0) {
        return seek_code;
    }
    let i32 tail_code = switch (fs::read(file, buffer)) {
        case .ok(read) : { if (read == 4 &&
            buffer[0] == 67 && buffer[1] == 111 &&
            buffer[2] == 100 && buffer[3] == 101) { 0 } else { 46 } }
        case .err(_) : { 47 }
    };
    if (tail_code != 0) {
        return tail_code;
    }
    let i32 end_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 9) { 0 } else { 48 } }
        case .err(_) : { 49 }
    };
    if (end_code != 0) {
        return end_code;
    }
    return switch (fs::seek_start(file, 0)) {
        case .ok(position) : { if (position == 0) { 0 } else { 50 } }
        case .err(_) : { 51 }
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
        case .ok(position) : { if (position == 5) { 0 } else { 63 } }
        case .err(_) : { 64 }
    };
    let i32 operation_code = if (seek_code != 0) {
        seek_code
    } else {
        let i32 write_code = switch (fs::write_all(ref mut file, patch)) {
            case .ok(written) : { if (written == 4) { 0 } else { 65 } }
            case .err(_) : { 66 }
        };
        if (write_code != 0) {
            write_code
        } else {
            let i32 size_code = switch (fs::size(ref file)) {
                case .ok(size) : { if (size == 9) { 0 } else { 67 } }
                case .err(_) : { 68 }
            };
            if (size_code != 0) {
                size_code
            } else {
                let usize invalid_length = 18446744073709551615;
                let i32 invalid_length_code = switch (fs::set_len(ref mut file, invalid_length)) {
                    case .ok(_) : { 76 }
                    case .err(error) : { if (fs::error_status(error) == 2 &&
                        fs::error_os_code(error) == 0 && fs::error_completed(error) == 0) {
                        0
                    } else {
                        77
                    } }
                };
                if (invalid_length_code != 0) {
                    invalid_length_code
                } else {
                    let i32 preserved_size_code = switch (fs::size(ref file)) {
                        case .ok(size) : { if (size == 9) { 0 } else { 78 } }
                        case .err(_) : { 79 }
                    };
                    if (preserved_size_code != 0) {
                        preserved_size_code
                    } else {
                        let i32 resize_code = resize_round_trip(ref mut file, patch);
                        if (resize_code != 0) {
                            resize_code
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
                                        69
                                    } }
                                    case .err(_) : { 70 }
                                }
                            }
                        }
                    }
                }
            }
        }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 72 }
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
                case .ok(size) : { if (size == 9) { 0 } else { 53 } }
                case .err(_) : { 54 }
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
    let i32 read_only_length_code = switch (fs::set_len(ref mut file, 9)) {
        case .ok(_) : { 92 }
        case .err(error) : { if (fs::error_status(error) == 1 &&
            fs::error_os_code(error) == 5 && fs::error_completed(error) == 0) {
            0
        } else {
            93
        } }
    };
    let i32 size_code = if (read_only_length_code != 0) {
        read_only_length_code
    } else { switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 9) { 0 } else { 57 } }
        case .err(_) : { 58 }
    } };
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
        case .ok(size) : { if (size == 0) { 0 } else { 61 } }
        case .err(_) : { 62 }
    };
}

@effects(filesystem)
fn i32 check_invalid_path() {
    return switch (fs::open_read(c"")) {
        case .ok(_) : { 40 }
        case .err(error) : {
            if (fs::error_status(error) == 2 &&
                fs::error_os_code(error) == 0 &&
                fs::error_completed(error) == 0) {
                0
            } else {
                41
            }
        }
    };
}

@effects(filesystem)
fn i32 check_missing_update() {
    return switch (fs::open_update(c"fairyc-std-fs-missing.tmp")) {
        case .ok(_) : { 73 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 2 &&
                fs::error_completed(error) == 0) {
                0
            } else {
                74
            }
        }
    };
}

@effects(filesystem)
fn i32 initialize_open_or_create(fs::File input) {
    var fs::File file = input;
    let ref u8[] seed = b"Seed";
    let i32 size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 0) { 0 } else { 94 } }
        case .err(_) : { 95 }
    };
    let i32 operation_code = if (size_code != 0) {
        size_code
    } else {
        let i32 position_code = switch (fs::position(ref mut file)) {
            case .ok(position) : { if (position == 0) { 0 } else { 96 } }
            case .err(_) : { 97 }
        };
        if (position_code != 0) {
            position_code
        } else {
            switch (fs::write_all(ref mut file, seed)) {
                case .ok(written) : { if (written == 4) { 0 } else { 98 } }
                case .err(_) : { 99 }
            }
        }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 100 }
    };
    return if (operation_code != 0) {
        operation_code
    } else {
        close_code
    };
}

@effects(filesystem)
fn i32 verify_open_or_create_preserves(fs::File input) {
    var fs::File file = input;
    var u8[4] buffer = [0, 0, 0, 0];
    let i32 size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 4) { 0 } else { 101 } }
        case .err(_) : { 102 }
    };
    let i32 operation_code = if (size_code != 0) {
        size_code
    } else {
        let i32 position_code = switch (fs::position(ref mut file)) {
            case .ok(position) : { if (position == 0) { 0 } else { 103 } }
            case .err(_) : { 104 }
        };
        if (position_code != 0) {
            position_code
        } else {
            switch (fs::read(ref mut file, ref mut buffer)) {
                case .ok(read) : { if (read == 4 &&
                    buffer[0] == 83 && buffer[1] == 101 &&
                    buffer[2] == 101 && buffer[3] == 100) { 0 } else { 105 } }
                case .err(_) : { 106 }
            }
        }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 107 }
    };
    return if (operation_code != 0) {
        operation_code
    } else {
        close_code
    };
}

@effects(filesystem)
fn i32 main() {
    let ref CStr data_path = c"fairyc-std-fs-data.tmp";
    let ref CStr drop_path = c"fairyc-std-fs-drop.tmp";
    let ref CStr open_or_create_path = c"fairyc-std-fs-open-or-create.tmp";
    let i32 invalid_code = check_invalid_path();
    let i32 missing_update_code = check_missing_update();
    let i32 create_code = switch (fs::open_or_create(open_or_create_path)) {
        case .ok(file) : { initialize_open_or_create(file) }
        case .err(_) : { 108 }
    };
    let i32 preserve_code = if (create_code != 0) {
        create_code
    } else { switch (fs::open_or_create(open_or_create_path)) {
        case .ok(file) : { verify_open_or_create_preserves(file) }
        case .err(_) : { 109 }
    } };
    let i32 write_code = switch (fs::create_truncate(data_path)) {
        case .ok(file) : { write_then_close(file) }
        case .err(_) : { 30 }
    };
    return if (invalid_code != 0) {
        invalid_code
    } else if (missing_update_code != 0) {
        missing_update_code
    } else if (preserve_code != 0) {
        preserve_code
    } else if (write_code != 0) {
        write_code
    } else {
        let i32 update_code = switch (fs::open_update(data_path)) {
            case .ok(file) : { update_then_close(file) }
            case .err(_) : { 75 }
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
