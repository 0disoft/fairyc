module flags;

pub fn usize count_set(ref bool[] values) {
    var usize count = 0;
    for (let ref bool value in values) {
        if (*value) { count += 1; }
    }
    return count;
}

pub fn void fill(ref mut bool[] values, bool state) {
    for (let ref mut bool value in values) { *value = state; }
}

pub fn Option<usize> first_clear(ref bool[] values) {
    var usize index = 0;
    for (let bool value in values) {
        if (!value) { return .some(index); }
        index += 1;
    }
    return .none;
}
