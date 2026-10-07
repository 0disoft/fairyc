module module_check;
import module_registry as modules;
import name_resolution as names;
import name_cli as name_output;
import type_check as typing;
import type_cli as type_output;
import syntax as syntax;
import token_output as output;
import std::io as io;

@effects(console)
fn void message(u32 code) {
    discard io::write_stderr(if(code==7101){"E7101 invalid effects declaration"}else if(code==4201){"E4201 invalid must_use declaration"}else if (code == 1) { "duplicate module" } else if (code == 2) { "duplicate name in scope" }
        else if (code == 3 || code == 4) { "unresolved import" }
        else if (code == 5) { "unresolved imported item" } else if (code == 6) { "private imported item" }
        else if (code == 7) { "duplicate import alias" }
        else if (code == 10) { "module-check limit" } else if (code == 11) { "buffer allocation failed" }
        else { "unsupported module-check syntax" });
}
@effects(console)
pub fn i32 error(ref syntax::Parsed parsed, ref u8[] path, modules::Problem problem) {
    output::prefix(syntax::source_view(parsed),path,problem.start);
    message(problem.code); output::range(problem.start,problem.end); return 2;
}
// Every file, including imported/private function bodies, reaches the existing
// resolver and type walker. Import signatures affect ordinary expression typing.
@effects(console)
pub fn i32 run(syntax::Parsed parsed, ref modules::Registry registry, usize module_index, ref u8[] path) {
    let names::Resolved resolved = names::resolve_modules(parsed,registry,module_index);
    let names::Problem name_problem = names::problem(ref resolved);
    if (name_problem.code != 0) {
        if (name_problem.code < 7 || name_problem.code==4201 || name_problem.code==7101) { return name_output::errors(ref resolved,path); }
        output::prefix(names::source_view(ref resolved),path,name_problem.start);
        message(if (name_problem.code == 7) { 4 } else if (name_problem.code == 8) { 5 } else if (name_problem.code == 9) { 6 } else { 9 });
        output::range(name_problem.start,name_problem.end); return 2;
    }
    let typing::Typed typed = typing::check(resolved);
    return type_output::errors(ref typed,path);
}
