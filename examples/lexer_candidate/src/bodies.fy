module bodies;
import scanner as scanner;
import std::vec as v;
import types as types;
import parse_tokens as tokens;
import body_state as state;

pub struct Entry { u32 kind; usize start; usize end; usize name_start; usize name_end; usize type_start; usize type_end; usize type_root; usize expression; usize id; }

fn bool word(ref scanner::Stream stream, usize index, ref u8[] text) { return tokens::word(stream,index,text); }
fn usize expression_inner(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize minimum, usize depth, bool allow_record) {
    if (depth >= 64) { cursor.status = 4; return 0; }
    let usize token_index = cursor.index;
    let scanner::TokenInfo token = state::current(stream, cursor);
    var usize left = 0;
    if (tokens::atom(stream,cursor.index)) {
        state::advance(stream, cursor);
        if (token.kind == 3) {
            while (state::take(stream, cursor, b"::")) {
                if (state::current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
                state::advance(stream, cursor);
            }
        }
        left = state::append(nodes, cursor, state::node_value(if (token.kind == 3) { 2 } else { 1 },token.start,cursor.previous_end,token_index,0,0));
        if (left == 0) { return 0; }
    } else if (state::take(stream,cursor,b".")) {
        let usize name = cursor.index;
        if (state::current(stream,cursor).kind != 3) { cursor.expected = 1; return 0; }
        state::advance(stream,cursor);
        left = state::append(nodes,cursor,state::node_value(28,token.start,cursor.previous_end,name,0,0));
    } else if (word(stream,cursor.index,b"if")) {
        left = conditional(stream,cursor,nodes,entries,statements,type_nodes,depth);
    } else if (word(stream,cursor.index,b"switch")) {
        left = switch_value(stream,cursor,nodes,entries,statements,type_nodes,depth);
    } else if (tokens::block_start(stream,cursor.index)) {
        let bool unsafe_block = state::take(stream,cursor,b"unsafe");
        let usize body = block(stream,cursor,nodes,entries,statements,type_nodes,cursor.block_depth,true);
        if (body == 0) { return 0; }
        left = state::append(nodes,cursor,state::node_value(if (unsafe_block) { 31 } else { 12 },token.start,cursor.previous_end,token_index,body,0));
    } else if (state::take(stream, cursor, b"(")) {
        let usize child = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, depth + 1,true);
        if (child == 0 || !state::expect(stream, cursor, b")", 5)) { return 0; }
        left = state::append(nodes, cursor, state::node_value(3,token.start,cursor.previous_end,token_index,child,0));
    } else if (state::take(stream,cursor,b"[")) {
        var usize elements = 0;
        var usize repeated = 0; var usize repeat_count = 0;
        while (!word(stream,cursor.index,b"]")) {
            let usize element_token = cursor.index;
            let usize element_start = state::current(stream,cursor).start;
            let usize value = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1,true);
            if (value == 0) { return 0; }
            if (state::take(stream,cursor,b";")) {
                if (elements != 0) { cursor.expected = 11; return 0; }
                let usize count_token = cursor.index;
                let scanner::TokenInfo count = state::current(stream,cursor);
                if (count.kind != 5 && count.kind != 3) { cursor.expected = 16; return 0; }
                state::advance(stream,cursor);
                if (count.kind == 3) {
                    while (state::take(stream,cursor,b"::")) {
                        if (state::current(stream,cursor).kind != 3) { cursor.expected = 1; return 0; }
                        state::advance(stream,cursor);
                    }
                }
                repeat_count = state::append(nodes,cursor,state::node_value(if (count.kind == 5) { 1 } else { 2 },count.start,cursor.previous_end,count_token,0,0));
                if (repeat_count == 0) { return 0; }
                repeated = value; break;
            }
            elements = state::append(nodes,cursor,state::node_value(25,element_start,cursor.previous_end,element_token,value,elements));
            if (elements == 0) { return 0; }
            if (!state::take(stream,cursor,b",")) { break; }
        }
        if (!state::expect(stream,cursor,b"]",11)) { return 0; }
        left = state::append(nodes,cursor,state::node_value(if (repeated == 0) { 24 } else { 29 },token.start,cursor.previous_end,token_index,repeated,if (repeated == 0) { elements } else { repeat_count }));
    } else if (state::take(stream,cursor,b"ref")) {
        let bool mutable = state::take(stream,cursor,b"mut");
        let usize child = expression(stream,cursor,nodes,entries,statements,type_nodes,11,depth + 1,allow_record);
        if (child == 0) { return 0; }
        left = state::append(nodes,cursor,state::node_value(if (mutable) { 22 } else { 21 },token.start,cursor.previous_end,token_index,child,0));
    } else if (tokens::prefix(stream,cursor.index)) {
        state::advance(stream, cursor);
        let usize child = expression(stream, cursor, nodes, entries, statements, type_nodes, 11, depth + 1,allow_record);
        if (child == 0) { return 0; }
        left = state::append(nodes, cursor, state::node_value(4,token.start,cursor.previous_end,token_index,child,0));
    } else { cursor.expected = 3; return 0; }
    var bool comparison = false;
    while (left != 0) {
        if (!allow_record && word(stream,cursor.index,b"{")) { return left; }
        let usize opener = cursor.index;
        if (tokens::constructor_kind(cursor.expression_kind) && state::take(stream, cursor, b"{")) {
            var usize fields = 0;
            while (!word(stream, cursor.index, b"}")) {
                let usize start = state::current(stream, cursor).start;
                if (!state::expect(stream, cursor, b".", 14)) { return 0; }
                let usize name = cursor.index;
                if (state::current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
                state::advance(stream, cursor);
                var usize value = 0;
                if (word(stream,cursor.index,b",") || word(stream,cursor.index,b"}")) {
                    let scanner::TokenInfo identifier = scanner::token_info(stream,name);
                    value = state::append(nodes,cursor,state::node_value(2,identifier.start,identifier.end,name,0,0));
                } else {
                    if (!state::expect(stream, cursor, b"=", 9)) { return 0; }
                    value = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, depth + 1,true);
                }
                if (value == 0) { return 0; }
                fields = state::append(nodes, cursor, state::node_value(11,start,cursor.previous_end,name,value,fields));
                if (fields == 0) { return 0; }
                if (!state::take(stream, cursor, b",")) { break; }
            }
            if (!state::expect(stream, cursor, b"}", 7)) { return 0; }
            left = state::append(nodes, cursor, state::node_value(10,token.start,cursor.previous_end,token_index,left,fields));
            continue;
        }
        if (state::take(stream,cursor,b"?")) {
            left = state::append(nodes,cursor,state::node_value(30,token.start,cursor.previous_end,opener,left,0));
            continue;
        }
        if (word(stream,cursor.index,b"as")) {
            if (minimum > 10) { return left; }
            state::advance(stream,cursor);
            let usize target = read_type(stream,cursor,type_nodes);
            if (target == 0) { return 0; }
            left = state::append(nodes,cursor,state::node_value(23,token.start,cursor.previous_end,opener,left,target));
            continue;
        }
        if (cursor.expression_kind == 2 && word(stream,cursor.index,b"<") && types::call_ahead(stream,cursor.index)) {
            let types::Attempt result = types::parse_arguments(stream,cursor.index,cursor.types,type_nodes);
            cursor.index = result.next; cursor.previous_end = result.end; cursor.types = result.count;
            cursor.status = result.status; cursor.expected = result.expected;
            if (result.root == 0) { return 0; }
            left = state::append(nodes,cursor,state::node_value(20,token.start,cursor.previous_end,opener,left,result.root));
            if (left == 0) { return 0; }
            continue;
        }
        if (state::take(stream, cursor, b".")) {
            let usize member = cursor.index;
            if (state::current(stream, cursor).kind != 3) { cursor.expected = 1; return 0; }
            state::advance(stream, cursor);
            left = state::append(nodes, cursor, state::node_value(8,token.start,cursor.previous_end,member,left,0));
            continue;
        }
        if (state::take(stream, cursor, b"[")) {
            let usize bounds_start = state::current(stream,cursor).start;
            var usize index = 0;
            if (!word(stream,cursor.index,b"..")) {
                index = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1,true);
                if (index == 0) { return 0; }
            }
            let usize range_token = cursor.index;
            if (state::take(stream,cursor,b"..")) {
                var usize end = 0;
                if (!word(stream,cursor.index,b"]")) {
                    end = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1,true);
                    if (end == 0) { return 0; }
                }
                let usize bounds = state::append(nodes,cursor,state::node_value(27,bounds_start,cursor.previous_end,range_token,index,end));
                if (bounds == 0 || !state::expect(stream,cursor,b"]",11)) { return 0; }
                left = state::append(nodes,cursor,state::node_value(26,token.start,cursor.previous_end,opener,left,bounds));
            } else {
                if (!state::expect(stream, cursor, b"]", 11)) { return 0; }
                left = state::append(nodes, cursor, state::node_value(9,token.start,cursor.previous_end,opener,left,index));
            }
            continue;
        }
        if (state::take(stream, cursor, b"(")) {
            var usize arguments = 0;
            while (!word(stream, cursor.index, b")")) {
                var usize argument_token = cursor.index;
                let usize argument_start = state::current(stream, cursor).start;
                if (word(stream,cursor.index,b".")) {
                    let usize name = tokens::significant(stream,cursor.index + 1);
                    if (scanner::token_info(stream,name).kind == 3) {
                        let usize equals = tokens::significant(stream,name + 1);
                        if (word(stream,equals,b"=")) {
                            argument_token = name;
                            cursor.index = equals;
                            state::advance(stream,cursor);
                        }
                    }
                }
                let usize value = expression(stream, cursor, nodes, entries, statements, type_nodes, 1, depth + 1,true);
                if (value == 0) { return 0; }
                arguments = state::append(nodes, cursor, state::node_value(7,argument_start,cursor.previous_end,argument_token,value,arguments));
                if (arguments == 0) { return 0; }
                if (!state::take(stream, cursor, b",")) { break; }
            }
            if (!state::expect(stream, cursor, b")", 5)) { return 0; }
            left = state::append(nodes, cursor, state::node_value(6,token.start,cursor.previous_end,opener,left,arguments));
            continue;
        }
        let usize power = tokens::precedence(stream, cursor.index);
        if (power < minimum) { return left; }
        if (power == 3 && comparison) { return 0; }
        if (power == 3) { comparison = true; }
        let usize operator = cursor.index;
        state::advance(stream, cursor);
        let usize right = expression(stream, cursor, nodes, entries, statements, type_nodes, power + 1, depth + 1,allow_record);
        if (right == 0) { return 0; }
        left = state::append(nodes, cursor, state::node_value(5,token.start,cursor.previous_end,operator,left,right));
    }
    return 0;
}

