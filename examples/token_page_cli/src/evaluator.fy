module evaluator;
import tokens as tokens;
import arithmetic as arithmetic;

pub struct Binding { tokens::Span name; i64 value; }
pub struct Environment { Binding[16] bindings; usize used; }

pub fn Environment empty() {
    return Environment {
        .bindings = [Binding { .name = tokens::Span { .start = 0, .end = 0 }, .value = 0 }; 16],
        .used = 0
    };
}
fn bool same_name(ref u8[] source, tokens::Span left, tokens::Span right) {
    let usize length = left.end - left.start;
    if (length != right.end - right.start) { return false; }
    var usize index = 0;
    while (index < length) {
        if (source[left.start + index] != source[right.start + index]) { return false; }
        index += 1;
    }
    return true;
}
fn usize find(ref u8[] source, tokens::Span name, ref Environment environment) {
    let usize used = environment.used;
    var usize index = 0;
    for (let ref Binding binding in environment.bindings[..used]) {
        if (same_name(source, name, binding.name)) { return index; }
        index += 1;
    }
    return 16;
}

struct Expression { i64 value; usize consumed; }
fn Result<Expression, u32> expression(ref u8[] source, ref tokens::Token[] stream, ref Environment environment) {
    var arithmetic::Stack stack = arithmetic::empty();
    var usize cursor = 0;
    var usize depth = 0;
    var bool need_value = true;
    while (cursor < stream.len) {
        let tokens::Token token = stream[cursor];
        if (need_value) {
            if (token.kind == 7) {
                if (depth == 16) { return .err(12); }
                if (!arithmetic::push_operator(ref mut stack, 7)) { return .err(8); }
                depth += 1;
                cursor += 1;
                continue;
            }
            var i64 value = 0;
            if (token.kind == 2) { value = token.value; }
            else if (token.kind == 1) {
                let usize slot = find(source, token.position, environment);
                if (slot == 16) { return .err(9); }
                value = environment.bindings[slot].value;
            } else { return .err(8); }
            if (!arithmetic::push_value(ref mut stack, value)) { return .err(8); }
            need_value = false;
        } else if (token.kind == 3 || token.kind == 6) {
            discard switch(arithmetic::reduce_for(ref mut stack, token.kind)) {
                case .ok(value): { value } case .err(code): { return .err(code); }
            };
            if (!arithmetic::push_operator(ref mut stack, token.kind)) { return .err(8); }
            need_value = true;
        } else if (token.kind == 8) {
            if (depth == 0) { return .err(8); }
            discard switch(arithmetic::reduce_for(ref mut stack, 0)) {
                case .ok(value): { value } case .err(code): { return .err(code); }
            };
            if (arithmetic::top(ref stack) != 7) { return .err(8); }
            stack.operator_count -= 1;
            depth -= 1;
        } else if (token.kind == 5) {
            if (depth != 0) { return .err(8); }
            discard switch(arithmetic::reduce_for(ref mut stack, 0)) {
                case .ok(value): { value } case .err(code): { return .err(code); }
            };
            if (stack.value_count != 1) { return .err(8); }
            return .ok(Expression { .value = stack.values[0], .consumed = cursor + 1 });
        } else { return .err(8); }
        cursor += 1;
    }
    return .err(8);
}

/// Consume unrebased evaluation scanner tokens. Errors: 8 syntax, 9 undefined name,
/// 10 arithmetic overflow, 11 variable limit, 12 parenthesis limit. Reset on each call.
pub fn Result<usize, u32> evaluate(ref u8[] source, ref tokens::Token[] stream, ref mut Environment environment) {
    environment.used = 0;
    var usize cursor = 0;
    while (cursor < stream.len) {
        if (stream[cursor].kind != 1) { return .err(8); }
        let tokens::Span name = stream[cursor].position;
        cursor += 1;
        if (cursor == stream.len || stream[cursor].kind != 4) { return .err(8); }
        cursor += 1;
        let Expression parsed = switch(expression(source, stream[cursor..], environment)) {
            case .ok(value): { value } case .err(code): { return .err(code); }
        };
        let i64 value = parsed.value;
        cursor += parsed.consumed;
        // Finish the RHS before publication: self-reference sees the prior value.
        let usize slot = find(source, name, environment);
        if (slot == 16) {
            let usize used = environment.used;
            if (used == 16) { return .err(11); }
            environment.bindings[used] = Binding { .name, .value };
            environment.used = used + 1;
        } else { environment.bindings[slot].value = value; }
    }
    return .ok(environment.used);
}

pub fn ref Binding[] valid(ref Environment environment) {
    let usize used = environment.used;
    return ref environment.bindings[..used];
}
