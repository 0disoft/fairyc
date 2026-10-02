module list_options;
import std::process as process;
import std::bytes as bytes;
import format as format;
import output as output;

pub struct Options { u32 status; bool by_title; }

@effects(filesystem, console)
pub fn Result<Options, i32> parse(usize count) {
    if (count != 3 && count != 5 && count != 7) { return .err(output::usage("wrong command argument count; use --help")); }
    var u32 status = 0;
    var bool by_title = false;
    var bool seen_status = false;
    var bool seen_sort = false;
    var bytes::Buffer flag = bytes::empty();
    var bytes::Buffer value = bytes::empty();
    if (!format::buffer_ok(bytes::resize_zeroed(ref mut flag, 16)) ||
        !format::buffer_ok(bytes::resize_zeroed(ref mut value, 16))) {
        return .err(output::error("task-manager-cli", "cannot allocate arguments"));
    }
    var usize i = 3;
    while (i < count) {
        let usize flag_length = switch (process::arg(i, bytes::view_mut(ref mut flag))) {
            case .ok(n) : { n } case .err(_) : { return .err(output::usage("cannot read list option (maximum 15 UTF-8 bytes)")); }
        };
        if (flag_length > 15) { return .err(output::usage("cannot read list option (maximum 15 UTF-8 bytes)")); }
        let bool status_flag = bytes::equal(bytes::view(ref flag)[..flag_length], b"--status");
        let bool sort_flag = bytes::equal(bytes::view(ref flag)[..flag_length], b"--sort");
        if (!status_flag && !sort_flag) { return .err(output::usage("unknown list option; use --status or --sort")); }
        let usize value_length = switch (process::arg(i + 1, bytes::view_mut(ref mut value))) {
            case .ok(n) : { n } case .err(_) : { return .err(output::usage("cannot read list option (maximum 15 UTF-8 bytes)")); }
        };
        if (value_length > 15) { return .err(output::usage("cannot read list option (maximum 15 UTF-8 bytes)")); }
        let ref u8[] selected = bytes::view(ref value)[..value_length];
        if (status_flag) {
            if (seen_status) { return .err(output::usage("duplicate --status option")); }
            seen_status = true;
            if (bytes::equal(selected, b"all")) { status = 0; }
            else if (bytes::equal(selected, b"open")) { status = 1; }
            else if (bytes::equal(selected, b"done")) { status = 2; }
            else { return .err(output::usage("status must be open, done or all")); }
        } else {
            if (seen_sort) { return .err(output::usage("duplicate --sort option")); }
            seen_sort = true;
            if (bytes::equal(selected, b"id")) { by_title = false; }
            else if (bytes::equal(selected, b"title")) { by_title = true; }
            else { return .err(output::usage("sort must be id or title")); }
        }
        i += 2;
    }
    return .ok(Options { .status = status, .by_title = by_title });
}
