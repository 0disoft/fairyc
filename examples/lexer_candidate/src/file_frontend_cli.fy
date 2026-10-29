module file_frontend_cli;
import candidate_limits as limits;
import scanner as scanner;
import declaration_output as declarations;
import name_cli as names;
import type_cli as typing;
import hir_cli as hir;
import mir_cli as mir;
import c_cli as c;
import file_tokens as tokens;
import token_output as output;
import std::bytes as bytes;
import std::io as io;

@effects(console)
fn i32 error(ref str message) { discard io::write_stderr(message); return 2; }

pub fn u32 mode(ref u8[] flag) {
    return if(bytes::equal(flag,b"--tokens-file")){1}else if(bytes::equal(flag,b"--declarations-file")){2}
        else if(bytes::equal(flag,b"--check-names-file")){3}else if(bytes::equal(flag,b"--bindings-file")){4}
        else if(bytes::equal(flag,b"--check-types-file")){5}else if(bytes::equal(flag,b"--hir-file")){6}
        else if(bytes::equal(flag,b"--mir-file")){7}else if(bytes::equal(flag,b"--c-file")){8}else{0};
}
// Shared frontend after host-owned file loading; no target host calls.
@effects(console)
pub fn i32 run(bytes::Buffer input,ref u8[] path,u32 selected) {
    if(selected==1) { return tokens::run_input(input,path); }
    let bool declaration_mode=selected==2;
    let bool name_mode=selected==3;
    let bool binding_mode=selected==4;
    let bool type_mode=selected==5;
    let bool hir_mode=selected==6;
    let bool mir_mode=selected==7;
    let bool c_mode=selected==8;
    let scanner::Stream stream = switch(scanner::collect(input, false, limits::source_bytes(), 1048576, 256, 16777216)) {
        case .ok(value): { value }
        case .err(code): { return error(if (code == 4) { "error: lexer storage limit\n" }
            else if (code == 2) { "error: unsupported lexer input\n" } else { "error: buffer allocation failed\n" }); }
    };
    if (scanner::diagnostic_count(ref stream) > 0) { return output::errors(ref stream, path); }
    if (hir_mode) { return hir::run(stream,path); }
    if (mir_mode) { return mir::run(stream,path); }
    if (c_mode) { return c::run(stream,path); }
    if (type_mode) { return typing::run(stream,path); }
    if (name_mode || binding_mode) { return names::run(stream, path, binding_mode); }
    if (declaration_mode) { return declarations::run(stream, path); }
    return error("error: invalid file mode\n");
}
