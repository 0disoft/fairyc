module file_cli;
import candidate_limits as limits;
import public_cli_data as public;
import project_json_string as quoted;
import file_source as files;
import file_tokens as tokens;
import scanner as scanner;
import declaration_output as declarations;
import name_cli as names;
import type_cli as typing;
import module_cli as modules;
import module_project_cli as projects;
import project_check_cli as checking;
import project_emit_cli as workspace_c;
import native_cli as native;
import hir_cli as hir;
import mir_cli as mir;
import c_cli as c;
import token_output as output;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

@effects(console)
fn i32 error(ref str message) { discard io::write_stderr(message); return 2; }
fn ref str usage() { return "usage: lexer-candidate check [--locked] [--cache-root <existing directory>] [--cache-status]\n       lexer-candidate build [--release] [--locked] [--cache-root <existing directory>] [--cache-status]\n       lexer-candidate run [--release] [--locked] [--cache-root <existing directory>] [--cache-status] [-- <program arguments>]\n       lexer-candidate --tokens-file <path>\n       lexer-candidate --declarations-file <path>\n       lexer-candidate --check-names-file <path>\n       lexer-candidate --bindings-file <path>\n       lexer-candidate --check-types-file <path>\n       lexer-candidate --check-modules-file <manifest>\n       lexer-candidate --check-return-origins-file <manifest>\n       lexer-candidate --c-modules-file <manifest>\n       lexer-candidate --c-project-file <manifest>\n       lexer-candidate --c-workspace <root> [--locked]\n       lexer-candidate --hir-file <path>\n       lexer-candidate --mir-file <path>\n       lexer-candidate --c-file <path>\n"; }

@effects(console)
fn i32 invocation(ref str message){if(!io::write_stderr("fairyc: ").is_ok || !io::write_stderr(message).is_ok || !io::write_stderr("\n\n").is_ok || !io::write_stderr(public::help()).is_ok){return 4;}return 2;}
@effects(console)
fn i32 public_output(ref str message){return if(io::write(message).is_ok){0}else{4};}
fn bool is_help(ref u8[] flag){return bytes::equal(flag,b"--help") || bytes::equal(flag,b"-h") || bytes::equal(flag,b"help");}
@effects(console)
fn i32 quoted_error(ref str prefix,ref u8[] name,ref str suffix){var usize work=0;let spelling=switch(quoted::quote(name,ref mut work)){case .ok(p):{p}case .err(_):{return 4;}};if(!io::write_stderr("fairyc: ").is_ok || !io::write_stderr(prefix).is_ok || !io::write_stderr_bytes(bytes::view(ref spelling)).is_ok || !io::write_stderr(suffix).is_ok || !io::write_stderr("\n\n").is_ok || !io::write_stderr(public::help()).is_ok){return 4;}return 2;}

@effects(filesystem)
pub fn Result<bytes::Buffer, u32> load(ref u8[] path_bytes) {
    return files::load(path_bytes);
}

