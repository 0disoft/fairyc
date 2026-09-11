module bodies;
import scanner as scanner;
import std::vec as v;
import types as types;

pub struct Node { u32 kind; usize start; usize end; usize token; usize left; usize right; }
pub struct Cursor { usize index; usize previous_end; usize nodes; usize entries; usize statements; usize types; usize fields; usize expression; usize top_return; i32 status; u32 expected; usize active_expressions; usize block_depth; usize tail; usize last_entry; u32 expression_kind; }
pub struct Entry { u32 kind; usize start; usize end; usize name_start; usize name_end; usize type_start; usize type_end; usize type_root; usize expression; usize id; }
pub struct Statement { u32 kind; usize start; usize end; usize expression; usize target; usize first; usize second; usize entry; usize operator; }

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
fn usize append(ref mut v::Vec<Node> nodes, ref mut Cursor cursor, Node node) {
    if (cursor.nodes >= 4096) { cursor.status = 4; return 0; }
    if (!switch (v::push<Node>(nodes, node)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return 0; }
    cursor.nodes = cursor.nodes + 1;
    cursor.expression_kind = node.kind;
    return cursor.nodes;
}
fn usize expression_inner(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize minimum, usize depth) {
    if (depth >= 64) { cursor.status = 4; return 0; }
    let usize token_index = cursor.index;
    let scanner::TokenInfo token = current(stream, cursor);
    var usize left = 0;
    if (token.kind == 5 || token.kind == 7 || token.kind == 8 || token.kind == 3 || word(stream, cursor.index, b"true") || word(stream, cursor.index, b"false")) {
        advance(stream, cursor);
        if (token.kind == 3) {
            while (take(stream, cursor, b"::")) {
                if (current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
                advance(stream, cursor);
            }
        }
        left = append(nodes, cursor, Node { .kind = if (token.kind == 3) { 2 } else { 1 }, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = 0, .right = 0 });
        if (left == 0) { return 0; }
        if (token.kind == 3 && take(stream, cursor, b"{")) {
            var usize fields = 0;
            while (!word(stream, cursor.index, b"}")) {
                let usize start = current(stream, cursor).start;
                if (!expect(stream, cursor, b".", 14)) { return 0; }
                let usize name = cursor.index;
                if (current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
                advance(stream, cursor);
                if (!expect(stream, cursor, b"=", 9)) { return 0; }
                let usize value = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, depth + 1);
                if (value == 0) { return 0; }
                fields = append(nodes, cursor, Node { .kind = 11, .start = start, .end = cursor.previous_end, .token = name, .left = value, .right = fields });
                if (fields == 0) { return 0; }
                if (!take(stream, cursor, b",")) { break; }
            }
            if (!expect(stream, cursor, b"}", 7)) { return 0; }
            left = append(nodes, cursor, Node { .kind = 10, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = left, .right = fields });
        }
    } else if (take(stream,cursor,b".")) {
        let usize name = cursor.index;
        if (current(stream,cursor).kind != 3) { cursor.expected = 1; return 0; }
        advance(stream,cursor);
        left = append(nodes,cursor,Node { .kind = 28, .start = token.start, .end = cursor.previous_end, .token = name, .left = 0, .right = 0 });
    } else if (word(stream,cursor.index,b"if")) {
        left = conditional(stream,cursor,nodes,entries,statements,type_nodes,depth);
    } else if (word(stream,cursor.index,b"switch")) {
        left = switch_value(stream,cursor,nodes,entries,statements,type_nodes,depth);
    } else if (word(stream,cursor.index,b"{")) {
        let usize body = block(stream,cursor,nodes,entries,statements,type_nodes,cursor.block_depth,true);
        if (body == 0) { return 0; }
        left = append(nodes,cursor,Node { .kind = 12, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = body, .right = 0 });
    } else if (take(stream, cursor, b"(")) {
        let usize child = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, depth + 1);
        if (child == 0 || !expect(stream, cursor, b")", 5)) { return 0; }
        left = append(nodes, cursor, Node { .kind = 3, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = child, .right = 0 });
    } else if (take(stream,cursor,b"[")) {
        var usize elements = 0;
        while (!word(stream,cursor.index,b"]")) {
            let usize element_token = cursor.index;
            let usize element_start = current(stream,cursor).start;
            let usize value = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1);
            if (value == 0) { return 0; }
            elements = append(nodes,cursor,Node { .kind = 25, .start = element_start, .end = cursor.previous_end, .token = element_token, .left = value, .right = elements });
            if (elements == 0) { return 0; }
            if (!take(stream,cursor,b",")) { break; }
        }
        if (!expect(stream,cursor,b"]",11)) { return 0; }
        left = append(nodes,cursor,Node { .kind = 24, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = 0, .right = elements });
    } else if (take(stream,cursor,b"ref")) {
        let bool mutable = take(stream,cursor,b"mut");
        let usize child = expression(stream,cursor,nodes,entries,statements,type_nodes,11,depth + 1);
        if (child == 0) { return 0; }
        left = append(nodes,cursor,Node { .kind = if (mutable) { 22 } else { 21 }, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = child, .right = 0 });
    } else if (word(stream, cursor.index, b"-") || word(stream, cursor.index, b"+") || word(stream, cursor.index, b"!") || word(stream, cursor.index, b"~") || word(stream, cursor.index, b"*")) {
        advance(stream, cursor);
        let usize child = expression(stream, cursor, nodes, entries, statements, type_nodes, 11, depth + 1);
        if (child == 0) { return 0; }
        left = append(nodes, cursor, Node { .kind = 4, .start = token.start, .end = cursor.previous_end, .token = token_index, .left = child, .right = 0 });
    } else { cursor.expected = 3; return 0; }
    var bool comparison = false;
    while (left != 0) {
        let usize opener = cursor.index;
        if (word(stream,cursor.index,b"as")) {
            if (minimum > 10) { return left; }
            advance(stream,cursor);
            let usize target = read_type(stream,cursor,type_nodes);
            if (target == 0) { return 0; }
            left = append(nodes,cursor,Node { .kind = 23, .start = token.start, .end = cursor.previous_end, .token = opener, .left = left, .right = target });
            continue;
        }
        if (cursor.expression_kind == 2 && word(stream,cursor.index,b"<") && types::call_ahead(stream,cursor.index)) {
            let types::Attempt result = types::parse_arguments(stream,cursor.index,cursor.types,type_nodes);
            cursor.index = result.next; cursor.previous_end = result.end; cursor.types = result.count;
            cursor.status = result.status; cursor.expected = result.expected;
            if (result.root == 0) { return 0; }
            left = append(nodes,cursor,Node { .kind = 20, .start = token.start, .end = cursor.previous_end, .token = opener, .left = left, .right = result.root });
            if (left == 0) { return 0; }
            continue;
        }
        if (take(stream, cursor, b".")) {
            let usize member = cursor.index;
            if (current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
            advance(stream, cursor);
            left = append(nodes, cursor, Node { .kind = 8, .start = token.start, .end = cursor.previous_end, .token = member, .left = left, .right = 0 });
            continue;
        }
        if (take(stream, cursor, b"[")) {
            let usize bounds_start = current(stream,cursor).start;
            var usize index = 0;
            if (!word(stream,cursor.index,b"..")) {
                index = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1);
                if (index == 0) { return 0; }
            }
            let usize range_token = cursor.index;
            if (take(stream,cursor,b"..")) {
                var usize end = 0;
                if (!word(stream,cursor.index,b"]")) {
                    end = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1);
                    if (end == 0) { return 0; }
                }
                let usize bounds = append(nodes,cursor,Node { .kind = 27, .start = bounds_start, .end = cursor.previous_end, .token = range_token, .left = index, .right = end });
                if (bounds == 0 || !expect(stream,cursor,b"]",11)) { return 0; }
                left = append(nodes,cursor,Node { .kind = 26, .start = token.start, .end = cursor.previous_end, .token = opener, .left = left, .right = bounds });
            } else {
                if (!expect(stream, cursor, b"]", 11)) { return 0; }
                left = append(nodes, cursor, Node { .kind = 9, .start = token.start, .end = cursor.previous_end, .token = opener, .left = left, .right = index });
            }
            continue;
        }
        if (take(stream, cursor, b"(")) {
            var usize arguments = 0;
            while (!word(stream, cursor.index, b")")) {
                let usize argument_token = cursor.index;
                let usize argument_start = current(stream, cursor).start;
                let usize value = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, depth + 1);
                if (value == 0) { return 0; }
                arguments = append(nodes, cursor, Node { .kind = 7, .start = argument_start, .end = cursor.previous_end, .token = argument_token, .left = value, .right = arguments });
                if (arguments == 0) { return 0; }
                if (!take(stream, cursor, b",")) { break; }
            }
            if (!expect(stream, cursor, b")", 5)) { return 0; }
            left = append(nodes, cursor, Node { .kind = 6, .start = token.start, .end = cursor.previous_end, .token = opener, .left = left, .right = arguments });
            continue;
        }
        let usize power = precedence(stream, cursor.index);
        if (power < minimum) { return left; }
        if (power == 3 && comparison) { return 0; }
        if (power == 3) { comparison = true; }
        let usize operator = cursor.index;
        advance(stream, cursor);
        let usize right = expression(stream, cursor, nodes, entries, statements, type_nodes, power + 1, depth + 1);
        if (right == 0) { return 0; }
        left = append(nodes, cursor, Node { .kind = 5, .start = token.start, .end = cursor.previous_end, .token = operator, .left = left, .right = right });
    }
    return 0;
}

