module tests::arithmetic;
import evaluator as evaluator;
import tokens as tokens;

@test
fn void precedence_and_scanner_modes() {
    let ref u8[] source = b"  a=2+3*4; b=(2+3)*4; a=(a+b)*2;";
    var tokens::TokenPage page = tokens::empty();
    var evaluator::Environment environment = evaluator::empty();
    assert(tokens::scan_eval(source, ref mut page).is_ok);
    assert(evaluator::evaluate(source, tokens::valid(ref page), ref mut environment).is_ok);
    let ref evaluator::Binding[] bindings = evaluator::valid(ref environment);
    assert(bindings.len == 2 && bindings[0].value == 68 && bindings[1].value == 20);
    assert(bindings[0].name.start == 2 && bindings[0].name.end == 3);
    assert(!tokens::scan(b"x=2*3;", ref mut page).is_ok);
    assert(!tokens::scan(b"x=(1);", ref mut page).is_ok);
    assert(tokens::scan_eval(b"x=0*(9223372036854775807+1);", ref mut page).is_ok);
    assert(switch(evaluator::evaluate(b"x=0*(9223372036854775807+1);", tokens::valid(ref page), ref mut environment)) {
        case .ok(_): { false }
        case .err(problem): { problem.code == 10 }
    });
    assert(environment.used == 0);
    assert(tokens::scan_eval(b"x=2*(3+);", ref mut page).is_ok);
    assert(switch(evaluator::evaluate(b"x=2*(3+);", tokens::valid(ref page), ref mut environment)) {
        case .ok(_): { false }
        case .err(problem): { problem.code == 8 }
    });
    assert(tokens::scan_eval(b"x=(1+2)*3;", ref mut page).is_ok);
    assert(evaluator::evaluate(b"x=(1+2)*3;", tokens::valid(ref page), ref mut environment).is_ok);
    assert(environment.used == 1 && environment.bindings[0].value == 9);
}
