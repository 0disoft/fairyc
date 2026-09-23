module pages;
import scanner as scanner;
import diagnostics as diagnostics;
import std::bytes as bytes;
import std::io as io;

struct Byte { usize count; u8 value; bool ok; }
struct Word { usize value; bool ok; }
struct Window { usize start; usize position; }

@effects(console)
fn Byte read_byte() {
    var u8[1] data = [0];
    return switch (io::read_stdin(ref mut data)) {
        case .ok(count): { Byte { .count = count, .value = data[0], .ok = true } }
        case .err(_): { Byte { .count = 0, .value = 0, .ok = false } }
    };
}
@effects(console)
fn Word read_word() {
    var usize value = 0; var usize factor = 1; var usize index = 0;
    while (index < 4) {
        let Byte part = read_byte();
        if (!part.ok || part.count != 1) { return Word { .value = 0, .ok = false }; }
        value += (part.value as usize) * factor; index += 1;
        if (index < 4) { factor *= 256; }
    }
    return Word { .value = value, .ok = true };
}
@effects(console)
fn void number(ref mut Window window, usize value) {
    if (window.position >= window.start && window.position - window.start < 4096) {
        assert(switch (io::write_usize(value)) { case .ok(_): { true } case .err(_): { false } });
        assert(switch (io::write(" ")) { case .ok(_): { true } case .err(_): { false } });
    }
    window.position = window.position + 1;
}
@effects(console)
fn void text(ref mut Window window, ref u8[] value) {
    number(window, value.len);
    for (let u8 byte in value) { number(window, byte as usize); }
}

// Each independent request scans the entire input and selects at most 4096
// observation integers. No hash, token selection or cooked-byte truncation.
@effects(console)
pub fn void report(ref scanner::Stream stream, usize start) {
    var Window window = Window { .start = start, .position = 0 };
    number(ref mut window, scanner::token_count(stream));
    var usize index = 0;
    while (index < scanner::token_count(stream)) {
        let scanner::TokenInfo token = scanner::token_info(stream, index);
        number(ref mut window, token.kind as usize); number(ref mut window, token.trivia as usize);
        number(ref mut window, token.start); number(ref mut window, token.end);
        text(ref mut window, scanner::token_cooked(stream, index));
        index += 1;
    }
    number(ref mut window, scanner::diagnostic_count(stream));
    index = 0;
    while (index < scanner::diagnostic_count(stream)) {
        let diagnostics::Info info = scanner::diagnostic_info(stream, index);
        number(ref mut window, info.code as usize); number(ref mut window, info.start); number(ref mut window, info.end);
        number(ref mut window, if (info.exit_repair) { 1 } else { 0 });
        text(ref mut window, scanner::diagnostic_token(stream, index));
        index += 1;
    }
}

// Private one-request protocol: mode u8, source length u32 LE, observation start
// u32 LE, exact source bytes, then EOF. Bounds precede source allocation.
@effects(console)
pub fn i32 run() {
    let Byte mode = read_byte(); let Word length = read_word(); let Word start = read_word();
    if (!mode.ok || mode.count != 1 || mode.value > 1 || !length.ok || !start.ok || start.value > 1048576) { return 8; }
    if (length.value > 32768) { return 4; }
    var bytes::Buffer input = bytes::empty(); var usize index = 0;
    while (index < length.value) {
        let Byte part = read_byte();
        if (!part.ok || part.count != 1) { return 8; }
        if (!switch (bytes::append_byte(ref mut input, part.value)) { case .ok(_): { true } case .err(_): { false } }) { return 3; }
        index += 1;
    }
    let Byte end = read_byte();
    if (!end.ok || end.count != 0) { return 8; }
    return switch (scanner::collect(input, mode.value == 1, 32768, 1048576, 256, 16777216)) {
        case .ok(stream): { report(ref stream, start.value); 0 }
        case .err(status): { status }
    };
}