fn bool inferred_binding(ref scanner::Stream stream, usize index) {
    if (scanner::token_info(stream,index).kind != 3) { return false; }
    var usize next = index + 1;
    while (next < scanner::token_count(stream) && scanner::token_info(stream,next).kind == 2) { next += 1; }
    return next < scanner::token_count(stream) && word(stream,next,b"=");
}
pub fn usize read_type(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<types::Node> nodes) {
    let types::Attempt result = types::parse(stream,cursor.index,cursor.types,nodes);
    cursor.index = result.next; cursor.previous_end = result.end; cursor.types = result.count;
    cursor.status = result.status; cursor.expected = result.expected;
    return result.root;
}
pub fn bool entry(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, u32 kind) {
    if (cursor.entries >= 1024) { cursor.status = 4; return false; }
    cursor.entries = cursor.entries + 1;
    let usize id = cursor.entries;
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
            root = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, 0);
            if (root == 0) { return false; }
        }
        if (!expect(stream, cursor, b";", 8)) { return false; }
    }
    let Entry value = Entry { .kind = kind, .start = start, .end = cursor.previous_end, .name_start = name_start, .name_end = name_end, .type_start = type_start, .type_end = type_end, .type_root = type_root, .expression = root, .id = id };
    if (!switch (v::push<Entry>(entries, value)) { case .ok(_): { true } case .err(_): { false } }) { cursor.status = 3; return false; }
    cursor.last_entry = id;
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
fn usize block_inner(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth, bool allow_tail) {
    if (depth >= 32) { cursor.status = 4; return 0; }
    let usize start = current(stream, cursor).start;
    if (!expect(stream, cursor, b"{", 6)) { return 0; }
    var usize tail = 0; var usize count = 0; var usize tail_value = 0;
    while (!word(stream, cursor.index, b"}")) {
        if (current(stream, cursor).kind == 1) { cursor.expected = 7; return 0; }
        let usize child_start = current(stream, cursor).start;
        cursor.tail = 0;
        let usize child = statement(stream, cursor, nodes, entries, statements, type_nodes, depth, allow_tail);
        if (child == 0) {
            if (allow_tail && cursor.tail != 0) { tail_value = cursor.tail; cursor.tail = 0; break; }
            return 0;
        }
        var Statement link = statement_value(7, child_start, cursor.previous_end);
        link.first = child; link.second = tail;
        tail = append_statement(statements, cursor, link);
        if (tail == 0) { return 0; }
        count += 1;
    }
    advance(stream, cursor);
    var Statement value = statement_value(1, start, cursor.previous_end);
    value.first = tail; value.second = count; value.expression = tail_value;
    return append_statement(statements, cursor, value);
}
fn usize statement(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth, bool allow_tail) {
    if (depth >= 32) { cursor.status = 4; return 0; }
    let usize start = current(stream, cursor).start;
    if (!allow_tail && word(stream, cursor.index, b"{")) { return block(stream, cursor, nodes, entries, statements, type_nodes, depth + 1, true); }
    var Statement value = statement_value(0, start, 0);
    if (word(stream, cursor.index, b"break") || word(stream, cursor.index, b"continue")) {
        value.kind = if (word(stream, cursor.index, b"break")) { 9 } else { 10 };
        advance(stream, cursor);
        if (!expect(stream, cursor, b";", 8)) { return 0; }
    } else if (take(stream,cursor,b"assert")) {
        value.kind = 11;
        if (!expect(stream,cursor,b"(",4)) { return 0; }
        value.expression = expression(stream,cursor,nodes,entries,statements,type_nodes,1,0);
        if (value.expression == 0 || !expect(stream,cursor,b")",5) || !expect(stream,cursor,b";",8)) { return 0; }
    } else if (take(stream,cursor,b"discard")) {
        value.kind = 12;
        value.expression = expression(stream,cursor,nodes,entries,statements,type_nodes,1,0);
        if (value.expression == 0 || !expect(stream,cursor,b";",8)) { return 0; }
    } else if (take(stream, cursor, b"while")) {
        value.kind = 8;
        if (!expect(stream, cursor, b"(", 4)) { return 0; }
        value.expression = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, 0);
        if (value.expression == 0 || !expect(stream, cursor, b")", 5)) { return 0; }
        value.first = block(stream, cursor, nodes, entries, statements, type_nodes, depth + 1, false);
        if (value.first == 0) { return 0; }
    } else if (!allow_tail && take(stream, cursor, b"if")) {
        value.kind = 5;
        if (!expect(stream, cursor, b"(", 4)) { return 0; }
        value.expression = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, 0);
        if (value.expression == 0 || !expect(stream, cursor, b")", 5)) { return 0; }
        value.first = block(stream, cursor, nodes, entries, statements, type_nodes, depth + 1, false);
        if (value.first == 0) { return 0; }
        if (take(stream, cursor, b"else")) {
            if (word(stream, cursor.index, b"if")) { value.second = statement(stream, cursor, nodes, entries, statements, type_nodes, depth + 1, false); }
            else { value.second = block(stream, cursor, nodes, entries, statements, type_nodes, depth + 1, false); }
            if (value.second == 0) { return 0; }
        }
    } else {
        let u32 kind = if (word(stream, cursor.index, b"let")) { 2 } else if (word(stream, cursor.index, b"var")) { 3 } else if (word(stream, cursor.index, b"return")) { 4 } else { 0 };
        if (kind != 0) {
            if (!entry(stream, cursor, nodes, entries, statements, type_nodes, kind)) { return 0; }
            value.kind = if (kind == 4) { 3 } else { 2 };
            value.entry = cursor.last_entry; value.expression = cursor.expression;
            if (kind == 4 && depth == 0) { cursor.top_return = cursor.expression; }
        } else {
            let usize left = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, 0);
            if (left == 0) { return 0; }
            value.kind = 6; value.expression = left;
            if (assignment(stream, cursor.index)) {
                value.kind = 4; value.target = left; value.operator = cursor.index;
                advance(stream, cursor);
                value.expression = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, 0);
                if (value.expression == 0) { return 0; }
            }
            if (allow_tail && value.kind == 6 && word(stream,cursor.index,b"}")) { cursor.tail = left; return 0; }
            if (value.kind == 6 && (cursor.expression_kind == 12 || cursor.expression_kind == 13) && !word(stream,cursor.index,b";")) {
                value.end = cursor.previous_end; return append_statement(statements,cursor,value);
            }
            if (!expect(stream, cursor, b";", 8)) { return 0; }
        }
    }
    value.end = cursor.previous_end;
    return append_statement(statements, cursor, value);
}

