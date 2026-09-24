module budget;

// Values are returned explicitly: the candidate does not require mutable fields.
pub struct Work { usize remaining; bool exhausted; }

pub fn Work create(usize limit) {
    return Work { .remaining = limit, .exhausted = false };
}

pub fn void charge(ref mut Work work, usize units) {
    if (work.exhausted || units > work.remaining) {
        work.exhausted = true;
        return;
    }
    work.remaining = work.remaining - units;
}
