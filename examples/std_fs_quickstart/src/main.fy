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
fn i32 expect_disk(ref fs::File file) {
    return switch (fs::kind(file)) {
        case .ok(kind) : { switch (kind) {
            case .Unknown : { 138 }
            case .Disk : { if (fs::kind_is_disk(kind) &&
                !fs::kind_is_character(kind) && !fs::kind_is_pipe(kind) &&
                !fs::kind_is_unknown(kind)) { 0 } else { 139 } }
            case .Character : { 140 }
            case .Pipe : { 143 }
        } }
        case .err(_) : { 141 }
    };
}

@effects(filesystem)
fn i32 verify_character_then_close(fs::File input) {
    var fs::File file = input;
    let i32 operation_code = switch (fs::kind(ref file)) {
        case .ok(kind) : { switch (kind) {
            case .Unknown : { 142 }
            case .Disk : { 143 }
            case .Character : { if (!fs::kind_is_disk(kind) &&
                fs::kind_is_character(kind) && !fs::kind_is_pipe(kind) &&
                !fs::kind_is_unknown(kind)) { 0 } else { 144 } }
            case .Pipe : { 146 }
        } }
        case .err(_) : { 145 }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 146 }
    };
    return if (operation_code != 0) { operation_code } else { close_code };
}

@effects(filesystem)
fn i32 write_then_close(fs::File input) {
    var fs::File file = input;
    let ref u8[] bytes = b"FairyFile";
    let i32 kind_code = expect_disk(ref file);
    let i32 operation_code = if (kind_code != 0) {
        kind_code
    } else { switch (fs::write_all(ref mut file, bytes)) {
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
    } };
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
fn i32 read_exact_then_close(fs::File input) {
    var fs::File file = input;
    var u8[9] exact = [0, 0, 0, 0, 0, 0, 0, 0, 0];
    let i32 exact_code = switch (fs::read_exact(ref mut file, ref mut exact)) {
        case .ok(read) : { if (read == 9 &&
            exact[0] == 70 && exact[1] == 97 && exact[2] == 105 &&
            exact[3] == 114 && exact[4] == 121 && exact[5] == 67 &&
            exact[6] == 111 && exact[7] == 100 && exact[8] == 101) {
            0
        } else {
            148
        } }
        case .err(_) : { 149 }
    };
    let i32 seek_code = if (exact_code != 0) {
        exact_code
    } else { switch (fs::seek_start(ref mut file, 0)) {
        case .ok(position) : { if (position == 0) { 0 } else { 150 } }
        case .err(_) : { 151 }
    } };
    var u8[12] oversized = [
        170, 170, 170, 170, 170, 170,
        170, 170, 170, 170, 170, 170,
    ];
    let i32 incomplete_code = if (seek_code != 0) {
        seek_code
    } else { switch (fs::read_exact(ref mut file, ref mut oversized)) {
        case .ok(_) : { 152 }
        case .err(error) : { if (fs::error_status(error) == 7 &&
            fs::error_os_code(error) == 0 && fs::error_completed(error) == 9 &&
            oversized[0] == 70 && oversized[1] == 97 && oversized[2] == 105 &&
            oversized[3] == 114 && oversized[4] == 121 && oversized[5] == 67 &&
            oversized[6] == 111 && oversized[7] == 100 && oversized[8] == 101 &&
            oversized[9] == 170 && oversized[10] == 170 && oversized[11] == 170) {
            0
        } else {
            153
        } }
    } };
    let i32 position_code = if (incomplete_code != 0) {
        incomplete_code
    } else { switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 9) { 0 } else { 154 } }
        case .err(_) : { 155 }
    } };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 156 }
    };
    return if (position_code != 0) { position_code } else { close_code };
}

