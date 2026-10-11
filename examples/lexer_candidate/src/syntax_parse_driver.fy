module syntax_parse_driver;
import syntax as owner;
import scanner as scanner;
import std::vec as v;
import types as types;
import records as records;
import bodies as bodies;
import body_state as state;
import attributes as attributes;
import cleanup_syntax as cleanups;

struct Attempt { bool ok; owner::Declaration value; owner::Problem problem; }

fn u32 item_kind(ref scanner::Stream stream, usize index) {
    var usize next = index;
    if (state::word(stream,next,b"pub")) {
        next += 1;
        while (next < scanner::token_count(stream) && scanner::token_info(stream,next).kind == 2) { next += 1; }
    }
    if (next >= scanner::token_count(stream)) { return 5; }
    if (state::word(stream,next,b"fn")) { return 3; }
    if (state::word(stream,next,b"struct")) { return 4; }
    if (state::word(stream,next,b"resource")) { return 6; }
    if (state::word(stream,next,b"const")) { return 7; }
    if (state::word(stream,next,b"static_assert")) { return 8; }
    if (state::word(stream,next,b"unsafe")) { return 9; }
    if (state::word(stream,next,b"state")) { return 10; }
    if (state::word(stream,next,b"enum")) { return 11; }
    return 5;
}

fn Attempt failed(ref scanner::Stream stream, ref mut state::Cursor cursor, u32 code) {
    let scanner::TokenInfo token = state::current(stream, cursor);
    return Attempt { .ok = false,
        .value = owner::Declaration { .cleanup_attribute_start = 0, .cleanup_attribute_count = 0, .unsafe_start = 0, .unsafe_end = 0, .owner_start = 0, .owner_end = 0, .kind = 0, .start = 0, .end = 0, .name_start = 0, .name_end = 0, .expression = 0, .entry_start = 0, .entry_count = 0, .body = 0, .result_type = 0, .field_start = 0, .field_count = 0, .public = false, .public_start = 0, .public_end = 0, .attribute_start = 0, .attribute_count = 0, .cleanup_start = 0, .cleanup_end = 0, .cleanup_body = 0, .generic_parameters = 0, .generic_start = 0, .generic_end = 0 },
        .problem = owner::Problem { .code, .start = token.start, .end = token.end, .expected = cursor.expected, .token = cursor.index } };
}

