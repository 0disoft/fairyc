module mir_cli;
import scanner as scanner;
import syntax as syntax;
import name_resolution as names;
import name_cli as name_output;
import type_check as typing;
import type_cli as type_output;
import hir_model as h;
import hir_lower as hir;
import mir_model as m;
import mir_lower as lower;
import mir_output as output;
import declaration_output as declarations;
import std::bytes as bytes;
import std::io as io;

@effects(console)
fn void verification_error(u32 code) {
    discard io::write_stderr(if (code == 1001) { "error: MIR verification V1001\n" } else if (code == 1002) { "error: MIR verification V1002\n" }
        else if (code == 1501) { "error: MIR verification V1501\n" } else if (code == 2) { "error: MIR limit\n" }
        else if (code == 3) { "error: buffer allocation failed\n" } else { "error: invalid or unsupported MIR input\n" });
}

@effects(console)
pub fn i32 run(scanner::Stream stream, ref u8[] path) {
    let syntax::Parsed parsed = switch(syntax::parse(stream,512)) {
        case .ok(value): { value }
        case .err(code): { discard io::write_stderr(if (code == 4) { "error: parser limit\n" } else if (code == 3) { "error: buffer allocation failed\n" } else { "error: parser failed\n" }); return 2; }
    };
    if (syntax::problem_count(ref parsed) != 0) { return declarations::errors(ref parsed,path); }
    let names::Resolved resolved = names::resolve(parsed);
    if (names::problem(ref resolved).code != 0) { return name_output::errors(ref resolved,path); }
    let typing::Typed typed = typing::check(resolved);
    if (typing::problem(ref typed).code != 0) { return type_output::errors(ref typed,path); }
    let h::Program input = switch(hir::lower(typed,0)) { case .ok(value): { value } case .err(_): { discard io::write_stderr("error: HIR lowering failed\n"); return 2; } };
    let m::Program program = switch(lower::lower(input)) {
        case .ok(value): { value }
        case .err(problem): { verification_error(problem.code); return 2; }
    };
    var bytes::Buffer rendered = bytes::empty(); let u32 code = output::render(ref program,ref mut rendered);
    if (code != 0) { if (code == 4) { discard io::write_stderr("error: MIR output limit (2097152 bytes)\n"); } else { verification_error(code); } return 2; }
    let ref u8[] data = bytes::view(ref rendered);
    return if (switch(io::write_bytes(data)) { case .ok(count): { count == data.len } case .err(_): { false } }) { 0 } else { discard io::write_stderr("error: stdout write failed\n"); 2 };
}
