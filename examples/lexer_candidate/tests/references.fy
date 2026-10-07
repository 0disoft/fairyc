module tests::references;
import scanner as scanner;
import syntax as syntax;
import name_resolution as names;
import type_check as typing;
import hir_lower as lower;
import std::bytes as bytes;
import std::io as io;

@test @effects(console) fn void mutable_reference_argument_keeps_one_generation(){
    var bytes::Buffer input=bytes::empty();assert(bytes::append(ref mut input,b"module main;fn void mark(i32 n){}fn i32 read(ref mut i32 p){return *p;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;return read(p);}").is_ok);
    let stream=switch(scanner::collect(input,false,32768,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let p=switch(lower::lower(typing::check(names::resolve(parsed)),0)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);discard io::write_stderr_usize(e.start);discard io::write_stderr_usize(e.end);assert(false);return;}};
}