fn usize expression(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize minimum, usize depth) {
    if (cursor.active_expressions >= 64) { cursor.status = 4; return 0; }
    cursor.active_expressions = cursor.active_expressions + 1;
    let usize result = expression_inner(stream,cursor,nodes,entries,statements,type_nodes,minimum,depth);
    cursor.active_expressions = cursor.active_expressions - 1;
    return result;
}

pub fn usize block(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth, bool allow_tail) {
    if (depth >= 32) { cursor.status = 4; return 0; }
    let usize previous = cursor.block_depth; cursor.block_depth = depth + 1;
    let usize result = block_inner(stream,cursor,nodes,entries,statements,type_nodes,depth,allow_tail);
    cursor.block_depth = previous;
    return result;
}

fn usize conditional(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth) {
    if (depth >= 64) { cursor.status = 4; return 0; }
    let usize token = cursor.index; let usize start = current(stream,cursor).start;
    advance(stream,cursor);
    if (!expect(stream,cursor,b"(",4)) { return 0; }
    let usize condition = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1);
    if (condition == 0 || !expect(stream,cursor,b")",5)) { return 0; }
    let usize branch_start = current(stream,cursor).start;
    let usize consequent = block(stream,cursor,nodes,entries,statements,type_nodes,cursor.block_depth,true);
    if (consequent == 0) { return 0; }
    var usize alternative = 0;
    if (take(stream,cursor,b"else")) {
        if (word(stream,cursor.index,b"if")) {
            if (cursor.active_expressions >= 64) { cursor.status = 4; return 0; }
            cursor.active_expressions = cursor.active_expressions + 1;
            alternative = conditional(stream,cursor,nodes,entries,statements,type_nodes,depth + 1);
            cursor.active_expressions = cursor.active_expressions - 1;
        } else {
            let usize opener = cursor.index; let usize body_start = current(stream,cursor).start;
            let usize body = block(stream,cursor,nodes,entries,statements,type_nodes,cursor.block_depth,true);
            if (body == 0) { return 0; }
            alternative = append(nodes,cursor,Node { .kind = 12, .start = body_start, .end = cursor.previous_end, .token = opener, .left = body, .right = 0 });
        }
        if (alternative == 0) { return 0; }
    }
    let usize branches = append(nodes,cursor,Node { .kind = 14, .start = branch_start, .end = cursor.previous_end, .token = token, .left = consequent, .right = alternative });
    if (branches == 0) { return 0; }
    return append(nodes,cursor,Node { .kind = 13, .start = start, .end = cursor.previous_end, .token = token, .left = condition, .right = branches });
}

