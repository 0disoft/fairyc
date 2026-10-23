module main;

import std::bytes as bytes;
import std::fs as fs;
import std::io as io;

@effects(filesystem, console)
fn i32 main() {
    var bytes::Buffer data = bytes::empty();
    assert(bytes::resize_zeroed(ref mut data, 7).is_ok);
    var fs::File input = switch (fs::open_read(c"input.bin")) {
        case .ok(file): { file }
        case .err(_): { return 1; }
    };
    assert(fs::read_exact(ref mut input, bytes::view_mut(ref mut data)).is_ok);
    assert(fs::close(input).is_ok);
    let ref u8[] view = bytes::view(ref data);
    assert(view[0] == 0 && view[1] == 1 && view[2] == 127 && view[3] == 128);
    assert(view[4] == 255 && view[5] == 10 && view[6] == 42);
    var fs::File output = switch (fs::create_new(c"output.bin")) {
        case .ok(file): { file }
        case .err(_): { return 2; }
    };
    assert(fs::write_all(ref mut output, view).is_ok);
    assert(fs::sync(ref mut output).is_ok);
    assert(fs::close(output).is_ok);
    assert(io::write("Fairy C portability\n").is_ok);
    return 0;
}
