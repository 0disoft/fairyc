module tests::diagnostics;
import diagnostics as diagnostics;
import tokens as tokens;
import evaluator as evaluator;

fn bool fails_at(ref u8[] source, u32 code, usize offset) {
    var tokens::TokenPage page = tokens::empty();
    discard switch(tokens::scan_eval(source, ref mut page)) {
        case .ok(count): { count }
        case .err(actual): { return actual == code && page.error_offset == offset; }
    };
    var evaluator::Environment environment = evaluator::empty();
    return switch(evaluator::evaluate(source, tokens::valid(ref page), ref mut environment)) {
        case .ok(_): { false }
        case .err(problem): { problem.code == code && problem.offset == offset }
    };
}

@test
fn void error_offsets_and_byte_locations() {
    assert(fails_at(b"price=10;\ntotal=price+;", 8, 22));
    assert(fails_at(b"x=missing;", 9, 2));
    assert(fails_at(b"x=9223372036854775807+1+0;", 10, 21));
    assert(fails_at(b"x=3037000500*3037000500;", 10, 12));
    assert(fails_at(b"x=9223372036854775808;", 2, 20));
    assert(fails_at(b"x=1;\r\n\t@", 1, 7));
    assert(fails_at(b"x=(1+2", 8, 6));
    assert(fails_at(b"x", 8, 1));
    let diagnostics::Location lf = diagnostics::locate(b"a\nb", 2);
    assert(lf.line == 2 && lf.column == 1);
    let diagnostics::Location crlf = diagnostics::locate(b"a\r\n\tb", 4);
    assert(crlf.line == 2 && crlf.column == 2);
    let diagnostics::Location cr = diagnostics::locate(b"a\rb", 3);
    assert(cr.line == 2 && cr.column == 2);
    let diagnostics::Location eof = diagnostics::locate(b"a\r\n", 3);
    assert(eof.line == 2 && eof.column == 1);
    let diagnostics::Location empty = diagnostics::locate(b"", 0);
    assert(empty.line == 1 && empty.column == 1);
    var tokens::TokenPage page = tokens::empty();
    assert(!tokens::scan_eval(b"x=1;@", ref mut page).is_ok && page.error_offset == 4);
    assert(tokens::scan_eval(b"x=1;", ref mut page).is_ok && page.error_offset == 0);
}
