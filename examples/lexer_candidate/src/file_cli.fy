module file_cli;
import project_manifest_values as sc_values;
import project_sdk_bundle as sc_sdk;
import windows_host_paths as sc_host;
import windows_command as sc_command;
import std::fs as sc_fs;
import std::text as sc_text;
import std::vec as sc_vec;
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
fn i32 check_status(i32 code){if(code!=2){return code;}if(!io::write_stderr("\n").is_ok || !io::write_stderr(public::help()).is_ok){return 4;}return 2;}
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
    if(bytes::equal(flag,b"new")){return scaffold();}
    if(argc==2 && is_help(flag)){return public_output(public::help());}
    if(argc==3){var u8[4096] other=[0;4096];let n=switch(process::arg(2,ref mut other)){case .ok(p):{p}case .err(_):{return invocation("argument exceeds the supported byte limit");}};let ref u8[] value=ref other;
        if(is_help(value[..n]) && (bytes::equal(flag,b"check") || bytes::equal(flag,b"build") || bytes::equal(flag,b"run") || bytes::equal(flag,b"new") || bytes::equal(flag,b"test") || bytes::equal(flag,b"explain") || bytes::equal(flag,b"targets") || bytes::equal(flag,b"experimental"))){return public_output(public::help());}
        if(bytes::equal(flag,b"explain")){let code=value[..n];if(code.len!=5 || (code[0]!=69 && code[0]!=65 && code[0]!=86) || code[1]<48 || code[1]>57 || code[2]<48 || code[2]>57 || code[3]<48 || code[3]>57 || code[4]<48 || code[4]>57){return quoted_error("diagnostic code ",code," is not in canonical E0001, A2001, or V1001 form");}let message=public::explanation(code);if(message.len==0){return quoted_error("unknown diagnostic code ",code,"");}return public_output(message);}
    }
    if(bytes::equal(flag,b"--version")){return if(argc==2){public_output(public::version())}else{invocation("--version does not accept arguments")};}
    if(bytes::equal(flag,b"targets")){return if(argc==2){public_output(public::targets())}else{invocation("command \"targets\" does not accept arguments")};}
    if(bytes::equal(flag,b"explain")){return invocation("command \"explain\" expects exactly one diagnostic code");}
    if(bytes::equal(flag,b"emit")){return invocation("command \"emit\" is not a stable product command; for compiler debugging use \"fairyc experimental emit-workspace-c --target <target-id> [--locked]\"");}
    if (bytes::equal(flag,b"check")) { if(argc==3){var u8[64] option=[0;64];let n=switch(process::arg(2,ref mut option)){case .ok(p):{p}case .err(_):{0}};let ref u8[] value=ref option;if(bytes::equal(value[..n],b"--cache-status")){return invocation("command \"check\" accepts only optional --locked");}}return check_status(checking::arguments()); }
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

