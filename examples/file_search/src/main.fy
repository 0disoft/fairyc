module main;
import search as search;
import std::fs as fs;
import std::io as io;
import std::process as process;
import std::text as text;

@effects(console)
fn i32 fail(ref str message, i32 code) {
    discard io::write_stderr(message);
    return code;
}

@effects(console)
fn i32 report(search::Scan result) {
    if (!io::write("lines=").is_ok || !io::write_usize(result.lines).is_ok ||
        !io::write(" matches=").is_ok || !io::write_usize(result.matches).is_ok ||
        !io::write("\n").is_ok) {
        return fail("file-search: cannot write result\n", 1);
    }
    return 0;
}

@effects(filesystem, console)
fn i32 scan_file(fs::File input) {
    var fs::File file = input;
    var u8[4096] buffer = [0; 4096];
    var search::Scan state = search::start();
    while (true) {
        let usize count = switch (fs::read(ref mut file, ref mut buffer)) {
            case .ok(n) : { n }
            case .err(_) : { return fail("file-search: cannot read input\n", 1); }
        };
        if (count == 0) { break; }
        let ref u8[] bytes = ref buffer;
        state = search::feed(state, bytes[..count]);
    }
    if (!fs::close(file).is_ok) {
        return fail("file-search: cannot close input\n", 1);
    }
    return report(search::finish(state));
}

@effects(filesystem, console)
fn i32 main() {
    if (process::arg_count() != 2) {
        return fail("usage: file-search FILE\n", 2);
    }
    var u8[4096] path = [0; 4096];
    let usize count = switch (process::arg(1, ref mut path)) {
        case .ok(n) : { n }
        case .err(_) : { return fail("file-search: cannot read path argument\n", 2); }
    };
    let ref u8[] bytes = ref path;
    return switch (text::validate_utf8(bytes[..count])) {
        case .ok(name) : {
            switch (fs::open_read_utf8(name)) {
                case .ok(file) : { scan_file(file) }
                case .err(_) : { fail("file-search: cannot open input\n", 1) }
            }
        }
        case .err(_) : { fail("file-search: path must be UTF-8\n", 2) }
    };
}
