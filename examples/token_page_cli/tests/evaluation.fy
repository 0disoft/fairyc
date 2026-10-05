module tests::evaluation;
import evaluator as evaluator;
import tokens as tokens;

@test
fn void bindings_and_reuse() {
    let ref u8[] source = b"  x=4000000000; x=x+7; total=x+3;";
    var tokens::TokenPage page = tokens::empty();
    var evaluator::Environment environment = evaluator::empty();
    assert(tokens::scan(source, ref mut page).is_ok);
    assert(switch(evaluator::evaluate(source, tokens::valid(ref page), ref mut environment)) {
        case .ok(count): { count == 2 }
        case .err(_): { false }
    });
    let ref evaluator::Binding[] bindings = evaluator::valid(ref environment);
    assert(bindings.len == 2 && bindings[0].value == 4000000007 && bindings[1].value == 4000000010);
    assert(bindings[0].name.start == 2 && bindings[0].name.end == 3);
    assert(tokens::scan(b"x=1; x=missing;", ref mut page).is_ok);
    assert(switch(evaluator::evaluate(b"x=1; x=missing;", tokens::valid(ref page), ref mut environment)) {
        case .ok(_): { false }
        case .err(problem): { problem.code == 9 }
    });
    // A failed RHS does not overwrite the earlier binding; no CLI output is emitted.
    assert(environment.used == 1 && environment.bindings[0].value == 1);
    assert(tokens::scan(b"", ref mut page).is_ok);
    assert(evaluator::evaluate(b"", tokens::valid(ref page), ref mut environment).is_ok);
    assert(environment.used == 0);
    assert(tokens::scan(b"x=x+1;", ref mut page).is_ok);
    assert(switch(evaluator::evaluate(b"x=x+1;", tokens::valid(ref page), ref mut environment)) {
        case .ok(_): { false }
        case .err(problem): { problem.code == 9 }
    });
}
