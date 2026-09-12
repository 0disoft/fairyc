module syntax;
import scanner as scanner;
import std::vec as v;
import types as types;
import records as records;
import bodies as bodies;
import attributes as attributes;

// Private declarations: 1 module, 2 import, 3 function, 4 struct, 6 resource, 7 const; problem 4 is
// outside the admitted subset. These are not public compiler diagnostic IDs.
pub struct Declaration { u32 kind; usize start; usize end; usize name_start; usize name_end; usize expression; usize entry_start; usize entry_count; usize body; usize result_type; usize field_start; usize field_count; bool public; usize public_start; usize public_end; usize attribute_start; usize attribute_count; usize cleanup_start; usize cleanup_end; usize cleanup_body; }
// Function entries in source order: 1 parameter, 2 let, 3 var, 4 return.
// Zero name/type spans mean unwritten; expression zero means no initializer/value.
pub struct Entry { u32 kind; usize start; usize end; usize name_start; usize name_end; usize type_start; usize type_end; usize type_root; usize expression; }
// One-based child IDs; zero means absent. Nodes are postorder, with byte spans
// and the source token index retaining exact literal/name/operator spelling.
pub struct Expression { u32 kind; usize start; usize end; usize token; usize left; usize right; }
pub struct Problem { u32 code; usize start; usize end; u32 expected; usize token; }
// 1 block, 2 binding, 3 return, 4 assignment, 5 if, 6 expression, 7 list link.
// 8 while (expression condition, first body), 9 break, 10 continue.
// 11 assert and 12 discard retain their operands in expression.
pub struct Statement { u32 kind; usize start; usize end; usize expression; usize target; usize first; usize second; usize entry; usize operator; }
pub resource Parsed {
    scanner::Stream source;
    v::Vec<Declaration> declarations;
    v::Vec<Problem> problems;
    v::Vec<bodies::Node> expressions;
    v::Vec<bodies::Entry> entries;
    v::Vec<bodies::Statement> statements;
    v::Vec<types::Node> types;
    v::Vec<records::Field> fields;
    v::Vec<attributes::Attribute> attributes;
    v::Vec<attributes::Argument> attribute_arguments;
    cleanup { }
}
struct Attempt { bool ok; Declaration value; Problem problem; }

