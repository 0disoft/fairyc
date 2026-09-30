module files;
import std::fs as fs;
import std::bytes as bytes;
import std::text as text;
import format as format;
import output as output;

@effects(console)
pub fn Result<text::CString, i32> cpath(ref str path) {
    return switch (text::cstring_from_text(path)) {
        case .ok(value) : { .ok(value) }
        case .err(_) : { .err(output::error(path, "cannot allocate file path")) }
    };
}
@effects(console)
pub fn Result<text::String, i32> suffix(ref str path, ref str ending) {
    var text::String result = switch (text::copy(path)) {
        case .ok(value) : { value } case .err(_) : { return .err(output::error(path, "cannot allocate sidecar path")); }
    };
    let bool added = switch (text::append(ref mut result, ending)) { case .ok(_) : { true } case .err(_) : { false } };
    if (!added) { return .err(output::error(path, "cannot allocate sidecar path")); }
    return .ok(result);
}

@effects(filesystem, console)
pub fn i32 close(ref str path, fs::File file) {
    return switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(error) : { output::file_error(path, "close failed", fs::error_status(error), fs::error_os_code(error)) }
    };
}
@effects(filesystem, console)
pub fn i32 remove(ref str path) {
    let text::CString name = switch (cpath(path)) { case .ok(v) : { v } case .err(code) : { return code; } };
    return switch (fs::remove_file(text::cstring_as_cstr(ref name))) {
        case .ok(_) : { 0 }
        case .err(error) : { output::file_error(path, "remove sidecar failed; inspect recovery files", fs::error_status(error), fs::error_os_code(error)) }
    };
}
@effects(filesystem, console)
pub fn i32 absent(ref str path) {
    return switch (fs::open_read_utf8(path)) {
        case .ok(file) : {
            let i32 closed = close(path, file);
            output::error(path, "recovery sidecar exists; inspect .tmp/.bak before retry")
        }
        case .err(error) : {
            if (fs::error_os_code(error) == 2) { 0 } else { output::file_error(path, "cannot inspect recovery sidecar", fs::error_status(error), fs::error_os_code(error)) }
        }
    };
}

@effects(filesystem, console)
fn bool fill(ref str path, ref mut fs::File file, ref mut u8[] data) {
    let bool ready = switch (fs::read_exact(file, data)) {
        case .ok(n) : { n == data.len }
        case .err(error) : { let i32 code = output::file_error(path, "read failed", fs::error_status(error), fs::error_os_code(error)); false }
    };
    if (!ready) { return false; }
    var u8[1] probe = [0];
    return switch (fs::read(file, ref mut probe)) {
        case .ok(n) : { if (n == 0) { true } else { let i32 code = output::error(path, "store grew while reading"); false } }
        case .err(error) : { let i32 code = output::file_error(path, "read failed", fs::error_status(error), fs::error_os_code(error)); false }
    };
}
@effects(filesystem, console)
pub fn Result<bytes::Buffer, i32> read(ref str path, fs::File input) {
    var fs::File file = input;
    let usize size = switch (fs::size(ref file)) {
        case .ok(n) : { n }
        case .err(error) : { return .err(output::file_error(path, "size query failed", fs::error_status(error), fs::error_os_code(error))); }
    };
    if (size > 262144) { return .err(output::error(path, "store exceeds 262144 bytes")); }
    var bytes::Buffer data = bytes::empty();
    if (!format::buffer_ok(bytes::resize_zeroed(ref mut data, size))) { return .err(output::error(path, "cannot allocate store buffer")); }
    let bool filled = fill(path, ref mut file, bytes::view_mut(ref mut data));
    let i32 closed = close(path, file);
    if (!filled || closed != 0) { return .err(1); }
    return .ok(data);
}

// The cooperative lock owns this exclusive sibling staging slot.
// Publish only after a complete write, sync and checked consuming close.
@effects(filesystem, console)
pub fn i32 save(ref str path, ref str temporary, ref u8[] data) {
    let text::CString main_name = switch (cpath(path)) { case .ok(v) : { v } case .err(code) : { return code; } };
    let text::CString temp_name = switch (cpath(temporary)) { case .ok(v) : { v } case .err(code) : { return code; } };
    var fs::File file = switch (fs::create_new(text::cstring_as_cstr(ref temp_name))) {
        case .ok(v) : { v }
        case .err(error) : { return output::file_error(temporary, "create temporary store failed", fs::error_status(error), fs::error_os_code(error)); }
    };
    let bool written = switch (fs::write_all(ref mut file, data)) {
        case .ok(n) : { n == data.len }
        case .err(error) : { let i32 code = output::file_error(temporary, "write failed; temporary file preserved", fs::error_status(error), fs::error_os_code(error)); false }
    };
    var bool synced = false;
    if (written) {
        synced = switch (fs::sync(ref mut file)) {
            case .ok(_) : { true }
            case .err(error) : { let i32 code = output::file_error(temporary, "sync failed; temporary file preserved", fs::error_status(error), fs::error_os_code(error)); false }
        };
    }
    let i32 closed = close(temporary, file);
    if (!written || !synced || closed != 0) { return output::error(path, "save not installed; temporary file preserved"); }
    return switch (fs::replace_file(text::cstring_as_cstr(ref temp_name), text::cstring_as_cstr(ref main_name))) {
        case .ok(_) : { 0 }
        case .err(error) : { output::file_error(path, "replace failed; live store unchanged; temporary file preserved", fs::error_status(error), fs::error_os_code(error)) }
    };
}
