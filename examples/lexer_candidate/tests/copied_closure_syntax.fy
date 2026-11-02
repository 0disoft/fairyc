module tests::copied_closure_syntax;
import scanner as scanner;
import syntax as syntax;
import generic_source_function_json as functions;
import std::bytes as bytes;
import std::io as io;
fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
@effects(console) pub fn void emit(ref u8[] raw){let p=parsed(raw);var usize work=0;let before=switch(functions::encode(ref p,1,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let copied=switch(syntax::copy(ref p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let after=switch(functions::encode(ref copied,1,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref before),bytes::view(ref after)));assert(io::write_bytes(bytes::view(ref after)).is_ok && io::write("\n").is_ok);}
