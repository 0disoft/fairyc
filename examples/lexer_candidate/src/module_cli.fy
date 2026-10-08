module module_cli;
import candidate_limits as limits;
import module_registry as modules;
import module_check as checking;
import file_source as files;
import scanner as scanner;
import syntax as syntax;
import token_output as output;
import declaration_output as declarations;
import module_program as programs;
import c_emit as c;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;
import std::io as io;

pub struct Path { usize start; usize end; }
@effects(console)
fn i32 error(ref str message) { discard io::write_stderr(message); return 2; }
@effects(filesystem)
fn Result<bytes::Buffer,u32> manifest(ref u8[] path_bytes) {
    return files::manifest(path_bytes);
}
// Exact, untrimmed UTF-8 relative path lines. LF/CRLF and no final newline;
// no empty lines, BOM, control bytes, colon/ADS, or leading slash/backslash.
// Relative traversal is allowed: this is a cwd resolver, not a sandbox.
// 1 invalid manifest, 3 allocation, 4 path bound, 5 module bound.
pub fn u32 paths(ref u8[] source, ref mut v::Vec<Path> result) {
    if (source.len == 0 || source.len > limits::manifest_bytes() || !text::validate_utf8(source).is_ok) { return 1; }
    if (source.len >= 3 && source[0] == 239 && source[1] == 187 && source[2] == 191) { return 1; }
    var usize start = 0;
    while (start < source.len) {
        var usize end = start;
        while (end < source.len && source[end] != 10) { end += 1; }
        var usize stop = end;
        if (end < source.len && stop > start && source[stop - 1] == 13) { stop -= 1; }
        if (stop == start || source[start] == 47 || source[start] == 92) { return 1; }
        if (stop - start > 4095) { return 4; }
        var usize byte = start;
        while (byte < stop) { if (source[byte] < 32 || source[byte] == 127 || source[byte] == 58) { return 1; } byte += 1; }
        if (v::len<Path>(result) >= limits::source_files()) { return 5; }
        if (!v::push<Path>(result,Path { .start, .end = stop }).is_ok) { return 3; }
        start = if (end == source.len) { end } else { end + 1 };
    }
    return 0;
}
@effects(filesystem,console)
fn i32 execute(ref u8[] path,bool program_mode) {
    let bytes::Buffer manifest_source = switch(manifest(path)) {
        case .ok(value): { value }
        case .err(code): { return error(if (code == 4) { "error: manifest limit (65536 bytes)\n" } else if (code == 3) { "error: buffer allocation failed\n" } else { "error: manifest file read failed\n" }); }
    };
    let ref u8[] source = bytes::view(ref manifest_source);
    var v::Vec<Path> entries = v::empty<Path>();
    let u32 status = paths(source,ref mut entries);
    if (status != 0) { return error(if (status == 3) { "error: buffer allocation failed\n" } else if (status == 4) { "error: file path limit (4095 UTF-8 bytes)\n" } else if (status == 5) { "error: module limit (256 files)\n" } else { "error: invalid module manifest\n" }); }
    var v::Vec<syntax::Parsed> parsed_files = v::empty<syntax::Parsed>();
    var usize index = 0;
    while (index < v::len<Path>(ref entries)) {
        let Path entry = *v::at<Path>(ref entries,index);
        let ref u8[] filename = source[entry.start..entry.end];
        let bytes::Buffer input = switch(files::load(filename)) {
            case .ok(value): { value }
            case .err(code): { return error(if (code == 4) { "error: source limit (65536 bytes)\n" } else if (code == 3) { "error: buffer allocation failed\n" } else { "error: input file read failed\n" }); }
        };
        let scanner::Stream stream = switch(scanner::collect(input,false,limits::source_bytes(),1048576,256,16777216)) {
            case .ok(value): { value } case .err(code): { return error(if (code == 4) { "error: lexer storage limit\n" } else if (code == 2) { "error: unsupported lexer input\n" } else { "error: buffer allocation failed\n" }); }
        };
        if (scanner::diagnostic_count(ref stream) > 0) { return output::errors(ref stream,filename); }
        let syntax::Parsed parsed = switch(syntax::parse(stream,512)) {
            case .ok(value): { value } case .err(code): { return error(if (code == 4) { "error: parser limit\n" } else if (code == 3) { "error: buffer allocation failed\n" } else { "error: parser failed\n" }); }
        };
        if (syntax::problem_count(ref parsed) != 0) { return declarations::errors(ref parsed,filename); }
        if (!v::push<syntax::Parsed>(ref mut parsed_files,parsed).is_ok) { return error("error: buffer allocation failed\n"); }
        index += 1;
    }
    if(program_mode){
        let program=switch(programs::compile(ref parsed_files)){
            case .ok(p):{p}
            case .err(p):{
                let Path entry=*v::at<Path>(ref entries,p.module_index);
                output::prefix(syntax::source_view(v::at<syntax::Parsed>(ref parsed_files,p.module_index)),source[entry.start..entry.end],p.start);
                discard io::write_stderr("module program phase ");discard io::write_stderr_u32(p.phase);
                discard io::write_stderr(" code ");discard io::write_stderr_u32(p.code);output::range(p.start,p.end);return 2;
            }
        };
        var bytes::Buffer out=bytes::empty();let u32 code=c::render(ref program,ref mut out);
        if(code!=0){return error(if(code==1){"error: C entry requires exactly one fn i32 main()\n"}else if(code==4){"error: C output limit (2097152 bytes)\n"}else if(code==3){"error: buffer allocation failed\n"}else{"error: invalid module C input\n"});}
        let data=bytes::view(ref out);
        return if(switch(io::write_bytes(data)){case .ok(n):{n==data.len}case .err(_):{false}}){0}else{error("error: stdout write failed\n")};
    }
    let modules::Registry registry = switch(modules::build(ref parsed_files)) {
        case .ok(value): { value }
        case .err(problem): { let Path entry = *v::at<Path>(ref entries,problem.module_index); return checking::error(v::at<syntax::Parsed>(ref parsed_files,problem.module_index),source[entry.start..entry.end],problem); }
    };
    // Move exact syntax owners into checking in manifest order, without rereading
    // files or rebuilding source. Public Vec pop/push owns all transfers.
    var v::Vec<syntax::Parsed> pending = v::empty<syntax::Parsed>();
    while (v::len<syntax::Parsed>(ref parsed_files) > 0) {
        discard switch(v::pop<syntax::Parsed>(ref mut parsed_files)) { case .some(parsed): { if (!v::push<syntax::Parsed>(ref mut pending,parsed).is_ok) { return error("error: buffer allocation failed\n"); } } case .none: { return error("error: parser failed\n"); } };
    }
    index = 0;
    while (v::len<syntax::Parsed>(ref pending) > 0) {
        let Path entry = *v::at<Path>(ref entries,index);
        let i32 checked = switch(v::pop<syntax::Parsed>(ref mut pending)) { case .some(parsed): { checking::run(parsed,ref registry,index,source[entry.start..entry.end]) } case .none: { 2 } };
        if (checked != 0) { return checked; } index += 1;
    }
    return if (io::write("modules ok\n").is_ok) { 0 } else { error("error: stdout write failed\n") };
}

@effects(filesystem,console)
pub fn i32 run(ref u8[] path){return execute(path,false);}
@effects(filesystem,console)
pub fn i32 emit(ref u8[] path){return execute(path,true);}
