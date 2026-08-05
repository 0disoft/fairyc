module path_ops;

import std::fs as fs;

@effects(filesystem)
fn i32 close_after_path_open_error(fs::File file, i32 code) {
    let fs::File owned = file;
    return switch (fs::close(owned)) {
        case .ok(_) : { code }
        case .err(_) : { 185 }
    };
}

@effects(filesystem)
fn i32 remove_while_open_then_close(fs::File input, ref CStr path) {
    var fs::File file = input;
    let i32 sharing_code = switch (fs::remove_file(path)) {
        case .ok(_) : { 197 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 32 &&
                fs::error_completed(error) == 0) { 0 } else { 198 }
        }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 199 }
    };
    return if (sharing_code != 0) { sharing_code } else { close_code };
}

@effects(filesystem)
pub fn i32 check_remove_file(ref CStr path) {
    let i32 create_code = switch (fs::create_truncate(path)) {
        case .ok(file) : { remove_while_open_then_close(file, path) }
        case .err(_) : { 200 }
    };
    let i32 remove_code = if (create_code != 0) {
        create_code
    } else { switch (fs::remove_file(path)) {
        case .ok(value) : { if (value == 0) { 0 } else { 201 } }
        case .err(_) : { 202 }
    } };
    let i32 open_code = if (remove_code != 0) {
        remove_code
    } else { switch (fs::open_read(path)) {
        case .ok(file) : { close_after_path_open_error(file, 203) }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 2 &&
                fs::error_completed(error) == 0) { 0 } else { 204 }
        }
    } };
    return if (open_code != 0) {
        open_code
    } else { switch (fs::remove_file(path)) {
        case .ok(_) : { 205 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 2 &&
                fs::error_completed(error) == 0) { 0 } else { 206 }
        }
    } };
}

@effects(filesystem)
fn i32 seed_rename_file(fs::File input, ref u8[] bytes, i32 write_error, i32 close_error) {
    var fs::File file = input;
    let i32 write_code = switch (fs::write_all(ref mut file, bytes)) {
        case .ok(written) : { if (written == bytes.len) { 0 } else { write_error } }
        case .err(_) : { write_error }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { close_error }
    };
    return if (write_code != 0) { write_code } else { close_code };
}

@effects(filesystem)
fn i32 verify_rename_source_then_close(fs::File input, i32 byte_error, i32 close_error) {
    var fs::File file = input;
    var u8[6] buffer = [0, 0, 0, 0, 0, 0];
    let i32 read_code = switch (fs::read_exact(ref mut file, ref mut buffer)) {
        case .ok(read) : {
            if (read == 6 &&
                buffer[0] == 83 && buffer[1] == 111 && buffer[2] == 117 &&
                buffer[3] == 114 && buffer[4] == 99 && buffer[5] == 101) {
                0
            } else {
                byte_error
            }
        }
        case .err(_) : { byte_error }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { close_error }
    };
    return if (read_code != 0) { read_code } else { close_code };
}

@effects(filesystem)
fn i32 verify_rename_destination_then_close(fs::File input) {
    var fs::File file = input;
    var u8[4] buffer = [0, 0, 0, 0];
    let i32 read_code = switch (fs::read_exact(ref mut file, ref mut buffer)) {
        case .ok(read) : {
            if (read == 4 &&
                buffer[0] == 68 && buffer[1] == 101 &&
                buffer[2] == 115 && buffer[3] == 116) { 0 } else { 215 }
        }
        case .err(_) : { 216 }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 217 }
    };
    return if (read_code != 0) { read_code } else { close_code };
}

@effects(filesystem)
fn i32 rename_while_open_then_close(
    fs::File input,
    ref CStr source,
    ref CStr destination
) {
    var fs::File file = input;
    let i32 rename_code = switch (fs::rename_file(source, destination)) {
        case .ok(_) : { 218 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 32 &&
                fs::error_completed(error) == 0) { 0 } else { 219 }
        }
    };
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 220 }
    };
    return if (rename_code != 0) { rename_code } else { close_code };
}

@effects(filesystem)
pub fn i32 check_rename_file(ref CStr source, ref CStr destination) {
    let ref u8[] source_bytes = b"Source";
    let ref u8[] destination_bytes = b"Dest";
    let i32 source_code = switch (fs::create_truncate(source)) {
        case .ok(file) : { seed_rename_file(file, source_bytes, 207, 208) }
        case .err(_) : { 209 }
    };
    let i32 destination_code = if (source_code != 0) {
        source_code
    } else { switch (fs::create_truncate(destination)) {
        case .ok(file) : { seed_rename_file(file, destination_bytes, 210, 211) }
        case .err(_) : { 212 }
    } };
    let i32 collision_code = if (destination_code != 0) {
        destination_code
    } else { switch (fs::rename_file(source, destination)) {
        case .ok(_) : { 213 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 183 &&
                fs::error_completed(error) == 0) { 0 } else { 214 }
        }
    } };
    let i32 source_preserved_code = if (collision_code != 0) {
        collision_code
    } else { switch (fs::open_read(source)) {
        case .ok(file) : { verify_rename_source_then_close(file, 221, 222) }
        case .err(_) : { 223 }
    } };
    let i32 destination_preserved_code = if (source_preserved_code != 0) {
        source_preserved_code
    } else { switch (fs::open_read(destination)) {
        case .ok(file) : { verify_rename_destination_then_close(file) }
        case .err(_) : { 224 }
    } };
    let i32 remove_destination_code = if (destination_preserved_code != 0) {
        destination_preserved_code
    } else { switch (fs::remove_file(destination)) {
        case .ok(_) : { 0 }
        case .err(_) : { 225 }
    } };
    let i32 sharing_code = if (remove_destination_code != 0) {
        remove_destination_code
    } else { switch (fs::open_read(source)) {
        case .ok(file) : { rename_while_open_then_close(file, source, destination) }
        case .err(_) : { 226 }
    } };
    let i32 rename_code = if (sharing_code != 0) {
        sharing_code
    } else { switch (fs::rename_file(source, destination)) {
        case .ok(value) : { if (value == 0) { 0 } else { 227 } }
        case .err(_) : { 228 }
    } };
    let i32 source_missing_code = if (rename_code != 0) {
        rename_code
    } else { switch (fs::open_read(source)) {
        case .ok(file) : { close_after_path_open_error(file, 229) }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 2 &&
                fs::error_completed(error) == 0) { 0 } else { 230 }
        }
    } };
    let i32 final_bytes_code = if (source_missing_code != 0) {
        source_missing_code
    } else { switch (fs::open_read(destination)) {
        case .ok(file) : { verify_rename_source_then_close(file, 231, 232) }
        case .err(_) : { 233 }
    } };
    return if (final_bytes_code != 0) {
        final_bytes_code
    } else { switch (fs::remove_file(destination)) {
        case .ok(_) : { 0 }
        case .err(_) : { 234 }
    } };
}