pub fn usize read_type(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<types::Node> nodes) {
    let types::Attempt result = types::parse(stream,cursor.index,cursor.types,nodes);
    cursor.index = result.next; cursor.previous_end = result.end; cursor.types = result.count;
    cursor.status = result.status; cursor.expected = result.expected;
    return result.root;
}
pub fn bool entry(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, u32 kind) {
    if (cursor.entries >= 1024) { cursor.status = 4; return false; }
    cursor.entries = cursor.entries + 1;
    let usize id = cursor.entries;
    let usize start = state::current(stream, cursor).start;
    var usize name_start = 0; var usize name_end = 0;
    var usize type_start = 0; var usize type_end = 0; var usize root = 0; var usize type_root = 0;
    if (kind != 1) { state::advance(stream, cursor); }
    if (kind != 4) {
        if (kind == 1 || !tokens::inferred_binding(stream, cursor.index)) {
            type_start = state::current(stream, cursor).start;
            type_root = read_type(stream, cursor, type_nodes);
            if (type_root == 0) { return false; }
            type_end = cursor.previous_end;
        }
        let scanner::TokenInfo name = state::current(stream, cursor);
        if (name.kind != 3) { cursor.expected = 1; return false; }
        name_start = name.start; name_end = name.end; state::advance(stream, cursor);
        if (kind != 1 && !state::expect(stream, cursor, b"=", 9)) { return false; }
    }
    if (kind != 1) {
        if (kind != 4 || !word(stream, cursor.index, b";")) {
            root = expression(stream, cursor, nodes, entries, statements, type_nodes, .minimum = 1, .depth = 0,.allow_record = true);
            if (root == 0) { return false; }
        }
        if (!state::expect(stream, cursor, b";", 8)) { return false; }
    }
    let Entry value = Entry { .kind, .start, .end = cursor.previous_end, .name_start, .name_end, .type_start, .type_end, .type_root, .expression = root, .id };
    if (!v::push<Entry>(entries, value).is_ok) { cursor.status = 3; return false; }
    cursor.last_entry = id;
    cursor.expression = root;
    return true;
}
fn usize block_inner(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth, bool allow_tail) {
    if (depth >= 32) { cursor.status = 4; return 0; }
    let usize start = state::current(stream, cursor).start;
    if (!state::expect(stream, cursor, b"{", 6)) { return 0; }
    var usize tail = 0; var usize count = 0; var usize tail_value = 0;
    while (!word(stream, cursor.index, b"}")) {
        if (state::current(stream, cursor).kind == 1) { cursor.expected = 7; return 0; }
        let usize child_start = state::current(stream, cursor).start;
        cursor.tail = 0;
        let usize child = statement(stream, cursor, nodes, entries, statements, type_nodes, depth, allow_tail);
        if (child == 0) {
            if (allow_tail && cursor.tail != 0) { tail_value = cursor.tail; cursor.tail = 0; break; }
            return 0;
        }
        var state::Statement link = state::statement_value(7, child_start, cursor.previous_end);
        link.first = child; link.second = tail;
        tail = state::append_statement(statements, cursor, link);
        if (tail == 0) { return 0; }
        count += 1;
    }
    state::advance(stream, cursor);
    var state::Statement value = state::statement_value(1, start, cursor.previous_end);
    value.first = tail; value.second = count; value.expression = tail_value;
    return state::append_statement(statements, cursor, value);
}
fn usize statement(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth, bool allow_tail) {
    if (depth >= 32) { cursor.status = 4; return 0; }
    let usize start = state::current(stream, cursor).start;
    var state::Statement value = state::statement_value(0, start, 0);
    if (tokens::jump(stream,cursor.index)) {
        value.kind = if (word(stream, cursor.index, b"break")) { 9 } else { 10 };
        state::advance(stream, cursor);
        if (!state::expect(stream, cursor, b";", 8)) { return 0; }
    } else if (state::take(stream,cursor,b"assert")) {
        value.kind = 11;
        if (!state::expect(stream,cursor,b"(",4)) { return 0; }
        value.expression = expression(stream,cursor,nodes,entries,statements,type_nodes,.minimum = 1,.depth = 0,.allow_record = true);
        if (value.expression == 0 || !state::expect(stream,cursor,b")",5) || !state::expect(stream,cursor,b";",8)) { return 0; }
    } else if (tokens::operand_statement(stream,cursor.index) != 0) {
        value.kind = tokens::operand_statement(stream,cursor.index);
        state::advance(stream,cursor);
        value.expression = expression(stream,cursor,nodes,entries,statements,type_nodes,.minimum = 1,.depth = 0,.allow_record = true);
        if (value.expression == 0 || !state::expect(stream,cursor,b";",8)) { return 0; }
    } else if (state::take(stream,cursor,b"region")) {
        value.kind = 17;
        let usize name_token = cursor.index; let scanner::TokenInfo name = state::current(stream,cursor);
        if (name.kind != 3) { cursor.expected = 1; return 0; }
        state::advance(stream,cursor);
        value.target = state::append(nodes,cursor,state::node_value(17,name.start,name.end,name_token,0,0));
        if (value.target == 0 || !state::take(stream,cursor,b"in")) { cursor.expected = 23; return 0; }
        value.expression = expression(stream,cursor,nodes,entries,statements,type_nodes,.minimum = 1,.depth = 0,.allow_record = false);
        if (value.expression == 0) { return 0; }
        value.first = block(stream,cursor,nodes,entries,statements,type_nodes,depth + 1,false);
        if (value.first == 0) { return 0; }
    } else if (state::take(stream,cursor,b"for")) {
        let usize header_start = state::current(stream,cursor).start;
        if (!state::expect(stream,cursor,b"(",4)) { return 0; }
        if (tokens::byte_for_header(stream,cursor.index)) {
            value.kind = 18;
            state::advance(stream,cursor);
            value.second = read_type(stream,cursor,type_nodes);
            if (value.second == 0) { return 0; }
            let usize name_token = cursor.index;
            let scanner::TokenInfo name = state::current(stream,cursor);
            if (name.kind != 3) { cursor.expected = 1; return 0; }
            state::advance(stream,cursor);
            value.target = state::append(nodes,cursor,state::node_value(17,name.start,name.end,name_token,0,0));
            if (value.target == 0 || !state::expect(stream,cursor,b"in",23)) { return 0; }
            value.expression = expression(stream,cursor,nodes,entries,statements,type_nodes,.minimum = 1,.depth = 0,.allow_record = true);
            if (value.expression == 0 || !state::expect(stream,cursor,b")",5)) { return 0; }
            value.first = block(stream,cursor,nodes,entries,statements,type_nodes,depth + 1,false);
            if (value.first == 0) { return 0; }
        } else {
            value.kind = 14;
            var usize initial = 0; var usize post = 0;
            if (!state::take(stream,cursor,b";")) {
                initial = statement(stream,cursor,nodes,entries,statements,type_nodes,depth + 1,false);
                if (initial == 0) { return 0; }
                if (cursor.statement_kind != 2 && cursor.statement_kind != 4) { return 0; }
            }
            if (!word(stream,cursor.index,b";")) {
                value.expression = expression(stream,cursor,nodes,entries,statements,type_nodes,.minimum = 1,.depth = 0,.allow_record = true);
                if (value.expression == 0) { return 0; }
            }
            if (!state::expect(stream,cursor,b";",8)) { return 0; }
            if (!word(stream,cursor.index,b")")) {
                var state::Statement clause = state::statement_value(4,state::current(stream,cursor).start,0);
                clause.target = expression(stream,cursor,nodes,entries,statements,type_nodes,.minimum = 1,.depth = 0,.allow_record = true);
                if (clause.target == 0 || !tokens::assignment(stream,cursor.index)) { return 0; }
                clause.operator = cursor.index; state::advance(stream,cursor);
                clause.expression = expression(stream,cursor,nodes,entries,statements,type_nodes,.minimum = 1,.depth = 0,.allow_record = true);
                if (clause.expression == 0) { return 0; }
                clause.end = cursor.previous_end;
                post = state::append_statement(statements,cursor,clause);
                if (post == 0) { return 0; }
            }
            if (!state::expect(stream,cursor,b")",5)) { return 0; }
            var state::Statement header = state::statement_value(15,header_start,cursor.previous_end);
            header.first = initial; header.second = post;
            value.second = state::append_statement(statements,cursor,header);
            if (value.second == 0) { return 0; }
            value.first = block(stream,cursor,nodes,entries,statements,type_nodes,depth + 1,false);
            if (value.first == 0) { return 0; }
        }
    } else if (state::take(stream, cursor, b"while")) {
        value.kind = 8;
        if (!state::expect(stream, cursor, b"(", 4)) { return 0; }
        value.expression = expression(stream, cursor, nodes, entries, statements, type_nodes, .minimum = 1, .depth = 0,.allow_record = true);
        if (value.expression == 0 || !state::expect(stream, cursor, b")", 5)) { return 0; }
        value.first = block(stream, cursor, nodes, entries, statements, type_nodes, depth + 1, false);
        if (value.first == 0) { return 0; }
    } else {
        let u32 kind = if (word(stream, cursor.index, b"let")) { 2 } else if (word(stream, cursor.index, b"var")) { 3 } else if (word(stream, cursor.index, b"return")) { 4 } else { 0 };
        if (kind != 0) {
            if (!entry(stream, cursor, nodes, entries, statements, type_nodes, kind)) { return 0; }
            value.kind = if (kind == 4) { 3 } else { 2 };
            value.entry = cursor.last_entry; value.expression = cursor.expression;
            if (kind == 4 && depth == 0) { cursor.top_return = cursor.expression; }
        } else {
            let usize left = expression(stream, cursor, nodes, entries, statements, type_nodes, .minimum = 1, .depth = 0,.allow_record = true);
            if (left == 0) { return 0; }
            value.kind = 6; value.expression = left;
            if (tokens::assignment(stream, cursor.index)) {
                value.kind = 4; value.target = left; value.operator = cursor.index;
                state::advance(stream, cursor);
                value.expression = expression(stream, cursor, nodes, entries, statements, type_nodes, .minimum = 1, .depth = 0,.allow_record = true);
                if (value.expression == 0) { return 0; }
            }
            if (allow_tail && value.kind == 6 && word(stream,cursor.index,b"}")) { cursor.tail = left; return 0; }
            if (value.kind == 6 && tokens::block_kind(cursor.expression_kind) && !word(stream,cursor.index,b";")) {
                value.end = cursor.previous_end; return state::append_statement(statements,cursor,value);
            }
            if (!state::expect(stream, cursor, b";", 8)) { return 0; }
        }
    }
    value.end = cursor.previous_end;
    return state::append_statement(statements, cursor, value);
}

