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
    discard io::write_stderr(if(code==5106){"error: MIR E5106 invalid C linker signature\n"}else if(code==5102){"error: MIR E5102 unsupported scalar C ABI\n"}else if(code==5101){"error: MIR E5101 missing unsafe call provenance\n"}else if(code==7102){"error: E7102 cleanup requires declared effects\n"}else if (code == 1001) { "error: MIR verification V1001\n" } else if (code == 1002) { "error: MIR verification V1002\n" }
        else if (code == 1501) { "error: MIR verification V1501\n" } else if (code == 2) { "error: MIR limit\n" }
        else if (code == 3) { "error: buffer allocation failed\n" } else { "error: invalid or unsupported MIR input\n" });
}

@effects(console)
pub fn Result<m::Program,i32> compile(scanner::Stream stream, ref u8[] path) {
    let syntax::Parsed parsed = switch(syntax::parse(stream,512)) {
        case .ok(value): { value }
        case .err(code): { discard io::write_stderr(if (code == 4) { "error: parser limit\n" } else if (code == 3) { "error: buffer allocation failed\n" } else { "error: parser failed\n" }); return .err(2); }
    };
    if (syntax::problem_count(ref parsed) != 0) { return .err(declarations::errors(ref parsed,path)); }
    let names::Resolved resolved = names::resolve(parsed);
    if (names::problem(ref resolved).code != 0) { return .err(name_output::errors(ref resolved,path)); }
    let typing::Typed typed = typing::check(resolved);
    if (typing::problem(ref typed).code != 0) { return .err(type_output::errors(ref typed,path)); }
    let h::Program input = switch(hir::lower(typed,0)) { case .ok(value): { value } case .err(problem): { discard io::write_stderr("HIR bytes ");discard io::write_stderr_usize(problem.start);discard io::write_stderr("..");discard io::write_stderr_usize(problem.end);discard io::write_stderr("\n"); discard io::write_stderr(if(problem.code==5106){"error: HIR E5106 conflicting C linker signature\n"}else if(problem.code==5102){"error: HIR E5102 unsupported scalar C ABI\n"}else if(problem.code==2101){"error: HIR E2101 active byte-view loan\n"}else if(problem.code==2102){"error: HIR E2102 local view escape or mutable descriptor\n"}else if(problem.code==2103){"error: HIR E2103 ambiguous or recursive borrow return origin\n"}else{"error: HIR lowering failed\n"}); return .err(2); } };
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
    return if (switch(io::write_bytes(data)) { case .ok(count): { count == data.len } case .err(_): { false } }) { 0 } else { discard io::write_stderr("error: stdout write failed\n"); 2 };
}
