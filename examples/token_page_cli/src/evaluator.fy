module evaluator;
import tokens as tokens;

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

/// Consume unrebased scanner tokens. Errors: 8 syntax, 9 undefined name,
/// 10 checked addition overflow, 11 variable limit. Reset bindings on every call.
pub fn Result<usize, u32> evaluate(ref u8[] source, ref tokens::Token[] stream, ref mut Environment environment) {
    environment.used = 0;
    var usize cursor = 0;
    while (cursor < stream.len) {
        if (stream[cursor].kind != 1) { return .err(8); }
        let tokens::Span name = stream[cursor].position;
        cursor += 1;
        if (cursor == stream.len || stream[cursor].kind != 4) { return .err(8); }
        cursor += 1;
        var i64 value = 0;
        while (true) {
            if (cursor == stream.len) { return .err(8); }
            let tokens::Token token = stream[cursor];
            var i64 part = 0;
            if (token.kind == 2) { part = token.value; }
            else if (token.kind == 1) {
                let usize slot = find(source, token.position, environment);
                if (slot == 16) { return .err(9); }
                part = environment.bindings[slot].value;
            } else { return .err(8); }
            if (part > 9223372036854775807 - value) { return .err(10); }
            value += part;
            cursor += 1;
            if (cursor == stream.len) { return .err(8); }
            let u32 next = stream[cursor].kind;
            cursor += 1;
            if (next == 5) { break; }
            if (next != 3) { return .err(8); }
        }
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
