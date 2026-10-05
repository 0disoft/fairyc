module file_cli;
import scanner as scanner;
import declaration_output as declarations;
import token_output as output;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;
import std::io as io;
import std::process as process;

@effects(console)
fn i32 error(ref str message) { discard io::write_stderr(message); return 2; }

@effects(filesystem)
fn Result<bytes::Buffer, u32> load(ref u8[] path_bytes) {
    let ref str path = switch(text::validate_utf8(path_bytes)) {
        case .ok(value): { value } case .err(_): { return .err(1); }
    };
    var fs::File file = switch(fs::open_read_utf8(path)) {
        case .ok(value): { value } case .err(_): { return .err(1); }
    };
    var bytes::Buffer input = bytes::empty();
    if (!bytes::resize_zeroed(ref mut input, 32769).is_ok) { return .err(3); }
    var usize used = 0;
    while (true) {
        let usize count = switch(fs::read(ref mut file, bytes::view_mut(ref mut input)[used..])) {
            case .ok(value): { value } case .err(_): { return .err(1); }
        };
        used += count;
        if (used > 32768) { return .err(4); }
        if (count == 0) { break; }
    }
    if (!fs::close(file).is_ok) { return .err(1); }
    if (!bytes::resize_zeroed(ref mut input, used).is_ok) { return .err(3); }
    return .ok(input);
}

@effects(filesystem, console)
pub fn i32 run() {
    let usize argc = process::arg_count();
    if (argc != 2 && argc != 3) { return error("usage: lexer-candidate --tokens-file <path>\n       lexer-candidate --declarations-file <path>\n"); }
    var bytes::Buffer argument = bytes::empty();
    if (!bytes::resize_zeroed(ref mut argument, 4096).is_ok) { return error("error: buffer allocation failed\n"); }
    let usize flag_length = switch(process::arg(1, bytes::view_mut(ref mut argument))) {
        case .ok(value): { value } case .err(_): { return error("usage: lexer-candidate --tokens-file <path>\n       lexer-candidate --declarations-file <path>\n"); }
    };
    let ref u8[] flag = bytes::view(ref argument)[..flag_length];
    if (argc == 2 && bytes::equal(flag, b"--help")) {
        return if (io::write("usage: lexer-candidate --tokens-file <path>\n       lexer-candidate --declarations-file <path>\n       lexer-candidate --help\nNo arguments: private binary stdin protocol.\n").is_ok) { 0 }
            else { error("error: stdout write failed\n") };
    }
    let bool declaration_mode = bytes::equal(flag, b"--declarations-file");
    if (argc != 3 || (!bytes::equal(flag, b"--tokens-file") && !declaration_mode)) { return error("usage: lexer-candidate --tokens-file <path>\n       lexer-candidate --declarations-file <path>\n"); }
    let usize length = switch(process::arg(2, bytes::view_mut(ref mut argument))) {
        case .ok(value): { value } case .err(_): { return error("error: file path limit (4095 UTF-8 bytes)\n"); }
    };
    if (length > 4095) { return error("error: file path limit (4095 UTF-8 bytes)\n"); }
    let ref u8[] path = bytes::view(ref argument)[..length];
    let bytes::Buffer input = switch(load(path)) {
        case .ok(value): { value }
        case .err(code): { return error(if (code == 4) { "error: source limit (32768 bytes)\n" }
            else if (code == 3) { "error: buffer allocation failed\n" } else { "error: input file read failed\n" }); }
    };
    let scanner::Stream stream = switch(scanner::collect(input, false, 32768, 1048576, 256, 16777216)) {
        case .ok(value): { value }
        case .err(code): { return error(if (code == 4) { "error: lexer storage limit\n" }
            else if (code == 2) { "error: unsupported lexer input\n" } else { "error: buffer allocation failed\n" }); }
    };
    if (scanner::diagnostic_count(ref stream) > 0) { return output::errors(ref stream, path); }
    if (declaration_mode) { return declarations::run(stream, path); }
    var bytes::Buffer result = bytes::empty();
    let u32 status = output::tokens(ref stream, ref mut result);
    if (status != 0) { return error(if (status == 4) { "error: output limit (1048576 bytes)\n" }
        else if (status == 5) { "error: invalid token coverage\n" } else { "error: buffer allocation failed\n" }); }
    let ref u8[] data = bytes::view(ref result);
    return if (switch(io::write_bytes(data)) { case .ok(count): { count == data.len } case .err(_): { false } }) { 0 }
        else { error("error: stdout write failed\n") };
}