fn Attempt declaration(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<bodies::Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, ref mut v::Vec<records::Field> fields, ref mut v::Vec<attributes::Attribute> attribute_nodes, ref mut v::Vec<attributes::Argument> attribute_arguments, usize attribute_base, usize argument_base, u32 initial_kind) {
    var u32 kind = initial_kind;
    let usize start = scanner::token_info(stream, cursor.index).start;
    let usize entry_start = cursor.entries;
    let bool public = kind >= 3 && state::take(stream, cursor, b"pub");
    let usize field_start = cursor.fields;
    let usize unsafe_start = if (kind == 9) { state::current(stream,cursor).start } else { 0 };
    let usize unsafe_end = if (kind == 9) { state::current(stream,cursor).end } else { 0 };
    state::advance(stream, cursor);
    if (kind == 9 && state::take(stream,cursor,b"fn")) { kind = 3; }
    var usize result_type = 0;
    if (kind == 9 && (!state::expect(stream,cursor,b"extern",19) || !state::expect(stream,cursor,b"C",20) || !state::expect(stream,cursor,b"fn",21))) { return failed(stream,cursor,kind); }
    if (kind == 3 || kind == 7 || kind == 9) {
        result_type = bodies::read_type(stream, cursor, type_nodes);
        if (result_type == 0) { return failed(stream, cursor, kind); }
    }
    let scanner::TokenInfo name = scanner::token_info(stream, cursor.index);
    if (kind != 8 && kind != 10 && name.kind != 3) { cursor.expected = 1; return failed(stream, cursor, kind); }
    var usize name_start = if (kind == 8 || kind == 10) { 0 } else { name.start };
    var usize name_end = if (kind == 8 || kind == 10) { 0 } else { name.end };
    var usize root = 0;
    var usize cleanup_attribute_start = 0; var usize cleanup_attribute_count = 0;
    var usize body = 0; var usize cleanup_start = 0; var usize cleanup_end = 0; var usize cleanup_body = 0;
    if (kind != 8 && kind != 10) { state::advance(stream, cursor); }
    var usize owner_start = 0; var usize owner_end = 0;
    if (kind == 3 && state::take(stream,cursor,b"::")) {
        owner_start = name_start; owner_end = name_end;
        let scanner::TokenInfo method = state::current(stream,cursor);
        if (method.kind != 3) { cursor.expected = 1; return failed(stream,cursor,kind); }
        name_start = method.start; name_end = method.end; state::advance(stream,cursor);
    }
    var usize generic_parameters = 0; var usize generic_start = 0; var usize generic_end = 0;
    if ((kind == 3 || kind == 4 || kind == 6) && state::word(stream,cursor.index,b"<")) {
        generic_start = state::current(stream,cursor).start;
        let types::Attempt parameters = types::parse_parameters(stream,cursor.index,cursor.types,type_nodes);
        cursor.index = parameters.next; cursor.previous_end = parameters.end; cursor.types = parameters.count;
        cursor.status = parameters.status; cursor.expected = parameters.expected;
        if (parameters.root == 0) { return failed(stream,cursor,kind); }
        generic_parameters = parameters.root; generic_end = parameters.end;
    }
    if (kind < 3) {
        while (state::take(stream, cursor, b"::")) {
            let scanner::TokenInfo part = scanner::token_info(stream, cursor.index);
            if (part.kind != 3) { cursor.expected = 1; return failed(stream, cursor, kind); }
            name_end = part.end; state::advance(stream, cursor);
        }
        if (kind == 2) {
            if (!state::expect(stream, cursor, b"as", 10)) { return failed(stream, cursor, kind); }
            let scanner::TokenInfo alias = scanner::token_info(stream, cursor.index);
            if (alias.kind != 3) { cursor.expected = 1; return failed(stream, cursor, kind); }
            name_start = alias.start; name_end = alias.end; state::advance(stream, cursor);
        }
        if (!state::expect(stream, cursor, b";", 8)) { return failed(stream, cursor, kind); }
    } else if (kind == 8) {
        if (!state::expect(stream,cursor,b"(",4)) { return failed(stream,cursor,kind); }
        root = bodies::read_expression(stream,cursor,nodes,entries,statements,type_nodes);
        if (root == 0 || !state::expect(stream,cursor,b")",5) || !state::expect(stream,cursor,b";",8)) { return failed(stream,cursor,kind); }
    } else if (kind == 7) {
        if (!state::expect(stream,cursor,b"=",9)) { return failed(stream,cursor,kind); }
        root = bodies::read_expression(stream,cursor,nodes,entries,statements,type_nodes);
        if (root == 0 || !state::expect(stream,cursor,b";",8)) { return failed(stream,cursor,kind); }
    } else if (kind == 4 || kind == 6 || kind == 10 || kind == 11) {
        let records::Attempt record = records::parse(stream,cursor.index,cursor.types,cursor.fields,type_nodes,fields,kind);
        cursor.index = record.next; cursor.previous_end = record.end; cursor.types = record.types; cursor.fields = record.fields;
        cursor.expected = record.expected; cursor.status = record.status;
        if (!record.ok) { return failed(stream,cursor,kind); }
        cleanup_start = record.cleanup_start;
        if (cleanup_start != 0) {
            let cleanups::Attempt c = cleanups::parse(stream,cursor,nodes,entries,statements,type_nodes,attribute_nodes,attribute_arguments,attribute_base,argument_base);
            if (!c.ok) { return failed(stream,cursor,kind); }
            cleanup_start = c.start; cleanup_body = c.body; cleanup_end = c.end;
            cleanup_attribute_start = c.attribute_start; cleanup_attribute_count = c.attribute_count;
            if (!state::expect(stream,cursor,b"}",7)) { return failed(stream,cursor,kind); }
        }
    } else {
        if (!state::expect(stream, cursor, b"(", 4)) { return failed(stream, cursor, kind); }
        while (!state::word(stream, cursor.index, b")")) {
            if (!bodies::entry(stream, cursor, nodes, entries, statements, type_nodes, 1)) { return failed(stream, cursor, kind); }
            if (!state::take(stream, cursor, b",")) { break; }
        }
        if (!state::expect(stream, cursor, b")", 5)) { return failed(stream, cursor, kind); }
        if (kind == 9) {
            if (!state::expect(stream,cursor,b";",8)) { return failed(stream,cursor,kind); }
        } else {
        cursor.top_return = 0;
        body = bodies::block(stream, cursor, nodes, entries, statements, type_nodes, 0, false);
        if (body == 0) { return failed(stream, cursor, kind); }
        root = cursor.top_return;
        }
    }
    return Attempt { .ok = true,
        .value = owner::Declaration { .cleanup_attribute_start, .cleanup_attribute_count, .unsafe_start, .unsafe_end, .owner_start, .owner_end, .kind, .start, .end = cursor.previous_end, .name_start, .name_end, .expression = root, .entry_start, .entry_count = cursor.entries - entry_start, .body, .result_type, .field_start, .field_count = cursor.fields - field_start, .public, .public_start = if (public) { start } else { 0 }, .public_end = if (public) { start + 3 } else { 0 }, .attribute_start = 0, .attribute_count = 0, .cleanup_start, .cleanup_end, .cleanup_body, .generic_parameters, .generic_start, .generic_end },
        .problem = owner::Problem { .code = 0, .start = 0, .end = 0, .expected = 0, .token = 0 } };
}