pub fn usize read_expression(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes) {
    return expression(stream,cursor,nodes,entries,statements,type_nodes,.minimum = 1,.depth = 0,.allow_record = true);
}
fn usize expression(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize minimum, usize depth, bool allow_record) {
    if (cursor.active_expressions >= 64) { cursor.status = 4; return 0; }
    cursor.active_expressions = cursor.active_expressions + 1;
    let usize result = expression_inner(stream,cursor,nodes,entries,statements,type_nodes,minimum,depth,allow_record);
    cursor.active_expressions = cursor.active_expressions - 1;
    return result;
}

pub fn usize block(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth, bool allow_tail) {
    if (depth >= 32) { cursor.status = 4; return 0; }
    let usize previous = cursor.block_depth; cursor.block_depth = depth + 1;
    let usize result = block_inner(stream,cursor,nodes,entries,statements,type_nodes,depth,allow_tail);
    cursor.block_depth = previous;
    return result;
}

fn usize conditional(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth) {
    if (depth >= 64) { cursor.status = 4; return 0; }
    let usize token = cursor.index; let usize start = state::current(stream,cursor).start;
    state::advance(stream,cursor);
    if (!state::expect(stream,cursor,b"(",4)) { return 0; }
    let usize condition = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1,true);
    if (condition == 0 || !state::expect(stream,cursor,b")",5)) { return 0; }
    let usize branch_start = state::current(stream,cursor).start;
    let usize consequent = block(stream,cursor,nodes,entries,statements,type_nodes,cursor.block_depth,true);
    if (consequent == 0) { return 0; }
    var usize alternative = 0;
    if (state::take(stream,cursor,b"else")) {
        if (word(stream,cursor.index,b"if")) {
            if (cursor.active_expressions >= 64) { cursor.status = 4; return 0; }
            cursor.active_expressions = cursor.active_expressions + 1;
            alternative = conditional(stream,cursor,nodes,entries,statements,type_nodes,depth + 1);
            cursor.active_expressions = cursor.active_expressions - 1;
        } else {
            let usize opener = cursor.index; let usize body_start = state::current(stream,cursor).start;
            let usize body = block(stream,cursor,nodes,entries,statements,type_nodes,cursor.block_depth,true);
            if (body == 0) { return 0; }
            alternative = state::append(nodes,cursor,state::node_value(12,body_start,cursor.previous_end,opener,body,0));
        }
        if (alternative == 0) { return 0; }
    }
    let usize branches = state::append(nodes,cursor,state::node_value(14,branch_start,cursor.previous_end,token,consequent,alternative));
    if (branches == 0) { return 0; }
    return state::append(nodes,cursor,state::node_value(13,start,cursor.previous_end,token,condition,branches));
}

