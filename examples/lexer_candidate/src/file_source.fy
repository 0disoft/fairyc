module file_source;
import candidate_limits as limits;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;

// This leaf owns file reading without importing scanner, parser or CLI code.
// One extra byte detects overflow before publishing any partial input.
@effects(filesystem)
fn Result<bytes::Buffer, u32> read(ref u8[] path_bytes, usize maximum) {
    let ref str path = switch(text::validate_utf8(path_bytes)) {
        case .ok(value): { value } case .err(_): { return .err(1); }
    };
    var fs::File file = switch(fs::open_read_utf8(path)) {
        case .ok(value): { value } case .err(_): { return .err(1); }
    };
    var bytes::Buffer input = bytes::empty();
    if (!bytes::resize_zeroed(ref mut input, maximum+1).is_ok) { return .err(3); }
    var usize used = 0;
    while (true) {
        let usize count = switch(fs::read(ref mut file, bytes::view_mut(ref mut input)[used..])) {
            case .ok(value): { value } case .err(_): { return .err(1); }
        };
        used += count;
        if (used > maximum) { return .err(4); }
        if (count == 0) { break; }
    }
    if (!fs::close(file).is_ok) { return .err(1); }
    if (!bytes::resize_zeroed(ref mut input, used).is_ok) { return .err(3); }
    return .ok(input);
}

@effects(filesystem)
pub fn Result<bytes::Buffer, u32> load(ref u8[] path_bytes) {
    return read(path_bytes, limits::source_bytes());
}
@effects(filesystem)
pub fn Result<bytes::Buffer, u32> manifest(ref u8[] path_bytes) {
    if (path_bytes.len == 0 || path_bytes.len > 4095) { return .err(1); }
    return read(path_bytes, limits::manifest_bytes());
}
