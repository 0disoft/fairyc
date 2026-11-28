module main;
import search as search;
import std::bytes as bytes;
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
fn i32 usage() {
    return fail("usage: file-search FILE | file-search --dir DIR\n", 2);
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
fn Result<search::Scan, i32> scan_file(ref str path) {
    let fs::File input = switch (fs::open_read_utf8(path)) {
        case .ok(file) : { file }
        case .err(_) : { return .err(fail("file-search: cannot open input\n", 1)); }
    };
    var fs::File file = input;
    var u8[4096] buffer = [0; 4096];
    var search::Scan state = search::start();
    while (true) {
        let usize count = switch (fs::read(ref mut file, ref mut buffer)) {
            case .ok(n) : { n }
            case .err(_) : { return .err(fail("file-search: cannot read input\n", 1)); }
        };
        if (count == 0) { break; }
        let ref u8[] bytes = ref buffer;
        state = search::feed(state, bytes[..count]);
    }
    if (!fs::close(file).is_ok) {
        return .err(fail("file-search: cannot close input\n", 1));
    }
    return .ok(search::finish(state));
}

@effects(filesystem, console)
fn i32 scan_directory(ref str path) {
    let ref u8[] root = text::as_bytes(path);
    if (root.len == 0) { return fail("file-search: cannot open directory\n", 1); }
    var fs::ReadDir directory = switch (fs::read_dir_utf8(path)) {
        case .ok(opened) : { opened }
        case .err(_) : { return fail("file-search: cannot open directory\n", 1); }
    };
    var bytes::Buffer child = bytes::empty();
    if (!bytes::append(ref mut child, root).is_ok ||
        (root[root.len - 1] != 47 && !bytes::append_byte(ref mut child, 47).is_ok)) {
        return fail("file-search: cannot build child path\n", 1);
    }
    let usize prefix = bytes::len(ref child);
    var u8[4096] name = [0; 4096];
    var search::Scan total = search::start();
    while (true) {
        let fs::DirEntry entry = switch (fs::next_dir(ref mut directory, ref mut name)) {
            case .ok(value) : { value }
            case .err(_) : { return fail("file-search: cannot read directory\n", 1); }
        };
        if (!entry.has_entry) { break; }
        if (!fs::path_kind_is_file(entry.kind) || entry.is_reparse) { continue; }
        let ref u8[] name_bytes = ref name;
        if (!bytes::truncate(ref mut child, prefix).is_ok ||
            !bytes::append(ref mut child, name_bytes[..entry.name_length]).is_ok) {
            return fail("file-search: cannot build child path\n", 1);
        }
        let ref str child_path = switch (text::validate_utf8(bytes::view(ref child))) {
            case .ok(value) : { value }
            case .err(_) : { return fail("file-search: child path must be UTF-8\n", 1); }
        };
        let search::Scan file = switch (scan_file(child_path)) {
            case .ok(value) : { value }
            case .err(code) : { return code; }
        };
        total.lines += file.lines;
        total.matches += file.matches;
    }
    if (!fs::close_dir(directory).is_ok) {
        return fail("file-search: cannot close directory\n", 1);
    }
    return report(total);
}

@effects(filesystem, console)
fn i32 main() {
    let usize argc = process::arg_count();
    if (argc != 2 && argc != 3) { return usage(); }
    var u8[4096] path = [0; 4096];
    var usize count = switch (process::arg(1, ref mut path)) {
        case .ok(n) : { n }
        case .err(_) : { return fail("file-search: cannot read path argument\n", 2); }
    };
    let ref u8[] first = ref path;
    let bool directory_mode = bytes::equal(first[..count], b"--dir");
    if (directory_mode != (argc == 3)) { return usage(); }
    if (directory_mode) {
        count = switch (process::arg(2, ref mut path)) {
            case .ok(n) : { n }
            case .err(_) : { return fail("file-search: cannot read path argument\n", 2); }
        };
    }
    let ref u8[] bytes = ref path;
    return switch (text::validate_utf8(bytes[..count])) {
        case .ok(name) : {
            if (directory_mode) { scan_directory(name) } else {
                switch (scan_file(name)) {
                    case .ok(result) : { report(result) }
                    case .err(code) : { code }
                }
            }
        }
        case .err(_) : { fail("file-search: path must be UTF-8\n", 2) }
    };
}