fn usize pattern(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes) {
    let usize start = current(stream,cursor).start;
    if (!expect(stream,cursor,b".",14)) { return 0; }
    let usize token = cursor.index;
    if (current(stream,cursor).kind != 3) { cursor.expected = 1; return 0; }
    advance(stream,cursor);
    var usize binding = 0;
    if (take(stream,cursor,b"(")) {
        let usize name = cursor.index; let scanner::TokenInfo info = current(stream,cursor);
        if (info.kind != 3) { cursor.expected = 1; return 0; }
        advance(stream,cursor);
        binding = append(nodes,cursor,Node { .kind = 17, .start = info.start, .end = info.end, .token = name, .left = 0, .right = 0 });
        if (binding == 0 || !expect(stream,cursor,b")",5)) { return 0; }
    }
    return append(nodes,cursor,Node { .kind = 16, .start = start, .end = cursor.previous_end, .token = token, .left = binding, .right = 0 });
}
fn usize switch_value(ref scanner::Stream stream, ref mut Cursor cursor, ref mut v::Vec<Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth) {
    let usize token = cursor.index; let usize start = current(stream,cursor).start;
    advance(stream,cursor);
    if (!expect(stream,cursor,b"(",4)) { return 0; }
    let usize subject = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1);
    if (subject == 0 || !expect(stream,cursor,b")",5) || !expect(stream,cursor,b"{",6)) { return 0; }
    var usize cases = 0;
    while (!word(stream,cursor.index,b"}")) {
        let usize case_token = cursor.index; let usize case_start = current(stream,cursor).start;
        if (!expect(stream,cursor,b"case",18)) { return 0; }
        let usize match = pattern(stream,cursor,nodes);
        if (match == 0 || !expect(stream,cursor,b":",17)) { return 0; }
        let usize body = block(stream,cursor,nodes,entries,statements,type_nodes,cursor.block_depth,true);
        if (body == 0) { return 0; }
        let usize arm = append(nodes,cursor,Node { .kind = 18, .start = case_start, .end = cursor.previous_end, .token = case_token, .left = match, .right = body });
        if (arm == 0) { return 0; }
        cases = append(nodes,cursor,Node { .kind = 19, .start = case_start, .end = cursor.previous_end, .token = case_token, .left = arm, .right = cases });
        if (cases == 0) { return 0; }
    }
    advance(stream,cursor);
    return append(nodes,cursor,Node { .kind = 15, .start = start, .end = cursor.previous_end, .token = token, .left = subject, .right = cases });
}
