module budget;

// Values are returned explicitly: the candidate does not require mutable fields.
pub struct Work { usize used; usize limit; bool exhausted; }

pub fn Work create(usize limit) {
    return Work { .used = 0, .limit = limit, .exhausted = false };
}

pub fn Work charge(Work work, usize units) {
    if (work.exhausted || work.used > work.limit || units > work.limit - work.used) {
        return Work { .used = work.used, .limit = work.limit, .exhausted = true };
    }
    return Work { .used = work.used + units, .limit = work.limit, .exhausted = false };
}
