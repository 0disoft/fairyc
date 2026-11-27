module main;

import std::bytes as bytes;
import std::fs as fs;
import std::io as io;

@effects(clock) unsafe extern C fn usize parser_ticks();
@effects(clock) unsafe extern C fn usize parser_frequency();

// Each record is a little-endian u16 length followed by that many bytes.
fn Result<usize, i32> decode(ref u8[] input) {
    var usize offset = 0;
    var usize sum = 0;
    while (offset < input.len) {
        if (input.len - offset < 2) {
            return .err(2);
        }
        let usize length = (input[offset] as usize) + (input[offset + 1] as usize) * 256;
        offset += 2;
        if (input.len - offset < length) {
            return .err(3);
        }
        let usize end = offset + length;
        while (offset < end) {
            sum += input[offset] as usize;
            offset += 1;
        }
    }
    return .ok(sum);
}

@effects(filesystem)
fn Result<bytes::Buffer, i32> load() {
    var fs::File file = switch (fs::open_read_utf8("input.bin")) {
        case .ok(value): { value }
        case .err(_): { return .err(4); }
    };
    let usize size = switch (fs::size(ref file)) {
        case .ok(value): { value }
        case .err(_): { return .err(4); }
    };
    if (size > 67108864) {
        return .err(4);
    }
    var bytes::Buffer buffer = bytes::empty();
    switch (bytes::resize_zeroed(ref mut buffer, size)) {
        case .ok(_): {}
        case .err(_): { return .err(4); }
    };
    switch (fs::read_exact(ref mut file, bytes::view_mut(ref mut buffer))) {
        case .ok(_): {}
        case .err(_): { return .err(4); }
    };
    switch (fs::close(file)) {
        case .ok(_): {}
        case .err(_): { return .err(4); }
    };
    return .ok(buffer);
}

@effects(console)
fn i32 report(usize sum, usize elapsed, usize frequency) {
    discard io::write_usize(sum);
    discard io::write(" ");
    discard io::write_usize(elapsed);
    discard io::write(" ");
    discard io::write_usize(frequency);
    discard io::write("\n");
    return 0;
}

@effects(clock, console, filesystem)
fn i32 main() {
    let bytes::Buffer buffer = switch (load()) {
        case .ok(value): { value }
        case .err(code): { return code; }
    };
    let ref u8[] input = bytes::view(ref buffer);
    let usize start = unsafe { parser_ticks() };
    return switch (decode(input)) {
        case .ok(sum): {
            let usize end = unsafe { parser_ticks() };
            let usize frequency = unsafe { parser_frequency() };
            if (start == 0 || end < start || frequency == 0) {
                return 5;
            }
            report(sum, end - start, frequency)
        }
        case .err(code): { code }
    };
}
