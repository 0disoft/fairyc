module budget;

// Callers stop on a failed charge; no exhausted state must cross a return.
pub struct Work { usize remaining; }

pub fn Work create(usize limit) {
    return Work { .remaining = limit };
}

pub fn bool charge(ref mut Work work, usize units) {
    if (units > work.remaining) { return false; }
    work.remaining = work.remaining - units;
    return true;
}
