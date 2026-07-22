module tests::answer;

import answer as answer;

@test
fn void dependency_answer_is_42() {
    assert(answer::value() == 42);
}
