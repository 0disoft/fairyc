module main;

import std::fs as fs;
import std::process as process;
import std::text as text;
import std::bytes as bytes;
import csv as csv;
import diagnostics as diag;

fn bool buffer_ok(Result<usize, bytes::BufferError> result) {
    return switch (result) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
}

@effects(filesystem, console)
fn Result<bytes::Buffer, i32> read_contents(ref str path, ref mut fs::File file) {
    var bytes::Buffer contents = bytes::empty();
    var bytes::Buffer chunk = bytes::empty();
    if (!buffer_ok(bytes::resize_zeroed(ref mut chunk, 4096))) {
        return .err(diag::error(path, 0, 0, "cannot allocate read buffer"));
    }
    let usize maximum = 1048576;
    while (true) {
        let usize count = switch (fs::read(file, bytes::view_mut(ref mut chunk))) {
            case .ok(read) : { read }
            case .err(error) : {
                return .err(diag::file_error(path, "read failed", fs::error_status(error), fs::error_os_code(error)));
            }
        };
        if (count == 0) { break; }
        if (count > maximum - bytes::len(ref contents)) {
            return .err(diag::error(path, 0, 0, "file exceeds 1048576 byte limit"));
        }
        if (!buffer_ok(bytes::append(ref mut contents, bytes::view(ref chunk)[..count]))) {
            return .err(diag::error(path, 0, 0, "cannot allocate file buffer"));
        }
    }
    return .ok(contents);
}

@effects(filesystem, console)
fn i32 aggregate_file(ref str path, ref u8[] column, fs::File input) {
    var fs::File file = input;
    let Result<bytes::Buffer, i32> contents = read_contents(path, ref mut file);
    let i32 close_code = switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(error) : { diag::file_error(path, "close failed", fs::error_status(error), fs::error_os_code(error)) }
    };
    return switch (contents) {
        case .ok(buffer) : {
            if (close_code != 0) { close_code } else { csv::aggregate(path, bytes::view(ref buffer), column) }
        }
        case .err(code) : { code }
    };
}

@effects(filesystem, console)
fn i32 run_arguments(ref u8[] path_bytes, ref u8[] column) {
    if (path_bytes.is_empty) { return diag::usage("file path is empty"); }
    if (path_bytes.len > 4095) { return diag::usage("file path exceeds 4095 UTF-8 bytes"); }
    if (path_bytes.len >= 2 && path_bytes[0] == 45 && path_bytes[1] == 45) {
        return diag::usage("unknown option; use --help (prefix a -- filename with ./)");
    }
    if (column.is_empty) { return diag::usage("column name is empty"); }
    let bool valid_column = switch (text::validate_utf8(column)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
    if (!valid_column) { return diag::usage("column name is not valid UTF-8"); }
    return switch (text::validate_utf8(path_bytes)) {
        case .ok(path) : { switch (fs::open_read_utf8(path)) {
            case .ok(file) : { aggregate_file(path, column, file) }
            case .err(error) : { diag::file_error(path, "open failed", fs::error_status(error), fs::error_os_code(error)) }
        } }
        case .err(_) : { diag::usage("file path is not valid UTF-8") }
    };
}

@effects(filesystem, console)
fn i32 main() {
    let usize count = process::arg_count();
    if (count != 2 && count != 3) { return diag::usage("expected file path and column name; use --help"); }
    var bytes::Buffer path = bytes::empty();
    if (!buffer_ok(bytes::resize_zeroed(ref mut path, 4096))) {
        return diag::error("csv-aggregator", 0, 0, "cannot allocate argument buffer");
    }
    let usize path_length = switch (process::arg(1, bytes::view_mut(ref mut path))) {
        case .ok(length) : { length }
        case .err(_) : { return diag::usage("cannot read file path (maximum 4095 UTF-8 bytes)"); }
    };
    let ref u8[] path_value = bytes::view(ref path)[..path_length];
    if (count == 2) {
        if (csv::equals(path_value, b"--help")) {
            return if (diag::stdout_text("Usage: csv-aggregator <file-path> <column-name>\n       csv-aggregator --help\nValues and running sum: signed decimal i64 (-9223372036854775808..9223372036854775807); no floats/exponents\nLimits: file 1048576 bytes, record 65536 raw bytes, field 16384 decoded bytes, 64 columns\n")) {
                0
            } else { diag::error("csv-aggregator", 0, 0, "stdout write failed") };
        }
        return diag::usage("expected file path and column name; use --help");
    }
    var bytes::Buffer column = bytes::empty();
    if (!buffer_ok(bytes::resize_zeroed(ref mut column, 16384))) {
        return diag::error("csv-aggregator", 0, 0, "cannot allocate argument buffer");
    }
    let usize column_length = switch (process::arg(2, bytes::view_mut(ref mut column))) {
        case .ok(length) : { length }
        case .err(_) : { return diag::usage("cannot read column name (maximum 16384 UTF-8 bytes)"); }
    };
    return run_arguments(path_value, bytes::view(ref column)[..column_length]);
}
