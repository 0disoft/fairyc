module mir_ownership_snapshot;
import mir_state as state;
import std::bytes as bytes;

// Keep branch snapshots off recursive native frames. Allocation failure is a
// lowering failure; no partially restored ownership state may be published.
pub fn bytes::Buffer capture(ref mut state::Context c, usize start, usize end) {
    var bytes::Buffer saved = bytes::empty();
    if (!bytes::append(ref mut saved,ref c.live[..]).is_ok || !bytes::append(ref mut saved,ref c.pending[..]).is_ok) {
        discard state::fail(c,3,start,end);
    }
    return saved;
}
pub fn void restore(ref mut state::Context c, ref bytes::Buffer saved) {
    let ref u8[] values = bytes::view(saved); assert(values.len == 5249);
    var usize i = 0; while (i < 1153) { c.live[i] = values[i]; i += 1; }
    i = 0; while (i < 4096) { c.pending[i] = values[1153+i]; i += 1; }
}
