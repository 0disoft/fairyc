module budget;

// Values are returned explicitly: the candidate does not require mutable fields.
pub struct Work { usize remaining; bool exhausted; }

pub fn Work create(usize limit) {
    return Work { .remaining = limit, .exhausted = false };
}

pub fn Work charge(Work work, usize units) {
    if (work.exhausted || units > work.remaining) {
        return Work { .remaining = work.remaining, .exhausted = true };
    }
    return Work { .remaining = work.remaining - units, .exhausted = false };
}