fn usize switch_value(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, usize depth) {
    let usize token = cursor.index; let usize start = state::current(stream,cursor).start;
    state::advance(stream,cursor);
    if (!state::expect(stream,cursor,b"(",4)) { return 0; }
    let usize subject = expression(stream,cursor,nodes,entries,statements,type_nodes,1,depth + 1,true);
    if (subject == 0 || !state::expect(stream,cursor,b")",5) || !state::expect(stream,cursor,b"{",6)) { return 0; }
    var usize cases = 0;
    while (!word(stream,cursor.index,b"}")) {
        let usize case_token = cursor.index; let usize case_start = state::current(stream,cursor).start;
        if (!state::expect(stream,cursor,b"case",18)) { return 0; }
        let usize match = state::pattern(stream,cursor,nodes);
        if (match == 0 || !state::expect(stream,cursor,b":",17)) { return 0; }
        let usize body = block(stream,cursor,nodes,entries,statements,type_nodes,cursor.block_depth,true);
        if (body == 0) { return 0; }
        let usize arm = state::append(nodes,cursor,state::node_value(18,case_start,cursor.previous_end,case_token,match,body));
        if (arm == 0) { return 0; }
        cases = state::append(nodes,cursor,state::node_value(19,case_start,cursor.previous_end,case_token,arm,cases));
        if (cases == 0) { return 0; }
    }
    state::advance(stream,cursor);
    return state::append(nodes,cursor,state::node_value(15,start,cursor.previous_end,token,subject,cases));
}
