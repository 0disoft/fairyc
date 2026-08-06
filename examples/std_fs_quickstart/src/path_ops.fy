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

@effects(filesystem)
fn i32 close_directory_file(fs::File file, i32 close_error) {
    return switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { close_error }
    };
}

@effects(filesystem)
pub fn i32 check_directory_lifecycle(
    ref CStr directory,
    ref CStr child,
    ref CStr file_collision,
    ref CStr missing_parent
) {
    let i32 missing_parent_code = switch (fs::create_dir(missing_parent)) {
        case .ok(_) : { 235 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 3 &&
                fs::error_completed(error) == 0) { 0 } else { 236 }
        }
    };
    let i32 file_setup_code = if (missing_parent_code != 0) {
        missing_parent_code
    } else { switch (fs::create_truncate(file_collision)) {
        case .ok(file) : { close_directory_file(file, 237) }
        case .err(_) : { 238 }
    } };
    let i32 file_collision_code = if (file_setup_code != 0) {
        file_setup_code
    } else { switch (fs::create_dir(file_collision)) {
        case .ok(_) : { 239 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 183 &&
                fs::error_completed(error) == 0) { 0 } else { 240 }
        }
    } };
    let i32 file_cleanup_code = if (file_collision_code != 0) {
        file_collision_code
    } else { switch (fs::remove_file(file_collision)) {
        case .ok(_) : { 0 }
        case .err(_) : { 241 }
    } };
    let i32 create_code = if (file_cleanup_code != 0) {
        file_cleanup_code
    } else { switch (fs::create_dir(directory)) {
        case .ok(value) : { if (value == 0) { 0 } else { 242 } }
        case .err(_) : { 243 }
    } };
    let i32 collision_code = if (create_code != 0) {
        create_code
    } else { switch (fs::create_dir(directory)) {
        case .ok(_) : { 244 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 183 &&
                fs::error_completed(error) == 0) { 0 } else { 245 }
        }
    } };
    let i32 child_code = if (collision_code != 0) {
        collision_code
    } else { switch (fs::create_truncate(child)) {
        case .ok(file) : { close_directory_file(file, 246) }
        case .err(_) : { 247 }
    } };
    let i32 nonempty_code = if (child_code != 0) {
        child_code
    } else { switch (fs::remove_dir(directory)) {
        case .ok(_) : { 248 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 145 &&
                fs::error_completed(error) == 0) { 0 } else { 249 }
        }
    } };
    let i32 child_cleanup_code = if (nonempty_code != 0) {
        nonempty_code
    } else { switch (fs::remove_file(child)) {
        case .ok(_) : { 0 }
        case .err(_) : { 250 }
    } };
    let i32 remove_code = if (child_cleanup_code != 0) {
        child_cleanup_code
    } else { switch (fs::remove_dir(directory)) {
        case .ok(value) : { if (value == 0) { 0 } else { 251 } }
        case .err(_) : { 252 }
    } };
    return if (remove_code != 0) {
        remove_code
    } else { switch (fs::remove_dir(directory)) {
        case .ok(_) : { 253 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 2 &&
                fs::error_completed(error) == 0) { 0 } else { 254 }
        }
    } };
}

