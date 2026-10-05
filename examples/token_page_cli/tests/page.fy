module tests::page;
import tokens as tokens;

@test
fn void normalized_spans_and_reuse() {
    var tokens::TokenPage page = tokens::empty();
    assert(switch(tokens::scan(b" \talpha=7;", ref mut page)) { case .ok(count): { count == 4 } case .err(_): { false } });
    assert(page.tokens[0].position.start == 2 && page.tokens[3].position.end == 10);
    tokens::rebase(ref mut page);
    let ref tokens::Token[] valid = tokens::valid(ref page);
    assert(valid.len == 4);
    assert(valid[0].position.start == 0 && valid[0].position.end == 5);
    assert(valid[1].position.start == 5 && valid[1].position.end == 6);
    assert(valid[2].position.start == 6 && valid[2].position.end == 7 && valid[2].value == 7);
    assert(valid[3].position.start == 7 && valid[3].position.end == 8);
    assert(tokens::first_end(valid) == 4);
    let tokens::Stats stats = switch(tokens::analyze(valid)) { case .ok(value): { value } case .err(_): { assert(false); tokens::Stats { .identifiers = 0, .numbers = 0, .sum = 0 } } };
    assert(stats.identifiers == 1 && stats.numbers == 1 && stats.sum == 7);
    assert(tokens::scan(b"", ref mut page).is_ok);
    let ref tokens::Token[] empty = tokens::valid(ref page);
    assert(empty.is_empty);
    assert(tokens::scan(b";", ref mut page).is_ok);
    let ref tokens::Token[] fresh = tokens::valid(ref page);
    assert(fresh.len == 1 && fresh[0].kind == 5 && fresh[0].position.start == 0 && fresh[0].position.end == 1);
}
