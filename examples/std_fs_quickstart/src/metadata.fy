module metadata;

import std::fs as fs;

@effects(filesystem)
fn i32 inspect_then_close(fs::File input, bool expected_read_only, usize expected_links, i32 base) {
    var fs::File file = input;
    var usize before_position = 0;
    let i32 before_position_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { before_position = position; 0 }
        case .err(_) : { base }
    };
    var usize before_size = 0;
    let i32 metadata_code = if (before_position_code != 0) {
        before_position_code
    } else { switch (fs::metadata(ref file)) {
        case .ok(metadata) : {
            before_size = metadata.size;
            if (fs::kind_is_disk(metadata.kind) && metadata.read_only == expected_read_only &&
                metadata.hard_link_count == expected_links &&
                metadata.allocated_size >= metadata.size) { 0 } else { base + 1 }
        }
        case .err(_) : { base + 2 }
    } };
    let i32 position_code = if (metadata_code != 0) {
        metadata_code
    } else { switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == before_position) { 0 } else { base + 3 } }
        case .err(_) : { base + 4 }
    } };
    let i32 size_code = if (position_code != 0) {
        position_code
    } else { switch (fs::size(ref file)) {
        case .ok(size) : { if (size == before_size) { 0 } else { base + 5 } }
        case .err(_) : { base + 6 }
    } };
    return switch (fs::close(file)) {
        case .ok(_) : { size_code }
        case .err(_) : { base + 7 }
    };
}

@effects(filesystem)
fn i32 mutate_read_only_then_close(fs::File input) {
    var fs::File file = input;
    let i32 enable_code = switch (fs::set_read_only(ref mut file, true)) {
        case .ok(enabled) : { if (enabled) { 0 } else { 415 } }
        case .err(_) : { 416 }
    };
    let i32 enabled_code = if (enable_code != 0) {
        enable_code
    } else { switch (fs::is_read_only(ref file)) {
        case .ok(enabled) : { if (enabled) { 0 } else { 417 } }
        case .err(_) : { 418 }
    } };
    let i32 disable_code = if (enabled_code != 0) {
        enabled_code
    } else { switch (fs::set_read_only(ref mut file, false)) {
        case .ok(enabled) : { if (!enabled) { 0 } else { 419 } }
        case .err(_) : { 420 }
    } };
    let i32 disabled_code = if (disable_code != 0) {
        disable_code
    } else { switch (fs::is_read_only(ref file)) {
        case .ok(enabled) : { if (!enabled) { 0 } else { 421 } }
        case .err(_) : { 422 }
    } };
    let i32 position_code = if (disabled_code != 0) {
        disabled_code
    } else { switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 0) { 0 } else { 423 } }
        case .err(_) : { 424 }
    } };
    let i32 size_code = if (position_code != 0) {
        position_code
    } else { switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 0) { 0 } else { 425 } }
        case .err(_) : { 426 }
    } };
    return switch (fs::close(file)) {
        case .ok(_) : { size_code }
        case .err(_) : { 427 }
    };
}

@effects(filesystem)
fn i32 reject_read_only_mutation_then_close(fs::File input) {
    var fs::File file = input;
    let i32 failure_code = switch (fs::set_read_only(ref mut file, true)) {
        case .ok(_) : { 428 }
        case .err(error) : {
            if (fs::error_status(error) == 1 && fs::error_os_code(error) == 5 &&
                fs::error_completed(error) == 0) { 0 } else { 429 }
        }
    };
    let i32 unchanged_code = if (failure_code != 0) {
        failure_code
    } else { switch (fs::is_read_only(ref file)) {
        case .ok(enabled) : { if (!enabled) { 0 } else { 430 } }
        case .err(_) : { 431 }
    } };
    return switch (fs::close(file)) {
        case .ok(_) : { unchanged_code }
        case .err(_) : { 432 }
    };
}

@effects(filesystem)
fn i32 check_read_only_mutation() {
    let ref CStr path = c"fairyc-std-fs-read-only-mutation.tmp";
    let i32 mutation_code = switch (fs::create_truncate(path)) {
        case .ok(file) : { mutate_read_only_then_close(file) }
        case .err(_) : { 433 }
    };
    let i32 denied_code = if (mutation_code != 0) {
        mutation_code
    } else { switch (fs::open_read(path)) {
        case .ok(file) : { reject_read_only_mutation_then_close(file) }
        case .err(_) : { 434 }
    } };
    return if (denied_code != 0) {
        denied_code
    } else { switch (fs::remove_file(path)) {
        case .ok(_) : { 0 }
        case .err(_) : { 435 }
    } };
}

@effects(filesystem)
pub fn i32 check_metadata(
    ref CStr normal_path,
    ref CStr read_only_path,
    ref CStr hard_link_path,
    ref CStr hard_link_alias_path
) {
    let i32 normal_code = switch (fs::open_read(normal_path)) {
        case .ok(file) : { inspect_then_close(file, false, 1, 333) }
        case .err(_) : { 344 }
    };
    if (normal_code != 0) {
        return normal_code;
    }
    let i32 read_only_code = switch (fs::open_read(read_only_path)) {
        case .ok(file) : { inspect_then_close(file, true, 1, 345) }
        case .err(_) : { 356 }
    };
    if (read_only_code != 0) {
        return read_only_code;
    }
    let i32 link_code = switch (fs::open_read(hard_link_path)) {
        case .ok(file) : { inspect_then_close(file, false, 2, 357) }
        case .err(_) : { 368 }
    };
    if (link_code != 0) {
        return link_code;
    }
    let i32 alias_code = switch (fs::open_read(hard_link_alias_path)) {
        case .ok(file) : { inspect_then_close(file, false, 2, 369) }
        case .err(_) : { 380 }
    };
    return if (alias_code != 0) { alias_code } else { check_read_only_mutation() };
}
