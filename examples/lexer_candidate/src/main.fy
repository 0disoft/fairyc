module main;
import scanner as scanner;
import std::io as io;
import std::bytes as bytes;

struct Byte { usize count; u8 value; bool ok; }

@effects(console)
fn Byte read_byte() {
    var u8[1] data = [0];
    return switch (io::read_stdin(ref mut data)) {
        case .ok(count) : { Byte { .count = count, .value = data[0], .ok = true } }
        case .err(_) : { Byte { .count = 0, .value = 0, .ok = false } }
    };
}

// Private corpus framing: mode byte (0=current, 1=previous), u16 LE length, bytes.
// Clean EOF is accepted only between frames; malformed framing exits 8.
@effects(console)
fn i32 main() {
    var usize frames = 0;
    while (true) {
        let Byte mode = read_byte();
        if (!mode.ok) { return 8; }
        if (mode.count == 0) { return 0; }
        if (mode.value > 1 || frames == 2048) { return 8; }
        let Byte low = read_byte(); let Byte high = read_byte();
        if (!low.ok || !high.ok || low.count != 1 || high.count != 1) { return 8; }
        let usize length = (low.value as usize) + (high.value as usize) * 256;
        if (length > 4096) { return 4; }
        var bytes::Buffer input = bytes::empty();
        var usize index = 0;
        while (index < length) {
            let Byte value = read_byte();
            if (!value.ok || value.count != 1) { return 8; }
            if (!switch (bytes::append_byte(ref mut input, value.value)) { case .ok(_) : { true } case .err(_) : { false } }) { return 3; }
            index += 1;
        }
        let i32 status = scanner::report_mode(bytes::view(ref input), mode.value == 1);
        if (status != 0) { return status; }
        frames += 1;
    }
    return 8;
}