fn scanner::TokenInfo current(ref scanner::Stream stream, ref mut bodies::Cursor cursor) {
    return scanner::token_info(stream, cursor.index);
}
fn void trivia(ref scanner::Stream stream, ref mut bodies::Cursor cursor) {
    while (scanner::token_info(stream, cursor.index).kind == 2 && cursor.index + 1 < scanner::token_count(stream)) {
        cursor.index = cursor.index + 1;
    }
}
fn void advance(ref scanner::Stream stream, ref mut bodies::Cursor cursor) {
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
fn bool take(ref scanner::Stream stream, ref mut bodies::Cursor cursor, ref u8[] expected) {
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
    if (word(stream,next,b"resource")) { return 6; }
    if (word(stream,next,b"const")) { return 7; }
    return 5;
}
fn Attempt failed(ref scanner::Stream stream, ref mut bodies::Cursor cursor, u32 code) {
    let scanner::TokenInfo token = current(stream, cursor);
    return Attempt { .ok = false,
        .value = Declaration { .kind = 0, .start = 0, .end = 0, .name_start = 0, .name_end = 0, .expression = 0, .entry_start = 0, .entry_count = 0, .body = 0, .result_type = 0, .field_start = 0, .field_count = 0, .public = false, .public_start = 0, .public_end = 0, .attribute_start = 0, .attribute_count = 0, .cleanup_start = 0, .cleanup_end = 0, .cleanup_body = 0 },
        .problem = Problem { .code = code, .start = token.start, .end = token.end, .expected = cursor.expected, .token = cursor.index } };
}
fn bool expect(ref scanner::Stream stream, ref mut bodies::Cursor cursor, ref u8[] text, u32 code) {
    if (take(stream, cursor, text)) { return true; }
    cursor.expected = code;
    return false;
}
fn Attempt declaration(ref scanner::Stream stream, ref mut bodies::Cursor cursor, ref mut v::Vec<bodies::Node> nodes, ref mut v::Vec<bodies::Entry> entries, ref mut v::Vec<bodies::Statement> statements, ref mut v::Vec<types::Node> type_nodes, ref mut v::Vec<records::Field> fields, u32 kind) {
    let usize start = scanner::token_info(stream, cursor.index).start;
    let usize entry_start = cursor.entries;
    let bool public = kind >= 3 && take(stream, cursor, b"pub");
    let usize field_start = cursor.fields;
    advance(stream, cursor);
    var usize result_type = 0;
    if (kind == 3 || kind == 7) {
        result_type = bodies::read_type(stream, cursor, type_nodes);
        if (result_type == 0) { return failed(stream, cursor, kind); }
    }
    let scanner::TokenInfo name = scanner::token_info(stream, cursor.index);
    if (name.kind != 3) { cursor.expected = 1; return failed(stream, cursor, kind); }
    var usize name_start = name.start; var usize name_end = name.end;
    var usize root = 0;
    var usize body = 0; var usize cleanup_start = 0; var usize cleanup_end = 0; var usize cleanup_body = 0;
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
    } else if (kind == 7) {
        if (!expect(stream,cursor,b"=",9)) { return failed(stream,cursor,kind); }
        root = bodies::read_expression(stream,cursor,nodes,entries,statements,type_nodes);
        if (root == 0 || !expect(stream,cursor,b";",8)) { return failed(stream,cursor,kind); }
    } else if (kind == 4 || kind == 6) {
        let records::Attempt record = records::parse(stream,cursor.index,cursor.types,cursor.fields,type_nodes,fields,kind == 6);
        cursor.index = record.next; cursor.previous_end = record.end; cursor.types = record.types; cursor.fields = record.fields;
        cursor.expected = record.expected; cursor.status = record.status;
        if (!record.ok) { return failed(stream,cursor,kind); }
        cleanup_start = record.cleanup_start;
        if (cleanup_start != 0) {
            advance(stream,cursor);
            cleanup_body = bodies::block(stream,cursor,nodes,entries,statements,type_nodes,0,false);
            if (cleanup_body == 0) { return failed(stream,cursor,kind); }
            cleanup_end = cursor.previous_end;
            if (!expect(stream,cursor,b"}",7)) { return failed(stream,cursor,kind); }
        }
    } else {
        if (!expect(stream, cursor, b"(", 4)) { return failed(stream, cursor, kind); }
        while (!word(stream, cursor.index, b")")) {
            if (!bodies::entry(stream, cursor, nodes, entries, statements, type_nodes, 1)) { return failed(stream, cursor, kind); }
            if (!take(stream, cursor, b",")) { break; }
        }
        if (!expect(stream, cursor, b")", 5)) { return failed(stream, cursor, kind); }
        cursor.top_return = 0;
        body = bodies::block(stream, cursor, nodes, entries, statements, type_nodes, 0, false);
        if (body == 0) { return failed(stream, cursor, kind); }
        root = cursor.top_return;
    }
    return Attempt { .ok = true,
        .value = Declaration { .kind = kind, .start = start, .end = cursor.previous_end, .name_start = name_start, .name_end = name_end, .expression = root, .entry_start = entry_start, .entry_count = cursor.entries - entry_start, .body = body, .result_type = result_type, .field_start = field_start, .field_count = cursor.fields - field_start, .public = public, .public_start = if (public) { start } else { 0 }, .public_end = if (public) { start + 3 } else { 0 }, .attribute_start = 0, .attribute_count = 0, .cleanup_start = cleanup_start, .cleanup_end = cleanup_end, .cleanup_body = cleanup_body },
        .problem = Problem { .code = 0, .start = 0, .end = 0, .expected = 0, .token = 0 } };
}