@effects(filesystem)
pub fn i32 check_rename_dir(
    ref CStr source,
    ref CStr destination,
    ref CStr source_child,
    ref CStr destination_child
) {
    let i32 source_code = switch (fs::create_dir(source)) {
        case .ok(_) : { 0 }
        case .err(_) : { 255 }
    };
    let i32 child_code = if (source_code != 0) {
        source_code
    } else { switch (fs::create_truncate(source_child)) {
        case .ok(file) : { close_directory_file(file, 256) }
        case .err(_) : { 257 }
    } };
    let i32 destination_code = if (child_code != 0) {
        child_code
    } else { switch (fs::create_dir(destination)) {
        case .ok(_) : { 0 }
        case .err(_) : { 258 }
    } };
    let i32 collision_code = if (destination_code != 0) {
        destination_code
    } else { switch (fs::rename_dir(source, destination)) {
        case .ok(_) : { 259 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 183 &&
                fs::error_completed(error) == 0) { 0 } else { 260 }
        }
    } };
    let i32 destination_cleanup_code = if (collision_code != 0) {
        collision_code
    } else { switch (fs::remove_dir(destination)) {
        case .ok(_) : { 0 }
        case .err(_) : { 261 }
    } };
    let i32 rename_code = if (destination_cleanup_code != 0) {
        destination_cleanup_code
    } else { switch (fs::rename_dir(source, destination)) {
        case .ok(value) : { if (value == 0) { 0 } else { 262 } }
        case .err(_) : { 263 }
    } };
    let i32 old_missing_code = if (rename_code != 0) {
        rename_code
    } else { switch (fs::remove_dir(source)) {
        case .ok(_) : { 264 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 2 &&
                fs::error_completed(error) == 0) { 0 } else { 265 }
        }
    } };
    let i32 child_preserved_code = if (old_missing_code != 0) {
        old_missing_code
    } else { switch (fs::open_read(destination_child)) {
        case .ok(file) : { close_directory_file(file, 266) }
        case .err(_) : { 267 }
    } };
    let i32 child_cleanup_code = if (child_preserved_code != 0) {
        child_preserved_code
    } else { switch (fs::remove_file(destination_child)) {
        case .ok(_) : { 0 }
        case .err(_) : { 268 }
    } };
    let i32 directory_cleanup_code = if (child_cleanup_code != 0) {
        child_cleanup_code
    } else { switch (fs::remove_dir(destination)) {
        case .ok(_) : { 0 }
        case .err(_) : { 269 }
    } };
    return if (directory_cleanup_code != 0) {
        directory_cleanup_code
    } else { switch (fs::rename_dir(source, destination)) {
        case .ok(_) : { 270 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 2 &&
                fs::error_completed(error) == 0) { 0 } else { 271 }
        }
    } };
}

@effects(filesystem)
pub fn i32 check_create_dir_all(
    ref CStr root,
    ref CStr nested,
    ref CStr leaf,
    ref CStr collision,
    ref CStr collision_child,
    ref CStr partial_failure,
    ref CStr partial_prefix,
    ref CStr partial_root
) {
    let i32 create_code = switch (fs::create_dir_all(leaf)) {
        case .ok(value) : { if (value == 0) { 0 } else { 272 } }
        case .err(_) : { 273 }
    };
    let i32 existing_code = if (create_code != 0) {
        create_code
    } else { switch (fs::create_dir_all(leaf)) {
        case .ok(value) : { if (value == 0) { 0 } else { 274 } }
        case .err(_) : { 275 }
    } };
    let i32 collision_setup_code = if (existing_code != 0) {
        existing_code
    } else { switch (fs::create_truncate(collision)) {
        case .ok(file) : { close_directory_file(file, 276) }
        case .err(_) : { 277 }
    } };
    let i32 collision_code = if (collision_setup_code != 0) {
        collision_setup_code
    } else { switch (fs::create_dir_all(collision_child)) {
        case .ok(_) : { 278 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 183 &&
                fs::error_completed(error) == 0) { 0 } else { 279 }
        }
    } };
    let i32 collision_cleanup_code = if (collision_code != 0) {
        collision_code
    } else { switch (fs::remove_file(collision)) {
        case .ok(_) : { 0 }
        case .err(_) : { 280 }
    } };
    let i32 leaf_cleanup_code = if (collision_cleanup_code != 0) {
        collision_cleanup_code
    } else { switch (fs::remove_dir(leaf)) {
        case .ok(_) : { 0 }
        case .err(_) : { 281 }
    } };
    let i32 nested_cleanup_code = if (leaf_cleanup_code != 0) {
        leaf_cleanup_code
    } else { switch (fs::remove_dir(nested)) {
        case .ok(_) : { 0 }
        case .err(_) : { 282 }
    } };
    let i32 root_cleanup_code = if (nested_cleanup_code != 0) {
        nested_cleanup_code
    } else { switch (fs::remove_dir(root)) {
        case .ok(_) : { 0 }
        case .err(_) : { 283 }
    } };
    let i32 partial_code = if (root_cleanup_code != 0) {
        root_cleanup_code
    } else { switch (fs::create_dir_all(partial_failure)) {
        case .ok(_) : { 284 }
        case .err(error) : {
            if (fs::error_status(error) == 1 &&
                fs::error_os_code(error) == 123 &&
                fs::error_completed(error) == 0) { 0 } else { 285 }
        }
    } };
    let i32 partial_prefix_code = if (partial_code != 0) {
        partial_code
    } else { switch (fs::remove_dir(partial_prefix)) {
        case .ok(_) : { 0 }
        case .err(_) : { 286 }
    } };
    return if (partial_prefix_code != 0) {
        partial_prefix_code
    } else { switch (fs::remove_dir(partial_root)) {
        case .ok(_) : { 0 }
        case .err(_) : { 287 }
    } };
}
