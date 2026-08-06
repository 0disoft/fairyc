module metadata;

import std::fs as fs;

@effects(filesystem)
fn i32 inspect_then_close(fs::File input, bool expected, i32 base) {
    var fs::File file = input;
    var usize before_position = 0;
    let i32 before_position_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { before_position = position; 0 }
        case .err(_) : { base }
    };
    var usize before_size = 0;
    let i32 before_size_code = if (before_position_code != 0) {
        before_position_code
    } else { switch (fs::size(ref file)) {
        case .ok(size) : { before_size = size; 0 }
        case .err(_) : { base + 1 }
    } };
    let i32 attribute_code = if (before_size_code != 0) {
        before_size_code
    } else { switch (fs::is_read_only(ref file)) {
        case .ok(read_only) : { if (read_only == expected) { 0 } else { base + 2 } }
        case .err(_) : { base + 3 }
    } };
    let i32 position_code = if (attribute_code != 0) {
        attribute_code
    } else { switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == before_position) { 0 } else { base + 4 } }
        case .err(_) : { base + 5 }
    } };
    let i32 size_code = if (position_code != 0) {
        position_code
    } else { switch (fs::size(ref file)) {
        case .ok(size) : { if (size == before_size) { 0 } else { base + 6 } }
        case .err(_) : { base + 7 }
    } };
    return switch (fs::close(file)) {
        case .ok(_) : { size_code }
        case .err(_) : { base + 8 }
    };
}

@effects(filesystem)
pub fn i32 check_read_only(ref CStr normal_path, ref CStr read_only_path) {
    let i32 normal_code = switch (fs::open_read(normal_path)) {
        case .ok(file) : { inspect_then_close(file, false, 333) }
        case .err(_) : { 342 }
    };
    if (normal_code != 0) {
        return normal_code;
    }
    return switch (fs::open_read(read_only_path)) {
        case .ok(file) : { inspect_then_close(file, true, 343) }
        case .err(_) : { 352 }
    };
}
