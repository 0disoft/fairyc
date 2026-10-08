module name_cli;
import scanner as scanner;
import syntax as syntax;
import name_resolution as names;
import declaration_output as declarations;
import token_output as output;
import binding_output as bindings;
import std::io as io;

@effects(console)
pub fn Result<names::Resolved,i32> compile(scanner::Stream stream,ref u8[] path){
    let syntax::Parsed parsed=switch(declarations::compile(stream,path)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let names::Resolved resolved=names::resolve(parsed);
    if(names::problem(ref resolved).code!=0){return .err(errors(ref resolved,path));}
    return .ok(resolved);
}

@effects(console)
pub fn i32 run(scanner::Stream stream, ref u8[] path, bool dump) {
    let names::Resolved resolved=switch(compile(stream,path)){case .ok(p):{p}case .err(code):{return code;}};
    if (dump) { return bindings::run(ref resolved); }
    return if (io::write("names ok\n").is_ok) { 0 } else { discard io::write_stderr("error: stdout write failed\n"); 2 };
}

@effects(console)
pub fn i32 errors(ref names::Resolved resolved, ref u8[] path) {
    let names::Problem problem = names::problem(resolved);
    if (problem.code != 0) {
        output::prefix(names::source_view(resolved),path,problem.start);
        discard io::write_stderr(if(problem.code==5102){"E5102 unsupported C ABI type"}else if(problem.code==5106){"E5106 invalid C linker symbol"}else if(problem.code==4201){"E4201 invalid must_use declaration"}else if(problem.code==7101){"E7101 invalid effects declaration"}else if(problem.code==4501){"E4501 invalid explicit generic request"}else if(problem.code==4502){"E4502 unsupported generic scalar"}else if(problem.code==4503){"E4503 recursive specialization or expansion limit"}else if (problem.code == 1) { "unresolved name" }
            else if (problem.code == 2) { "duplicate name in scope" }
            else if (problem.code == 3) { "unsupported name-check syntax" }
            else if (problem.code == 5) { "buffer allocation failed" }
            else if (problem.code == 6) { "invalid parsed source" } else { "name-check limit" });
        output::range(problem.start,problem.end); return 2;
    }
    return 0;
}
