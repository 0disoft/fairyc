module tasks;
import std::fs as fs;
import std::bytes as bytes;
import std::text as text;
import files as files;
import format as format;
import output as output;

@effects(filesystem, console)
fn i32 process(ref str path, ref str temporary, ref u8[] data, u32 command, ref u8[] argument, i32 requested, ref mut bytes::Buffer response) {
    let i32 next = switch (format::validate(path, data)) { case .ok(n) : { n } case .err(code) : { return code; } };
    if (command == 0) { return format::listing(path, data, response); }
    let bytes::Buffer changed = switch (format::change(path, data, next, command, argument, requested, response)) {
        case .ok(value) : { value } case .err(code) : { return code; }
    };
    return files::save(path, temporary, bytes::view(ref changed));
}

@effects(filesystem, console)
fn i32 locked(ref str path, ref str temporary, ref str backup, u32 command, ref u8[] argument, i32 requested, ref mut bytes::Buffer response) {
    if (files::absent(temporary) != 0 || files::absent(backup) != 0) { return 1; }
    return switch (fs::open_read_utf8(path)) {
        case .ok(file) : {
            switch (files::read(path, file)) {
                case .ok(data) : { process(path, temporary, bytes::view(ref data), command, argument, requested, response) }
                case .err(code) : { code }
            }
        }
        case .err(error) : {
            if (fs::error_os_code(error) == 2) {
                switch (bytes::copy(b"FYTASK1\t1\n")) {
                    case .ok(data) : { process(path, temporary, bytes::view(ref data), command, argument, requested, response) }
                    case .err(_) : { output::error(path, "cannot allocate store buffer") }
                }
            } else { output::file_error(path, "open store failed", fs::error_status(error), fs::error_os_code(error)) }
        }
    };
}

@effects(filesystem, console)
pub fn i32 run(ref str path, u32 command, ref u8[] argument, i32 requested) {
    let text::String tmp = switch (files::suffix(path, ".tmp")) { case .ok(v) : { v } case .err(code) : { return code; } };
    let text::String bak = switch (files::suffix(path, ".bak")) { case .ok(v) : { v } case .err(code) : { return code; } };
    let text::String guard = switch (files::suffix(path, ".lock")) { case .ok(v) : { v } case .err(code) : { return code; } };
    let ref str guard_path = text::string_as_str(ref guard);
    let text::CString guard_name = switch (files::cpath(guard_path)) { case .ok(v) : { v } case .err(code) : { return code; } };
    let fs::File lock_file = switch (fs::create_new(text::cstring_as_cstr(ref guard_name))) {
        case .ok(v) : { v }
        case .err(error) : { return output::file_error(guard_path, "create lock failed; another run or stale lock", fs::error_status(error), fs::error_os_code(error)); }
    };
    let i32 lock_closed = files::close(guard_path, lock_file);
    var bytes::Buffer response = bytes::empty();
    var i32 result = 1;
    if (lock_closed == 0) {
        result = locked(path, text::string_as_str(ref tmp), text::string_as_str(ref bak), command, argument, requested, ref mut response);
    }
    let i32 released = switch (fs::remove_file(text::cstring_as_cstr(ref guard_name))) {
        case .ok(_) : { 0 }
        case .err(error) : { output::file_error(guard_path, "lock cleanup failed; mutation may already be saved", fs::error_status(error), fs::error_os_code(error)) }
    };
    if (result != 0 || released != 0) { return 1; }
    return output::response(ref response);
}
