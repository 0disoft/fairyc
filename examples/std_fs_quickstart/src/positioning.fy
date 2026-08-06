module positioning;

import std::fs as fs;

@effects(filesystem)
pub fn i32 write_position_probe(ref mut fs::File file) {
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
    let i32 relative_code = switch (fs::seek_current(file, -4)) {
        case .ok(position) : { if (position == 5) { 0 } else { 319 } }
        case .err(_) : { 320 }
    };
    if (relative_code != 0) {
        return relative_code;
    }
    let i32 from_end_code = switch (fs::seek_end(file, -9)) {
        case .ok(position) : { if (position == 0) { 0 } else { 321 } }
        case .err(_) : { 322 }
    };
    if (from_end_code != 0) {
        return from_end_code;
    }
    let i32 negative_code = switch (fs::seek_current(file, -1)) {
        case .ok(_) : { 323 }
        case .err(error) : { if (fs::error_status(error) == 1 &&
            fs::error_os_code(error) != 0 && fs::error_completed(error) == 0) {
            0
        } else {
            324
        } }
    };
    if (negative_code != 0) {
        return negative_code;
    }
    let i32 negative_preserved_code = switch (fs::position(file)) {
        case .ok(position) : { if (position == 0) { 0 } else { 325 } }
        case .err(_) : { 326 }
    };
    if (negative_preserved_code != 0) {
        return negative_preserved_code;
    }
    let i32 beyond_end_code = switch (fs::seek_end(file, 3)) {
        case .ok(position) : { if (position == 12) { 0 } else { 327 } }
        case .err(_) : { 328 }
    };
    if (beyond_end_code != 0) {
        return beyond_end_code;
    }
    let i32 reset_from_end_code = switch (fs::seek_end(file, -9)) {
        case .ok(position) : { if (position == 0) { 0 } else { 331 } }
        case .err(_) : { 332 }
    };
    if (reset_from_end_code != 0) {
        return reset_from_end_code;
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
