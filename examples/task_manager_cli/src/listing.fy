module listing;
import std::bytes as bytes;
import format as format;
import output as output;

// The complete store must pass format::validate before filtering or slicing.
@effects(console)
pub fn i32 render(ref str path, ref u8[] data, u32 status, bool by_title, ref mut bytes::Buffer response) {
    var usize[256] starts = [0; 256];
    var usize[256] tabs = [0; 256];
    var usize[256] ends = [0; 256];
    var usize[256] order = [0; 256];
    var usize count = 0;
    var usize start = format::end(data, 0, 10) + 1;
    while (start < data.len) {
        let usize line_end = format::end(data, start, 10);
        let usize tab = format::end(data, start, 9);
        let bool done = data[tab + 1] == 49;
        if (status == 0 || (status == 1 && !done) || (status == 2 && done)) {
            starts[count] = start;
            tabs[count] = tab;
            ends[count] = line_end;
            order[count] = count;
            count += 1;
        }
        start = line_end + 1;
    }
    // Strict-less insertion keeps ascending IDs when titles compare equal.
    if (by_title) {
        var usize i = 1;
        while (i < count) {
            let usize current = order[i];
            var usize j = i;
            while (j > 0) {
                let usize prior = order[j - 1];
                if (bytes::compare(data[tabs[current] + 3..ends[current]], data[tabs[prior] + 3..ends[prior]]) >= 0) { break; }
                order[j] = prior;
                j -= 1;
            }
            order[j] = current;
            i += 1;
        }
    }
    if (count == 0) {
        return if (format::buffer_ok(bytes::append(response, b"empty\n"))) { 0 } else { output::error(path, "cannot allocate response") };
    }
    var usize i = 0;
    while (i < count) {
        let usize row = order[i];
        let usize tab = tabs[row];
        let bool a = format::buffer_ok(bytes::append(response, data[starts[row]..tab + 1]));
        let bool b = format::buffer_ok(bytes::append(response, if (data[tab + 1] == 49) { b"done\t" } else { b"open\t" }));
        let bool c = format::buffer_ok(bytes::append(response, data[tab + 3..ends[row] + 1]));
        if (!a || !b || !c) { return output::error(path, "cannot allocate response"); }
        i += 1;
    }
    return 0;
}