// Restart at the failed declaration, then consume with brace depth. Never accept
// a nested function as the next top-level declaration or repeat a zero-width step.
fn void recover(ref scanner::Stream stream, ref mut bodies::Cursor cursor, usize start) {
    cursor.index = start;
    var usize boundary = start;
    if (word(stream,cursor.index,b"@")) {
        while (current(stream,cursor).kind != 1 && !word(stream,cursor.index,b"fn") && !word(stream,cursor.index,b"pub") && !word(stream,cursor.index,b"struct") && !word(stream,cursor.index,b"resource") && !word(stream,cursor.index,b"const") && !word(stream,cursor.index,b"import")) { advance(stream,cursor); }
        boundary = cursor.index;
    }
    if (take(stream, cursor, b"pub")) {
        if (word(stream, cursor.index, b"fn") || word(stream, cursor.index, b"struct") || word(stream, cursor.index, b"resource") || word(stream, cursor.index, b"const") || word(stream, cursor.index, b"import")) { advance(stream, cursor); }
    }
    var usize depth = 0;
    while (scanner::token_info(stream, cursor.index).kind != 1) {
        if (depth == 0 && cursor.index > boundary && (word(stream, cursor.index, b"fn") || item_kind(stream, cursor.index) != 5 || word(stream, cursor.index, b"import") || word(stream,cursor.index,b"@"))) { return; }
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
    var v::Vec<bodies::Node> nodes = v::empty<bodies::Node>();
    var v::Vec<bodies::Entry> entries = v::empty<bodies::Entry>();
    var v::Vec<bodies::Statement> statements = v::empty<bodies::Statement>();
    var v::Vec<types::Node> type_nodes = v::empty<types::Node>();
    var v::Vec<records::Field> fields = v::empty<records::Field>();
    var v::Vec<attributes::Attribute> attribute_nodes = v::empty<attributes::Attribute>();
    var v::Vec<attributes::Argument> attribute_arguments = v::empty<attributes::Argument>();
    var bodies::Cursor cursor = bodies::Cursor { .index = 0, .previous_end = 0, .nodes = 0, .entries = 0, .statements = 0, .types = 0, .fields = 0, .expression = 0, .top_return = 0, .status = 0, .expected = 0, .active_expressions = 0, .block_depth = 0, .tail = 0, .last_entry = 0, .expression_kind = 0 };
    trivia(ref input, ref mut cursor);
    var bool first = true; var bool functions = false; var usize items = 0; var usize errors = 0;
    while (first || scanner::token_info(ref input, cursor.index).kind != 1) {
        cursor.expected = if (first) { 12 } else { 0 };
        let usize start = cursor.index;
        let usize node_start = v::len<bodies::Node>(ref nodes);
        let usize entry_start = cursor.entries;
        let usize statement_start = cursor.statements;
        let usize type_start = cursor.types;
        let usize field_start = cursor.fields;
        let usize attribute_start = v::len<attributes::Attribute>(ref attribute_nodes);
        let usize argument_start = v::len<attributes::Argument>(ref attribute_arguments);
        var bool attributes_ok = true;
        if (!first && word(ref input,cursor.index,b"@")) {
            let attributes::Attempt prefix = attributes::parse(ref input,cursor.index,attribute_start,argument_start,ref mut attribute_nodes,ref mut attribute_arguments);
            cursor.index = prefix.next; cursor.previous_end = prefix.end; cursor.status = prefix.status; cursor.expected = prefix.expected;
            attributes_ok = prefix.ok;
        }
        let usize item_start = cursor.index;
        let usize attribute_count = v::len<attributes::Attribute>(ref attribute_nodes) - attribute_start;
        let u32 kind = if (first) { 1 } else if (attribute_count == 0 && !functions && word(ref input, cursor.index, b"import")) { 2 } else { item_kind(ref input,cursor.index) };
        let Attempt attempt = if (!attributes_ok || (first && !word(ref input, cursor.index, b"module")) || kind == 5) { failed(ref input, ref mut cursor, if (kind == 5) { 4 } else { kind }) } else { declaration(ref input, ref mut cursor, ref mut nodes, ref mut entries, ref mut statements, ref mut type_nodes, ref mut fields, kind) };
        if (cursor.status != 0) { return .err(cursor.status); }
        if (attempt.ok) {
            if (items >= max_items) { return .err(4); }
            var Declaration value = attempt.value;
            value.attribute_start = attribute_start; value.attribute_count = attribute_count;
            if (attribute_count > 0) { value.start = scanner::token_info(ref input,start).start; }
            if (!switch (v::push<Declaration>(ref mut declarations, value)) { case .ok(_): { true } case .err(_): { false } }) { return .err(3); }
            items += 1;
        } else {
            discard v::truncate<attributes::Attribute>(ref mut attribute_nodes,attribute_start);
            discard v::truncate<attributes::Argument>(ref mut attribute_arguments,argument_start);
            discard v::truncate<bodies::Node>(ref mut nodes, node_start);
            cursor.nodes = node_start;
            discard v::truncate<bodies::Entry>(ref mut entries, entry_start);
            cursor.entries = entry_start;
            discard v::truncate<bodies::Statement>(ref mut statements, statement_start);
            cursor.statements = statement_start;
            discard v::truncate<types::Node>(ref mut type_nodes, type_start);
            cursor.types = type_start;
            discard v::truncate<records::Field>(ref mut fields, field_start);
            cursor.fields = field_start;
            if (errors >= 32) { return .err(4); }
            if (!switch (v::push<Problem>(ref mut problems, attempt.problem)) { case .ok(_): { true } case .err(_): { false } }) { return .err(3); }
            errors += 1;
            if (!first || word(ref input, start, b"module")) { recover(ref input, ref mut cursor, if (attributes_ok) { item_start } else { start }); }
        }
        if (kind == 3 || kind == 4 || kind == 6 || kind == 7) { functions = true; }
        first = false;
    }
    return .ok(Parsed { .source = input, .declarations = declarations, .problems = problems, .expressions = nodes, .entries = entries, .statements = statements, .types = type_nodes, .fields = fields, .attributes = attribute_nodes, .attribute_arguments = attribute_arguments });
}
pub fn usize source_token_count(ref Parsed parsed) { return scanner::token_count(ref parsed.source); }
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
    if (code == 14) { return b"."; }
    if (code == 15) { return b"attribute argument"; }
    if (code == 16) { return b"array length"; }
    if (code == 17) { return b":"; }
    if (code == 18) { return b"case"; }
    return b"supported syntax";
}
pub fn ref u8[] source_view(ref Parsed parsed) { return scanner::source_view(ref parsed.source); }
pub fn usize expression_count(ref Parsed parsed) { return v::len<bodies::Node>(ref parsed.expressions); }
pub fn usize entry_count(ref Parsed parsed) { return v::len<bodies::Entry>(ref parsed.entries); }
pub fn usize statement_count(ref Parsed parsed) { return v::len<bodies::Statement>(ref parsed.statements); }
pub fn Statement statement_at(ref Parsed parsed, usize id) {
    let bodies::Statement s = *v::at<bodies::Statement>(ref parsed.statements,id - 1);
    return Statement { .kind = s.kind, .start = s.start, .end = s.end, .expression = s.expression, .target = s.target, .first = s.first, .second = s.second, .entry = s.entry, .operator = s.operator };
}
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
pub fn Entry entry_at(ref Parsed parsed, usize index) {
    let usize count = v::len<bodies::Entry>(ref parsed.entries);
    assert(index < count);
    var usize slot = 0;
    while (slot < count && v::at<bodies::Entry>(ref parsed.entries,slot).id != index + 1) { slot += 1; }
    assert(slot < count);
    let bodies::Entry e = *v::at<bodies::Entry>(ref parsed.entries,slot);
    return Entry { .kind = e.kind, .start = e.start, .end = e.end, .name_start = e.name_start, .name_end = e.name_end, .type_start = e.type_start, .type_end = e.type_end, .type_root = e.type_root, .expression = e.expression };
}
pub fn Expression expression_at(ref Parsed parsed, usize id) {
    let bodies::Node node = *v::at<bodies::Node>(ref parsed.expressions, id - 1);
    return Expression { .kind = node.kind, .start = node.start, .end = node.end, .token = node.token, .left = node.left, .right = node.right };
}
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
pub fn u32 literal_kind(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed, id); assert(node.kind == 1);
    return scanner::token_info(ref parsed.source, node.token).kind;
}
pub fn ref u8[] literal_cooked(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed, id); assert(node.kind == 1);
    return scanner::token_cooked(ref parsed.source, node.token);
}
pub fn usize call_argument(ref Parsed parsed, usize call, usize index) {
    let usize count = call_argument_count(parsed, call); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed, call).right;
    while (remaining > index) { link = expression_at(parsed, link).right; remaining -= 1; }
    return expression_at(parsed, link).left;
}


