module tokens;

pub struct Span { usize start; usize end; }
pub struct Token { u32 kind; i64 value; Span position; }

pub fn Option<ref Token> find(ref Token[] tokens, u32 kind) {
    for (let ref Token token in tokens) {
        if (token.kind == kind) { return .some(token); }
    }
    return .none;
}
