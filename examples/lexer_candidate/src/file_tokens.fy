module file_tokens;
import candidate_limits as limits;
import file_source as files;
import scanner as scanner;
import token_output as output;
import std::bytes as bytes;
import std::io as io;

@effects(console)
fn i32 error(ref str message) { discard io::write_stderr(message); return 2; }

// The original --tokens-file path, independent of parser/type/module dispatch.
@effects(filesystem,console)
pub fn i32 run(ref u8[] path) {
    let bytes::Buffer input = switch(files::load(path)) {
        case .ok(value): { value }
        case .err(code): { return error(if (code == 4) { "error: source limit (65536 bytes)\n" }
            else if (code == 3) { "error: buffer allocation failed\n" } else { "error: input file read failed\n" }); }
    };
    let scanner::Stream stream = switch(scanner::collect(input, false, limits::source_bytes(), 1048576, 256, 16777216)) {
        case .ok(value): { value }
        case .err(code): { return error(if (code == 4) { "error: lexer storage limit\n" }
            else if (code == 2) { "error: unsupported lexer input\n" } else { "error: buffer allocation failed\n" }); }
    };
    if (scanner::diagnostic_count(ref stream) > 0) { return output::errors(ref stream, path); }
    var bytes::Buffer result = bytes::empty();
    let u32 status = output::tokens(ref stream, ref mut result);
    if (status != 0) { return error(if (status == 4) { "error: output limit (1048576 bytes)\n" }
        else if (status == 5) { "error: invalid token coverage\n" } else { "error: buffer allocation failed\n" }); }
    let ref u8[] data = bytes::view(ref result);
    return if (switch(io::write_bytes(data)) { case .ok(count): { count == data.len } case .err(_): { false } }) { 0 }
        else { error("error: stdout write failed\n") };
}
