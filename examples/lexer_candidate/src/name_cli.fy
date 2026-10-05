module name_cli;
import scanner as scanner;
import syntax as syntax;
import name_resolution as names;
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
    let names::Problem problem = names::check(ref parsed);
    if (problem.code != 0) {
        output::prefix(syntax::source_view(ref parsed),path,problem.start);
        discard io::write_stderr(if (problem.code == 1) { "unresolved name" }
            else if (problem.code == 2) { "duplicate name in scope" }
            else if (problem.code == 3) { "unsupported name-check syntax" } else { "name-check limit" });
        output::range(problem.start,problem.end); return 2;
    }
    return if (io::write("names ok\n").is_ok) { 0 } else { discard io::write_stderr("error: stdout write failed\n"); 2 };
}
