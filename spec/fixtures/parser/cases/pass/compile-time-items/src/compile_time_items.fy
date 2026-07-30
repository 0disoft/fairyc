module compile_time_items;

const i32 ANSWER = 40 + 2;
static_assert(ANSWER == 42);

fn i32 static_assert() {
    return ANSWER;
}