@effects(filesystem, console, environment, clock)
pub fn i32 run() {
    let usize argc = process::arg_count();
    if (argc < 2) { return invocation("a command is required"); }
    var bytes::Buffer argument = bytes::empty();
    if (!bytes::resize_zeroed(ref mut argument, 4096).is_ok) { return error("error: buffer allocation failed\n"); }
    let usize flag_length = switch(process::arg(1, bytes::view_mut(ref mut argument))) {
        case .ok(value): { value } case .err(_): { return error(usage()); }
    };
    let ref u8[] flag = bytes::view(ref argument)[..flag_length];
    if(argc==2 && is_help(flag)){return public_output(public::help());}
    if(argc==3){var u8[4096] other=[0;4096];let n=switch(process::arg(2,ref mut other)){case .ok(p):{p}case .err(_):{return invocation("argument exceeds the supported byte limit");}};let ref u8[] value=ref other;
        if(is_help(value[..n]) && (bytes::equal(flag,b"check") || bytes::equal(flag,b"build") || bytes::equal(flag,b"run") || bytes::equal(flag,b"new") || bytes::equal(flag,b"test") || bytes::equal(flag,b"explain") || bytes::equal(flag,b"targets") || bytes::equal(flag,b"experimental"))){return public_output(public::help());}
        if(bytes::equal(flag,b"explain")){let code=value[..n];if(code.len!=5 || (code[0]!=69 && code[0]!=65 && code[0]!=86) || code[1]<48 || code[1]>57 || code[2]<48 || code[2]>57 || code[3]<48 || code[3]>57 || code[4]<48 || code[4]>57){return quoted_error("diagnostic code ",code," is not in canonical E0001, A2001, or V1001 form");}let message=public::explanation(code);if(message.len==0){return quoted_error("unknown diagnostic code ",code,"");}return public_output(message);}
    }
    if(bytes::equal(flag,b"--version")){return if(argc==2){public_output(public::version())}else{invocation("--version does not accept arguments")};}
    if(bytes::equal(flag,b"targets")){return if(argc==2){public_output(public::targets())}else{invocation("command \"targets\" does not accept arguments")};}
    if(bytes::equal(flag,b"explain")){return invocation("command \"explain\" expects exactly one diagnostic code");}
    if(bytes::equal(flag,b"emit")){return invocation("command \"emit\" is not a stable product command; for compiler debugging use \"fairyc experimental emit-workspace-c --target <target-id> [--locked]\"");}
    if (bytes::equal(flag,b"check")) { return checking::arguments(); }
    if (bytes::equal(flag,b"build")) { return native::run(false); }
    if (bytes::equal(flag,b"run")) { return native::run(true); }
    if (bytes::equal(flag,b"--c-workspace")) { return workspace_c::arguments(); }
    if(flag.len<2 || flag[0]!=45 || flag[1]!=45){return quoted_error("unknown command ",flag,"");}
    if (argc != 2 && argc != 3) { return error(usage()); }
    if (argc == 2 && bytes::equal(flag, b"--help")) {
        return if (io::write(usage()).is_ok && io::write("       lexer-candidate --help\nNo arguments: private binary stdin protocol.\n").is_ok) { 0 }
            else { error("error: stdout write failed\n") };
    }
    let bool token_mode = bytes::equal(flag,b"--tokens-file");
    let bool binding_mode = bytes::equal(flag,b"--bindings-file");
    let bool type_mode = bytes::equal(flag,b"--check-types-file");
    let bool module_mode = bytes::equal(flag,b"--check-modules-file");
    let bool origin_mode = bytes::equal(flag,b"--check-return-origins-file");
    let bool program_mode = bytes::equal(flag,b"--c-modules-file");
    let bool project_mode = bytes::equal(flag,b"--c-project-file");
    let bool hir_mode = bytes::equal(flag,b"--hir-file");
    let bool mir_mode = bytes::equal(flag,b"--mir-file");
    let bool c_mode = bytes::equal(flag,b"--c-file");
    let bool name_mode = bytes::equal(flag, b"--check-names-file");
    let bool declaration_mode = bytes::equal(flag, b"--declarations-file");
    if (argc != 3 || (!bytes::equal(flag, b"--tokens-file") && !declaration_mode && !name_mode && !binding_mode && !type_mode && !module_mode && !origin_mode && !program_mode && !project_mode && !hir_mode && !mir_mode && !c_mode)) { return quoted_error("unknown command ",flag,""); }
    let usize length = switch(process::arg(2, bytes::view_mut(ref mut argument))) {
        case .ok(value): { value } case .err(_): { return error("error: file path limit (4095 UTF-8 bytes)\n"); }
    };
    if (length > 4095) { return error("error: file path limit (4095 UTF-8 bytes)\n"); }
    let ref u8[] path = bytes::view(ref argument)[..length];
    if (token_mode) { return tokens::run(path); }
    if (module_mode) { return modules::run(path); }
    if (origin_mode) { return modules::check_return_origins(path); }
    if (program_mode) { return modules::emit(path); }
    if (project_mode) { return projects::run(path); }
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