// Restart at the failed declaration, then consume with brace depth. Never accept
// a nested function as the next top-level declaration or repeat a zero-width step.

fn void recover(ref scanner::Stream stream, ref mut state::Cursor cursor, usize start) {
    let usize failed_index = cursor.index;
    let bool missing_pattern_close = cursor.expected == 25 && state::word(stream,failed_index,b":");
    cursor.index = start;
    var usize boundary = start;
    if (state::word(stream,cursor.index,b"@")) {
        while (state::current(stream,cursor).kind != 1 && !state::word(stream,cursor.index,b"fn") && !state::word(stream,cursor.index,b"pub") && !state::word(stream,cursor.index,b"struct") && !state::word(stream,cursor.index,b"resource") && !state::word(stream,cursor.index,b"enum") && !state::word(stream,cursor.index,b"const") && !state::word(stream,cursor.index,b"static_assert") && !state::word(stream,cursor.index,b"import")) { state::advance(stream,cursor); }
        boundary = cursor.index;
    }
    if (state::take(stream, cursor, b"pub")) {
        if (state::word(stream, cursor.index, b"fn") || state::word(stream, cursor.index, b"struct") || state::word(stream, cursor.index, b"resource") || state::word(stream, cursor.index, b"enum") || state::word(stream, cursor.index, b"const") || state::word(stream, cursor.index, b"static_assert") || state::word(stream, cursor.index, b"import")) { state::advance(stream, cursor); }
    }
    var usize depth = 0;
    if (state::take(stream,cursor,b"unsafe")) {
        discard state::take(stream,cursor,b"extern");
        if (state::current(stream,cursor).kind == 3) { state::advance(stream,cursor); }
        discard state::take(stream,cursor,b"fn");
        boundary = cursor.index;
    }
    var bool type_header = false;
    while (scanner::token_info(stream, cursor.index).kind != 1) {
        if (missing_pattern_close && cursor.index == failed_index && depth > 0) { depth -= 1; }
        if (depth == 0 && cursor.index < failed_index && state::word(stream,cursor.index,b"unsafe")) { type_header = true; }
        if (type_header && state::word(stream,cursor.index,b"fn")) { type_header = false; state::advance(stream,cursor); continue; }
        if (depth == 0 && cursor.index > boundary && cursor.index >= failed_index && (state::word(stream, cursor.index, b"fn") || item_kind(stream, cursor.index) != 5 || state::word(stream, cursor.index, b"import") || state::word(stream,cursor.index,b"@"))) { return; }
        if (state::word(stream, cursor.index, b"{")) { depth += 1; }
        else if (state::word(stream, cursor.index, b"}")) {
            state::advance(stream, cursor);
            if (depth == 0) { return; }
            depth -= 1;
            if (depth == 0) { return; }
            continue;
        } else if (depth == 0 && state::word(stream, cursor.index, b";")) { state::advance(stream, cursor); return; }
        state::advance(stream, cursor);
    }
}

// Errors 3 allocation, 4 candidate item/diagnostic bound, 5 lexical diagnostics.
// Syntax problems return an owned incomplete result, never an executable program.

