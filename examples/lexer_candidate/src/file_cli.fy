module file_cli;
import candidate_limits as limits;
import file_source as files;
import file_tokens as tokens;
import scanner as scanner;
import declaration_output as declarations;
import name_cli as names;
import type_cli as typing;
import module_cli as modules;
import hir_cli as hir;
import mir_cli as mir;
import c_cli as c;
import token_output as output;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

@effects(console)
fn i32 error(ref str message) { discard io::write_stderr(message); return 2; }
fn ref str usage() { return "usage: lexer-candidate --tokens-file <path>\n       lexer-candidate --declarations-file <path>\n       lexer-candidate --check-names-file <path>\n       lexer-candidate --bindings-file <path>\n       lexer-candidate --check-types-file <path>\n       lexer-candidate --check-modules-file <manifest>\n       lexer-candidate --c-modules-file <manifest>\n       lexer-candidate --hir-file <path>\n       lexer-candidate --mir-file <path>\n       lexer-candidate --c-file <path>\n"; }

@effects(filesystem)
pub fn Result<bytes::Buffer, u32> load(ref u8[] path_bytes) {
    return files::load(path_bytes);
}

@effects(filesystem, console)
pub fn i32 run() {
    let usize argc = process::arg_count();
    if (argc != 2 && argc != 3) { return error(usage()); }
    var bytes::Buffer argument = bytes::empty();
    if (!bytes::resize_zeroed(ref mut argument, 4096).is_ok) { return error("error: buffer allocation failed\n"); }
    let usize flag_length = switch(process::arg(1, bytes::view_mut(ref mut argument))) {
        case .ok(value): { value } case .err(_): { return error(usage()); }
    };
    let ref u8[] flag = bytes::view(ref argument)[..flag_length];
    if (argc == 2 && bytes::equal(flag, b"--help")) {
        return if (io::write(usage()).is_ok && io::write("       lexer-candidate --help\nNo arguments: private binary stdin protocol.\n").is_ok) { 0 }
            else { error("error: stdout write failed\n") };
    }
    let bool token_mode = bytes::equal(flag,b"--tokens-file");
    let bool binding_mode = bytes::equal(flag,b"--bindings-file");
    let bool type_mode = bytes::equal(flag,b"--check-types-file");
    let bool module_mode = bytes::equal(flag,b"--check-modules-file");
    let bool program_mode = bytes::equal(flag,b"--c-modules-file");
    let bool hir_mode = bytes::equal(flag,b"--hir-file");
    let bool mir_mode = bytes::equal(flag,b"--mir-file");
    let bool c_mode = bytes::equal(flag,b"--c-file");
    let bool name_mode = bytes::equal(flag, b"--check-names-file");
    let bool declaration_mode = bytes::equal(flag, b"--declarations-file");
    if (argc != 3 || (!bytes::equal(flag, b"--tokens-file") && !declaration_mode && !name_mode && !binding_mode && !type_mode && !module_mode && !program_mode && !hir_mode && !mir_mode && !c_mode)) { return error(usage()); }
    let usize length = switch(process::arg(2, bytes::view_mut(ref mut argument))) {
        case .ok(value): { value } case .err(_): { return error("error: file path limit (4095 UTF-8 bytes)\n"); }
    };
    if (length > 4095) { return error("error: file path limit (4095 UTF-8 bytes)\n"); }
    let ref u8[] path = bytes::view(ref argument)[..length];
    if (token_mode) { return tokens::run(path); }
    if (module_mode) { return modules::run(path); }
    if (program_mode) { return modules::emit(path); }
    let bytes::Buffer input = switch(load(path)) {
        case .ok(value): { value }
        case .err(code): { return error(if (code == 4) { "error: source limit (65536 bytes)\n" }
            else if (code == 3) { "error: buffer allocation failed\n" } else { "error: input file read failed\n" }); }
    };
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
