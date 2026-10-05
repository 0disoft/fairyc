module type_cli;
import scanner as scanner;
import syntax as syntax;
import name_resolution as names;
import name_cli as name_output;
import type_check as typing;
import type_state as state;
import declaration_output as declarations;
import token_output as output;
import std::io as io;

@effects(console)
pub fn i32 run(scanner::Stream stream, ref u8[] path) {
    let syntax::Parsed parsed = switch(syntax::parse(stream,512)) {
        case .ok(value): { value }
        case .err(code): {
            discard io::write_stderr(if (code == 4) { "error: parser limit\n" }
                else if (code == 3) { "error: buffer allocation failed\n" } else { "error: parser failed\n" }); return 2;
        }
    };
    if (syntax::problem_count(ref parsed) != 0) { return declarations::errors(ref parsed,path); }
    let names::Resolved resolved = names::resolve(parsed);
    if (names::problem(ref resolved).code != 0) { return name_output::errors(ref resolved,path); }
    let typing::Typed typed = typing::check(resolved); let state::Problem problem = typing::problem(ref typed);
    if (problem.code != 0) {
        output::prefix(typing::source_view(ref typed),path,problem.start);
        discard io::write_stderr(if (problem.code == 1) { "type mismatch" } else if (problem.code == 2) { "integer literal out of range" }
            else if (problem.code == 3) { "expected bool operand" } else if (problem.code == 4) { "assignment target is immutable" }
            else if (problem.code == 5) { "call argument count mismatch" } else if (problem.code == 6) { "expression is not callable" }
            else if (problem.code == 7) { "missing return value" } else if (problem.code == 8) { "non-scalar binding value" }
            else if (problem.code == 9) { "invalid scalar operator" } else if (problem.code == 10) { "type-check limit" } else if (problem.code == 12) { "buffer allocation failed" } else { "unsupported type-check syntax" });
        output::range(problem.start,problem.end); return 2;
    }
    return if (io::write("types ok\n").is_ok) { 0 } else { discard io::write_stderr("error: stdout write failed\n"); 2 };
}
