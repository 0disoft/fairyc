module syntax;
import scanner as scanner;
import std::vec as v;

// Private candidate categories: 1 module, 2 import, 3 function; problem 4 is
// outside the admitted subset. These are not public compiler diagnostic IDs.
pub struct Declaration { u32 kind; usize start; usize end; usize name_start; usize name_end; usize expression; }
// One-based child IDs; zero means absent. Nodes are postorder, with byte spans
// and the source token index retaining exact literal/name/operator spelling.
pub struct Expression { u32 kind; usize start; usize end; usize token; usize left; usize right; }
pub struct Problem { u32 code; usize start; usize end; }
pub resource Parsed {
    scanner::Stream source;
    v::Vec<Declaration> declarations;
    v::Vec<Problem> problems;
    v::Vec<Expression> expressions;
    cleanup { }
}
struct Cursor { usize index; usize previous_end; usize nodes; i32 status; }
struct Attempt { bool ok; Declaration value; Problem problem; }

fn scanner::TokenInfo current(ref scanner::Stream stream, ref mut Cursor cursor) {
    return scanner::token_info(stream, cursor.index);
}
fn void trivia(ref scanner::Stream stream, ref mut Cursor cursor) {
    while (scanner::token_info(stream, cursor.index).kind == 2 && cursor.index + 1 < scanner::token_count(stream)) {
        cursor.index = cursor.index + 1;
    }
}
fn void advance(ref scanner::Stream stream, ref mut Cursor cursor) {
    cursor.previous_end = scanner::token_info(stream, cursor.index).end;
    if (cursor.index + 1 < scanner::token_count(stream)) { cursor.index = cursor.index + 1; }
    trivia(stream, cursor);
}
fn bool word(ref scanner::Stream stream, usize index, ref u8[] expected) {
    let ref u8[] raw = scanner::token_raw(stream, index);
    if (raw.len != expected.len) { return false; }
    var usize byte_index = 0;
    while (byte_index < raw.len) {
        if (raw[byte_index] != expected[byte_index]) { return false; }
        byte_index += 1;
    }
    return true;
}
fn bool take(ref scanner::Stream stream, ref mut Cursor cursor, ref u8[] expected) {
    if (!word(stream, cursor.index, expected)) { return false; }
    advance(stream, cursor);
    return true;
}
fn Attempt failed(ref scanner::Stream stream, ref mut Cursor cursor, u32 code) {
    let scanner::TokenInfo token = current(stream, cursor);
    return Attempt { .ok = false,
        .value = Declaration { .kind = 0, .start = 0, .end = 0, .name_start = 0, .name_end = 0, .expression = 0 },
        .problem = Problem { .code = code, .start = token.start, .end = token.end } };
}
fn usize precedence(ref scanner::Stream stream, usize index) {
    if (word(stream, index, b"||")) { return 1; }
    if (word(stream, index, b"&&")) { return 2; }
    if (word(stream, index, b"==") || word(stream, index, b"!=") || word(stream, index, b"<") || word(stream, index, b"<=") || word(stream, index, b">") || word(stream, index, b">=")) { return 3; }
    if (word(stream, index, b"|")) { return 4; }
    if (word(stream, index, b"^")) { return 5; }
    if (word(stream, index, b"&")) { return 6; }
    if (word(stream, index, b"<<") || word(stream, index, b">>")) { return 7; }
    if (word(stream, index, b"+") || word(stream, index, b"-")) { return 8; }
    if (word(stream, index, b"*") || word(stream, index, b"/") || word(stream, index, b"%")) { return 9; }
    return 0;
}
fn usize append(ref mut v::Vec<Expression> nodes, ref mut Cursor cursor, Expression node) {
    if (cursor.nodes >= 4096) { cursor.status = 4; return 0; }
    if (!switch (v::push<Expression>(nodes, node)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return 0; }
    cursor.nodes = cursor.nodes + 1;
    return cursor.nodes;
}
fn usize expression(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Expression> nodes, usize minimum, usize depth) {
    if (depth >= 64) { cursor.status = 4; return 0; }
    let usize token_index = cursor.index;
    let scanner::TokenInfo token = current(stream, cursor);
    var usize left = 0;
    if (token.kind == 5 || token.kind == 3) {
        advance(stream, cursor);
        left = append(nodes, cursor, Expression { .kind = if (token.kind == 5) { 1 } else { 2 }, .start = token.start, .end = token.end, .token = token_index, .left = 0, .right = 0 });
    } else if (take(stream, cursor, b"(")) {
        let usize child = expression(stream, cursor, nodes, 1, depth + 1);
        if (child == 0 || !take(stream, cursor, b")")) { return 0; }
        left = append(nodes, cursor, Expression { .kind = 3, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = child, .right = 0 });
    } else if (word(stream, cursor.index, b"-") || word(stream, cursor.index, b"+") || word(stream, cursor.index, b"!") || word(stream, cursor.index, b"~") || word(stream, cursor.index, b"*")) {
        advance(stream, cursor);
        let usize child = expression(stream, cursor, nodes, 10, depth + 1);
        if (child == 0) { return 0; }
        left = append(nodes, cursor, Expression { .kind = 4, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = child, .right = 0 });
    } else { return 0; }
    var bool comparison = false;
    while (left != 0) {
        let usize power = precedence(stream, cursor.index);
        if (power < minimum) { return left; }
        if (power == 3 && comparison) { return 0; }
        if (power == 3) { comparison = true; }
        let usize operator = cursor.index;
        advance(stream, cursor);
        let usize right = expression(stream, cursor, nodes, power + 1, depth + 1);
        if (right == 0) { return 0; }
        left = append(nodes, cursor, Expression { .kind = 5, .start = token.start, .end = cursor.previous_end, .token = operator, .left = left, .right = right });
    }
    return 0;
}
fn Attempt declaration(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Expression> nodes, u32 kind) {
    let usize start = scanner::token_info(stream, cursor.index).start;
    advance(stream, cursor);
    if (kind == 3) {
        if (!word(stream, cursor.index, b"void") && !word(stream, cursor.index, b"i32")) { return failed(stream, cursor, kind); }
        advance(stream, cursor);
    }
    let scanner::TokenInfo name = scanner::token_info(stream, cursor.index);
    if (name.kind != 3) { return failed(stream, cursor, kind); }
    var usize name_start = name.start; var usize name_end = name.end;
    var usize root = 0;
    advance(stream, cursor);
    if (kind != 3) {
        while (take(stream, cursor, b"::")) {
            let scanner::TokenInfo part = scanner::token_info(stream, cursor.index);
            if (part.kind != 3) { return failed(stream, cursor, kind); }
            name_end = part.end; advance(stream, cursor);
        }
        if (kind == 2) {
            if (!take(stream, cursor, b"as")) { return failed(stream, cursor, kind); }
            let scanner::TokenInfo alias = scanner::token_info(stream, cursor.index);
            if (alias.kind != 3) { return failed(stream, cursor, kind); }
            name_start = alias.start; name_end = alias.end; advance(stream, cursor);
        }
        if (!take(stream, cursor, b";")) { return failed(stream, cursor, kind); }
    } else {
        if (!take(stream, cursor, b"(") || !take(stream, cursor, b")") || !take(stream, cursor, b"{")) { return failed(stream, cursor, kind); }
        if (!word(stream, cursor.index, b"}")) {
            if (!take(stream, cursor, b"return")) { return failed(stream, cursor, kind); }
            root = expression(stream, cursor, nodes, 1, 0);
            if (root == 0) { return failed(stream, cursor, kind); }
            if (!take(stream, cursor, b";")) { return failed(stream, cursor, kind); }
        }
        if (!take(stream, cursor, b"}")) { return failed(stream, cursor, kind); }
    }
    return Attempt { .ok = true,
        .value = Declaration { .kind = kind, .start = start, .end = cursor.previous_end, .name_start = name_start, .name_end = name_end, .expression = root },
        .problem = Problem { .code = 0, .start = 0, .end = 0 } };
}

// Restart at the failed declaration, then consume with brace depth. Never accept
// a nested function as the next top-level declaration or repeat a zero-width step.
fn void recover(ref scanner::Stream stream, ref mut Cursor cursor, usize start) {
    cursor.index = start;
    var usize depth = 0;
    while (scanner::token_info(stream, cursor.index).kind != 1) {
        if (depth == 0 && cursor.index > start && (word(stream, cursor.index, b"fn") || word(stream, cursor.index, b"import"))) { return; }
        if (word(stream, cursor.index, b"{")) { depth += 1; }
        else if (word(stream, cursor.index, b"}")) {
            advance(stream, cursor);
            if (depth == 0) { return; }
            depth -= 1;
            if (depth == 0) { return; }
            continue;
        } else if (depth == 0 && word(stream, cursor.index, b";")) { advance(stream, cursor); return; }
        advance(stream, cursor);
    }
}

// Errors 3 allocation, 4 candidate item/diagnostic bound, 5 lexical diagnostics.
// Syntax problems return an owned incomplete result, never an executable program.
pub fn Result<Parsed, i32> parse(scanner::Stream input, usize max_items) {
    if (scanner::diagnostic_count(ref input) != 0) { return .err(5); }
    if (max_items > 512) { return .err(4); }
    var v::Vec<Declaration> declarations = v::empty<Declaration>();
    var v::Vec<Problem> problems = v::empty<Problem>();
    var v::Vec<Expression> nodes = v::empty<Expression>();
    var Cursor cursor = Cursor { .index = 0, .previous_end = 0, .nodes = 0, .status = 0 };
    trivia(ref input, ref mut cursor);
    var bool first = true; var bool functions = false; var usize items = 0; var usize errors = 0;
    while (first || scanner::token_info(ref input, cursor.index).kind != 1) {
        let usize start = cursor.index;
        let usize node_start = v::len<Expression>(ref nodes);
        let u32 kind = if (first) { 1 } else if (!functions && word(ref input, cursor.index, b"import")) { 2 } else if (word(ref input, cursor.index, b"fn")) { 3 } else { 4 };
        let Attempt attempt = if ((first && !word(ref input, cursor.index, b"module")) || kind == 4) { failed(ref input, ref mut cursor, kind) } else { declaration(ref input, ref mut cursor, ref mut nodes, kind) };
        if (cursor.status != 0) { return .err(cursor.status); }
        if (attempt.ok) {
            if (items >= max_items) { return .err(4); }
            if (!switch (v::push<Declaration>(ref mut declarations, attempt.value)) { case .ok(_): { true } case .err(_): { false } }) { return .err(3); }
            items += 1;
        } else {
            discard v::truncate<Expression>(ref mut nodes, node_start);
            cursor.nodes = node_start;
            if (errors >= 32) { return .err(4); }
            if (!switch (v::push<Problem>(ref mut problems, attempt.problem)) { case .ok(_): { true } case .err(_): { false } }) { return .err(3); }
            errors += 1;
            if (!first || word(ref input, start, b"module")) { recover(ref input, ref mut cursor, start); }
        }
        if (kind == 3) { functions = true; }
        first = false;
    }
    return .ok(Parsed { .source = input, .declarations = declarations, .problems = problems, .expressions = nodes });
}
pub fn usize declaration_count(ref Parsed parsed) { return v::len<Declaration>(ref parsed.declarations); }
pub fn usize problem_count(ref Parsed parsed) { return v::len<Problem>(ref parsed.problems); }
pub fn Declaration declaration_at(ref Parsed parsed, usize index) { return *v::at<Declaration>(ref parsed.declarations, index); }
pub fn Problem problem_at(ref Parsed parsed, usize index) { return *v::at<Problem>(ref parsed.problems, index); }
pub fn ref u8[] source_view(ref Parsed parsed) { return scanner::source_view(ref parsed.source); }
pub fn usize expression_count(ref Parsed parsed) { return v::len<Expression>(ref parsed.expressions); }
pub fn Expression expression_at(ref Parsed parsed, usize id) { return *v::at<Expression>(ref parsed.expressions, id - 1); }
pub fn ref u8[] expression_text(ref Parsed parsed, usize id) {
    return scanner::token_raw(ref parsed.source, v::at<Expression>(ref parsed.expressions, id - 1).token);
}
