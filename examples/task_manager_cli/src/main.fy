module main;
import std::process as process;
import std::bytes as bytes;
import std::text as text;
import format as format;
import output as output;
import tasks as tasks;
import list_options as list_options;

@effects(filesystem, console)
fn i32 arguments(ref u8[] path, ref u8[] command, ref u8[] value, usize count, u32 status, bool by_title) {
    if (path.is_empty || path.len > 4000) { return output::usage("store path must contain 1..4000 UTF-8 bytes"); }
    var usize i = 0;
    while (i < path.len) { if (path[i] == 0) { return output::usage("store path contains NUL"); } i += 1; }
    var u32 operation = 0;
    if (bytes::equal(command, b"list")) { operation = 0; }
    else if (bytes::equal(command, b"add")) { operation = 1; }
    else if (bytes::equal(command, b"done")) { operation = 2; }
    else if (bytes::equal(command, b"delete")) { operation = 3; }
    else { return output::usage("unknown command; use list, add, done or delete"); }
    if ((operation == 0 && count != 3 && count != 5 && count != 7) || (operation != 0 && count != 4)) { return output::usage("wrong command argument count; use --help"); }
    var i32 requested = 0;
    if (operation == 1) {
        if (!format::title(value)) { return output::usage("title must be valid UTF-8, 1..512 bytes, nonblank, without ASCII controls"); }
    } else if (operation != 0) {
        requested = switch (format::id(value)) {
            case .ok(n) : { n } case .err(_) : { return output::usage("ID must be canonical decimal 1..2147483647"); }
        };
    }
    return switch (text::validate_utf8(path)) {
        case .ok(name) : { tasks::run(name, operation, value, requested, status, by_title) }
        case .err(_) : { output::usage("store path is not valid UTF-8") }
    };
}

@effects(filesystem, console)
fn i32 main() {
    let usize count = process::arg_count();
    if (count < 2 || count > 7) { return output::usage("expected store path and command; use --help"); }
    var bytes::Buffer path = bytes::empty();
    if (!format::buffer_ok(bytes::resize_zeroed(ref mut path, 4001))) { return output::error("task-manager-cli", "cannot allocate arguments"); }
    let usize path_length = switch (process::arg(1, bytes::view_mut(ref mut path))) {
        case .ok(n) : { n } case .err(_) : { return output::usage("cannot read store path (maximum 4000 UTF-8 bytes)"); }
    };
    let ref u8[] name = bytes::view(ref path)[..path_length];
    if (count == 2) {
        if (bytes::equal(name, b"--help")) {
            return output::text("Usage: task-manager-cli <store-path> list [--status open|done|all] [--sort id|title]\n       task-manager-cli <store-path> add \"title\"\n       task-manager-cli <store-path> done <id>\n       task-manager-cli <store-path> delete <id>\n       task-manager-cli --help\nIDs: canonical decimal, assigned 1..2147483646; never reused after deletion\nList: defaults all/ID; title uses UTF-8 byte order with ID ties; no matches prints empty\nList output: tab-separated ID/open|done/title; options may appear in either order\nStore: UTF-8 no BOM, LF TSV, FYTASK1<TAB>next<LF> then id<TAB>0|1<TAB>title<LF>\nTitles: spaces/Korean preserved; nonblank UTF-8, no ASCII controls\nSave: exclusive .lock, checked write/sync/close of .tmp, then replace_file; no backup gap\nPending .lock/.tmp/.bak needs manual inspection/recovery; no automatic deletion\nLimits: 256 tasks, title 512 bytes, store 262144 bytes, path 4000 UTF-8 bytes\nExit: 0 success, 1 store/data/allocation/cleanup error, 2 CLI usage, 3 stdout failure\n");
        }
        return output::usage("expected store path and command; use --help");
    }
    var bytes::Buffer command = bytes::empty();
    if (!format::buffer_ok(bytes::resize_zeroed(ref mut command, 16))) { return output::error("task-manager-cli", "cannot allocate arguments"); }
    let usize command_length = switch (process::arg(2, bytes::view_mut(ref mut command))) {
        case .ok(n) : { n } case .err(_) : { return output::usage("cannot read command"); }
    };
    let ref u8[] command_name = bytes::view(ref command)[..command_length];
    if (bytes::equal(command_name, b"list")) {
        let list_options::Options options = switch (list_options::parse(count)) {
            case .ok(v) : { v } case .err(code) : { return code; }
        };
        return arguments(name, command_name, b"", count, options.status, options.by_title);
    }
    var bytes::Buffer value = bytes::empty();
    if (!format::buffer_ok(bytes::resize_zeroed(ref mut value, 513))) { return output::error("task-manager-cli", "cannot allocate arguments"); }
    var usize value_length = 0;
    if (count == 4) {
        value_length = switch (process::arg(3, bytes::view_mut(ref mut value))) {
            case .ok(n) : { n } case .err(_) : { return output::usage("cannot read title/ID (maximum title 512 UTF-8 bytes)"); }
        };
    }
    return arguments(name, bytes::view(ref command)[..command_length], bytes::view(ref value)[..value_length], count, 0, false);
}