// Pin each directory prefix without delete sharing. The SDK path APIs below
// then resolve through these retained, non-reparse directories. No new C glue.
@effects(filesystem) @link_name("CreateFileW") unsafe extern C fn usize scaffold_open_pin(ptr u8 path,u32 access,u32 share,ptr void security,u32 disposition,u32 flags,usize template_handle);
@effects(filesystem) @link_name("GetFileInformationByHandleEx") unsafe extern C fn i32 scaffold_pin_info(usize handle,u32 kind,ptr mut void out,u32 size);
@effects(filesystem) @link_name("GetFinalPathNameByHandleW") unsafe extern C fn u32 scaffold_final_path(usize handle,ptr mut u8 data,u32 count,u32 flags);
@effects(filesystem) @link_name("CloseHandle") unsafe extern C fn i32 scaffold_close_pin(usize handle);
@link_name("GetLastError") unsafe extern C fn u32 scaffold_last_error();
@repr(C) struct ScaffoldAttributes { u32 attributes; u32 tag; }
@repr(C) struct ScaffoldIdentity { usize volume; usize low; usize high; }
@effects(filesystem) resource ScaffoldPin {
    usize handle;
    cleanup { if(this.handle!=0){unsafe{discard scaffold_close_pin(this.handle);};} }
}
// Paths use slots 0 target, 1 src, 2 tests, then the five template files.
// kind: 1 parent, 2 existing target, 3 permission, 4 filesystem, 5 rollback,
// 8 allocation/internal. Native codes never reach product output.
struct ScaffoldProblem { u32 kind; usize slot; bool owns_file; }
fn ScaffoldProblem scaffold_problem(u32 kind,usize slot){return ScaffoldProblem{.kind,.slot,.owns_file=false};}
fn ScaffoldProblem scaffold_os(u32 code,usize slot,bool target){return scaffold_problem(if(target && (code==80 || code==183)){2}else if(code==5 || code==1314){3}else{4},slot);}
@effects(filesystem) fn bool scaffold_unpin(ref mut ScaffoldPin pin){let handle=pin.handle;pin.handle=0;return handle==0 || unsafe{scaffold_close_pin(handle)}!=0;}
fn Result<bytes::Buffer,ScaffoldProblem> scaffold_extended(ref u8[] path){
    var bytes::Buffer out=bytes::empty();var usize at=0;
    if(path.len>=4 && bytes::equal(path[..4],b"\\\\?\\")){}
    else if(path.len>=2 && path[0]==92 && path[1]==92){if(!bytes::append(ref mut out,b"\\\\?\\UNC\\").is_ok){return .err(scaffold_problem(8,0));}at=2;}
    else if(path.len>=3 && path[1]==58){if(!bytes::append(ref mut out,b"\\\\?\\").is_ok){return .err(scaffold_problem(8,0));}}
    while(at<path.len){if(!bytes::append_byte(ref mut out,if(path[at]==47){92}else{path[at]}).is_ok){return .err(scaffold_problem(8,0));}at+=1;}return .ok(out);
}
@effects(filesystem) fn Result<ScaffoldPin,ScaffoldProblem> scaffold_pin(ref u8[] path,usize slot){
    let extended=switch(scaffold_extended(path)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let wide=switch(sc_command::path(bytes::view(ref extended))){case .ok(p):{p}case .err(_):{return .err(scaffold_problem(8,slot));}};
    let handle=unsafe{scaffold_open_pin(bytes::view(ref wide) as ptr u8,128,1,null,3,35651584,0)};
    if(handle==18446744073709551615){return .err(scaffold_os(unsafe{scaffold_last_error()},slot,false));}
    var ScaffoldPin pin=ScaffoldPin{.handle};var ScaffoldAttributes info=ScaffoldAttributes{.attributes=0,.tag=0};
    if(unsafe{scaffold_pin_info(handle,9,ref mut info as ptr mut void,8)}==0){let code=unsafe{scaffold_last_error()};discard scaffold_unpin(ref mut pin);return .err(scaffold_os(code,slot,false));}
    if((info.attributes&16)==0 || (info.attributes&1024)!=0){discard scaffold_unpin(ref mut pin);return .err(scaffold_problem(4,slot));}
    return .ok(pin);
}
fn Result<bytes::Buffer,u32> scaffold_arg(usize index){
    var bytes::Buffer value=bytes::empty();
    let needed=switch(process::arg(index,bytes::view_mut(ref mut value))){case .ok(n):{n}case .err(e):{if(e.status!=3){return .err(1);}e.needed}};
    if(!bytes::resize_zeroed(ref mut value,needed).is_ok){return .err(1);}
    switch(process::arg(index,bytes::view_mut(ref mut value))){case .ok(n):{if(n!=needed){return .err(1);}}case .err(_):{return .err(1);}};
    return .ok(value);
}
@effects(console) fn bool scaffold_stderr(ref u8[] data){return switch(io::write_stderr_bytes(data)){case .ok(n):{n==data.len}case .err(_):{false}};}
@effects(console) fn i32 scaffold_output(ref u8[] data){
    if(switch(io::write_bytes(data)){case .ok(n):{n==data.len}case .err(_):{false}}){return 0;}
    discard io::write_stderr("fairyc: write output failed\n");return 4;
}
@effects(console) fn i32 scaffold_invocation(ref str message){
    if(!scaffold_stderr(b"fairyc: ") || !scaffold_stderr(sc_text::as_bytes(message)) || !scaffold_stderr(b"\n\n") || !scaffold_stderr(sc_text::as_bytes(public::help()))){return 4;}return 2;
}
fn bool scaffold_help(ref u8[] value){return bytes::equal(value,b"--help") || bytes::equal(value,b"-h") || bytes::equal(value,b"help");}
fn ref u8[] scaffold_leaf(usize slot){
    if(slot==1){return b"src";}if(slot==2){return b"tests";}if(slot==3){return b".gitignore";}if(slot==4){return b"README.md";}if(slot==5){return b"fairy.toml";}if(slot==6){return b"src/main.fy";}if(slot==7){return b"tests/main.fy";}return b"";
}
fn bool scaffold_append(ref mut bytes::Buffer out,ref u8[] data){return bytes::append(out,data).is_ok;}
fn Result<bytes::Buffer,ScaffoldProblem> scaffold_cstr(ref u8[] path,usize slot){
    var extended=switch(scaffold_extended(path)){case .ok(p):{p}case .err(e):{return .err(e);}};
    // Match the SDK's 4096-byte C path capacity, including its final NUL.
    // Validate before exposing any pointer; the Vec owns the moved allocation.
    {
        let view=bytes::view(ref extended);
        if(view.len==0 || view.len>4095){return .err(scaffold_problem(4,slot));}
        if(!sc_text::validate_utf8(view).is_ok){return .err(scaffold_problem(8,slot));}
        var usize at=0;while(at<view.len){if(view[at]==0){return .err(scaffold_problem(8,slot));}at+=1;}
    }
    if(!bytes::append_byte(ref mut extended,0).is_ok){return .err(scaffold_problem(8,slot));}return .ok(extended);
}
fn Result<bytes::Buffer,ScaffoldProblem> scaffold_body(usize index,ref u8[] name,ref u8[] uid){
    var bytes::Buffer out=bytes::empty();var bool ok=false;
    if(index==0){ok=scaffold_append(ref mut out,b"/build/\n");}
    if(index==1){ok=scaffold_append(ref mut out,b"# ") && scaffold_append(ref mut out,name) && scaffold_append(ref mut out,b"\n\nA Fairy C starter package.\n\n## Check and test\n\nRun these commands manually from this directory. The first check validates the workspace and\ncreates `fairy.lock`.\n\n```text\nfairyc check\nfairyc test --locked\n```\n\nThe default test command uses the fast semantic interpreter.\n\n## Run on Windows\n\nWith x64 MSVC and a Windows SDK installed:\n\n```text\nfairyc test --native --locked\nfairyc run --locked\nfairyc build --release --locked\n.\\build\\") && scaffold_append(ref mut out,name) && scaffold_append(ref mut out,b".exe\n```\n\n`fairyc test --native` compiles every test into a separate temporary Windows image.\n`fairyc run` forwards program output and returns `main`'s `i32` as the process status.\nThe generated starter returns zero after checking that `answer()` is 42.\n\n## Run on Linux\n\nWith Linux GNU amd64, glibc and GCC or Clang on PATH:\n\n```sh\nfairyc test --native --locked\nfairyc run --locked\nfairyc build --release --locked\n```\n\nRun the executable under `build/` with this package's name and no extension.\nLinux process status preserves the low eight bits of `main`'s return.\nUse default semantic tests for fast checks or `test --native` for isolated ELF execution.\n\n## Write to standard output\n\nReplace `src/main.fy` with:\n\n```fairyc\nmodule main;\n\nimport std::io as io;\n\n@effects(console)\nfn i32 main() {\n    let ref str message = \"Fairy C says hi\\n\";\n    return switch (io::write(message)) {\n        case .ok(written) : {\n            if (written == message.len) {\n                0\n            } else {\n                1\n            }\n        }\n        case .err(error) : {\n            if (error.status != 0 && error.written <= message.len) {\n                2\n            } else {\n                3\n            }\n        }\n    };\n}\n```\n\nThen refresh the lock and run the program directly:\n\n```text\nfairyc check\nfairyc run --release --locked\n```\n");}
    if(index==2){ok=scaffold_append(ref mut out,b"schema = \"fairyc.manifest/v1\"\n\n[package]\nuid = \"") && scaffold_append(ref mut out,uid) && scaffold_append(ref mut out,b"\"\nname = \"") && scaffold_append(ref mut out,name) && scaffold_append(ref mut out,b"\"\nversion = \"0.1.0\"\nlanguage = \"") && scaffold_append(ref mut out,sc_sdk::language()) && scaffold_append(ref mut out,b"\"\nsource = \"src\"\n");}
    if(index==3){ok=scaffold_append(ref mut out,b"module main;\n\npub fn i32 answer() {\n    return 42;\n}\n\nfn i32 main() {\n    return answer() - 42;\n}\n");}
    if(index==4){ok=scaffold_append(ref mut out,b"module tests::main;\n\nimport main as app;\n\n@test\nfn void answer_is_42() {\n    assert(app::answer() == 42);\n}\n");}
    if(!ok){return .err(scaffold_problem(8,index+3));}return .ok(out);
}
@effects(console) fn i32 scaffold_diagnostic(ScaffoldProblem problem,ref u8[] name){
    var bytes::Buffer out=bytes::empty();if(!scaffold_append(ref mut out,b"fairyc: project creation failed: ")){return 4;}
    if(problem.kind==1){if(!scaffold_append(ref mut out,b"project parent directory is unavailable\n")){return 4;}}
    else {
        let ref u8[] prefix=if(problem.kind==2){b"project path \""}else if(problem.kind==3){b"permission denied while creating \""}else if(problem.kind==5){b"project creation failed and \""}else{b"could not create \""};
        if(!scaffold_append(ref mut out,prefix) || !scaffold_append(ref mut out,name)){return 4;}
        if(problem.kind!=5 && problem.slot!=0 && (!scaffold_append(ref mut out,b"/") || !scaffold_append(ref mut out,scaffold_leaf(problem.slot)))){return 4;}
        if(!scaffold_append(ref mut out,if(problem.kind==2){b"\" already exists\n"}else if(problem.kind==5){b"\" could not be rolled back\n"}else{b"\"\n"})){return 4;}
    }
    return if(scaffold_stderr(bytes::view(ref out))){if(problem.kind==8){4}else{7}}else{4};
}
fn ref CStr scaffold_native(ref sc_vec::Vec<bytes::Buffer> paths,usize slot){
    let owner=sc_vec::at<bytes::Buffer>(paths,slot);let view=bytes::view(owner);
    assert(view.len>=2 && view.len<=4096 && view[view.len-1]==0);
    // Only scaffold_cstr constructs these immutable entries. Retain the exact
    // Buffer borrow origin, rather than an address into temporary text storage.
    return unsafe{CStr::from_raw_parts(owner,view as ptr u8,view.len)};
}
@effects(filesystem) fn Result<ScaffoldPin,ScaffoldProblem> scaffold_repin(ref sc_vec::Vec<bytes::Buffer> paths,usize slot){
    let owner=sc_vec::at<bytes::Buffer>(paths,slot);let view=bytes::view(owner);
    assert(view.len>=2 && view.len<=4096 && view[view.len-1]==0);return scaffold_pin(view[..view.len-1],slot);
}
@effects(filesystem) fn Result<bool,ScaffoldProblem> scaffold_write(ref CStr path,ref u8[] body,usize slot){
    var file=switch(sc_fs::create_new(path)){case .ok(p):{p}case .err(e):{return .err(scaffold_os(e.os_code,slot,false));}};
    var ScaffoldProblem problem=scaffold_problem(0,slot);
    switch(sc_fs::write_all(ref mut file,body)){case .ok(n):{if(n!=body.len){problem=scaffold_problem(4,slot);}}case .err(e):{problem=scaffold_os(e.os_code,slot,false);}};
    if(problem.kind==0){switch(sc_fs::sync(ref mut file)){case .ok(_):{}case .err(e):{problem=scaffold_os(e.os_code,slot,false);}};}
    switch(sc_fs::close(file)){case .ok(_):{}case .err(e):{if(problem.kind==0){problem=scaffold_os(e.os_code,slot,false);}}};
    problem.owns_file=true;return if(problem.kind==0){.ok(true)}else{.err(problem)};
}
// Remove only this attempt's fixed paths while their directory prefixes remain
// pinned. Unknown/nonempty replacements cause an explicit rollback failure.
// There is no recursive path traversal after releasing directory pins.
@effects(filesystem) fn ScaffoldProblem scaffold_rollback(ref sc_vec::Vec<bytes::Buffer> paths,ref mut ScaffoldPin target,ref mut ScaffoldPin src,ref mut ScaffoldPin tests,usize directories,usize files,ScaffoldProblem original){
    // A failed consuming close can leave all three pins cleared. Reestablish
    // containment before addressing any nested file; fail closed on a reparse.
    if(target.handle==0 && (directories!=0 || files!=0)){switch(scaffold_repin(paths,0)){case .ok(p):{let handle=p.handle;var pin=p;pin.handle=0;target.handle=handle;}case .err(_):{return scaffold_problem(5,0);}};}
    if(src.handle==0 && directories>=1){switch(scaffold_repin(paths,1)){case .ok(p):{let handle=p.handle;var pin=p;pin.handle=0;src.handle=handle;}case .err(_):{return scaffold_problem(5,0);}};}
    if(tests.handle==0 && directories>=2){switch(scaffold_repin(paths,2)){case .ok(p):{let handle=p.handle;var pin=p;pin.handle=0;tests.handle=handle;}case .err(_):{return scaffold_problem(5,0);}};}
    var bool ok=true;var usize at=files;
    while(at>0){at-=1;switch(sc_fs::remove_file(scaffold_native(paths,at+3))){case .ok(_):{}case .err(e):{if(e.os_code!=2 && e.os_code!=3){ok=false;}}};}
    if(!scaffold_unpin(tests)){ok=false;}if(directories>=2 && !sc_fs::remove_dir(scaffold_native(paths,2)).is_ok){ok=false;}
    if(!scaffold_unpin(src)){ok=false;}if(directories>=1 && !sc_fs::remove_dir(scaffold_native(paths,1)).is_ok){ok=false;}
    if(!scaffold_unpin(target)){ok=false;}if(!sc_fs::remove_dir(scaffold_native(paths,0)).is_ok){ok=false;}
    return if(ok){original}else{scaffold_problem(5,0)};
}
@effects(filesystem,environment) fn Result<bool,ScaffoldProblem> scaffold_create(ref u8[] name,ref u8[] uid){
    let cwd=switch(scaffold_pin(b".",0)){case .ok(p):{p}case .err(e):{return .err(scaffold_problem(if(e.kind==8){8}else{1},0));}};
    var bytes::Buffer wide=bytes::empty();if(!bytes::resize_zeroed(ref mut wide,65536).is_ok){return .err(scaffold_problem(8,0));}
    let count=unsafe{scaffold_final_path(cwd.handle,bytes::view_mut(ref mut wide) as ptr mut u8,32768,0)};
    if(count==0 || count>=32768){return .err(scaffold_problem(1,0));}
    let decoded=switch(sc_host::decode(bytes::view(ref wide)[..((count as usize)+1)*2])){case .ok(p):{p}case .err(e):{return .err(scaffold_problem(if(e.kind==2){8}else{1},0));}};
    let final=bytes::view(ref decoded);var bytes::Buffer parent=bytes::empty();
    if(final.len>=8 && bytes::equal(final[..8],b"\\\\?\\UNC\\")){if(!scaffold_append(ref mut parent,b"\\\\") || !scaffold_append(ref mut parent,final[8..])){return .err(scaffold_problem(8,0));}}
    else if(final.len>=4 && bytes::equal(final[..4],b"\\\\?\\")){if(!scaffold_append(ref mut parent,final[4..])){return .err(scaffold_problem(8,0));}}
    else{return .err(scaffold_problem(1,0));}
    let root=bytes::view(ref parent);
    // Current-path spelling is absolute. Reject unrecognized namespaces rather
    // than interpreting a drive-relative or device path as an ordinary parent.
    if(root.len<3 || root.len>4090){return .err(scaffold_problem(1,0));}
    var usize first=3;
    if(root[0]==92 && root[1]==92){first=2;while(first<root.len && root[first]!=92){first+=1;}first+=1;while(first<root.len && root[first]!=92){first+=1;}}
    else if(root[1]!=58 || (root[2]!=92 && root[2]!=47)){return .err(scaffold_problem(1,0));}
    return scaffold_parents(root,first,name,uid,cwd.handle);
}
// A lexical pin per prefix avoids effectful elements in the pure SDK Vec.
@effects(filesystem) fn Result<bool,ScaffoldProblem> scaffold_parents(ref u8[] root,usize begin,ref u8[] name,ref u8[] uid,usize cwd){
    var usize at=begin;while(at<root.len && root[at]!=92 && root[at]!=47){at+=1;}
    let pin=switch(scaffold_pin(root[..at],0)){case .ok(p):{p}case .err(_):{return .err(scaffold_problem(1,0));}};
    if(at==root.len){
        var ScaffoldIdentity expected=ScaffoldIdentity{.volume=0,.low=0,.high=0};var ScaffoldIdentity actual=ScaffoldIdentity{.volume=0,.low=0,.high=0};
        if(unsafe{scaffold_pin_info(cwd,18,ref mut expected as ptr mut void,24)}==0 || unsafe{scaffold_pin_info(pin.handle,18,ref mut actual as ptr mut void,24)}==0 || expected.volume!=actual.volume || expected.low!=actual.low || expected.high!=actual.high){return .err(scaffold_problem(1,0));}
        return scaffold_payload(root,name,uid);
    }return scaffold_parents(root,at+1,name,uid,cwd);
}
@effects(filesystem) fn Result<bool,ScaffoldProblem> scaffold_payload(ref u8[] root,ref u8[] name,ref u8[] uid){
    var bytes::Buffer destination=bytes::empty();if(!scaffold_append(ref mut destination,root) || root[root.len-1]!=92 && root[root.len-1]!=47 && !scaffold_append(ref mut destination,b"/") || !scaffold_append(ref mut destination,name)){return .err(scaffold_problem(8,0));}
    var sc_vec::Vec<bytes::Buffer> paths=sc_vec::empty<bytes::Buffer>();var sc_vec::Vec<bytes::Buffer> bodies=sc_vec::empty<bytes::Buffer>();var usize at=0;
    while(at<8){var bytes::Buffer path=bytes::empty();if(!scaffold_append(ref mut path,bytes::view(ref destination)) || at!=0 && (!scaffold_append(ref mut path,b"/") || !scaffold_append(ref mut path,scaffold_leaf(at)))){return .err(scaffold_problem(8,at));}let native=switch(scaffold_cstr(bytes::view(ref path),at)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!sc_vec::push<bytes::Buffer>(ref mut paths,native).is_ok){return .err(scaffold_problem(8,at));}at+=1;}
    at=0;while(at<5){let body=switch(scaffold_body(at,name,uid)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!sc_vec::push<bytes::Buffer>(ref mut bodies,body).is_ok){return .err(scaffold_problem(8,at+3));}at+=1;}
    switch(sc_fs::create_dir(scaffold_native(ref paths,0))){case .ok(_):{}case .err(e):{return .err(scaffold_os(e.os_code,0,true));}};
    var ScaffoldPin target=ScaffoldPin{.handle=0};var ScaffoldPin src=ScaffoldPin{.handle=0};var ScaffoldPin tests=ScaffoldPin{.handle=0};var usize directories=0;var usize files=0;
    let acquired=scaffold_pin(bytes::view(ref destination),0);
    switch(acquired){case .ok(p):{target=p;}case .err(e):{return .err(scaffold_rollback(ref paths,ref mut target,ref mut src,ref mut tests,0,0,e));}};
    at=1;while(at<=2){
        switch(sc_fs::create_dir(scaffold_native(ref paths,at))){case .ok(_):{directories=at;}case .err(e):{return .err(scaffold_rollback(ref paths,ref mut target,ref mut src,ref mut tests,directories,files,scaffold_os(e.os_code,at,false)));}};
        var bytes::Buffer path=bytes::empty();if(!scaffold_append(ref mut path,bytes::view(ref destination)) || !scaffold_append(ref mut path,b"/") || !scaffold_append(ref mut path,scaffold_leaf(at))){return .err(scaffold_rollback(ref paths,ref mut target,ref mut src,ref mut tests,directories,files,scaffold_problem(8,at)));}
        switch(scaffold_pin(bytes::view(ref path),at)){case .ok(p):{if(at==1){src=p;}else{tests=p;}}case .err(e):{return .err(scaffold_rollback(ref paths,ref mut target,ref mut src,ref mut tests,directories,files,e));}};at+=1;
    }
    at=0;while(at<5){
        // Only an actual exclusive open confers ownership of a file; the write
        // helper closes it even when write/sync fails. Rollback may unlink a
        // failed partial file, but must preserve a preexisting collision.
        let slot=at+3;let body=bytes::view(sc_vec::at<bytes::Buffer>(ref bodies,at));
        switch(scaffold_write(scaffold_native(ref paths,slot),body,slot)){case .ok(_):{files=at+1;}case .err(e):{if(e.owns_file){files=at+1;}return .err(scaffold_rollback(ref paths,ref mut target,ref mut src,ref mut tests,directories,files,e));}};at+=1;
    }
    var bool closed=scaffold_unpin(ref mut tests);if(!scaffold_unpin(ref mut src)){closed=false;}if(!scaffold_unpin(ref mut target)){closed=false;}
    if(!closed){return .err(scaffold_rollback(ref paths,ref mut target,ref mut src,ref mut tests,directories,files,scaffold_problem(4,0)));}
    return .ok(true);
}
@effects(filesystem,console,environment) pub fn i32 scaffold(){
    let argc=process::arg_count();
    if(argc==3){let option=switch(scaffold_arg(2)){case .ok(p):{p}case .err(_):{return 4;}};if(scaffold_help(bytes::view(ref option))){return scaffold_output(sc_text::as_bytes(public::help()));}}
    if(argc!=5){return scaffold_invocation("command \"new\" expects <package-name> --uid <package-uid>");}
    let option=switch(scaffold_arg(3)){case .ok(p):{p}case .err(_):{return 4;}};
    if(!bytes::equal(bytes::view(ref option),b"--uid")){return scaffold_invocation("command \"new\" expects <package-name> --uid <package-uid>");}
    let name=switch(scaffold_arg(2)){case .ok(p):{p}case .err(_):{return 4;}};let spelling=bytes::view(ref name);
    if(!sc_values::name(spelling) || !sc_values::segment(spelling)){return scaffold_invocation("command \"new\": package name must use portable lowercase kebab-case");}
    let uid=switch(scaffold_arg(4)){case .ok(p):{p}case .err(_):{return 4;}};
    if(!sc_values::uid(bytes::view(ref uid))){return scaffold_invocation("command \"new\": package UID must use at least two lowercase dot-separated segments");}
    switch(scaffold_create(spelling,bytes::view(ref uid))){case .ok(_):{}case .err(e):{return scaffold_diagnostic(e,spelling);}};
    var bytes::Buffer message=bytes::empty();if(!scaffold_append(ref mut message,b"created Fairy C package \"") || !scaffold_append(ref mut message,spelling) || !scaffold_append(ref mut message,b"\" at ") || !scaffold_append(ref mut message,spelling) || !scaffold_append(ref mut message,b"\n")){return 4;}
    return scaffold_output(bytes::view(ref message));
}
