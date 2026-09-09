module main;
import scanner as scanner;
import std::io as io;
import std::bytes as bytes;

struct Byte { usize count; u8 value; bool ok; }
struct Word { usize value; bool ok; }

@effects(console)
fn Byte read_byte() {
    var u8[1] data = [0];
    return switch (io::read_stdin(ref mut data)) {
        case .ok(count) : { Byte { .count = count, .value = data[0], .ok = true } }
        case .err(_) : { Byte { .count = 0, .value = 0, .ok = false } }
    };
}

@effects(console)
fn Word read_word() {
    var usize value = 0;
    var usize factor = 1;
    var usize index = 0;
    while (index < 4) {
        let Byte part = read_byte();
        if (!part.ok || part.count != 1) { return Word { .value = 0, .ok = false }; }
        value += (part.value as usize) * factor;
        index += 1;
        if (index < 4) { factor *= 256; }
    }
    return Word { .value = value, .ok = true };
}

// Modes 0/1 use defaults; 2/3 add four u32 LE limits after the u16 LE length.
// Odd modes select the previous revision. Exact source bytes follow the header.
// Clean EOF is accepted only between frames; malformed framing exits 8.
@effects(console)
fn i32 main() {
    var usize frames = 0;
    while (true) {
        let Byte mode = read_byte();
        if (!mode.ok) { return 8; }
        if (mode.count == 0) { return 0; }
        if (mode.value > 3 || frames == 2048) { return 8; }
        let Byte low = read_byte(); let Byte high = read_byte();
        if (!low.ok || !high.ok || low.count != 1 || high.count != 1) { return 8; }
        let usize length = (low.value as usize) + (high.value as usize) * 256;
        if (length > 16384) { return 4; }
        var usize max_bytes = 16384;
        var usize max_tokens = 1048576;
        var usize max_diagnostics = 256;
        var usize max_work = 16777216;
        if (mode.value >= 2) {
            let Word byte_limit = read_word(); let Word token_limit = read_word();
            let Word diagnostic_limit = read_word(); let Word work_limit = read_word();
            if (!byte_limit.ok || !token_limit.ok || !diagnostic_limit.ok || !work_limit.ok) { return 8; }
            max_bytes = byte_limit.value; max_tokens = token_limit.value;
            max_diagnostics = diagnostic_limit.value; max_work = work_limit.value;
        }
        var bytes::Buffer input = bytes::empty();
        var usize index = 0;
        while (index < length) {
            let Byte value = read_byte();
            if (!value.ok || value.count != 1) { return 8; }
            if (!switch (bytes::append_byte(ref mut input, value.value)) { case .ok(_) : { true } case .err(_) : { false } }) { return 3; }
            index += 1;
        }
        let i32 status = scanner::report_work(bytes::view(ref input), mode.value == 1 || mode.value == 3, max_bytes, max_tokens, max_diagnostics, max_work);
        if (status != 0) { return status; }
        frames += 1;
    }
    return 8;
}