pub fn Result<owner::Parsed, i32> parse_diagnostics(scanner::Stream input, usize max_items) {
    if (max_items > 512) { return .err(4); }
    var v::Vec<owner::Declaration> declarations = v::empty<owner::Declaration>();
    var v::Vec<owner::Problem> problems = v::empty<owner::Problem>();
    var v::Vec<state::Node> nodes = v::empty<state::Node>();
    var v::Vec<bodies::Entry> entries = v::empty<bodies::Entry>();
    var v::Vec<state::Statement> statements = v::empty<state::Statement>();
    var v::Vec<types::Node> type_nodes = v::empty<types::Node>();
    var v::Vec<records::Field> fields = v::empty<records::Field>();
    var v::Vec<attributes::Attribute> attribute_nodes = v::empty<attributes::Attribute>();
    var v::Vec<attributes::Argument> attribute_arguments = v::empty<attributes::Argument>();
    var state::Cursor cursor = state::Cursor { .index = 0, .previous_end = 0, .nodes = 0, .entries = 0, .statements = 0, .types = 0, .fields = 0, .expression = 0, .top_return = 0, .status = 0, .expected = 0, .active_expressions = 0, .block_depth = 0, .tail = 0, .last_entry = 0, .expression_kind = 0, .statement_kind = 0 };
    state::trivia(ref input, ref mut cursor);
    var bool first = true; var bool functions = false; var usize items = 0; var usize errors = 0;
    while (first || scanner::token_info(ref input, cursor.index).kind != 1) {
        cursor.expected = if (first) { 12 } else { 0 };
        let usize start = cursor.index;
        let usize node_start = v::len<state::Node>(ref nodes);
        let usize entry_start = cursor.entries;
        let usize statement_start = cursor.statements;
        let usize type_start = cursor.types;
        let usize field_start = cursor.fields;
        let usize attribute_start = v::len<attributes::Attribute>(ref attribute_nodes);
        let usize argument_start = v::len<attributes::Argument>(ref attribute_arguments);
        var bool attributes_ok = true;
        if (!first && state::word(ref input,cursor.index,b"@")) {
            let attributes::Attempt prefix = attributes::parse(ref input,cursor.index,attribute_start,argument_start,ref mut attribute_nodes,ref mut attribute_arguments);
            cursor.index = prefix.next; cursor.previous_end = prefix.end; cursor.status = prefix.status; cursor.expected = prefix.expected;
            attributes_ok = prefix.ok;
        }
        let usize attribute_base = v::len<attributes::Attribute>(ref attribute_nodes);
        let usize argument_base = v::len<attributes::Argument>(ref attribute_arguments);
        let usize item_start = cursor.index;
        let usize attribute_count = v::len<attributes::Attribute>(ref attribute_nodes) - attribute_start;
        let u32 kind = if (first) { 1 } else if (attribute_count == 0 && !functions && state::word(ref input, cursor.index, b"import")) { 2 } else { item_kind(ref input,cursor.index) };
        let Attempt attempt = if (!attributes_ok || (first && !state::word(ref input, cursor.index, b"module")) || kind == 5) { failed(ref input, ref mut cursor, if (kind == 5) { 4 } else { kind }) } else { declaration(ref input, ref mut cursor, ref mut nodes, ref mut entries, ref mut statements, ref mut type_nodes, ref mut fields, ref mut attribute_nodes, ref mut attribute_arguments, attribute_base, argument_base, kind) };
        if (cursor.status != 0) { return .err(cursor.status); }
        if (attempt.ok) {
            if (items >= max_items) { return .err(4); }
            var owner::Declaration value = attempt.value;
            value.attribute_start = attribute_start; value.attribute_count = attribute_count;
            if (attribute_count > 0) { value.start = scanner::token_info(ref input,start).start; }
            if (!v::push<owner::Declaration>(ref mut declarations, value).is_ok) { return .err(3); }
            items += 1;
        } else {
            discard v::truncate<attributes::Attribute>(ref mut attribute_nodes,attribute_start);
            discard v::truncate<attributes::Argument>(ref mut attribute_arguments,argument_start);
            discard v::truncate<state::Node>(ref mut nodes, node_start);
            cursor.nodes = node_start;
            discard v::truncate<bodies::Entry>(ref mut entries, entry_start);
            cursor.entries = entry_start;
            discard v::truncate<state::Statement>(ref mut statements, statement_start);
            cursor.statements = statement_start;
            discard v::truncate<types::Node>(ref mut type_nodes, type_start);
            cursor.types = type_start;
            discard v::truncate<records::Field>(ref mut fields, field_start);
            cursor.fields = field_start;
            if (errors >= 32) { return .err(4); }
            if (!v::push<owner::Problem>(ref mut problems, attempt.problem).is_ok) { return .err(3); }
            errors += 1;
            if (!first || state::word(ref input, start, b"module")) { recover(ref input, ref mut cursor, if (attributes_ok) { item_start } else { start }); }
        }
        if (kind >= 3 && kind != 5) { functions = true; }
        first = false;
    }
    return .ok(owner::assemble(input,declarations,problems,nodes,entries,statements,type_nodes,fields,attribute_nodes,attribute_arguments));
}
