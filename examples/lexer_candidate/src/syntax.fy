module syntax;
import scanner as scanner;
import std::vec as v;
import types as types;
import records as records;

// Private declarations: 1 module, 2 import, 3 function, 4 struct; problem 4 is
// outside the admitted subset. These are not public compiler diagnostic IDs.
pub struct Declaration { u32 kind; usize start; usize end; usize name_start; usize name_end; usize expression; usize entry_start; usize entry_count; usize body; usize result_type; usize field_start; usize field_count; bool public; usize public_start; usize public_end; }
// Function entries in source order: 1 parameter, 2 let, 3 var, 4 return.
// Zero name/type spans mean unwritten; expression zero means no initializer/value.
pub struct Entry { u32 kind; usize start; usize end; usize name_start; usize name_end; usize type_start; usize type_end; usize type_root; usize expression; }
// One-based child IDs; zero means absent. Nodes are postorder, with byte spans
// and the source token index retaining exact literal/name/operator spelling.
pub struct Expression { u32 kind; usize start; usize end; usize token; usize left; usize right; }
pub struct Problem { u32 code; usize start; usize end; u32 expected; usize token; }
// 1 block, 2 binding, 3 return, 4 assignment, 5 if, 6 expression, 7 list link.
// 8 while (expression condition, first body), 9 break, 10 continue.
pub struct Statement { u32 kind; usize start; usize end; usize expression; usize target; usize first; usize second; usize entry; usize operator; }
pub resource Parsed {
    scanner::Stream source;
    v::Vec<Declaration> declarations;
    v::Vec<Problem> problems;
    v::Vec<Expression> expressions;
    v::Vec<Entry> entries;
    v::Vec<Statement> statements;
    v::Vec<types::Node> types;
    v::Vec<records::Field> fields;
    cleanup { }
}
struct Cursor { usize index; usize previous_end; usize nodes; usize entries; usize statements; usize types; usize fields; usize expression; usize top_return; i32 status; u32 expected; }
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
fn u32 item_kind(ref scanner::Stream stream, usize index) {
    var usize next = index;
    if (word(stream,next,b"pub")) {
        next += 1;
        while (next < scanner::token_count(stream) && scanner::token_info(stream,next).kind == 2) { next += 1; }
    }
    if (next >= scanner::token_count(stream)) { return 5; }
    if (word(stream,next,b"fn")) { return 3; }
    if (word(stream,next,b"struct")) { return 4; }
    return 5;
}
fn Attempt failed(ref scanner::Stream stream, ref mut Cursor cursor, u32 code) {
    let scanner::TokenInfo token = current(stream, cursor);
    return Attempt { .ok = false,
        .value = Declaration { .kind = 0, .start = 0, .end = 0, .name_start = 0, .name_end = 0, .expression = 0, .entry_start = 0, .entry_count = 0, .body = 0, .result_type = 0, .field_start = 0, .field_count = 0, .public = false, .public_start = 0, .public_end = 0 },
        .problem = Problem { .code = code, .start = token.start, .end = token.end, .expected = cursor.expected, .token = cursor.index } };
}
fn bool expect(ref scanner::Stream stream, ref mut Cursor cursor, ref u8[] text, u32 code) {
    if (take(stream, cursor, text)) { return true; }
    cursor.expected = code;
    return false;
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
    if (token.kind == 5 || token.kind == 3 || word(stream, cursor.index, b"true") || word(stream, cursor.index, b"false")) {
        advance(stream, cursor);
        if (token.kind == 3) {
            while (take(stream, cursor, b"::")) {
                if (current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
                advance(stream, cursor);
            }
        }
        left = append(nodes, cursor, Expression { .kind = if (token.kind == 3) { 2 } else { 1 }, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = 0, .right = 0 });
    } else if (take(stream, cursor, b"(")) {
        let usize child = expression(stream, cursor, nodes, 1, depth + 1);
        if (child == 0 || !expect(stream, cursor, b")", 5)) { return 0; }
        left = append(nodes, cursor, Expression { .kind = 3, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = child, .right = 0 });
    } else if (word(stream, cursor.index, b"-") || word(stream, cursor.index, b"+") || word(stream, cursor.index, b"!") || word(stream, cursor.index, b"~") || word(stream, cursor.index, b"*")) {
        advance(stream, cursor);
        let usize child = expression(stream, cursor, nodes, 10, depth + 1);
        if (child == 0) { return 0; }
        left = append(nodes, cursor, Expression { .kind = 4, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = child, .right = 0 });
    } else { cursor.expected = 3; return 0; }
    var bool comparison = false;
    while (left != 0) {
        let usize opener = cursor.index;
        if (take(stream, cursor, b".")) {
            let usize member = cursor.index;
            if (current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
            advance(stream, cursor);
            left = append(nodes, cursor, Expression { .kind = 8, .start = token.start, .end = cursor.previous_end, .token = member, .left = left, .right = 0 });
            continue;
        }
        if (take(stream, cursor, b"[")) {
            let usize index = expression(stream, cursor, nodes, 1, depth + 1);
            if (index == 0 || !expect(stream, cursor, b"]", 11)) { return 0; }
            left = append(nodes, cursor, Expression { .kind = 9, .start = token.start, .end = cursor.previous_end, .token = opener, .left = left, .right = index });
            continue;
        }
        if (take(stream, cursor, b"(")) {
            var usize arguments = 0;
            while (!word(stream, cursor.index, b")")) {
                let usize argument_token = cursor.index;
                let usize argument_start = current(stream, cursor).start;
                let usize value = expression(stream, cursor, nodes, 1, depth + 1);
                if (value == 0) { return 0; }
                arguments = append(nodes, cursor, Expression { .kind = 7, .start = argument_start, .end = cursor.previous_end, .token = argument_token, .left = value, .right = arguments });
                if (arguments == 0) { return 0; }
                if (!take(stream, cursor, b",")) { break; }
            }
            if (!expect(stream, cursor, b")", 5)) { return 0; }
            left = append(nodes, cursor, Expression { .kind = 6, .start = token.start, .end = cursor.previous_end, .token = opener, .left = left, .right = arguments });
            continue;
        }
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
fn bool inferred_binding(ref scanner::Stream stream, usize index) {
    if (scanner::token_info(stream,index).kind != 3) { return false; }
    var usize next = index + 1;
    while (next < scanner::token_count(stream) && scanner::token_info(stream,next).kind == 2) { next += 1; }
    return next < scanner::token_count(stream) && word(stream,next,b"=");
}
fn usize read_type(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<types::Node> nodes) {
    let types::Attempt result = types::parse(stream,cursor.index,cursor.types,nodes);
    cursor.index = result.next; cursor.previous_end = result.end; cursor.types = result.count;
    cursor.status = result.status; cursor.expected = result.expected;
    return result.root;
}
fn bool entry(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Expression> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<types::Node> type_nodes, u32 kind) {
    if (cursor.entries >= 1024) { cursor.status = 4; return false; }
    let usize start = current(stream, cursor).start;
    var usize name_start = 0; var usize name_end = 0;
    var usize type_start = 0; var usize type_end = 0; var usize root = 0; var usize type_root = 0;
    if (kind != 1) { advance(stream, cursor); }
    if (kind != 4) {
        if (kind == 1 || !inferred_binding(stream, cursor.index)) {
            type_start = current(stream, cursor).start;
            type_root = read_type(stream, cursor, type_nodes);
            if (type_root == 0) { return false; }
            type_end = cursor.previous_end;
        }
        let scanner::TokenInfo name = current(stream, cursor);
        if (name.kind != 3) { cursor.expected = 1; return false; }
        name_start = name.start; name_end = name.end; advance(stream, cursor);
        if (kind != 1 && !expect(stream, cursor, b"=", 9)) { return false; }
    }
    if (kind != 1) {
        if (kind != 4 || !word(stream, cursor.index, b";")) {
            root = expression(stream, cursor, nodes, 1, 0);
            if (root == 0) { return false; }
        }
        if (!expect(stream, cursor, b";", 8)) { return false; }
    }
    let Entry value = Entry { .kind = kind, .start = start, .end = cursor.previous_end, .name_start = name_start, .name_end = name_end, .type_start = type_start, .type_end = type_end, .type_root = type_root, .expression = root };
    if (!switch (v::push<Entry>(entries, value)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return false; }
    cursor.entries = cursor.entries + 1;
    cursor.expression = root;
    return true;
}
fn Statement statement_value(u32 kind, usize start, usize end) {
    return Statement { .kind = kind, .start = start, .end = end, .expression = 0, .target = 0, .first = 0, .second = 0, .entry = 0, .operator = 0 };
}
fn usize append_statement(ref mut v::Vec<Statement> statements, ref mut Cursor cursor, Statement value) {
    if (cursor.statements >= 4096) { cursor.status = 4; return 0; }
    if (!switch (v::push<Statement>(statements, value)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return 0; }
    cursor.statements = cursor.statements + 1;
    return cursor.statements;
}
fn bool assignment(ref scanner::Stream stream, usize index) {
    return word(stream, index, b"=") || word(stream, index, b"+=") || word(stream, index, b"-=") || word(stream, index, b"*=") || word(stream, index, b"/=") || word(stream, index, b"%=") || word(stream, index, b"<<=") || word(stream, index, b">>=") || word(stream, index, b"&=") || word(stream, index, b"^=") || word(stream, index, b"|=");
}
fn usize block(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Expression> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth) {
    if (depth >= 32) { cursor.status = 4; return 0; }
    let usize start = current(stream, cursor).start;
    if (!expect(stream, cursor, b"{", 6)) { return 0; }
    var usize tail = 0; var usize count = 0;
    while (!word(stream, cursor.index, b"}")) {
        if (current(stream, cursor).kind == 1) { cursor.expected = 7; return 0; }
        let usize child_start = current(stream, cursor).start;
        let usize child = statement(stream, cursor, nodes, entries, statements, type_nodes, depth);
        if (child == 0) { return 0; }
        var Statement link = statement_value(7, child_start, cursor.previous_end);
        link.first = child; link.second = tail;
        tail = append_statement(statements, cursor, link);
        if (tail == 0) { return 0; }
        count += 1;
    }
    advance(stream, cursor);
    var Statement value = statement_value(1, start, cursor.previous_end);
    value.first = tail; value.second = count;
    return append_statement(statements, cursor, value);
}
fn usize statement(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Expression> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth) {
    if (depth >= 32) { cursor.status = 4; return 0; }
    let usize start = current(stream, cursor).start;
    if (word(stream, cursor.index, b"{")) { return block(stream, cursor, nodes, entries, statements, type_nodes, depth + 1); }
    var Statement value = statement_value(0, start, 0);
    if (word(stream, cursor.index, b"break") || word(stream, cursor.index, b"continue")) {
        value.kind = if (word(stream, cursor.index, b"break")) { 9 } else { 10 };
        advance(stream, cursor);
        if (!expect(stream, cursor, b";", 8)) { return 0; }
    } else if (take(stream, cursor, b"while")) {
        value.kind = 8;
        if (!expect(stream, cursor, b"(", 4)) { return 0; }
        value.expression = expression(stream, cursor, nodes, 1, 0);
        if (value.expression == 0 || !expect(stream, cursor, b")", 5)) { return 0; }
        value.first = block(stream, cursor, nodes, entries, statements, type_nodes, depth + 1);
        if (value.first == 0) { return 0; }
    } else if (take(stream, cursor, b"if")) {
        value.kind = 5;
        if (!expect(stream, cursor, b"(", 4)) { return 0; }
        value.expression = expression(stream, cursor, nodes, 1, 0);
        if (value.expression == 0 || !expect(stream, cursor, b")", 5)) { return 0; }
        value.first = block(stream, cursor, nodes, entries, statements, type_nodes, depth + 1);
        if (value.first == 0) { return 0; }
        if (take(stream, cursor, b"else")) {
            if (word(stream, cursor.index, b"if")) { value.second = statement(stream, cursor, nodes, entries, statements, type_nodes, depth + 1); }
            else { value.second = block(stream, cursor, nodes, entries, statements, type_nodes, depth + 1); }
            if (value.second == 0) { return 0; }
        }
    } else {
        let u32 kind = if (word(stream, cursor.index, b"let")) { 2 } else if (word(stream, cursor.index, b"var")) { 3 } else if (word(stream, cursor.index, b"return")) { 4 } else { 0 };
        if (kind != 0) {
            if (!entry(stream, cursor, nodes, entries, type_nodes, kind)) { return 0; }
            value.kind = if (kind == 4) { 3 } else { 2 };
            value.entry = cursor.entries; value.expression = cursor.expression;
            if (kind == 4 && depth == 0) { cursor.top_return = cursor.expression; }
        } else {
            let usize left = expression(stream, cursor, nodes, 1, 0);
            if (left == 0) { return 0; }
            value.kind = 6; value.expression = left;
            if (assignment(stream, cursor.index)) {
                value.kind = 4; value.target = left; value.operator = cursor.index;
                advance(stream, cursor);
                value.expression = expression(stream, cursor, nodes, 1, 0);
                if (value.expression == 0) { return 0; }
            }
            if (!expect(stream, cursor, b";", 8)) { return 0; }
        }
    }
    value.end = cursor.previous_end;
    return append_statement(statements, cursor, value);
}
fn Attempt declaration(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Expression> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, ref mut v::Vec<records::Field> fields, u32 kind) {
    let usize start = scanner::token_info(stream, cursor.index).start;
    let usize entry_start = cursor.entries;
    let bool public = kind >= 3 && take(stream, cursor, b"pub");
    let usize field_start = cursor.fields;
    advance(stream, cursor);
    var usize result_type = 0;
    if (kind == 3) {
        result_type = read_type(stream, cursor, type_nodes);
        if (result_type == 0) { return failed(stream, cursor, kind); }
    }
    let scanner::TokenInfo name = scanner::token_info(stream, cursor.index);
    if (name.kind != 3) { cursor.expected = 1; return failed(stream, cursor, kind); }
    var usize name_start = name.start; var usize name_end = name.end;
    var usize root = 0;
    var usize body = 0;
    advance(stream, cursor);
    if (kind < 3) {
        while (take(stream, cursor, b"::")) {
            let scanner::TokenInfo part = scanner::token_info(stream, cursor.index);
            if (part.kind != 3) { cursor.expected = 1; return failed(stream, cursor, kind); }
            name_end = part.end; advance(stream, cursor);
        }
        if (kind == 2) {
            if (!expect(stream, cursor, b"as", 10)) { return failed(stream, cursor, kind); }
            let scanner::TokenInfo alias = scanner::token_info(stream, cursor.index);
            if (alias.kind != 3) { cursor.expected = 1; return failed(stream, cursor, kind); }
            name_start = alias.start; name_end = alias.end; advance(stream, cursor);
        }
        if (!expect(stream, cursor, b";", 8)) { return failed(stream, cursor, kind); }
    } else if (kind == 4) {
        let records::Attempt record = records::parse(stream,cursor.index,cursor.types,cursor.fields,type_nodes,fields);
        cursor.index = record.next; cursor.previous_end = record.end; cursor.types = record.types; cursor.fields = record.fields;
        cursor.expected = record.expected; cursor.status = record.status;
        if (!record.ok) { return failed(stream,cursor,4); }
    } else {
        if (!expect(stream, cursor, b"(", 4)) { return failed(stream, cursor, kind); }
        while (!word(stream, cursor.index, b")")) {
            if (!entry(stream, cursor, nodes, entries, type_nodes, 1)) { return failed(stream, cursor, kind); }
            if (!take(stream, cursor, b",")) { break; }
        }
        if (!expect(stream, cursor, b")", 5)) { return failed(stream, cursor, kind); }
        cursor.top_return = 0;
        body = block(stream, cursor, nodes, entries, statements, type_nodes, 0);
        if (body == 0) { return failed(stream, cursor, kind); }
        root = cursor.top_return;
    }
    return Attempt { .ok = true,
        .value = Declaration { .kind = kind, .start = start, .end = cursor.previous_end, .name_start = name_start, .name_end = name_end, .expression = root, .entry_start = entry_start, .entry_count = cursor.entries - entry_start, .body = body, .result_type = result_type, .field_start = field_start, .field_count = cursor.fields - field_start, .public = public, .public_start = if (public) { start } else { 0 }, .public_end = if (public) { start + 3 } else { 0 } },
        .problem = Problem { .code = 0, .start = 0, .end = 0, .expected = 0, .token = 0 } };
}

// Restart at the failed declaration, then consume with brace depth. Never accept
// a nested function as the next top-level declaration or repeat a zero-width step.
fn void recover(ref scanner::Stream stream, ref mut Cursor cursor, usize start) {
    cursor.index = start;
    if (take(stream, cursor, b"pub")) {
        if (word(stream, cursor.index, b"fn") || word(stream, cursor.index, b"struct") || word(stream, cursor.index, b"import")) { advance(stream, cursor); }
    }
    var usize depth = 0;
    while (scanner::token_info(stream, cursor.index).kind != 1) {
        if (depth == 0 && cursor.index > start && (word(stream, cursor.index, b"fn") || item_kind(stream, cursor.index) < 5 || word(stream, cursor.index, b"import"))) { return; }
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
    var v::Vec<Entry> entries = v::empty<Entry>();
    var v::Vec<Statement> statements = v::empty<Statement>();
    var v::Vec<types::Node> type_nodes = v::empty<types::Node>();
    var v::Vec<records::Field> fields = v::empty<records::Field>();
    var Cursor cursor = Cursor { .index = 0, .previous_end = 0, .nodes = 0, .entries = 0, .statements = 0, .types = 0, .fields = 0, .expression = 0, .top_return = 0, .status = 0, .expected = 0 };
    trivia(ref input, ref mut cursor);
    var bool first = true; var bool functions = false; var usize items = 0; var usize errors = 0;
    while (first || scanner::token_info(ref input, cursor.index).kind != 1) {
        cursor.expected = if (first) { 12 } else { 0 };
        let usize start = cursor.index;
        let usize node_start = v::len<Expression>(ref nodes);
        let usize entry_start = cursor.entries;
        let usize statement_start = cursor.statements;
        let usize type_start = cursor.types;
        let usize field_start = cursor.fields;
        let u32 kind = if (first) { 1 } else if (!functions && word(ref input, cursor.index, b"import")) { 2 } else { item_kind(ref input,cursor.index) };
        let Attempt attempt = if ((first && !word(ref input, cursor.index, b"module")) || kind == 5) { failed(ref input, ref mut cursor, if (kind == 5) { 4 } else { kind }) } else { declaration(ref input, ref mut cursor, ref mut nodes, ref mut entries, ref mut statements, ref mut type_nodes, ref mut fields, kind) };
        if (cursor.status != 0) { return .err(cursor.status); }
        if (attempt.ok) {
            if (items >= max_items) { return .err(4); }
            if (!switch (v::push<Declaration>(ref mut declarations, attempt.value)) { case .ok(_): { true } case .err(_): { false } }) { return .err(3); }
            items += 1;
        } else {
            discard v::truncate<Expression>(ref mut nodes, node_start);
            cursor.nodes = node_start;
            discard v::truncate<Entry>(ref mut entries, entry_start);
            cursor.entries = entry_start;
            discard v::truncate<Statement>(ref mut statements, statement_start);
            cursor.statements = statement_start;
            discard v::truncate<types::Node>(ref mut type_nodes, type_start);
            cursor.types = type_start;
            discard v::truncate<records::Field>(ref mut fields, field_start);
            cursor.fields = field_start;
            if (errors >= 32) { return .err(4); }
            if (!switch (v::push<Problem>(ref mut problems, attempt.problem)) { case .ok(_): { true } case .err(_): { false } }) { return .err(3); }
            errors += 1;
            if (!first || word(ref input, start, b"module")) { recover(ref input, ref mut cursor, start); }
        }
        if (kind == 3 || kind == 4) { functions = true; }
        first = false;
    }
    return .ok(Parsed { .source = input, .declarations = declarations, .problems = problems, .expressions = nodes, .entries = entries, .statements = statements, .types = type_nodes, .fields = fields });
}
pub fn usize declaration_count(ref Parsed parsed) { return v::len<Declaration>(ref parsed.declarations); }
pub fn usize problem_count(ref Parsed parsed) { return v::len<Problem>(ref parsed.problems); }
pub fn Declaration declaration_at(ref Parsed parsed, usize index) { return *v::at<Declaration>(ref parsed.declarations, index); }
pub fn Problem problem_at(ref Parsed parsed, usize index) { return *v::at<Problem>(ref parsed.problems, index); }
pub fn ref u8[] problem_actual(ref Parsed parsed, usize index) {
    return scanner::token_raw(ref parsed.source, v::at<Problem>(ref parsed.problems, index).token);
}
pub fn ref u8[] expected_text(u32 code) {
    if (code == 1) { return b"identifier"; }
    if (code == 2) { return b"type"; }
    if (code == 3) { return b"expression"; }
    if (code == 4) { return b"("; }
    if (code == 5) { return b")"; }
    if (code == 6) { return b"{"; }
    if (code == 7) { return b"}"; }
    if (code == 8) { return b";"; }
    if (code == 9) { return b"="; }
    if (code == 10) { return b"as"; }
    if (code == 11) { return b"]"; }
    if (code == 12) { return b"module"; }
    if (code == 13) { return b">"; }
    return b"supported syntax";
}
pub fn ref u8[] source_view(ref Parsed parsed) { return scanner::source_view(ref parsed.source); }
pub fn usize expression_count(ref Parsed parsed) { return v::len<Expression>(ref parsed.expressions); }
pub fn usize entry_count(ref Parsed parsed) { return v::len<Entry>(ref parsed.entries); }
pub fn usize statement_count(ref Parsed parsed) { return v::len<Statement>(ref parsed.statements); }
pub fn Statement statement_at(ref Parsed parsed, usize id) { return *v::at<Statement>(ref parsed.statements, id - 1); }
pub fn ref u8[] assignment_operator(ref Parsed parsed, usize id) {
    let Statement value = statement_at(parsed, id); assert(value.kind == 4);
    return scanner::token_raw(ref parsed.source, value.operator);
}
pub fn usize block_statement_count(ref Parsed parsed, usize id) {
    let Statement value = statement_at(parsed, id); assert(value.kind == 1); return value.second;
}
pub fn usize block_statement(ref Parsed parsed, usize id, usize index) {
    let Statement value = statement_at(parsed, id); assert(value.kind == 1 && index < value.second);
    var usize link = value.first; var usize remaining = value.second - 1;
    while (remaining > index) { link = statement_at(parsed, link).second; remaining -= 1; }
    return statement_at(parsed, link).first;
}
pub fn Entry entry_at(ref Parsed parsed, usize index) { return *v::at<Entry>(ref parsed.entries, index); }
pub fn Expression expression_at(ref Parsed parsed, usize id) { return *v::at<Expression>(ref parsed.expressions, id - 1); }
pub fn ref u8[] expression_text(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed, id);
    if (node.kind == 2) { return scanner::source_view(ref parsed.source)[node.start..node.end]; }
    return scanner::token_raw(ref parsed.source, node.token);
}
pub fn usize call_argument_count(ref Parsed parsed, usize call) {
    let Expression node = expression_at(parsed, call); assert(node.kind == 6);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = expression_at(parsed, link).right; }
    return count;
}
pub fn usize call_argument(ref Parsed parsed, usize call, usize index) {
    let usize count = call_argument_count(parsed, call); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed, call).right;
    while (remaining > index) { link = expression_at(parsed, link).right; remaining -= 1; }
    return expression_at(parsed, link).left;
}


pub fn usize type_count(ref Parsed parsed) { return v::len<types::Node>(ref parsed.types); }
pub fn types::Node type_at(ref Parsed parsed, usize id) { return *v::at<types::Node>(ref parsed.types,id - 1); }
pub fn ref u8[] type_text(ref Parsed parsed, usize id) {
    let types::Node node = type_at(parsed,id);
    return scanner::source_view(ref parsed.source)[node.start..node.end];
}


pub fn usize field_count(ref Parsed parsed) { return v::len<records::Field>(ref parsed.fields); }
pub fn records::Field field_at(ref Parsed parsed, usize index) { return *v::at<records::Field>(ref parsed.fields,index); }
pub fn usize type_argument_count(ref Parsed parsed, usize id) {
    let types::Node node = type_at(parsed,id); assert(node.kind == 6);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = type_at(parsed,link).right; }
    return count;
}
pub fn usize type_argument(ref Parsed parsed, usize id, usize index) {
    let usize count = type_argument_count(parsed,id); assert(index < count);
    var usize remaining = count - 1; var usize link = type_at(parsed,id).right;
    while (remaining > index) { link = type_at(parsed,link).right; remaining -= 1; }
    return type_at(parsed,link).left;
}
