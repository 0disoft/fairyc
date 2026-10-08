module hir_cli;
import scanner as scanner;
import type_check as typing;
import type_cli as type_output;
import hir_model as model;
import hir_lower as lower;
import hir_output as output;
import std::bytes as bytes;
import std::io as io;
import token_output as console;

@effects(console)
pub fn Result<model::Program,i32> compile(scanner::Stream stream,ref u8[] path,bool legacy){
    let typing::Typed typed=switch(type_output::compile(stream,path)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let model::Program program = switch(lower::lower(typed,0)) {
        case .ok(value): { value }
        case .err(problem): {
            discard io::write_stderr("HIR bytes ");discard io::write_stderr_usize(problem.start);discard io::write_stderr("..");discard io::write_stderr_usize(problem.end);discard io::write_stderr("\n");
            discard io::write_stderr(if(problem.code==5106){"error: HIR E5106 conflicting C linker signature\n"}else if(problem.code==5102){"error: HIR E5102 unsupported C ABI\n"}else if (problem.code == 2101) { "error: HIR E2101 active byte-view loan\n" } else if (problem.code == 2102) { "error: HIR E2102 local view escape or mutable descriptor\n" } else if (problem.code == 2103) { "error: HIR E2103 ambiguous or recursive borrow return origin\n" } else if (legacy) { "error: HIR lowering failed\n" } else if (problem.code == 2) { "error: HIR limit\n" }
                else if (problem.code == 3) { "error: buffer allocation failed\n" }
                else { "error: invalid or unsupported HIR input\n" }); return .err(2);
        }
    };
    return .ok(program);
}

@effects(console)
pub fn i32 run(scanner::Stream stream, ref u8[] path) {
    let model::Program program=switch(compile(stream,path,false)){case .ok(p):{p}case .err(code):{return code;}};
    var bytes::Buffer rendered = bytes::empty();
    let u32 code = output::render(ref program,ref mut rendered);
    if (code != 0) { discard io::write_stderr(if (code == 4) { "error: HIR output limit (2097152 bytes)\n" } else { "error: buffer allocation failed\n" }); return 2; }
    let ref u8[] data = bytes::view(ref rendered);
    return console::write(data);
}