@effects(filesystem)
fn i32 compare_file_id_then_close(
    fs::File input,
    ref fs::FileId expected,
    bool should_equal,
    i32 mismatch_code,
    i32 observe_error_code,
    i32 close_error_code
) {
    var fs::File file = input;
    let i32 operation_code = switch (fs::file_id(ref file)) {
        case .ok(observed) : {
            if (fs::file_id_equal(ref observed, expected) == should_equal) {
                0
            } else {
                mismatch_code
            }
        }
        case .err(_) : { observe_error_code }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { close_error_code }
    };
    return if (operation_code != 0) { operation_code } else { close_code };
}

@effects(filesystem)
fn i32 verify_file_id_reopen(
    fs::File input,
    ref CStr same_path,
    ref CStr other_path
) {
    var fs::File file = input;
    let i32 operation_code = switch (fs::file_id(ref file)) {
        case .err(_) : { 158 }
        case .ok(original) : {
            let i32 same_code = switch (fs::open_read(same_path)) {
                case .ok(reopened) : {
                    compare_file_id_then_close(reopened, ref original, true, 160, 161, 162)
                }
                case .err(_) : { 163 }
            };
            if (same_code != 0) {
                same_code
            } else { switch (fs::create_truncate(other_path)) {
                case .ok(other) : {
                    compare_file_id_then_close(other, ref original, false, 164, 165, 166)
                }
                case .err(_) : { 167 }
            } }
        }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 159 }
    };
    return if (operation_code != 0) { operation_code } else { close_code };
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
fn i32 initialize_create_new(fs::File input) {
    var fs::File file = input;
    let ref u8[] seed = b"New";
    let i32 size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 0) { 0 } else { 110 } }
        case .err(_) : { 111 }
    };
    let i32 operation_code = if (size_code != 0) {
        size_code
    } else {
        let i32 position_code = switch (fs::position(ref mut file)) {
            case .ok(position) : { if (position == 0) { 0 } else { 112 } }
            case .err(_) : { 113 }
        };
        if (position_code != 0) {
            position_code
        } else { switch (fs::write_all(ref mut file, seed)) {
            case .ok(written) : { if (written == 3) { 0 } else { 114 } }
            case .err(_) : { 115 }
        } }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 116 }
    };
    return if (operation_code != 0) { operation_code } else { close_code };
}

@effects(filesystem)
fn i32 verify_create_new_preserves(fs::File input) {
    var fs::File file = input;
    var u8[3] buffer = [0, 0, 0];
    let i32 operation_code = switch (fs::read(ref mut file, ref mut buffer)) {
        case .ok(read) : {
            if (read == 3 && buffer[0] == 78 && buffer[1] == 101 && buffer[2] == 119) {
                0
            } else {
                117
            }
        }
        case .err(_) : { 118 }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 119 }
    };
    return if (operation_code != 0) { operation_code } else { close_code };
}

@effects(filesystem)
fn i32 check_create_new(ref CStr path) {
    let i32 create_code = switch (fs::create_new(path)) {
        case .ok(file) : { initialize_create_new(file) }
        case .err(_) : { 120 }
    };
    let i32 duplicate_code = if (create_code != 0) {
        create_code
    } else { switch (fs::create_new(path)) {
        case .ok(_) : { 121 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 80 &&
                fs::error_completed(error) == 0) { 0 } else { 122 }
        }
    } };
    return if (duplicate_code != 0) {
        duplicate_code
    } else { switch (fs::open_read(path)) {
        case .ok(file) : { verify_create_new_preserves(file) }
        case .err(_) : { 123 }
    } };
}

@effects(filesystem)
fn i32 append_with_reposition(fs::File input, usize expected_size, ref u8[] bytes) {
    var fs::File file = input;
    let i32 size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == expected_size) { 0 } else { 124 } }
        case .err(_) : { 125 }
    };
    let i32 operation_code = if (size_code != 0) {
        size_code
    } else {
        let i32 seek_code = switch (fs::seek_start(ref mut file, 0)) {
            case .ok(position) : { if (position == 0) { 0 } else { 126 } }
            case .err(_) : { 127 }
        };
        if (seek_code != 0) {
            seek_code
        } else { switch (fs::write_all(ref mut file, bytes)) {
            case .ok(written) : { if (written == bytes.len) { 0 } else { 128 } }
            case .err(_) : { 129 }
        } }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 130 }
    };
    return if (operation_code != 0) { operation_code } else { close_code };
}

