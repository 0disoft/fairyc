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
    let typing::Typed typed = typing::check(resolved);
    if (typing::problem(ref typed).code != 0) { return errors(ref typed,path); }
    return if (io::write("types ok\n").is_ok) { 0 } else { discard io::write_stderr("error: stdout write failed\n"); 2 };
}

@effects(console)
pub fn i32 errors(ref typing::Typed typed, ref u8[] path) {
    let state::Problem problem = typing::problem(typed);
    if (problem.code != 0) {
        output::prefix(typing::source_view(typed),path,problem.start);
        if(problem.code==4201 || problem.code==4202){discard io::write_stderr(if(problem.code==4201){"E4201 invalid must_use result"}else{"E4202 ignored must_use call"});output::range(problem.start,problem.end);if(problem.move_end!=0){discard io::write_stderr("  must-use-declaration: ");output::range(problem.move_start,problem.move_end);}return 2;}
        if(problem.code==5101){discard io::write_stderr("E5101 call requires an unsafe boundary");output::range(problem.start,problem.end);return 2;}
        if(problem.code==7102){discard io::write_stderr("E7102 caller lacks required effects");output::range(problem.start,problem.end);return 2;}
        if(problem.code>=49 && problem.code<=51){discard io::write_stderr(if(problem.code==49){"E1107 invalid resource abilities"}else if(problem.code==50){"E1108 implicit destruction of no-drop resource"}else{"E1109 unauthorized burn"});output::range(problem.start,problem.end);if(problem.move_end!=0){discard io::write_stderr("  ability-declaration: ");output::range(problem.move_start,problem.move_end);}return 2;}
        if(problem.code==48){discard io::write_stderr("E1102 cleanup receiver cannot transfer or replace its obligation");output::range(problem.start,problem.end);return 2;}
        if(problem.code==45 || problem.code==46 || problem.code==47){discard io::write_stderr(if(problem.code==47){"E1105 incompatible resource state at loop edge"}else if(problem.code==46){"E1104 incompatible resource states at branch join"}else{"E1101 resource used after move"});output::range(problem.start,problem.end);
            if(problem.loop_end!=0){discard io::write_stderr("  loop-header: ");output::prefix(typing::source_view(typed),path,problem.loop_start);output::range(problem.loop_start,problem.loop_end);}
            if(problem.move_end!=0){discard io::write_stderr("  move: ");output::prefix(typing::source_view(typed),path,problem.move_start);output::range(problem.move_start,problem.move_end);}return 2;}
        if(problem.code==43){discard io::write_stderr("resource-bearing inert struct or array field");output::range(problem.start,problem.end);return 2;}
        if(problem.code==42){discard io::write_stderr("Result value must be handled");output::range(problem.start,problem.end);return 2;}
        if (problem.code == 40 || problem.code == 41) {
            discard io::write_stderr(if (problem.code == 40) { "? requires Result operand" }
                else { "? requires enclosing Result with identical error type" });
            output::range(problem.start,problem.end); return 2;
        }
        if (problem.code >= 36 && problem.code <= 39) {
            discard io::write_stderr(if (problem.code == 36) { "Result context required" }
                else if (problem.code == 37) { "invalid Result construction" }
                else if (problem.code == 38) { "invalid Result pattern" } else { "non-exhaustive Result switch" });
            output::range(problem.start,problem.end); return 2;
        }
        discard io::write_stderr(if(problem.code==52){"unsupported explicit cast"}else if (problem.code == 1) { "type mismatch" } else if (problem.code == 2) { "integer literal out of range" }
            else if (problem.code == 3) { "expected bool operand" } else if (problem.code == 4) { "assignment target is immutable" }
            else if (problem.code == 5) { "call argument count mismatch" } else if (problem.code == 6) { "expression is not callable" }
            else if (problem.code == 7) { "missing return value" } else if (problem.code == 8) { "non-scalar binding value" }
            else if (problem.code == 9) { "invalid scalar operator" } else if (problem.code == 10) { "type-check limit" } else if (problem.code == 12) { "buffer allocation failed" } else if (problem.code == 13) { "loop control outside loop" } else if (problem.code == 14) { "unsupported for header" } else if (problem.code == 15) { "unsupported defer call" } else if (problem.code == 16) { "invalid nominal type" } else if (problem.code == 17) { "duplicate field" } else if (problem.code == 18) { "unknown field" } else if (problem.code == 19) { "missing struct fields" } else if (problem.code == 20) { "recursive value type" } else if (problem.code == 21) { "text indexing and slicing are not supported" } else if (problem.code == 22) { "duplicate enum member" } else if (problem.code == 23) { "expected nonempty enum (1..128 members)" } else if (problem.code == 24) { "enum context required" } else if (problem.code == 25) { "unknown enum member" } else if (problem.code == 26) { "duplicate switch case" } else if (problem.code == 27) { "non-exhaustive enum switch" } else if (problem.code == 28) { "unsupported enum pattern" } else if (problem.code == 29) { "enum defer arguments are not supported" } else if (problem.code == 30) { "enum payload requires 1..16 integer or bool fields" } else if (problem.code == 31) { "enum payload fields must match exactly" } else if(problem.code==44){"E1103 partial resource field move"} else if(problem.code==32){"Option context required"} else if(problem.code==33){"invalid Option construction"} else if(problem.code==34){"invalid Option pattern"} else if(problem.code==35){"non-exhaustive Option switch"} else { "unsupported type-check syntax" });
        output::range(problem.start,problem.end); return 2;
    }
    return 0;
}
