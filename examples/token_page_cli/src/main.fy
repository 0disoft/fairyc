module main;
import tokens as tokens;
import evaluator as evaluator;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

fn bool equal(ref u8[] left, ref u8[] right) {
    if (left.len != right.len) { return false; }
    var usize index = 0;
    while (index < left.len) {
        if (left[index] != right[index]) { return false; }
        index += 1;
    }
    return true;
}
@effects(console)
fn i32 error(u32 code) {
    let ref str message = if (code == 1) { "error: invalid ASCII input\n" }
        else if (code == 2) { "error: integer overflow\n" }
        else if (code == 3) { "error: token limit (128)\n" }
        else if (code == 4) { "error: sum overflow\n" }
        else if (code == 5) { "error: buffer allocation failed\n" }
        else if (code == 6) { "error: input limit (4095 UTF-8 bytes)\n" }
        else if (code == 7) { "error: stdout write failed\n" }
        else if (code == 8) { "error: invalid evaluation syntax\n" }
        else if (code == 9) { "error: undefined variable\n" }
        else if (code == 10) { "error: evaluation overflow\n" }
        else if (code == 11) { "error: variable limit (16)\n" }
        else { "usage: token-page-cli [ASCII-source]\n" };
    discard io::write_stderr(message);
    return 2;
}
fn bool count_row(ref mut bytes::Buffer output, ref u8[] label, usize count) {
    return bytes::append(output, label).is_ok && bytes::append_usize(output, count).is_ok && bytes::append(output, b"\n").is_ok;
}
fn bool sum_row(ref mut bytes::Buffer output, ref u8[] label, i64 sum) {
    return bytes::append(output, label).is_ok && bytes::append_i64(output, sum).is_ok && bytes::append(output, b"\n").is_ok;
}
@effects(console)
fn i32 analyze_source(ref u8[] source) {
    var tokens::TokenPage page = tokens::empty();
    let usize used = switch(tokens::scan(source, ref mut page)) {
        case .ok(count): { count }
        case .err(code): { return error(code); }
    };
    tokens::rebase(ref mut page);
    let ref tokens::Token[] valid = tokens::valid(ref page);
    assert(valid.len == used && (valid.is_empty || valid[0].position.start == 0));
    let tokens::Stats all = switch(tokens::analyze(valid)) {
        case .ok(stats): { stats }
        case .err(code): { return error(code); }
    };
    let usize first_end = tokens::first_end(valid);
    let ref tokens::Token[] first = ref page.tokens[..first_end];
    let tokens::Stats section = switch(tokens::analyze(first)) {
        case .ok(stats): { stats }
        case .err(code): { return error(code); }
    };
    var bytes::Buffer output = bytes::empty();
    if (!count_row(ref mut output, b"tokens\t", used) ||
        !count_row(ref mut output, b"identifiers\t", all.identifiers) ||
        !count_row(ref mut output, b"numbers\t", all.numbers) ||
        !sum_row(ref mut output, b"sum\t", all.sum) ||
        !count_row(ref mut output, b"first_tokens\t", first_end) ||
        !count_row(ref mut output, b"first_identifiers\t", section.identifiers) ||
        !count_row(ref mut output, b"first_numbers\t", section.numbers) ||
        !sum_row(ref mut output, b"first_sum\t", section.sum)) { return error(5); }
    let ref u8[] result = bytes::view(ref output);
    return if (switch(io::write_bytes(result)) { case .ok(count): { count == result.len } case .err(_): { false } }) { 0 } else { error(7) };
}
@effects(console)
fn i32 evaluate_source(ref u8[] source) {
    var tokens::TokenPage page = tokens::empty();
    discard switch(tokens::scan(source, ref mut page)) {
        case .ok(count): { count }
        case .err(code): { return error(code); }
    };
    // Identifier spans retain offsets into the original source; do not rebase.
    let ref tokens::Token[] stream = tokens::valid(ref page);
    var evaluator::Environment environment = evaluator::empty();
    discard switch(evaluator::evaluate(source, stream, ref mut environment)) {
        case .ok(count): { count }
        case .err(code): { return error(code); }
    };
    var bytes::Buffer output = bytes::empty();
    let ref evaluator::Binding[] bindings = evaluator::valid(ref environment);
    for (let ref evaluator::Binding binding in bindings) {
        let ref u8[] name = source[binding.name.start..binding.name.end];
        if (!bytes::append(ref mut output, name).is_ok || !sum_row(ref mut output, b"\t", binding.value)) { return error(5); }
    }
    let ref u8[] result = bytes::view(ref output);
    return if (switch(io::write_bytes(result)) { case .ok(count): { count == result.len } case .err(_): { false } }) { 0 } else { error(7) };
}
@effects(console)
fn i32 main() {
    let usize argc = process::arg_count();
    if (argc == 1) { return analyze_source(b"price = 4000000000 + fee; other = 7;"); }
    if (argc != 2 && argc != 3) { return error(0); }
    var bytes::Buffer input = bytes::empty();
    if (!bytes::resize_zeroed(ref mut input, 4096).is_ok) { return error(5); }
    let usize length = switch(process::arg(1, bytes::view_mut(ref mut input))) {
        case .ok(count): { count }
        case .err(_): { return error(if (argc == 3) { 0 } else { 6 }); }
    };
    if (length > 4095) { return error(if (argc == 3) { 0 } else { 6 }); }
    let ref u8[] source = bytes::view(ref input)[..length];
    if (equal(source, b"--eval")) {
        if (argc != 3) { return error(0); }
        let usize eval_length = switch(process::arg(2, bytes::view_mut(ref mut input))) {
            case .ok(count): { count }
            case .err(_): { return error(6); }
        };
        if (eval_length > 4095) { return error(6); }
        return evaluate_source(bytes::view(ref input)[..eval_length]);
    }
    if (argc != 2) { return error(0); }
    if (equal(source, b"--help")) {
        return if (io::write("usage: token-page-cli [ASCII-source]\n       token-page-cli --eval <ASCII-source>\n").is_ok) { 0 } else { error(7) };
    }
    return analyze_source(source);
}
