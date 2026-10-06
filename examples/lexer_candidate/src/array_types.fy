module array_types;
import syntax as syntax;
import type_state as state;

// Private canonical type IDs, disjoint from scalar and nominal IDs.
// Six scalar element types, 1..64 items. Views and nested arrays stay closed.
pub fn bool valid(u32 type) { return type >= 1033 && type <= 1542 && (type - 1024) % 8 >= 1 && (type - 1024) % 8 <= 6; }
pub fn u32 element(u32 type) { assert(valid(type)); return (type - 1024) % 8; }
pub fn usize length(u32 type) { assert(valid(type)); return ((type - 1024) / 8) as usize; }
pub fn u32 make(u32 element, usize count) { return if (element >= 1 && element <= 6 && count >= 1 && count <= 64) { 1024 + state::small_id(count) * 8 + element } else { 0 }; }
pub fn usize count(ref u8[] text) {
    if (text.len == 0) { return 0; }
    var usize base = 10; var usize i = 0; var usize n = 0;
    if (text.len >= 2 && text[0] == 48) { if (text[1] == 120 || text[1] == 88) { base = 16; i = 2; } else if (text[1] == 98 || text[1] == 66) { base = 2; i = 2; } }
    while (i < text.len) {
        let u8 b = text[i]; i += 1; if (b == 95) { continue; }
        if (!(b >= 48 && b <= 57) && !(b >= 65 && b <= 70) && !(b >= 97 && b <= 102)) { return 0; }
        let usize digit = if (b >= 48 && b <= 57) { (b - 48) as usize } else if (b >= 65 && b <= 70) { (b - 65 + 10) as usize } else { (b - 97 + 10) as usize };
        if (digit >= base || n > (64 - digit) / base) { return 0; }
        n = n * base + digit;
    }
    return if (n >= 1 && n <= 64) { n } else { 0 };
}
pub fn u32 resolve(ref syntax::Parsed p, usize root) {
    let node = syntax::type_at(p,root);
    if (node.kind != 8 || syntax::type_at(p,node.left).kind != 1 || syntax::type_at(p,node.right).kind != 9) { return 0; }
    return make(state::primitive(p,node.left),count(syntax::type_text(p,node.right)));
}
