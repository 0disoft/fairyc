module declaration_output;
import syntax as syntax;
import scanner as scanner;
import token_output as output;
import std::bytes as bytes;
import std::text as text;
import std::io as io;

fn bool label(ref mut bytes::Buffer result, u32 kind) {
    let ref str name = if (kind == 1) { "module" } else if (kind == 2) { "import" }
        else if (kind == 3) { "function" } else if (kind == 4) { "struct" }
        else if (kind == 6) { "resource" } else if (kind == 7) { "const" }
        else if (kind == 8) { "static-assert" } else if (kind == 9) { "foreign-function" }
        else if (kind == 10) { "state" } else { "enum" };
    return bytes::append(result, text::as_bytes(name)).is_ok;
}

/// Validate every declaration before rendering. Parser names are ASCII spans;
/// imports expose the alias, while state/static_assert have an empty 0..0 name.
pub fn u32 declarations(ref syntax::Parsed parsed, ref mut bytes::Buffer result) {
    let ref u8[] source = syntax::source_view(parsed);
    let usize count = syntax::declaration_count(parsed);
    if (syntax::problem_count(parsed) != 0 || count == 0) { return 5; }
    var usize previous = 0; var usize index = 0;
    while (index < count) {
        let syntax::Declaration item = syntax::declaration_at(parsed, index);
        if (item.kind == 0 || item.kind == 5 || item.kind > 11 ||
            item.start < previous || item.start >= item.end || item.end > source.len) { return 5; }
        if (item.kind == 8 || item.kind == 10) {
            if (item.name_start != 0 || item.name_end != 0) { return 5; }
        } else if (item.name_start < item.start || item.name_start >= item.name_end || item.name_end > item.end) { return 5; }
        previous = item.end; index += 1;
    }
    if (!bytes::append(result, b"kind\tname\tstart\tend\tname_start\tname_end\n").is_ok) { return 3; }
    index = 0;
    while (index < count) {
        let syntax::Declaration item = syntax::declaration_at(parsed, index);
        let ref u8[] name = source[item.name_start..item.name_end];
        if (128 + name.len > 1048576 - bytes::len(result)) { return 4; }
        if (!label(result, item.kind) || !bytes::append(result, b"\t").is_ok || !bytes::append(result, name).is_ok ||
            !bytes::append(result, b"\t").is_ok || !bytes::append_usize(result, item.start).is_ok ||
            !bytes::append(result, b"\t").is_ok || !bytes::append_usize(result, item.end).is_ok ||
            !bytes::append(result, b"\t").is_ok || !bytes::append_usize(result, item.name_start).is_ok ||
            !bytes::append(result, b"\t").is_ok || !bytes::append_usize(result, item.name_end).is_ok ||
            !bytes::append(result, b"\n").is_ok) { return 3; }
        index += 1;
    }
    return 0;
}

@effects(console)
pub fn i32 errors(ref syntax::Parsed parsed, ref u8[] path) {
    let ref u8[] source = syntax::source_view(parsed);
    var usize index = 0;
    while (index < syntax::problem_count(parsed)) {
        let syntax::Problem problem = syntax::problem_at(parsed, index);
        output::prefix(source, path, problem.start);
        if (problem.code == 4 && problem.expected == 0) { discard io::write_stderr("unsupported syntax"); }
        else {
            discard io::write_stderr("expected ");
            discard io::write_stderr_bytes(syntax::expected_text(problem.expected));
        }
        output::range(problem.start, problem.end);
        index += 1;
    }
    return 2;
}

@effects(console)
pub fn Result<syntax::Parsed,i32> compile(scanner::Stream stream,ref u8[] path){
    let syntax::Parsed parsed = switch(syntax::parse(stream, 512)) {
        case .ok(value): { value }
        case .err(code): {
            discard io::write_stderr(if (code == 4) { "error: parser limit\n" }
                else if (code == 3) { "error: buffer allocation failed\n" } else { "error: parser failed\n" });
            return .err(2);
        }
    };
    if (syntax::problem_count(ref parsed) != 0) { return .err(errors(ref parsed, path)); }
    return .ok(parsed);
}

@effects(console)
pub fn i32 run(scanner::Stream stream, ref u8[] path) {
    let syntax::Parsed parsed=switch(compile(stream,path)){case .ok(p):{p}case .err(code):{return code;}};
    var bytes::Buffer result = bytes::empty();
    let u32 status = declarations(ref parsed, ref mut result);
    if (status != 0) {
        discard io::write_stderr(if (status == 4) { "error: output limit (1048576 bytes)\n" }
            else if (status == 5) { "error: invalid declaration ranges\n" } else { "error: buffer allocation failed\n" });
        return 2;
    }
    let ref u8[] data = bytes::view(ref result);
    return output::write(data);
}
