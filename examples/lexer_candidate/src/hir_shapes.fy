module hir_shapes;
import array_types as arrays;
import std::vec as v;

// Source-independent nominal layout facts. Field IDs are one-based; shape type
// IDs preserve 256 + declaration index. Every name/span indexes owned HIR bytes.
pub struct Shape { u32 type; usize binding; usize field_start; usize field_count; bool public; }
pub struct Field { u32 owner; u32 type; usize start; usize end; }
pub resource Table { v::Vec<Shape> shapes; v::Vec<Field> fields; cleanup { } }
pub fn Table empty() { return Table { .shapes = v::empty<Shape>(), .fields = v::empty<Field>() }; }
pub fn usize count(ref Table p) { return v::len<Shape>(ref p.shapes); }
pub fn usize field_count(ref Table p) { return v::len<Field>(ref p.fields); }
pub fn Shape at(ref Table p, usize i) { assert(i < count(p)); return *v::at<Shape>(ref p.shapes,i); }
pub fn Field field_at(ref Table p, usize id) { assert(id > 0 && id <= field_count(p)); return *v::at<Field>(ref p.fields,id - 1); }
pub fn usize find(ref Table p, u32 type) { var usize i = 0; while (i < count(p)) { if (at(p,i).type == type) { return i + 1; } i += 1; } return 0; }
pub fn bool valid_type(ref Table p, u32 type, bool allow_void) { return (type >= 1 && type <= if (allow_void) { 7 } else { 6 }) || arrays::valid(type) || (type >= 256 && find(p,type) != 0); }
fn bool visit(ref Table p, usize i, ref mut u8[] colors, usize depth) {
    if (colors[i] == 2) { return true; } if (colors[i] == 1 || depth >= 64) { return false; }
    colors[i] = 1; let Shape s = at(p,i); var usize j = 0;
    while (j < s.field_count) {
        let Field f = field_at(p,s.field_start + j + 1);
        if (f.type >= 256 && !visit(p,find(p,f.type) - 1,colors,depth + 1)) { return false; } j += 1;
    }
    colors[i] = 2; return true;
}
// Validation precedes DFS and all subtraction/indexing. A malformed/cyclic
// table is never published, including tables assembled by mutation tests.
pub fn bool valid(ref Table p, usize source_bytes, usize bindings) {
    if (count(p) > 512 || field_count(p) > 1024) { return false; }
    var usize offset = 0; var usize i = 0;
    while (i < count(p)) {
        let Shape s = at(p,i);
        if (s.type < 256 || s.type > 767 || s.binding == 0 || s.binding > bindings || s.field_start != offset || s.field_count > field_count(p) - offset) { return false; }
        var usize j = 0; while (j < i) { if (at(p,j).type == s.type || at(p,j).binding == s.binding) { return false; } j += 1; }
        j = 0; while (j < s.field_count) {
            let Field f = field_at(p,offset + j + 1);
            if (f.owner != s.type || arrays::valid(f.type) || !valid_type(p,f.type,false) || f.start >= f.end || f.end > source_bytes) { return false; } j += 1;
        }
        offset += s.field_count; i += 1;
    }
    if (offset != field_count(p)) { return false; }
    var u8[512] colors = [0;512]; i = 0;
    while (i < count(p)) { if (!visit(p,i,ref mut colors[..],0)) { return false; } i += 1; } return true;
}
pub fn Table assemble(v::Vec<Shape> shapes, v::Vec<Field> fields) { return Table { .shapes, .fields }; }
pub fn Result<Table,u32> copy(ref Table p) {
    var v::Vec<Shape> records = v::empty<Shape>(); var v::Vec<Field> fields = v::empty<Field>();
    var usize i = 0;
    while (i < count(p)) { if (!v::push<Shape>(ref mut records,at(p,i)).is_ok) { return .err(3); } i += 1; }
    i = 1; while (i <= field_count(p)) { if (!v::push<Field>(ref mut fields,field_at(p,i)).is_ok) { return .err(3); } i += 1; }
    return .ok(assemble(records,fields));
}
