module main;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

fn void increment(ref mut usize count) { *count = *count + 1; }
@effects(console)
fn bool write_count(usize value) {
    return switch(io::write_usize(value)) {case .ok(_):{true} case .err(_):{false}};
}
@effects(console)
fn bool write_text(ref str value) {
    return switch(io::write(value)) {case .ok(_):{true} case .err(_):{false}};
}

/// Counts UTF-8 bytes, not Unicode characters. Output is sorted by byte value.
@effects(console)
fn bool histogram(ref u8[] input) {
    var usize[256] bins = [0; 256];
    let ref mut usize[] writable = ref mut bins;
    var usize index = 0;
    while (index < input.len) {
        let ref u8 byte = ref input[index];
        increment(ref mut writable[(*byte) as usize]);
        index += 1;
    }
    let ref usize[] counts = ref bins;
    var usize code = 0;
    while (code < counts.len) {
        let ref usize count = ref counts[code];
        if (*count > 0) {
            if (!write_count(code) || !write_text("\t") ||
                !write_count(*count) || !write_text("\n")) { return false; }
        }
        code += 1;
    }
    return true;
}

@effects(console)
fn i32 main() {
    let usize argc = process::arg_count();
    if (argc == 1) { return if (histogram(b"Fairy C")) { 0 } else { 2 }; }
    if (argc != 2) { discard io::write_stderr("usage: integer-element-refs [UTF-8 text]\n"); return 2; }
    var bytes::Buffer input = bytes::empty();
    if (!switch(bytes::resize_zeroed(ref mut input, 4096)) {case .ok(_):{true} case .err(_):{false}}) {return 2;}
    let usize length = switch(process::arg(1, bytes::view_mut(ref mut input))) {case .ok(n):{n} case .err(_):{return 2;}};
    return if (histogram(bytes::view(ref input)[..length])) { 0 } else { 2 };
}
