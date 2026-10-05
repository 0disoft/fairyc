module c_cli;
import mir_cli as mir;
import mir_model as m;
import c_emit as emit;
import scanner as scanner;
import std::bytes as bytes;
import std::io as io;

@effects(console)
pub fn i32 run(scanner::Stream stream, ref u8[] path) {
    let m::Program program = switch(mir::compile(stream,path)) { case .ok(value): { value } case .err(code): { return code; } };
    var bytes::Buffer out = bytes::empty(); let u32 code = emit::render(ref program,ref mut out);
    if (code != 0) {
        discard io::write_stderr(if (code == 1) { "error: C entry requires fn i32 main()\n" } else if (code == 4) { "error: C output limit (2097152 bytes)\n" }
            else if (code == 3) { "error: buffer allocation failed\n" } else if (code == 2) { "error: C verification limit\n" } else { "error: invalid C input\n" }); return 2;
    }
    let ref u8[] data = bytes::view(ref out);
    if (switch(io::write_bytes(data)) { case .ok(count): { count == data.len } case .err(_): { false } }) { return 0; }
    discard io::write_stderr("error: stdout write failed\n"); return 2;
}
