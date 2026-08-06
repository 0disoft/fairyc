module link_ops;

import std::fs as fs;

@effects(filesystem)
fn i32 seed_source(fs::File input) {
    var fs::File file = input;
    let ref u8[] payload = b"FairyLink";
    let i32 write_code = switch (fs::write_all(ref mut file, payload)) {
        case .ok(written) : { if (written == 9) { 0 } else { 381 } }
        case .err(_) : { 382 }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 383 }
    };
    return if (write_code != 0) { write_code } else { close_code };
}

@effects(filesystem)
fn i32 verify_linked_pair(fs::File source_input, fs::File destination_input) {
    var fs::File source = source_input;
    var fs::File destination = destination_input;
    var fs::FileId source_id = fs::FileId {
        .volume_low = 0, .volume_high = 0,
        .file_low_0 = 0, .file_low_1 = 0,
        .file_high_0 = 0, .file_high_1 = 0,
    };
    let i32 source_id_code = switch (fs::file_id(ref source)) {
        case .ok(value) : { source_id = value; 0 }
        case .err(_) : { 384 }
    };
    var fs::FileId destination_id = fs::FileId {
        .volume_low = 0, .volume_high = 0,
        .file_low_0 = 0, .file_low_1 = 0,
        .file_high_0 = 0, .file_high_1 = 0,
    };
    let i32 destination_id_code = if (source_id_code != 0) {
        source_id_code
    } else { switch (fs::file_id(ref destination)) {
        case .ok(value) : { destination_id = value; 0 }
        case .err(_) : { 385 }
    } };
    let i32 identity_code = if (destination_id_code != 0) {
        destination_id_code
    } else if (fs::file_id_equal(ref source_id, ref destination_id)) { 0 } else { 386 };
    let i32 source_count_code = if (identity_code != 0) {
        identity_code
    } else { switch (fs::hard_link_count(ref source)) {
        case .ok(count) : { if (count == 2) { 0 } else { 387 } }
        case .err(_) : { 388 }
    } };
    let i32 destination_count_code = if (source_count_code != 0) {
        source_count_code
    } else { switch (fs::hard_link_count(ref destination)) {
        case .ok(count) : { if (count == 2) { 0 } else { 389 } }
        case .err(_) : { 390 }
    } };
    var u8[9] received = [0, 0, 0, 0, 0, 0, 0, 0, 0];
    let i32 bytes_code = if (destination_count_code != 0) {
        destination_count_code
    } else { switch (fs::read_exact(ref mut destination, ref mut received)) {
        case .ok(read) : {
            if (read == 9 &&
                received[0] == 70 && received[1] == 97 && received[2] == 105 &&
                received[3] == 114 && received[4] == 121 && received[5] == 76 &&
                received[6] == 105 && received[7] == 110 && received[8] == 107) { 0 } else { 391 }
        }
        case .err(_) : { 392 }
    } };
    let i32 source_close_code = switch (fs::close(source)) {
        case .ok(_) : { 0 }
        case .err(_) : { 393 }
    };
    let i32 destination_close_code = switch (fs::close(destination)) {
        case .ok(_) : { 0 }
        case .err(_) : { 394 }
    };
    return if (bytes_code != 0) {
        bytes_code
    } else if (source_close_code != 0) {
        source_close_code
    } else {
        destination_close_code
    };
}

@effects(filesystem)
fn i32 open_destination_then_verify(fs::File source, ref CStr destination) {
    return switch (fs::open_read(destination)) {
        case .ok(destination_file) : { verify_linked_pair(source, destination_file) }
        case .err(_) : { switch (fs::close(source)) {
            case .ok(_) : { 409 }
            case .err(_) : { 414 }
        } }
    };
}

@effects(filesystem)
fn i32 verify_surviving_alias(fs::File input) {
    var fs::File file = input;
    let i32 count_code = switch (fs::hard_link_count(ref file)) {
        case .ok(count) : { if (count == 1) { 0 } else { 395 } }
        case .err(_) : { 396 }
    };
    var u8[9] received = [0, 0, 0, 0, 0, 0, 0, 0, 0];
    let i32 bytes_code = if (count_code != 0) {
        count_code
    } else { switch (fs::read_exact(ref mut file, ref mut received)) {
        case .ok(read) : {
            if (read == 9 && received[0] == 70 && received[8] == 107) { 0 } else { 397 }
        }
        case .err(_) : { 398 }
    } };
    return switch (fs::close(file)) {
        case .ok(_) : { bytes_code }
        case .err(_) : { 399 }
    };
}

@effects(filesystem)
pub fn i32 check_hard_link() {
    let ref CStr source = c"fairyc-std-fs-hard-link-source.tmp";
    let ref CStr destination = c"fairyc-std-fs-hard-link-destination.tmp";
    let ref CStr missing_source = c"fairyc-std-fs-hard-link-missing.tmp";
    let ref CStr missing_destination = c"fairyc-std-fs-hard-link-missing-destination.tmp";
    let i32 seed_code = switch (fs::create_truncate(source)) {
        case .ok(file) : { seed_source(file) }
        case .err(_) : { 400 }
    };
    let i32 link_code = if (seed_code != 0) {
        seed_code
    } else { switch (fs::hard_link(source, destination)) {
        case .ok(value) : { if (value == 0) { 0 } else { 401 } }
        case .err(_) : { 402 }
    } };
    let i32 collision_code = if (link_code != 0) {
        link_code
    } else { switch (fs::hard_link(source, destination)) {
        case .ok(_) : { 403 }
        case .err(error) : {
            if (fs::error_status(error) == 1 && fs::error_os_code(error) == 183 &&
                fs::error_completed(error) == 0) { 0 } else { 404 }
        }
    } };
    let i32 missing_code = if (collision_code != 0) {
        collision_code
    } else { switch (fs::hard_link(missing_source, missing_destination)) {
        case .ok(_) : { 405 }
        case .err(error) : {
            if (fs::error_status(error) == 1 && fs::error_os_code(error) == 2 &&
                fs::error_completed(error) == 0) { 0 } else { 406 }
        }
    } };
    let i32 missing_destination_code = if (missing_code != 0) {
        missing_code
    } else { switch (fs::path_kind(missing_destination)) {
        case .ok(_) : { 407 }
        case .err(error) : {
            if (fs::error_status(error) == 1 && fs::error_os_code(error) == 2) { 0 } else { 408 }
        }
    } };
    var i32 pair_code = missing_destination_code;
    if (pair_code == 0) {
        pair_code = switch (fs::open_read(source)) {
            case .ok(source_file) : { open_destination_then_verify(source_file, destination) }
            case .err(_) : { 410 }
        };
    }
    let i32 remove_source_code = if (pair_code != 0) {
        pair_code
    } else { switch (fs::remove_file(source)) {
        case .ok(_) : { 0 }
        case .err(_) : { 411 }
    } };
    let i32 alias_code = if (remove_source_code != 0) {
        remove_source_code
    } else { switch (fs::open_read(destination)) {
        case .ok(file) : { verify_surviving_alias(file) }
        case .err(_) : { 412 }
    } };
    return if (alias_code != 0) {
        alias_code
    } else { switch (fs::remove_file(destination)) {
        case .ok(_) : { 0 }
        case .err(_) : { 413 }
    } };
}
