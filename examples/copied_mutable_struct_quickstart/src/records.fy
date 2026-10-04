module records;

pub struct Span { usize start; usize end; }
pub struct Record { i64 value; Span position; }

pub fn i64 clamp(i64 value) {
    if (value < 0) { return 0; }
    if (value > 10) { return 10; }
    return value;
}
