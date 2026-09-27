module file_syntax;
import scanner as scanner;
import syntax as syntax;
import parse_tokens as tokens;
import std::vec as v;

struct Segment { usize start; usize end; }

fn bool separator(u8 byte) { return byte == 47 || byte == 92; }
fn bool initial(u8 byte) { return byte == 95 || (byte >= 65 && byte <= 90) || (byte >= 97 && byte <= 122); }

// Caller-supplied source-root-relative identity, never a filesystem operation.
// Like source.File: slash/backslash, empty/dot segments and lexical parent cleanup.
// Errors: 4 private path budget, 6 invalid identity, 7 module mismatch.
fn i32 identity(ref scanner::Stream input, ref u8[] path) {
    if (path.len > 32768) { return 4; }
    if (path.len == 0 || separator(path[0])) { return 6; }
    var v::Vec<Segment> segments = v::empty<Segment>();
    var usize count = 0; var usize index = 0;
    while (index < path.len) {
        let usize start = index;
        while (index < path.len && !separator(path[index])) { index += 1; }
        let usize end = index;
        if (index < path.len) { index += 1; }
        if (end == start || (end - start == 1 && path[start] == 46)) { continue; }
        if (end - start == 2 && path[start] == 46 && path[start + 1] == 46) {
            if (count == 0) { return 6; }
            count -= 1; discard v::truncate<Segment>(ref mut segments,count); continue;
        }
        if (count >= 1024) { return 4; }
        if (!v::push<Segment>(ref mut segments,Segment { .start = start, .end = end }).is_ok) { return 3; }
        count += 1;
    }
    if (count == 0) { return 6; }
    let usize last = count - 1; let usize end = v::at<Segment>(ref segments,last).end;
    if (end - v::at<Segment>(ref segments,last).start < 4 || path[end - 3] != 46 || path[end - 2] != 102 || path[end - 1] != 121) { return 6; }
    index = 0;
    while (index < count) {
        var usize byte = v::at<Segment>(ref segments,index).start;
        let usize part_end = if (index == last) { end - 3 } else { v::at<Segment>(ref segments,index).end };
        if (!initial(path[byte])) { return 6; }
        byte += 1;
        while (byte < part_end) {
            if (!initial(path[byte]) && (path[byte] < 48 || path[byte] > 57)) { return 6; }
            byte += 1;
        }
        index += 1;
    }
    var usize token = tokens::significant(input,0);
    if (!tokens::word(input,token,b"module")) { return 7; }
    token = tokens::significant(input,token + 1);
    index = 0;
    while (index < count) {
        let usize part_start = v::at<Segment>(ref segments,index).start;
        let usize part_end = if (index == last) { end - 3 } else { v::at<Segment>(ref segments,index).end };
        if (!tokens::word(input,token,path[part_start..part_end])) { return 7; }
        token = tokens::significant(input,token + 1);
        index += 1;
        if (index < count) {
            if (!tokens::word(input,token,b"::")) { return 7; }
            token = tokens::significant(input,token + 1);
        }
    }
    return if (tokens::word(input,token,b";")) { 0 } else { 7 };
}

// The byte-only syntax::parse contract stays unchanged. Invalid paths fail first;
// lexical and syntax failures take precedence over a module mismatch.
fn Result<syntax::Parsed,i32> checked(syntax::Parsed parsed, i32 status) {
    if (status != 0 && syntax::problem_count(ref parsed) == 0) { return .err(status); }
    return .ok(parsed);
}
pub fn Result<syntax::Parsed,i32> parse(scanner::Stream input, usize max_items, ref u8[] relative_path) {
    let i32 status = identity(ref input,relative_path);
    if (status == 3 || status == 4 || status == 6) { return .err(status); }
    return switch (syntax::parse(input,max_items)) {
        case .ok(parsed): { checked(parsed,status) }
        case .err(error): { .err(error) }
    };
}