pub fn usize array_element_count(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed,id); assert(node.kind == 24);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = expression_at(parsed,link).right; }
    return count;
}
pub fn usize array_element(ref Parsed parsed, usize id, usize index) {
    let usize count = array_element_count(parsed,id); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed,id).right;
    while (remaining > index) { link = expression_at(parsed,link).right; remaining -= 1; }
    return expression_at(parsed,link).left;
}
pub fn usize type_count(ref Parsed parsed) { return v::len<types::Node>(ref parsed.types); }
pub fn usize record_field_count(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed, id); assert(node.kind == 10);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = expression_at(parsed, link).right; }
    return count;
}
pub fn usize record_field(ref Parsed parsed, usize id, usize index) {
    let usize count = record_field_count(parsed, id); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed, id).right;
    while (remaining > index) { link = expression_at(parsed, link).right; remaining -= 1; }
    return link;
}
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
pub fn usize attribute_count(ref Parsed parsed) { return v::len<attributes::Attribute>(ref parsed.attributes); }
pub fn usize specialization_argument_count(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed,id); assert(node.kind == 20);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = type_at(parsed,link).right; }
    return count;
}
pub fn usize specialization_argument(ref Parsed parsed, usize id, usize index) {
    let usize count = specialization_argument_count(parsed,id); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed,id).right;
    while (remaining > index) { link = type_at(parsed,link).right; remaining -= 1; }
    return type_at(parsed,link).left;
}
pub fn usize switch_case_count(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed,id); assert(node.kind == 15);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = expression_at(parsed,link).right; }
    return count;
}
pub fn usize switch_case(ref Parsed parsed, usize id, usize index) {
    let usize count = switch_case_count(parsed,id); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed,id).right;
    while (remaining > index) { link = expression_at(parsed,link).right; remaining -= 1; }
    return expression_at(parsed,link).left;
}
pub fn attributes::Attribute attribute_at(ref Parsed parsed, usize index) { return *v::at<attributes::Attribute>(ref parsed.attributes,index); }
pub fn attributes::Argument attribute_argument_at(ref Parsed parsed, usize index) { return *v::at<attributes::Argument>(ref parsed.attribute_arguments,index); }
pub fn ref u8[] attribute_argument_text(ref Parsed parsed, usize index) {
    let attributes::Argument argument = attribute_argument_at(parsed,index);
    return scanner::source_view(ref parsed.source)[argument.start..argument.end];
}
