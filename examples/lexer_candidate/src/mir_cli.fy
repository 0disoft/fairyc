module mir_cli;
import scanner as scanner;
import hir_model as h;
import hir_cli as hir;
import mir_model as m;
import mir_lower as lower;
import mir_output as output;
import std::bytes as bytes;
import std::io as io;
import token_output as console;

@effects(console)
fn void verification_error(u32 code) {
    discard io::write_stderr(if(code==5106){"error: MIR E5106 invalid C linker signature\n"}else if(code==5102){"error: MIR E5102 unsupported C ABI\n"}else if(code==5101){"error: MIR E5101 missing unsafe call provenance\n"}else if(code==7102){"error: E7102 cleanup requires declared effects\n"}else if (code == 1001) { "error: MIR verification V1001\n" } else if (code == 1002) { "error: MIR verification V1002\n" }
        else if (code == 1501) { "error: MIR verification V1501\n" } else if (code == 2) { "error: MIR limit\n" }
        else if (code == 3) { "error: buffer allocation failed\n" } else { "error: invalid or unsupported MIR input\n" });
}

@effects(console)
pub fn Result<m::Program,i32> compile(scanner::Stream stream, ref u8[] path) {
    let h::Program input=switch(hir::compile(stream,path,true)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let m::Program program = switch(lower::lower(input)) {
        case .ok(value): { value }
        case .err(problem): { verification_error(problem.code); return .err(2); }
    };
    return .ok(program);
}

@effects(console)
pub fn i32 run(scanner::Stream stream, ref u8[] path) {
    let m::Program program = switch(compile(stream,path)) { case .ok(value): { value } case .err(code): { return code; } };
    var bytes::Buffer rendered = bytes::empty(); let u32 code = output::render(ref program,ref mut rendered);
    if (code != 0) { if (code == 4) { discard io::write_stderr("error: MIR output limit (2097152 bytes)\n"); } else { verification_error(code); } return 2; }
    let ref u8[] data = bytes::view(ref rendered);
    return console::write(data);
}