@effects(filesystem)
fn i32 verify_append_file(fs::File input) {
    var fs::File file = input;
    var u8[7] buffer = [0, 0, 0, 0, 0, 0, 0];
    let i32 operation_code = switch (fs::read(ref mut file, ref mut buffer)) {
        case .ok(read) : {
            if (read == 7 &&
                buffer[0] == 79 && buffer[1] == 110 && buffer[2] == 101 &&
                buffer[3] == 84 && buffer[4] == 119 && buffer[5] == 111 &&
                buffer[6] == 33) { 0 } else { 131 }
        }
        case .err(_) : { 132 }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 133 }
    };
    return if (operation_code != 0) { operation_code } else { close_code };
}

@effects(filesystem)
fn i32 check_append(ref CStr path) {
    let ref u8[] first = b"One";
    let ref u8[] second = b"Two";
    let ref u8[] third = b"!";
    let i32 first_code = switch (fs::open_append(path)) {
        case .ok(file) : { append_with_reposition(file, 0, first) }
        case .err(_) : { 134 }
    };
    let i32 second_code = if (first_code != 0) {
        first_code
    } else { switch (fs::open_append(path)) {
        case .ok(file) : { append_with_reposition(file, 3, second) }
        case .err(_) : { 135 }
    } };
    let i32 third_code = if (second_code != 0) {
        second_code
    } else { switch (fs::open_append(path)) {
        case .ok(file) : { append_with_reposition(file, 6, third) }
        case .err(_) : { 136 }
    } };
    return if (third_code != 0) {
        third_code
    } else { switch (fs::open_read(path)) {
        case .ok(file) : { verify_append_file(file) }
        case .err(_) : { 137 }
    } };
}

@effects(filesystem)
fn i32 main() {
    let ref CStr data_path = c"fairyc-std-fs-data.tmp";
    let ref CStr drop_path = c"fairyc-std-fs-drop.tmp";
    let ref CStr open_or_create_path = c"fairyc-std-fs-open-or-create.tmp";
    let ref CStr create_new_path = c"fairyc-std-fs-create-new.tmp";
    let ref CStr append_path = c"fairyc-std-fs-append.tmp";
    let i32 invalid_code = check_invalid_path();
    let i32 missing_update_code = check_missing_update();
    let i32 character_code = switch (fs::open_read(c"NUL")) {
        case .ok(file) : { verify_character_then_close(file) }
        case .err(_) : { 147 }
    };
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
    let i32 create_new_code = check_create_new(create_new_path);
    let i32 append_code = check_append(append_path);
    let i32 write_code = switch (fs::create_truncate(data_path)) {
        case .ok(file) : { write_then_close(file) }
        case .err(_) : { 30 }
    };
    return if (invalid_code != 0) {
        invalid_code
    } else if (missing_update_code != 0) {
        missing_update_code
    } else if (character_code != 0) {
        character_code
    } else if (preserve_code != 0) {
        preserve_code
    } else if (create_new_code != 0) {
        create_new_code
    } else if (append_code != 0) {
        append_code
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
            let i32 exact_code = switch (fs::open_read(data_path)) {
                case .ok(file) : { read_exact_then_close(file) }
                case .err(_) : { 157 }
            };
            if (exact_code != 0) {
                exact_code
            } else {
                let i32 identity_code = switch (fs::open_read(data_path)) {
                    case .ok(file) : { verify_file_id_reopen(file, data_path, drop_path) }
                    case .err(_) : { 168 }
                };
                if (identity_code != 0) {
                    identity_code
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
            }
        }
    };
}
