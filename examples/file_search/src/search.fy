module search;

pub struct Scan {
    usize lines;
    usize matches;
    usize matched;
    bool found;
    bool pending;
}

pub fn Scan start() {
    return Scan{.lines=0, .matches=0, .matched=0, .found=false, .pending=false};
}

// Carry the current line and partial match across reads.
pub fn Scan feed(Scan previous, ref u8[] chunk) {
    var Scan state = previous;
    let u8[5] needle = [101, 114, 114, 111, 114]; // error
    var usize index = 0;
    while (index < chunk.len) {
        let u8 byte = chunk[index];
        if (byte == 10) {
            state.lines += 1;
            if (state.found) { state.matches += 1; }
            state.matched = 0;
            state.found = false;
            state.pending = false;
        } else {
            state.pending = true;
            if (!state.found) {
                if (byte == needle[state.matched]) {
                    state.matched += 1;
                } else {
                    state.matched = if (byte == 101) { 1 } else { 0 };
                }
                state.found = state.matched == 5;
            }
        }
        index += 1;
    }
    return state;
}

pub fn Scan finish(Scan previous) {
    var Scan state = previous;
    if (state.pending) {
        state.lines += 1;
        if (state.found) { state.matches += 1; }
        state.pending = false;
    }
    return state;
}
