module input_file;
import std::fs as fs;
import std::bytes as bytes;
import std::text as text;

/// Read at most 4096 bytes to distinguish the 4095-byte limit from EOF.
/// Errors: 5 allocation, 6 source limit, 13 open/read/close/path failure.
/// Read errors return before evaluation; File scope cleanup still closes the handle.
@effects(filesystem)
pub fn Result<usize, u32> load(ref u8[] path_bytes, ref mut bytes::Buffer input) {
    let ref str path = switch(text::validate_utf8(path_bytes)) {
        case .ok(value): { value }
        case .err(_): { return .err(13); }
    };
    var fs::File file = switch(fs::open_read_utf8(path)) {
        case .ok(value): { value }
        case .err(_): { return .err(13); }
    };
    if (!bytes::resize_zeroed(input, 4096).is_ok) { return .err(5); }
    var usize used = 0;
    while (true) {
        let usize count = switch(fs::read(ref mut file, bytes::view_mut(input)[used..])) {
            case .ok(value): { value }
            case .err(_): { return .err(13); }
        };
        used += count;
        if (used > 4095) { return .err(6); }
        if (count == 0) { break; }
    }
    // Close successfully before exposing the completed source to evaluation.
    if (!fs::close(file).is_ok) { return .err(13); }
    return .ok(used);
}
